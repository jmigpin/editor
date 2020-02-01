package godebug

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/osutil"
)

type Cmd struct {
	Client *Client
	Dir    string // "" will use current dir
	Stdout io.Writer
	Stderr io.Writer

	tmpDir       string
	tmpBuiltFile string   // file built and exec'd
	env          []string // set at start
	annset       *AnnotatorSet
	noModules    bool // go.mod's

	start struct {
		cancel    context.CancelFunc
		waitg     sync.WaitGroup
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

	m := &cmd.flags.mode

	if m.run || m.test || m.build {
		debug.SyncSend = cmd.flags.syncSend
		cmd.setupServerNetAddr()
		err := cmd.initAndAnnotate(ctx)
		if err != nil {
			return true, err
		}
	}

	// just building: inform the address used in the binary
	if m.build {
		cmd.Printf("build: %v (builtin address: %v, %v)\n",
			cmd.tmpBuiltFile,
			debug.ServerNetwork,
			debug.ServerAddress,
		)
		return true, err
	}

	if m.run || m.test || m.connect {
		err = cmd.startServerClient(ctx)
		return false, err
	}

	return false, nil
}

//------------

func (cmd *Cmd) Wait() error {
	cmd.start.waitg.Wait()
	cmd.start.cancel() // ensure resources are cleared
	return cmd.start.serverErr
}

//------------

func (cmd *Cmd) initAndAnnotate(ctx context.Context) error {
	// "files" not in cmd.* to allow early GC
	files := NewFiles(cmd.annset.FSet, cmd.noModules)
	files.Dir = cmd.Dir

	files.Add(cmd.flags.files...)
	files.Add(cmd.flags.dirs...)

	mainFilename := files.absFilename(cmd.flags.filename)

	// pre-build without annotations for better errors (result ignored)
	// done in sync (first pre-build, then annotate) in order to have the "go build" update go.mod's if necessary.
	if err := cmd.preBuild(ctx, mainFilename, cmd.flags.mode.test); err != nil {
		return err
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
	for filename := range files.annFilenames {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		dst := cmd.tmpDirBasedFilename(filename)
		typ := files.annTypes[filename]
		astFile, err := files.fullAstFile(filename)
		if err != nil {
			return err
		}
		if err := cmd.annset.AnnotateAstFile(astFile, typ); err != nil {
			return err
		}
		if err := cmd.mkdirAllWriteAstFile(dst, astFile); err != nil {
			return err
		}
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
	ctx2, cancel := context.WithCancel(ctx)
	cmd.start.cancel = cancel

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

	// start server
	var serverCmd *exec.Cmd
	if !cmd.flags.mode.connect {
		u, err := cmd.startCmd(ctx2, cmd.Dir, args, nil)
		if err != nil {
			// cmd.Wait() won't be called, need to clear resources
			cmd.start.cancel()
			return err
		}
		serverCmd = u

		// output cmd pid
		cmd.Printf("pid %d\n", serverCmd.Process.Pid)
	}

	// setup address to connect to
	if cmd.flags.mode.connect && cmd.flags.address != "" {
		debug.ServerNetwork = "tcp"
		debug.ServerAddress = cmd.flags.address
	}
	// start client (blocking connect)
	client, err := NewClient(ctx2)
	if err != nil {
		// cmd.Wait() won't be called, need to clear resources
		cmd.start.cancel()
		return err
	}
	cmd.Client = client

	// from this point, cmd.Wait() clears resources from cmd.start.cancel

	// server done
	if serverCmd != nil {
		cmd.start.waitg.Add(1)
		go func() {
			defer cmd.start.waitg.Done()
			// wait for server to finish
			cmd.start.serverErr = serverCmd.Wait()
		}()
	}

	// client done
	cmd.start.waitg.Add(1)
	go func() {
		defer cmd.start.waitg.Done()
		cmd.Client.Wait() // wait for client to finish
	}()
	// ensure client stops on context cancel (only for connect mode)
	if cmd.flags.mode.connect {
		go func() {
			select {
			case <-ctx.Done():
				_ = cmd.Client.Close()
			}
		}()
	}

	return nil
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

	// only used for tests, not for external use
	v = osutil.GetEnv(env, "EDITOR_GODEBUG_NOMODULES")
	if v == "true" {
		return true
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
	if debug.ServerNetwork == "unix" {
		if err := os.Remove(debug.ServerAddress); err != nil {
			if !os.IsNotExist(err) {
				cmd.Printf("cleanup err: %v\n", err)
			}
		}
	}

	if cmd.flags.work {
		// don't cleanup work dir
	} else {
		if cmd.tmpDir != "" {
			if err := os.RemoveAll(cmd.tmpDir); err != nil {
				cmd.Printf("cleanup err: %v\n", err)
			}
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
	filenameOut := replaceExt(filename, osutil.ExecName("_godebug"))

	args := []string{}
	if tests {
		args = []string{
			osutil.ExecName("go"), "test",
			"-c", // compile binary but don't run
			// TODO: faster dummy pre-builts?
			// "-toolexec", "", // don't run asm?
			"-o", filenameOut,
		}
		args = append(args, cmd.flags.otherArgs...)
	} else {
		args = []string{
			osutil.ExecName("go"), "build",
			"-o", filenameOut,
		}
		// insert otherargs before filename last arg: allows all "go build" to be used after the filename
		// TODO: accept -gobuild.* args?
		if cmd.flags.mode.build {
			args = append(args, cmd.flags.otherArgs...)
		}
		// filename is last arg
		args = append(args, filename)
	}

	dir := filepath.Dir(filenameOut)
	if cmd.flags.verbose {
		cmd.Printf("runBuildCmd: dir=%v\n", dir)
	}
	err := cmd.runCmd(ctx, dir, args, cmd.env)
	if err != nil {
		err = fmt.Errorf("runBuildCmd: %v", err)
	}
	return filenameOut, err
}

//------------

func (cmd *Cmd) runCmd(ctx context.Context, dir string, args, env []string) error {
	// ctx with early cancel for startcmd to clear inner goroutine resource
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	ecmd, err := cmd.startCmd(ctx2, dir, args, env)
	if err != nil {
		return err
	}
	return ecmd.Wait()
}

func (cmd *Cmd) startCmd(ctx context.Context, dir string, args, env []string) (*exec.Cmd, error) {
	cargs := osutil.ShellRunArgs(args...)
	ecmd := osutil.ExecCmdCtxWithAttr(ctx, cargs)

	ecmd.Env = env
	ecmd.Dir = dir
	ecmd.Stdout = cmd.Stdout
	ecmd.Stderr = cmd.Stderr

	if err := ecmd.Start(); err != nil {
		return nil, err
	}

	// ensure kill to child processes on context cancel
	// the ctx must be cancelable, otherwise it might kill the process on start
	go func() {
		select {
		case <-ctx.Done():
			if err := osutil.KillExecCmd(ecmd); err != nil {
				// commented: avoid over verbose errors before the full output comes out
				//cmd.Error(fmt.Errorf("kill: %v", err))
			}
		}
	}()

	return ecmd, nil
}

//------------

func (cmd *Cmd) setupTmpDir() (string, error) {
	if cmd.noModules {
		d := "editor_godebug_gopath_work"
		tmpDir, err := ioutil.TempDir(os.TempDir(), d)
		if err != nil {
			return "", err
		}
		return tmpDir, nil
	}

	d := "editor_godebug_mod_work"

	// The fixed directory will improve the file sync performance since modules require the whole directory to be there (not like gopath)
	// TODO: will have problems running more then one debug session in different editor sessions
	//d += "_"+md5.Sum([]byte(cmd.Dir))

	//tmpDir := filepath.Join(os.TempDir(), d)

	tmpDir, err := ioutil.TempDir(os.TempDir(), d)
	if err != nil {
		return "", err
	}
	return tmpDir, nil
}

//------------

func (cmd *Cmd) mkdirAllWriteAstFile(filename string, astFile *ast.File) error {
	buf := &bytes.Buffer{}
	if err := cmd.annset.Print(buf, astFile); err != nil {
		return err
	}
	return mkdirAllWriteFile(filename, buf.Bytes())
}

//------------

func (cmd *Cmd) writeGoDebugConfigFilesToTmpDir(ctx context.Context, files *Files) error {
	// godebugconfig pkg: config.go
	filename := files.GodebugconfigPkgFilename("config.go")
	src := cmd.annset.ConfigContent()
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
	fs.StringVar(&cmd.flags.toolExec, "toolexec", "", "execute cmd, useful to run a tool with the output file (ex: wine outputfilename")
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
	usage := fmt.Sprintf("`string` with env variables (ex: \"GOOS=os%c...\"'", filepath.ListSeparator)
	fs.Var(rf, "env", usage)
}

//------------

func (cmd *Cmd) setupServerNetAddr() {
	// find OS target
	goOs := osutil.GetEnv(cmd.env, "GOOS")
	if goOs == "" {
		goOs = runtime.GOOS
	}

	setupServerNetAddr(cmd.flags.address, goOs)
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
`
}

//------------

func mkdirAllWriteFile(filename string, src []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0770); err != nil {
		return err
	}
	return ioutil.WriteFile(filename, []byte(src), 0660)
}

func mkdirAllCopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0770); err != nil {
		return err
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer to.Close()
	_, err = io.Copy(to, from)
	return err
}

//------------

func mkdirAllCopyFileSync(src, dst string) error {
	// must exist in src
	info1, err := os.Stat(src)
	if os.IsNotExist(err) {
		return fmt.Errorf("not found in src: %v", src)
	}

	// already exists in dest with same modification time
	info2, err := os.Stat(dst)
	if !os.IsNotExist(err) {
		// compare modification time in src
		if info2.ModTime().Equal(info1.ModTime()) {
			return nil
		}
	}

	if err := mkdirAllCopyFile(src, dst); err != nil {
		return err
	}

	// set modtime equal to src to avoid copy next time
	t := info1.ModTime().Local()
	return os.Chtimes(dst, t, t)
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

func setupServerNetAddr(addr string, goOs string) {
	if addr != "" {
		debug.ServerNetwork = "tcp"
		debug.ServerAddress = addr
		return
	}

	// generate address: allows multiple editors to run debug sessions at the same time.

	seed := time.Now().UnixNano() + int64(os.Getpid())
	ra := rand.New(rand.NewSource(seed))
	min, max := 27000, 65535
	port := min + ra.Intn(max-min)

	switch goOs {
	case "linux":
		debug.ServerNetwork = "unix"
		p := "editor_godebug.sock" + fmt.Sprintf("%v", port)
		debug.ServerAddress = filepath.Join(os.TempDir(), p)
	default:
		debug.ServerNetwork = "tcp"
		debug.ServerAddress = fmt.Sprintf("127.0.0.1:%v", port)
	}
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
