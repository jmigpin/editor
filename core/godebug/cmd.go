package godebug

// Needs to run when there are changes on the ./debug pkg
//go:generate go run debugpack/debugpack.go

import (
	"bytes"
	"context"
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
)

type Cmd struct {
	Client *Client
	Dir    string // "" will use current dir
	Stdout io.Writer
	Stderr io.Writer

	tmpDir       string
	tmpBuiltFile string // file built and exec'd

	//FixedTmpDir    bool // re-use tmp dir to allow caching
	//FixedTmpDirPid int
	//noTmpCleanup   bool

	env       []string // set at start
	annset    *AnnotatorSet
	noModules bool // go.mod's

	NoPreBuild bool // useful for tests

	start struct {
		network   string
		address   string
		cancel    context.CancelFunc
		wait      sync.WaitGroup
		serverErr error
	}

	flags struct {
		mode struct {
			run     bool
			test    bool
			build   bool
			connect bool
		}
		verbose   bool
		filename  string
		work      bool
		output    string // ex: -o filename
		toolExec  string // ex: "wine" will run "wine args..."
		dirs      []string
		files     []string
		address   string   // build/connect
		env       []string // build
		syncSend  bool
		otherArgs []string
		runArgs   []string
	}
}

func NewCmd() *Cmd {
	cmd := &Cmd{
		annset: NewAnnotatorSet(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	//cmd.FixedTmpDir = true
	//cmd.FixedTmpDirPid = 2

	return cmd
}

//------------

func (cmd *Cmd) Error(err error) {
	cmd.Printf("error: %v\n", err)
}
func (cmd *Cmd) Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(cmd.Stdout, "# "+format, a...)
}

//------------

func (cmd *Cmd) Start(ctx context.Context, args []string) (done bool, _ error) {
	// parse arguments
	done, err := cmd.parseArgs(args)
	if done || err != nil {
		return done, err
	}

	// absolute dir
	if u, err := filepath.Abs(cmd.Dir); err == nil {
		cmd.Dir = u
	}

	cmd.noModules = cmd.detectNoModules()
	if cmd.flags.verbose {
		cmd.Printf("nomodules=%v\n", cmd.noModules)
	}

	// depends on: noModules
	tmpDir, err := cmd.setupTmpDir()
	if err != nil {
		return true, err
	}
	cmd.tmpDir = tmpDir

	// depends on: noModules, tmpDir
	cmd.env = cmd.detectEnviron()

	// print tmp dir if work flag is present
	if cmd.flags.work {
		cmd.Printf("work: %v\n", cmd.tmpDir)
	}

	if err := cmd.setupNetworkAddress(); err != nil {
		return true, err
	}

	m := &cmd.flags.mode
	if m.run || m.test || m.build {
		debug.SyncSend = cmd.flags.syncSend
		err := cmd.initAndAnnotate(ctx)
		if err != nil {
			return true, err
		}
	}

	// just building: inform the address used in the binary
	if m.build {
		cmd.Printf("build: %v (builtin address: %v, %v)\n", cmd.tmpBuiltFile, cmd.start.network, cmd.start.address)
		return true, err
	}

	if m.run || m.test || m.connect {
		err = cmd.startServerClient(ctx)
		return false, err
	}

	return false, nil
}

//------------

func (cmd *Cmd) initAndAnnotate(ctx context.Context) error {
	// "files" not in cmd.* to allow early GC
	files := NewFiles(cmd.annset.FSet, cmd.noModules)
	files.Dir = cmd.Dir
	//if cmd.FixedTmpDir {
	//	files.TmpDir = cmd.tmpDir
	//}

	files.Add(cmd.flags.files...)
	files.Add(cmd.flags.dirs...)

	mainFilename := files.absFilename(cmd.flags.filename)

	// pre-build without annotations for better errors with original filenames(result ignored)
	// done in sync (first pre-build, then annotate) in order to have the "go build" update go.mod's if necessary.
	if !cmd.NoPreBuild {
		if err := cmd.preBuild(ctx, mainFilename, cmd.flags.mode.test); err != nil {
			return err
		}
	}

	// TODO: force disable fetching since pre-build was successfull.
	//cmd.env = cmd.disableGoFetch(cmd.env)

	return cmd.initAndAnnotate2(ctx, files, mainFilename)
}

//func (cmd *Cmd) initAndAnnotate___concurrent(ctx context.Context) error {
//	files := NewFiles(cmd.annset.FSet) // not in cmd.* to allow early GC
//	files.Dir = cmd.Dir

//	files.Add(cmd.flags.files...)
//	files.Add(cmd.flags.dirs...)

//	mainFilename := files.absFilename(cmd.flags.filename)

//	ctx2, cancel := context.WithCancel(ctx)
//	defer cancel()

//	var wg sync.WaitGroup

//	// TODO: if the pre-build is changing the go.mod, then the annotate phase won't have the final go.mod available (also gives some warnings)
//	// pre-build without annotations for better errors (result is ignored)
//	wg.Add(1)
//	var preBuildErr error
//	go func() {
//		defer wg.Done()
//		if err := cmd.preBuild(ctx2, mainFilename, cmd.flags.mode.test); err != nil {
//			preBuildErr = err
//			cancel() // early cancel
//		}
//	}()

//	// continue with init and annotate
//	wg.Add(1)
//	var err2 error
//	go func() {
//		defer wg.Done()
//		if err := cmd.initAndAnnotate2(ctx2, files, mainFilename); err != nil {
//			err2 = err
//		}
//	}()

//	wg.Wait()

//	// send only the prebuild error if it happens
//	if preBuildErr != nil {
//		return preBuildErr
//	}
//	return err2
//}

//------------

func (cmd *Cmd) initAndAnnotate2(ctx context.Context, files *Files, mainFilename string) error {
	err := files.Do(ctx, mainFilename, cmd.flags.mode.test, cmd.env)
	if err != nil {
		return err
	}

	if cmd.flags.verbose {
		files.verbose(cmd)
	}

	// copy
	for filename := range files.copyFilenames {
		dst := cmd.tmpDirBasedFilename(filename)
		if err := mkdirAllCopyFileSync(filename, dst); err != nil {
			return err
		}
	}
	for filename := range files.modFilenames {
		dst := cmd.tmpDirBasedFilename(filename)
		if err := mkdirAllCopyFileSync(filename, dst); err != nil {
			return err
		}
	}

	// annotate
	if err := cmd.annotateFiles(ctx, files); err != nil {
		return err
	}

	// write config file after annotations
	if err := cmd.writeGoDebugConfigFilesToTmpDir(ctx, files); err != nil {
		return err
	}

	// create testmain file
	if cmd.flags.mode.test && !cmd.annset.InsertedExitIn.TestMain {
		if err := cmd.writeTestMainFilesToTmpDir(); err != nil {
			return err
		}
	}

	// main must have exit inserted
	if !cmd.flags.mode.test && !cmd.annset.InsertedExitIn.Main {
		return fmt.Errorf("have not inserted debug exit in main()")
	}

	if !cmd.noModules {
		if err := SetupGoMods(ctx, cmd, files); err != nil {
			return err
		}
	}

	return cmd.doBuild(ctx, mainFilename, cmd.flags.mode.test)
}

func (cmd *Cmd) doBuild(ctx context.Context, mainFilename string, tests bool) error {
	filename := cmd.filenameForBuild(mainFilename, tests)
	filenameAtTmp := cmd.tmpDirBasedFilename(filename)

	// create parent dirs
	if err := os.MkdirAll(filepath.Dir(filenameAtTmp), 0755); err != nil {
		return err
	}

	// build
	filenameAtTmpOut, err := cmd.runBuildCmd(ctx, filenameAtTmp, tests)
	if err != nil {
		return err
	}

	// move filename to working dir
	filenameWork := filepath.Join(cmd.Dir, filepath.Base(filenameAtTmpOut))
	// move filename to output option
	if cmd.flags.output != "" {
		o := cmd.flags.output
		if !filepath.IsAbs(o) {
			o = filepath.Join(cmd.Dir, o)
		}
		filenameWork = o
	}
	if err := os.Rename(filenameAtTmpOut, filenameWork); err != nil {
		return err
	}

	// keep moved filename that will run in working dir for later cleanup
	cmd.tmpBuiltFile = filenameWork

	return nil
}

func (cmd *Cmd) filenameForBuild(mainFilename string, tests bool) string {
	if tests {
		// final filename will include extension replacement with "_godebug"
		return filepath.Join(cmd.Dir, "pkgtest")
	}
	return mainFilename
}

func (cmd *Cmd) preBuild(ctx context.Context, mainFilename string, tests bool) error {
	filename := cmd.filenameForBuild(mainFilename, tests)
	filenameOut, err := cmd.runBuildCmd(ctx, filename, tests)
	defer os.Remove(filenameOut) // ensure removal even on error
	if err != nil {
		return err
	}
	return nil
}

//------------

func (cmd *Cmd) startServerClient(ctx context.Context) error {
	// server/client context to cancel the other when one of them ends
	ctx, cancel := context.WithCancel(ctx)
	cmd.start.cancel = cancel

	if err := cmd.startServerClient2(ctx); err != nil {
		// cmd.Wait() won't be called, clear resources
		cmd.start.cancel()
		cmd.start.wait.Wait()
	}
	return nil
}

func (cmd *Cmd) startServerClient2(ctx context.Context) error {
	// arguments (TODO: review normalize...)
	w := normalizeFilenameForExec(cmd.tmpBuiltFile)
	args := []string{w}
	if cmd.flags.mode.test {
		args = append(args, cmd.flags.runArgs...)
	} else {
		args = append(args, cmd.flags.otherArgs...)
	}

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
		// setup waiting for server to end
		cmd.start.wait.Add(1)
		go func() {
			defer cmd.start.wait.Done()
			cmd.start.serverErr = c.Wait()
			cmd.start.cancel()
		}()
	}

	// start client (blocks until connected)
	client, err := NewClient(ctx, cmd.start.network, cmd.start.address)
	if err != nil {
		return err
	}
	cmd.Client = client

	// setup waiting for client to finish
	cmd.start.wait.Add(1)
	go func() {
		defer cmd.start.wait.Done()
		cmd.Client.Wait() // wait for client to finish
		cmd.start.cancel()
	}()

	return nil
}

func (cmd *Cmd) Wait() error {
	cmd.start.wait.Wait()
	cmd.start.cancel() // ensure resources are cleared
	return cmd.start.serverErr
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

	if cmd.noModules {
		// trim filename when inside a src dir
		rhs := trimAtFirstSrcDir(filename)
		return filepath.Join(cmd.tmpDir, "src", rhs)
	}

	// just replicate on tmp dir
	return filepath.Join(cmd.tmpDir, filename)
}

//------------

func (cmd *Cmd) detectEnviron() []string {
	env := os.Environ()

	env = osutil.SetEnvs(env, cmd.flags.env)

	// after cmd.flags.env such that this result won't be overriden
	if s, ok := cmd.goPathStr(); ok {
		env = osutil.SetEnv(env, "GOPATH", s)
	}

	return env
}

func (cmd *Cmd) goPathStr() (string, bool) {
	u := []string{} // first has priority

	// add tmpdir for priority to the annotated files
	if cmd.noModules {
		u = append(u, cmd.tmpDir)
	}

	if s := osutil.GetEnv(cmd.flags.env, "GOPATH"); s != "" {
		u = append(u, s)
	}

	// always include default gopath last (includes entry that might not be defined anywhere, needs to be set)
	u = append(u, goutil.GoPath()...)

	return goutil.JoinPathLists(u...), true
}

//func (cmd *Cmd) disableGoFetch(env []string) []string {
//	// TODO: without this, windows is failing if there is no connection, but works if there is (checks local?)
//	// TODO: with this, linux fails with packages not being loaded (doesn't check local?)

//	return osutil.SetEnvs(cmd.env, []string{
//		//"GOPROXY=direct",
//		"GOPROXY=off", // don't use the net
//		//"GOSUMDB=off",
//	})
//}

//------------

func (cmd *Cmd) detectNoModules() bool {
	env := []string{}
	env = osutil.SetEnvs(env, os.Environ())
	env = osutil.SetEnvs(env, cmd.flags.env)

	v := osutil.GetEnv(env, "GO111MODULE")
	if v == "off" {
		return true
	}
	if v == "on" {
		return false
	}

	// auto: if it can't find a go.mod, it is noModules
	if _, ok := goutil.FindGoMod(cmd.Dir); !ok {
		return true
	}
	return false
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
		//} else if cmd.noTmpCleanup {
		//	// don't cleanup work dir
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

func (cmd *Cmd) runBuildCmd(ctx context.Context, filename string, tests bool) (string, error) {
	// TODO: faster dummy pre-builts?
	// "-toolexec", "", // don't run asm?

	filenameOut := replaceExt(filename, osutil.ExecName("_godebug"))

	bFlags := godebugBuildFlags(cmd.env)

	args := []string{}
	if tests {
		args = []string{
			osutil.ExecName("go"), "test",
			"-c", // compile binary but don't run
			"-o", filenameOut,
		}
		args = append(args, bFlags...)
		args = append(args, cmd.flags.otherArgs...) // ex
	} else {
		args = []string{
			osutil.ExecName("go"), "build",
			"-o", filenameOut,
		}
		//if cmd.flags.mode.build {
		//	args = append(args, cmd.flags.otherArgs...)
		//}
		args = append(args, bFlags...)
		args = append(args, filename) // last arg
	}

	if cmd.flags.verbose {
		cmd.Printf("runBuildCmd:  %v\n", args)
	}
	dir := filepath.Dir(filenameOut)
	err := cmd.runCmd(ctx, dir, args, cmd.env)
	if err != nil {
		err = fmt.Errorf("runBuildCmd: %v", err)
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

func (cmd *Cmd) annotateFiles(ctx context.Context, files *Files) error {
	var wg sync.WaitGroup
	var err1 error
	flow := make(chan struct{}, 5) // max concurrent
	for filename := range files.annFilenames {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		wg.Add(1)
		flow <- struct{}{}
		go func(filename string) {
			defer func() {
				wg.Done()
				<-flow
			}()
			err := cmd.annotateFile(ctx, files, filename)
			if err1 == nil {
				err1 = err // just keep first error
			}
		}(filename)
	}
	wg.Wait()
	return err1
}

func (cmd *Cmd) annotateFile(ctx context.Context, files *Files, filename string) error {

	dst := cmd.tmpDirBasedFilename(filename)
	astFile, err := files.fullAstFile(filename)
	if err != nil {
		return err
	}
	if err := cmd.annset.AnnotateAstFile(astFile, filename, files); err != nil {
		return err
	}

	// early stop
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := cmd.mkdirAllWriteAstFile(dst, astFile); err != nil {
		return err
	}
	return nil
}

//------------

func (cmd *Cmd) setupTmpDir() (string, error) {
	d := "editor_godebug_mod_work"
	if cmd.noModules {
		d = "editor_godebug_gopath_work"
	}
	//if cmd.FixedTmpDir {
	//	// use a fixed directory to allow "go build" to use the cache
	//	// there is only one godebug session per editor, so ok to use pid
	//	cmd.noTmpCleanup = true
	//	pid := os.Getpid()
	//	if cmd.FixedTmpDirPid != 0 {
	//		pid = cmd.FixedTmpDirPid
	//	}
	//	d += fmt.Sprintf("_pid%v", pid)
	//	tmpDir := filepath.Join(os.TempDir(), d)
	//	return tmpDir, nil
	//}
	return ioutil.TempDir(os.TempDir(), d)
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

func (cmd *Cmd) writeGoDebugConfigFilesToTmpDir(ctx context.Context, files *Files) error {
	// godebugconfig pkg: config.go
	filename := files.GodebugconfigPkgFilename("config.go")
	src := cmd.annset.ConfigContent(cmd.start.network, cmd.start.address)
	filenameAtTmp := cmd.tmpDirBasedFilename(filename)
	if err := mkdirAllWriteFile(filenameAtTmp, []byte(src)); err != nil {
		return err
	}
	// godebugconfig pkg: go.mod
	if !cmd.noModules {
		dir := files.GodebugconfigPkgFilename("")
		dirAtTmp := cmd.tmpDirBasedFilename(dir)
		if err := goutil.GoModInit(ctx, dirAtTmp, GodebugconfigPkgPath, cmd.env); err != nil {
			return err
		}
	}
	// debug pkg (pack into a file and add to godebugconfig pkg)
	for _, fp := range DebugFilePacks() {
		filename := files.DebugPkgFilename(fp.Name)
		filenameAtTmp := cmd.tmpDirBasedFilename(filename)
		if err := mkdirAllWriteFile(filenameAtTmp, []byte(fp.Data)); err != nil {
			return err
		}
	}
	// debug pkg: go.mod
	if !cmd.noModules {
		dir := files.DebugPkgFilename("")
		dirAtTmp := cmd.tmpDirBasedFilename(dir)
		if err := goutil.GoModInit(ctx, dirAtTmp, DebugPkgPath, cmd.env); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Cmd) writeTestMainFilesToTmpDir() error {
	u := cmd.annset.TestMainSources()
	for i, tms := range u {
		name := fmt.Sprintf("godebug_testmain%v_test.go", i)
		filename := filepath.Join(tms.Dir, name)
		filenameAtTmp := cmd.tmpDirBasedFilename(filename)
		return mkdirAllWriteFile(filenameAtTmp, []byte(tms.Src))
	}
	return nil
}

//------------

func (cmd *Cmd) parseArgs(args []string) (done bool, _ error) {
	if len(args) > 0 {
		switch args[0] {
		case "run":
			cmd.flags.mode.run = true
			return cmd.parseRunArgs(args[1:])
		case "test":
			cmd.flags.mode.test = true
			return cmd.parseTestArgs(args[1:])
		case "build":
			cmd.flags.mode.build = true
			return cmd.parseBuildArgs(args[1:])
		case "connect":
			cmd.flags.mode.connect = true
			return cmd.parseConnectArgs(args[1:])
		}
	}
	fmt.Fprint(cmd.Stderr, cmdUsage())
	return true, nil
}

func (cmd *Cmd) parseRunArgs(args []string) (done bool, _ error) {
	f := &flag.FlagSet{}
	f.SetOutput(cmd.Stderr)
	cmd.dirsFlag(f)
	cmd.filesFlag(f)
	cmd.workFlag(f)
	cmd.verboseFlag(f)
	cmd.toolExecFlag(f)
	cmd.syncSendFlag(f)
	cmd.envFlag(f)

	if err := f.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return true, nil
		}
		return true, err
	}

	cmd.flags.otherArgs = f.Args()

	if len(cmd.flags.otherArgs) > 0 {
		cmd.flags.filename = cmd.flags.otherArgs[0]
		cmd.flags.otherArgs = cmd.flags.otherArgs[1:]
	}

	return false, nil
}

func (cmd *Cmd) parseTestArgs(args []string) (done bool, _ error) {
	f := &flag.FlagSet{}
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
		if err == flag.ErrHelp {
			return true, nil
		}
		return true, err
	}

	cmd.flags.otherArgs = f.Args()

	// set test run flag at other flags to pass to the test exec
	if *run != "" {
		a := []string{"-test.run", *run}
		cmd.flags.runArgs = append(a, cmd.flags.runArgs...)
	}

	// verbose
	if *verboseTests {
		a := []string{"-test.v"}
		cmd.flags.runArgs = append(a, cmd.flags.runArgs...)
	}

	return false, nil
}

func (cmd *Cmd) parseBuildArgs(args []string) (done bool, _ error) {
	f := &flag.FlagSet{}
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
		if err == flag.ErrHelp {
			return true, nil
		}
		return true, err
	}

	cmd.flags.address = *addr
	cmd.flags.otherArgs = f.Args()
	if len(cmd.flags.otherArgs) > 0 {
		cmd.flags.filename = cmd.flags.otherArgs[0]
		cmd.flags.otherArgs = cmd.flags.otherArgs[1:]
	}

	return false, nil
}

func (cmd *Cmd) parseConnectArgs(args []string) (done bool, _ error) {
	f := &flag.FlagSet{}
	f.SetOutput(cmd.Stderr)
	addr := f.String("addr", "", "address to connect to, built into the binary")
	cmd.toolExecFlag(f)

	if err := f.Parse(args); err != nil {
		if err == flag.ErrHelp {
			f.SetOutput(cmd.Stderr)
			f.PrintDefaults()
			return true, nil
		}
		return true, err
	}

	cmd.flags.address = *addr

	return false, nil
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
	GoDebug build -addr=:8080 main.go
	GoDebug connect -addr=:8080
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

func normalizeFilenameForExec(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}

	// TODO: review
	if !strings.HasPrefix(filename, "./") {
		return "./" + filename
	}

	return filename
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
