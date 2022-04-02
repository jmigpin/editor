package godebug

// Needs to run when there are changes on the ./debug pkg
//go:generate go run debugpack/debugpack.go

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

type Cmd struct {
	Client *Client
	Dir    string
	Stdout io.Writer
	Stderr io.Writer

	tmpDir       string
	tmpBuiltFile string // file built and exec'd

	//fixedTmpDir struct { // re-use tmp dir to allow caching
	//	on  bool
	//	pid int
	//}

	env        []string // set at start
	annset     *AnnotatorSet
	gopathMode bool

	NoPreBuild bool // useful for tests

	start struct {
		network   string
		address   string
		cancel    context.CancelFunc
		serverCmd *osutil.Cmd // the annotated program
	}

	flags struct {
		mode struct {
			run     bool
			test    bool
			build   bool
			connect bool
		}
		verbose     bool
		filenames   []string // compile
		work        bool
		output      string   // ex: -o filename (build mode)
		toolExec    string   // ex: "wine" will run "wine args..."
		dirs        []string // annotate
		files       []string // annotate
		address     string   // build/connect
		env         []string // build
		syncSend    bool
		otherArgs   []string
		testRunArgs []string
	}
}

func NewCmd() *Cmd {
	cmd := &Cmd{
		annset: NewAnnotatorSet(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	//cmd.fixedTmpDir.on = true
	//cmd.fixedTmpDir.pid = 2
	return cmd
}

//------------

func (cmd *Cmd) Error(err error) {
	cmd.Printf("error: %v\n", err)
}
func (cmd *Cmd) Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(cmd.Stdout, "# "+format, a...)
}
func (cmd *Cmd) Vprintf(format string, a ...interface{}) (int, error) {
	if cmd.flags.verbose {
		return cmd.Printf(format, a...)
	}
	return 0, nil
}

//------------

func (cmd *Cmd) Start(ctx context.Context, args []string) (done bool, _ error) {
	// use absolute dir
	dir0, err := filepath.Abs(cmd.Dir)
	if err != nil {
		return true, err
	}
	cmd.Dir = dir0

	if err := cmd.parseArgs(args); err != nil {
		return true, err
	}
	if err := cmd.start2(ctx); err != nil {
		return true, err
	}
	done = cmd.flags.mode.build // just building, wait() should not be called
	return done, nil
}

func (cmd *Cmd) start2(ctx context.Context) error {
	// setup environment
	cmd.env = goutil.FullEnv()
	cmd.env = osutil.SetEnvs(cmd.env, cmd.flags.env)

	modsMode, err := cmd.detectModulesMode(cmd.env)
	if err != nil {
		return err
	}
	cmd.gopathMode = !modsMode
	cmd.Vprintf("gopathMode=%v\n", cmd.gopathMode)

	// depends on: gopathMode
	tmpDir, err := cmd.setupTmpDir()
	if err != nil {
		return err
	}
	cmd.tmpDir = tmpDir

	// depends on: gopathMode, tmpDir
	cmd.env = cmd.setGoPathEnv(cmd.env)

	// print tmp dir if work flag is present
	if cmd.flags.work {
		cmd.Printf("work: %v\n", cmd.tmpDir)
	}

	if err := cmd.setupNetworkAddress(); err != nil {
		return err
	}

	m := &cmd.flags.mode
	if m.run || m.test || m.build {
		if err := cmd.initAndAnnotate(ctx); err != nil {
			return err
		}
	}

	// inform the address used in the binary
	if m.build {
		cmd.Printf("build: %v (builtin address: %v, %v)\n", cmd.tmpBuiltFile, cmd.start.network, cmd.start.address)
	}

	if m.run || m.test || m.connect {
		return cmd.startServerClient(ctx)
	}

	return nil
}

//------------

func (cmd *Cmd) initAndAnnotate(ctx context.Context) error {
	// "files" not in cmd.* to allow early GC
	files := NewFiles(cmd.annset.FSet, cmd.Dir, cmd.flags.mode.test, cmd.gopathMode, cmd.Stderr)

	files.Add(cmd.flags.files...)
	files.Add(cmd.flags.dirs...)

	// make flag filenames absolute based on files.dir
	for i, f := range cmd.flags.filenames {
		cmd.flags.filenames[i] = files.absFilename(f)
	}

	// pre-build without annotations for better errors with original filenames (result ignored)
	// done in sync (first pre-build, then annotate) in order to have the "go build" update go.mod's if necessary.
	if !cmd.NoPreBuild {
		if err := cmd.preBuild(ctx); err != nil {
			return err
		}
	}

	return cmd.initAndAnnotate2(ctx, files)
}

func (cmd *Cmd) initAndAnnotate2(ctx context.Context, files *Files) error {
	err := files.Do(ctx, cmd.flags.filenames, cmd.env)
	if err != nil {
		return err
	}

	if cmd.flags.verbose {
		files.verbose(cmd)
	}

	//if err := cmd.updateFixedTmpDir(files); err != nil {
	//	return err
	//}

	if err := cmd.annotateFiles(ctx, files); err != nil {
		return err
	}

	if err := cmd.insertDebugExitInMain(ctx, files); err != nil {
		return err
	}

	// gets config content from annotations
	if err := files.setDebugConfigContent(cmd); err != nil {
		return err
	}

	if !cmd.gopathMode {
		if err := setupGoMods(ctx, cmd, files); err != nil {
			return err
		}
	}

	if err := files.writeToTmpDir(cmd); err != nil {
		return err
	}

	//if cmd.flags.verbose {
	//	files.verbose(cmd)
	//}

	return cmd.doBuild(ctx)
}

// os.Rename() doesn't work across partitions and tmpfs is often used for /tmp
// This is an alternative.
func moveFile(sourcePath, destPath string) error {

	// open files for copy
    inputFile, err := os.Open(sourcePath)
    if err != nil {
        return fmt.Errorf("Couldn't open source file: %s", err)
    }
    outputFile, err := os.Create(destPath)
    if err != nil {
        inputFile.Close()
        return fmt.Errorf("Couldn't open dest file: %s", err)
    }
    defer outputFile.Close()
    
    // do file copy
    _, err = io.Copy(outputFile, inputFile)
    if err != nil {
    	inputFile.Close()  // close early if we can't continue
        return fmt.Errorf("Writing to output file failed: %s", err)
    }
    
    // read file permissions
    stat, err := inputFile.Stat()
    if err != nil {
    	return fmt.Errorf("Unable to read input file permissions: %s", err)
    }
    inputFile.Close()
    
    // write file permissions
    err = os.Chmod(destPath, stat.Mode())
    if err != nil {
    	return fmt.Errorf("Unable to set output file permissions: %s", err)
    }
    
    // The copy was successful, so now delete the original file
    err = os.Remove(sourcePath)
    if err != nil {
        return fmt.Errorf("Failed removing original file: %s", err)
    }
    return nil
}

func (cmd *Cmd) doBuild(ctx context.Context) error {
	dirAtTmp := cmd.tmpDirBasedFilename(cmd.Dir)

	//// ensure it exists
	//if err := iout.MkdirAll(dirAtTmp); err != nil {
	//	return err
	//}

	filenamesAtTmp := []string{}
	for _, f := range cmd.flags.filenames {
		u := cmd.tmpDirBasedFilename(f)
		filenamesAtTmp = append(filenamesAtTmp, u)
	}

	// build
	filenameOutAtTmp, err := cmd.runBuildCmd(ctx, dirAtTmp, filenamesAtTmp, false)
	if err != nil {
		return err
	}

	// move filename to working dir
	filenameOut := filepath.Join(cmd.Dir, filepath.Base(filenameOutAtTmp))
	// move filename to output option
	if cmd.flags.output != "" {
		o := cmd.flags.output
		if !filepath.IsAbs(o) {
			o = filepath.Join(cmd.Dir, o)
		}
		filenameOut = o
	}
	
	// move temporary executable out of /tmp
	if err := moveFile(filenameOutAtTmp, filenameOut); err != nil {
		return err
	}

	// keep moved filename that will run in working dir for later cleanup
	cmd.tmpBuiltFile = filenameOut

	return nil
}

func (cmd *Cmd) preBuild(ctx context.Context) error {
	filenameOut, err := cmd.runBuildCmd(ctx, cmd.Dir, cmd.flags.filenames, true)
	defer os.Remove(filenameOut) // ensure removal even on error
	if err != nil {
		return fmt.Errorf("preBuild: %w", err)
	}
	return nil
}

//------------

func (cmd *Cmd) startServerClient(ctx context.Context) error {
	// server/client context to cancel the other when one of them ends
	ctx, cancel := context.WithCancel(ctx)
	cmd.start.cancel = cancel

	if err := cmd.startServerClient2(ctx); err != nil {
		cmd.start.cancel()
		_ = cmd.Wait()
		return err
	}
	return nil
}

func (cmd *Cmd) startServerClient2(ctx context.Context) error {
	args := []string{cmd.tmpBuiltFile}
	args = append(args, cmd.flags.testRunArgs...)
	args = append(args, cmd.flags.otherArgs...)

	// toolexec
	if cmd.flags.toolExec != "" {
		args = append([]string{cmd.flags.toolExec}, args...)
	}

	// start server (run the annotated program)
	if !cmd.flags.mode.connect {
		cb := func(c *osutil.Cmd) {
			cmd.Printf("pid %d\n", c.Cmd.Process.Pid)
		}
		c, err := cmd.startCmd(ctx, cmd.Dir, args, nil, cb)
		if err != nil {
			return err
		}
		cmd.start.serverCmd = c
	}

	// start client (blocks until connected)
	client, err := NewClient(ctx, cmd.start.network, cmd.start.address)
	if err != nil {
		return err
	}
	cmd.Client = client
	return nil
}

func (cmd *Cmd) Wait() error {
	defer cmd.start.cancel() // ensure resources are cleared
	var err error
	if cmd.start.serverCmd != nil { // might be nil: connect mode
		err = cmd.start.serverCmd.Wait()
	}
	if cmd.Client != nil { // might be nil if server failed to start
		cmd.Client.Wait()
	}
	return err
}

//------------

func (cmd *Cmd) RequestFileSetPositions() error {
	msg := &debug.ReqFilesDataMsg{}
	encoded, err := debug.EncodeMessage(msg)
	if err != nil {
		return err
	}
	_, err = cmd.Client.Conn.Write(encoded)
	return err
}

func (cmd *Cmd) RequestStart() error {
	msg := &debug.ReqStartMsg{}
	encoded, err := debug.EncodeMessage(msg)
	if err != nil {
		return err
	}
	_, err = cmd.Client.Conn.Write(encoded)
	return err
}

//------------

func (cmd *Cmd) tmpDirBasedFilename(filename string) string {
	// remove volume name
	v := filepath.VolumeName(filename)
	if len(v) > 0 {
		filename = filename[len(v):]
	}

	if cmd.gopathMode {
		// trim filename when inside a src dir
		rhs := trimAtFirstSrcDir(filename)
		return filepath.Join(cmd.tmpDir, "src", rhs)
	}

	// just replicate on tmp dir
	return filepath.Join(cmd.tmpDir, filename)
}

//------------

func (cmd *Cmd) setGoPathEnv(env []string) []string {
	// after cmd.flags.env such that this result won't be overriden

	s := cmd.fullGoPathStr(env)
	return osutil.SetEnv(env, "GOPATH", s)
}

func (cmd *Cmd) fullGoPathStr(env []string) string {
	u := []string{} // first has priority, use new slice

	// add tmpdir for priority to the annotated files
	if cmd.gopathMode {
		u = append(u, cmd.tmpDir)
	}

	if s := osutil.GetEnv(cmd.flags.env, "GOPATH"); s != "" {
		u = append(u, s)
	}

	// always include default gopath last (includes entry that might not be defined anywhere, needs to be set)
	u = append(u, goutil.GetGoPath(env)...)

	return goutil.JoinPathLists(u...)
}

//------------

func (cmd *Cmd) detectModulesMode(env []string) (bool, error) {
	v := osutil.GetEnv(env, "GO111MODULE")
	switch v {
	case "on":
		return true, nil
	case "off":
		return false, nil
	case "auto":
		return cmd.detectGoMod(), nil
	default:
		v, err := goutil.GoVersion()
		if err != nil {
			return false, err
		}
		// < go1.16, modules mode if go.mod present
		if parseutil.VersionLessThan(v, "1.16") {
			return cmd.detectGoMod(), nil
		}
		// >= go1.16, modules mode by default
		return true, nil
	}
}

func (cmd *Cmd) detectGoMod() bool {
	_, ok := goutil.FindGoMod(cmd.Dir)
	return ok
}

//------------

func (cmd *Cmd) Cleanup() {
	// cleanup unix socket in case of bad stop
	if cmd.start.network == "unix" {
		if err := os.Remove(cmd.start.address); err != nil {
			if !os.IsNotExist(err) {
				cmd.Printf("cleanup err: %v\n", err)
			}
		}
	}

	if cmd.flags.work {
		// don't cleanup work dir

		//} else if cmd.fixedTmpDir.on {
		// don't cleanup work dir
	} else if cmd.tmpDir != "" {
		if err := os.RemoveAll(cmd.tmpDir); err != nil {
			cmd.Printf("cleanup err: %v\n", err)
		}
	}

	if cmd.tmpBuiltFile != "" && !cmd.flags.mode.build {
		if err := os.Remove(cmd.tmpBuiltFile); err != nil {
			if !os.IsNotExist(err) {
				cmd.Printf("cleanup err: %v\n", err)
			}
		}
	}
}

//------------

func (cmd *Cmd) runBuildCmd(ctx context.Context, dir string, filenames []string, preBuild bool) (string, error) {
	filenameOut := filepath.Join(dir, cmd.baseFilenameOut())

	preFilesArgs := godebugBuildFlags(cmd.env)
	if cmd.flags.mode.test || cmd.flags.mode.build {
		preFilesArgs = append(preFilesArgs, cmd.flags.otherArgs...)
	}

	args := []string{}
	if cmd.flags.mode.test {
		args = []string{
			osutil.ExecName("go"), "test",
			"-c", // compile binary but don't run
			"-o", filenameOut,
		}
		args = append(args, preFilesArgs...)
		args = append(args, filenames...)
	} else {
		args = []string{
			osutil.ExecName("go"), "build",
			"-o", filenameOut,
		}
		if preBuild {
			// TODO: faster dummy pre-builts
			// TODO: test toolexec on other platforms
			//args = append(args, "-toolexec /bin/true")  // don't run asm?
		}
		args = append(args, preFilesArgs...)
		args = append(args, filenames...)
	}

	//cmd.Vprintf("runBuildCmd:  %v\n", args)

	err := cmd.runCmd(ctx, dir, args, cmd.env)
	if err != nil {
		err = fmt.Errorf("runBuildCmd: %w, args=%v, dir=%v", err, args, dir)
	}
	return filenameOut, err
}

//------------

func (cmd *Cmd) runCmd(ctx context.Context, dir string, args, env []string) error {
	c, err := cmd.startCmd(ctx, dir, args, env, nil)
	if err != nil {
		return err
	}
	return c.Wait()
}

func (cmd *Cmd) startCmd(ctx context.Context, dir string, args, env []string, cb func(*osutil.Cmd)) (*osutil.Cmd, error) {
	cargs := osutil.ShellRunArgs(args...)
	c := osutil.NewCmd(ctx, cargs...)
	c.Env = env
	c.Dir = dir
	if err := c.SetupStdio(nil, cmd.Stdout, cmd.Stderr); err != nil {
		return nil, err
	}
	if cb != nil {
		c.PreOutputCallback = func() { cb(c) }
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	return c, nil
}

//------------

func (cmd *Cmd) insertDebugExitInMain(ctx context.Context, files *Files) error {
	count := 0
	for _, f := range files.srcFiles {
		if f.mainFuncDecl != nil {
			astFile, err := files.fullAstFile(f.filename)
			if err != nil {
				return err
			}
			cmd.annset.InsertDebugExitInMain(f.mainFuncDecl, astFile, f)
			count++
		}
	}
	if count == 0 {
		return errors.New("unable to insertDebugExitInMain")
	}
	return nil
}

//------------

func (cmd *Cmd) annotateFiles(ctx context.Context, files *Files) error {
	fa := files.filesToAnnotate()
	if len(fa) == 0 {
		return errors.New("no files selected to annotate")
	}

	var wg sync.WaitGroup
	var err1 error
	flow := make(chan struct{}, 5) // max concurrent
	for _, f1 := range fa {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		wg.Add(1)
		flow <- struct{}{}
		go func(f2 *SrcFile) {
			defer func() {
				wg.Done()
				<-flow
			}()

			// early stop
			if err := ctx.Err(); err != nil {
				err1 = err
				return
			}

			err := cmd.annotateFile(f2)
			if err1 == nil {
				err1 = err // just keep first error
			}
		}(f1)
	}
	wg.Wait()
	return err1
}

func (cmd *Cmd) annotateFile(f *SrcFile) error {
	astFile, err := f.files.fullAstFile(f.filename)
	if err != nil {
		return err
	}
	return cmd.annset.AnnotateAstFile(astFile, f)
}

//------------

func (cmd *Cmd) mkdirAllWriteAstFile(filename string, astFile *ast.File) error {
	buf := &bytes.Buffer{}
	if err := goutil.PrintAstFile(buf, cmd.annset.FSet, astFile); err != nil {
		return err
	}
	return mkdirAllWriteFile(filename, buf.Bytes())
}

//------------

func (cmd *Cmd) baseFilenameOut() string {
	s := "godebug"
	if cmd.flags.mode.test {
		s = "test_godebug"
	}
	if len(cmd.flags.filenames) > 0 {
		base := filepath.Base(cmd.flags.filenames[0])
		s = replaceExt(base, "_"+s)
	}
	return osutil.ExecName(s)
}

func (cmd *Cmd) setupTmpDir() (string, error) {
	d := "editor_godebug_mod_work"
	if cmd.gopathMode {
		d = "editor_godebug_gopath_work"
	}
	//if cmd.fixedTmpDir.on {
	//	// use a fixed directory to allow "go build" to use the cache
	//	// there is only one godebug session per editor, so ok to use pid
	//	pid := os.Getpid()
	//	if cmd.fixedTmpDir.pid != 0 {
	//		pid = cmd.fixedTmpDir.pid
	//	}
	//	d += fmt.Sprintf("_pid%v", pid)
	//	return filepath.Join(os.TempDir(), d), nil
	//}
	return ioutil.TempDir(os.TempDir(), d)
}

//------------

//func (cmd *Cmd) updateFixedTmpDir(files *Files) error {
//	if !cmd.fixedTmpDir.on {
//		return nil
//	}

//	// visit all dirs of the loaded files
//	seen := map[string]bool{}
//	for _, f := range files.files {
//		// 1: src
//		// 2: tmp

//		fname1 := f.destFilename()
//		dir1 := filepath.Dir(fname1)
//		fname2 := cmd.tmpDirBasedFilename(fname1)
//		dir2 := filepath.Dir(fname2)
//		if seen[dir2] {
//			continue
//		}
//		seen[dir2] = true

//		// read all files in dir
//		fi2s, err := ioutil.ReadDir(dir2)
//		if err != nil {
//			continue
//		}
//		// compare files with original filenames
//		for _, fi2 := range fi2s {
//			// check if exists in source
//			fname3 := filepath.Join(dir1, fi2.Name())
//			fi1, err := os.Stat(fname3)
//			if err != nil {
//				// does not exist in source, remove in tmp
//				fname4 := filepath.Join(dir2, fi2.Name())
//				fmt.Printf("removing: %v\n", fname4)
//				if err := os.Remove(fname4); err != nil {
//					return err
//				}
//				continue
//			}
//			// exists in source, compare timestamps
//			if fi2.ModTime().After(fi1.ModTime()) {
//				// tmp is still newer than the src, keep it as is
//				if f.typ == FTSrc {
//					// TODO: need to check and compare new directive
//					fmt.Printf("setting to none: %v\n", f.filename)
//					f.action = FANone
//				}
//			}
//		}
//	}
//	return nil
//}

//------------

func (cmd *Cmd) parseArgs(args []string) error {
	if len(args) > 0 {
		name := "GoDebug " + args[0]
		switch args[0] {
		case "run":
			cmd.flags.mode.run = true
			return cmd.parseRunArgs(name, args[1:])
		case "test":
			cmd.flags.mode.test = true
			return cmd.parseTestArgs(name, args[1:])
		case "build":
			cmd.flags.mode.build = true
			return cmd.parseBuildArgs(name, args[1:])
		case "connect":
			cmd.flags.mode.connect = true
			return cmd.parseConnectArgs(name, args[1:])
		}
	}
	fmt.Fprint(cmd.Stderr, cmdUsage())
	return flag.ErrHelp
}

func (cmd *Cmd) parseRunArgs(name string, args []string) error {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(cmd.Stderr)
	cmd.dirsFlag(f)
	cmd.filesFlag(f)
	cmd.workFlag(f)
	cmd.verboseFlag(f)
	cmd.toolExecFlag(f)
	cmd.syncSendFlag(f)
	cmd.envFlag(f)

	if err := f.Parse(args); err != nil {
		return err
	}

	cmd.filenamesAndOtherArgs(f)

	return nil
}

func (cmd *Cmd) parseTestArgs(name string, args []string) error {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(cmd.Stderr)
	cmd.dirsFlag(f)
	cmd.filesFlag(f)
	cmd.workFlag(f)
	cmd.verboseFlag(f)
	cmd.toolExecFlag(f)
	cmd.syncSendFlag(f)
	cmd.envFlag(f)
	run := f.String("run", "", "run test")
	verboseTests := f.Bool("v", false, "verbose tests")

	if err := f.Parse(args); err != nil {
		return err
	}

	// set test run flag at other flags to pass to the test exec
	if *run != "" {
		a := []string{"-test.run", *run}
		cmd.flags.testRunArgs = append(a, cmd.flags.testRunArgs...)
	}
	// verbose
	if *verboseTests {
		a := []string{"-test.v"}
		cmd.flags.testRunArgs = append(a, cmd.flags.testRunArgs...)
	}

	cmd.filenamesAndOtherArgs(f)

	return nil
}

func (cmd *Cmd) parseBuildArgs(name string, args []string) error {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(cmd.Stderr)
	cmd.dirsFlag(f)
	cmd.filesFlag(f)
	cmd.workFlag(f)
	cmd.verboseFlag(f)
	cmd.syncSendFlag(f)
	cmd.envFlag(f)
	addr := f.String("addr", "", "address to serve from, built into the binary")
	f.StringVar(&cmd.flags.output, "o", "", "output filename (default: ${filename}_godebug")

	if err := f.Parse(args); err != nil {
		return err
	}

	cmd.flags.address = *addr
	cmd.filenamesAndOtherArgs(f)

	return nil
}

func (cmd *Cmd) parseConnectArgs(name string, args []string) error {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(cmd.Stderr)
	addr := f.String("addr", "", "address to connect to, built into the binary")
	cmd.toolExecFlag(f)

	if err := f.Parse(args); err != nil {
		return err
	}

	cmd.flags.address = *addr

	return nil
}

//------------

func (cmd *Cmd) filenamesAndOtherArgs(fs *flag.FlagSet) {
	args := fs.Args()

	// filenames
	f := []string{}
	for _, a := range args {
		if strings.HasSuffix(a, ".go") {
			f = append(f, a)
			continue
		}
		break
	}
	cmd.flags.filenames = f

	if len(args) > 0 {
		cmd.flags.otherArgs = args[len(f):] // keep rest
	}
}

//------------

func (cmd *Cmd) workFlag(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.flags.work, "work", false, "print workdir and don't cleanup on exit")
}
func (cmd *Cmd) verboseFlag(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.flags.verbose, "verbose", false, "verbose godebug")
}
func (cmd *Cmd) syncSendFlag(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.flags.syncSend, "syncsend", false, "Don't send msgs in chunks (slow). Useful to get msgs before a crash.")
}
func (cmd *Cmd) toolExecFlag(fs *flag.FlagSet) {
	fs.StringVar(&cmd.flags.toolExec, "toolexec", "", "execute cmd, useful to run a tool with the output file (ex: wine outputfilename)")
}
func (cmd *Cmd) dirsFlag(fs *flag.FlagSet) {
	fn := func(s string) error {
		cmd.flags.dirs = splitCommaList(s)
		return nil
	}
	rf := &runFnFlag{fn}
	fs.Var(rf, "dirs", "comma-separated `string` of directories to annotate")
}
func (cmd *Cmd) filesFlag(fs *flag.FlagSet) {
	fn := func(s string) error {
		cmd.flags.files = splitCommaList(s)
		return nil
	}
	rf := &runFnFlag{fn}
	fs.Var(rf, "files", "comma-separated `string` of files to annotate")
}
func (cmd *Cmd) envFlag(fs *flag.FlagSet) {
	fn := func(s string) error {
		cmd.flags.env = filepath.SplitList(s)
		return nil
	}
	rf := &runFnFlag{fn}
	// The type in usage is the backquoted "string" (detected by flagset)
	usage := fmt.Sprintf("`string` with env variables (ex: \"GOOS=os%c...\"'", filepath.ListSeparator)
	fs.Var(rf, "env", usage)
}

//------------

func (cmd *Cmd) setupNetworkAddress() error {
	// can't consider using stdin/out since the program could use it

	if cmd.flags.address != "" {
		cmd.start.network = "tcp"
		cmd.start.address = cmd.flags.address
		return nil
	}

	// find OS target
	goOs := osutil.GetEnv(cmd.env, "GOOS")
	if goOs == "" {
		goOs = runtime.GOOS
	}

	switch goOs {
	case "linux":
		cmd.start.network = "unix"
		port := osutil.RandomPort(1, 10000, 65000)
		p := "editor_godebug.sock" + fmt.Sprintf("%v", port)
		cmd.start.address = filepath.Join(os.TempDir(), p)
	default:
		//port, err := osutil.GetFreeTcpPort()
		//if err != nil {
		//	return err
		//}
		port := osutil.RandomPort(2, 10000, 65000)
		cmd.start.network = "tcp"
		cmd.start.address = fmt.Sprintf("127.0.0.1:%v", port)
	}
	return nil
}

//------------
//------------
//------------

type runFnFlag struct {
	fn func(string) error
}

func (v runFnFlag) String() string     { return "" }
func (v runFnFlag) Set(s string) error { return v.fn(s) }

//------------

func cmdUsage() string {
	return `Usage:
	GoDebug <command> [arguments]
The commands are:
	run		build and run program with godebug data
	test		test packages compiled with godebug data
	build 	build binary with godebug data (allows remote debug)
	connect	connect to a binary built with godebug data (allows remote debug)
Env variables:
	GODEBUG_BUILD_FLAGS	comma separated flags for build
Examples:
	GoDebug -help
	GoDebug run -help
	GoDebug run main.go -arg1 -arg2
	GoDebug run -dirs=dir1,dir2 -files=f1.go,f2.go main.go -arg1 -arg2
	GoDebug test -help
	GoDebug test
	GoDebug test -run mytest
	GoDebug build -addr=:8008 main.go
	GoDebug connect -addr=:8008
	GoDebug run -env=GODEBUG_BUILD_FLAGS=-tags=xproto main.go
`
}

//------------

func mkdirAllWriteFile(filename string, src []byte) error {
	return iout.MkdirAllWriteFile(filename, src, 0660)
}

func mkdirAllCopyFile(src, dst string) error {
	return iout.MkdirAllCopyFile(src, dst, 0660)
}
func mkdirAllCopyFileSync(src, dst string) error {
	return iout.MkdirAllCopyFileSync(src, dst, 0660)
}

func copyFile(src, dst string) error {
	return iout.CopyFile(src, dst, 0660)
}

//------------

func replaceExt(filename, ext string) string {
	// remove extension
	tmp := filename
	ext2 := filepath.Ext(tmp)
	if len(ext2) > 0 {
		tmp = tmp[:len(tmp)-len(ext2)]
	}
	// add new extension
	return tmp + ext
}

//------------

func splitCommaList(val string) []string {
	a := strings.Split(val, ",")
	u := []string{}
	for _, s := range a {
		// don't add empty strings
		s := strings.TrimSpace(s)
		if s == "" {
			continue
		}

		u = append(u, s)
	}
	return u
}

//------------

func trimAtFirstSrcDir(filename string) string {
	v := filename
	w := []string{}
	for {
		base := filepath.Base(v)
		if base == "src" {
			return filepath.Join(w...) // trimmed
		}
		w = append([]string{base}, w...)
		oldv := v
		v = filepath.Dir(v)
		isRoot := oldv == v
		if isRoot {
			break
		}
	}
	return filename
}

//----------

func godebugBuildFlags(env []string) []string {
	bfs := osutil.GetEnv(env, "GODEBUG_BUILD_FLAGS")
	if len(bfs) == 0 {
		return nil
	}
	return strings.Split(bfs, ",")
}
