package godebug

import (
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
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

	ann     *AnnotatorSet
	mainSrc interface{} // used for tests (at least)

	tmpDir       string
	tmpBuiltFile string // file built and exec'd

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
		filename  string
		work      bool
		dirs      []string
		files     []string
		address   string   // build/connect
		env       []string // build
		otherArgs []string
		runArgs   []string
	}
}

func NewCmd() *Cmd {
	return &Cmd{
		ann:    NewAnnotatorSet(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

//------------

func (cmd *Cmd) Start(ctx context.Context, args []string, mainSrc interface{}) (done bool, _ error) {
	cmd.mainSrc = mainSrc

	// parse arguments
	done, err := cmd.parseArgs(args)
	if done || err != nil {
		return done, err
	}

	// absolute dir
	if u, err := filepath.Abs(cmd.Dir); err == nil {
		cmd.Dir = u
	}

	// tmp dir for annotated files
	tmpDir, err := ioutil.TempDir(os.TempDir(), "editor_godebug")
	if err != nil {
		return true, err
	}
	cmd.tmpDir = tmpDir

	// print tmp dir if work flag is present
	if cmd.flags.work {
		fmt.Fprintf(cmd.Stdout, "work: %v\n", cmd.tmpDir)
	}

	m := &cmd.flags.mode

	if m.run || m.test || m.build {
		setupServerNetAddr(cmd.flags.address)
		err := cmd.initMode(ctx)
		if err != nil {
			return true, err
		}
	}
	if m.build {
		fmt.Fprintf(cmd.Stdout, "build: %v (builtin address: %v, %v)",
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

func (cmd *Cmd) initMode(ctx context.Context) error {
	// filename
	var filename string
	if cmd.flags.mode.test {
		filename = filepath.Join(cmd.Dir, "pkgtest")
	} else {
		filename = cmd.flags.filename
		if filename == "" {
			return fmt.Errorf("missing filename arg")
		}
		// base on workingdir
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(cmd.Dir, filename)
		}
	}

	// pre-build without annotations for better errors (result is ignored)
	if cmd.mainSrc == nil {
		if cmd.flags.mode.test {
			fout, err := cmd.buildTest(ctx, filename)
			if err != nil {
				return err
			}
			os.Remove(fout)
		} else {
			fout, err := cmd.build(ctx, filename)
			if err != nil {
				return err
			}
			os.Remove(fout)
		}
	}

	// annotate: main file
	if !cmd.flags.mode.test {
		if err := cmd.annotateFile(filename, cmd.mainSrc); err != nil {
			return err
		}
	}

	// annotate: files option
	for _, f := range cmd.flags.files {
		if !filepath.IsAbs(f) { // TODO: call func to solve filename
			f = filepath.Join(cmd.Dir, f)
		}
		if err := cmd.annotateFile(f, nil); err != nil {
			return err
		}
	}

	// annotate: auto include working dir in test mode
	if cmd.flags.mode.test {
		cmd.flags.dirs = append(cmd.flags.dirs, cmd.Dir)
	}

	// annotate: directories
	if err := cmd.annotateDirs(ctx); err != nil {
		return err
	}

	// write config file after annotations
	if err := cmd.writeConfigFileToTmpDir(); err != nil {
		return err
	}

	// populate missing go files in parent directories
	if err := cmd.populateParentDirectories(); err != nil {
		return err
	}

	// main/testmain exit
	if cmd.flags.mode.test {
		if !cmd.ann.InsertedExitIn.TestMain {
			if err := cmd.writeTestMainFilesToTmpDir(); err != nil {
				return err
			}
		}
	} else {
		if !cmd.ann.InsertedExitIn.Main {
			return fmt.Errorf("have not inserted debug exit in main()")
		}
	}

	filenameAtTmp := cmd.tmpDirBasedFilename(filename)

	// create parent dirs
	if err := os.MkdirAll(filepath.Dir(filenameAtTmp), 0755); err != nil {
		return err
	}

	// build
	var f2 string
	var err2 error
	if cmd.flags.mode.test {
		f2, err2 = cmd.buildTest(ctx, filenameAtTmp)
	} else {
		f2, err2 = cmd.build(ctx, filenameAtTmp)
	}
	if err2 != nil {
		return err2
	}
	filenameOut := f2

	// move filename to working dir
	filenameWork := filepath.Join(cmd.Dir, filepath.Base(filenameOut))
	if err := os.Rename(filenameOut, filenameWork); err != nil {
		return err
	}

	// keep moved filename that will run in working dir for later cleanup
	cmd.tmpBuiltFile = filenameWork

	return nil
}

//------------

func (cmd *Cmd) startServerClient(ctx context.Context) error {
	filenameWork := cmd.tmpBuiltFile

	// server/client context to cancel the other when one of them ends
	ctx2, cancel := context.WithCancel(ctx)
	cmd.start.cancel = cancel

	// arguments
	filenameWork2 := normalizeFilenameForExec(filenameWork)
	args := []string{filenameWork2}
	if cmd.flags.mode.test {
		args = append(args, cmd.flags.runArgs...)
	} else {
		args = append(args, cmd.flags.otherArgs...)
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
		fmt.Fprintf(cmd.Stdout, "# pid %d\n", serverCmd.Process.Pid)
	}

	// setup address to connect to
	if cmd.flags.mode.connect && cmd.flags.address != "" {
		debug.ServerNetwork = "tcp"
		debug.ServerAddress = cmd.flags.address
		fmt.Fprintf(cmd.Stdout, "# connect: %s, %s\n",
			debug.ServerNetwork,
			debug.ServerAddress)
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
	_, rest := goutil.ExtractSrcDir(filename)
	return filepath.Join(cmd.tmpDir, "src", rest)
}

//------------

func (cmd *Cmd) environ() []string {
	env := os.Environ()

	// add tmpdir to gopath to allow the compiler to give priority to the annotated files
	gopath := []string{}
	gopath = append(gopath, cmd.tmpDir)
	// add already defined gopath
	gopath = append(gopath, goutil.GoPath()...)
	// build gopath string
	s := "GOPATH=" + strings.Join(gopath, string(os.PathListSeparator))
	env = append(env, s)

	// add cmd line env vars
	for _, s := range cmd.flags.env {
		env = append(env, s)
	}

	return env
}

//------------

func (cmd *Cmd) Cleanup() {
	// cleanup unix socket in case of bad stop
	if debug.ServerNetwork == "unix" {
		_ = os.Remove(debug.ServerAddress)
	}

	// don't cleanup
	if cmd.flags.work {
		return
	}

	if cmd.tmpDir != "" {
		_ = os.RemoveAll(cmd.tmpDir)
	}
	if cmd.tmpBuiltFile != "" && !cmd.flags.mode.build {
		_ = os.Remove(cmd.tmpBuiltFile)
	}
}

//------------

func (cmd *Cmd) build(ctx context.Context, filename string) (string, error) {
	filenameOut := replaceExt(filename, "_godebug")
	args := []string{
		osutil.GoExec(), "build",
		"-o", filenameOut,
		filename,
	}
	dir := filepath.Dir(filenameOut)
	err := cmd.runCmd(ctx, dir, args, cmd.environ())
	return filenameOut, err
}

func (cmd *Cmd) buildTest(ctx context.Context, filename string) (string, error) {
	filenameOut := replaceExt(filename, "_godebug")
	args := []string{
		osutil.GoExec(), "test",
		"-c", // compile binary but don't run
		// "-toolexec", "", // don't run asm // TODO: faster dummy pre-builts?
		"-o", filenameOut,
	}

	// append otherargs in test mode
	args = append(args, cmd.flags.otherArgs...)

	dir := filepath.Dir(filenameOut)
	err := cmd.runCmd(ctx, dir, args, cmd.environ())
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
	ecmd := exec.CommandContext(ctx, cargs[0], cargs[1:]...)

	ecmd.Env = env
	ecmd.Dir = dir
	ecmd.Stdout = cmd.Stdout
	ecmd.Stderr = cmd.Stderr
	osutil.SetupExecCmdSysProcAttr(ecmd)

	if err := ecmd.Start(); err != nil {
		return nil, err
	}

	// ensure kill to child processes on context cancel
	// the ctx must be cancelable, otherwise it might kill the process on start
	go func() {
		select {
		case <-ctx.Done():
			_ = osutil.KillExecCmd(ecmd)
		}
	}()

	return ecmd, nil
}

//------------

func (cmd *Cmd) annotateDirs(ctx context.Context) error {
	seen := map[string]bool{}
	for _, d := range cmd.flags.dirs {
		if seen[d] {
			continue
		}
		seen[d] = true
		if err := cmd.annotateDir(d); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Cmd) annotateDir(dir string) error {
	// if dir is not absolute, check if exists in cmd.dir
	if !filepath.IsAbs(dir) {
		//fmt.Printf("** %v + %v\n", cmd.Dir, dir)
		t := filepath.Join(cmd.Dir, dir)
		fi, err := os.Stat(t)
		if err == nil {
			if fi.IsDir() {
				dir = t
			}
		}
	}

	// dir files
	dir2, _, names, err := goutil.PkgFilenames(dir, true)
	if err != nil {
		return err
	}
	// annotate files
	for _, name := range names {
		filename := filepath.Join(dir2, name)
		if err := cmd.annotateFile(filename, nil); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Cmd) annotateFile(filename string, src interface{}) error {
	astFile, err := cmd.ann.AnnotateFile(filename, src)
	if err != nil {
		return err
	}
	return cmd.writeAstFileToTmpDir(astFile)
}

//------------

func (cmd *Cmd) writeAstFileToTmpDir(astFile *ast.File) error {
	// filename
	tokFile := cmd.ann.FSet.File(astFile.Package)
	if tokFile == nil {
		return fmt.Errorf("unable to get pos token file")
	}
	filename := tokFile.Name()

	// create path directories in destination
	destFilename := cmd.tmpDirBasedFilename(filename)
	if err := os.MkdirAll(filepath.Dir(destFilename), 0770); err != nil {
		return err
	}

	// write file
	f, err := os.Create(destFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	return cmd.ann.Print(f, astFile)
}

func (cmd *Cmd) writeConfigFileToTmpDir() error {
	// content
	src, filename := cmd.ann.ConfigSource()

	// create path directories in destination
	filenameAtTmp := cmd.tmpDirBasedFilename(filename)
	if err := os.MkdirAll(filepath.Dir(filenameAtTmp), 0770); err != nil {
		return err
	}

	// write
	return ioutil.WriteFile(filenameAtTmp, []byte(src), os.ModePerm)
}

func (cmd *Cmd) writeTestMainFilesToTmpDir() error {
	u := cmd.ann.TestMainSources()
	for i, tms := range u {
		name := fmt.Sprintf("godebug_testmain%v_test.go", i)
		filename := filepath.Join(tms.Dir, name)

		// create path directories in destination
		filenameAtTmp := cmd.tmpDirBasedFilename(filename)
		if err := os.MkdirAll(filepath.Dir(filenameAtTmp), 0770); err != nil {
			return err
		}

		// write
		if err := ioutil.WriteFile(filenameAtTmp, []byte(tms.Src), os.ModePerm); err != nil {
			return err
		}
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
	work := f.Bool("work", false, "print workdir and don't cleanup on exit")
	dirs := f.String("dirs", "", "comma-separated list of directories")
	files := f.String("files", "", "comma-separated list of files to avoid annotating big directories")

	if err := f.Parse(args); err != nil {
		if err == flag.ErrHelp {
			f.SetOutput(cmd.Stderr)
			f.PrintDefaults()
			return true, nil
		}
		return true, err
	}

	cmd.flags.work = *work
	cmd.flags.dirs = splitCommaList(*dirs)
	cmd.flags.files = splitCommaList(*files)
	cmd.flags.otherArgs = f.Args()

	if len(cmd.flags.otherArgs) > 0 {
		cmd.flags.filename = cmd.flags.otherArgs[0]
		cmd.flags.otherArgs = cmd.flags.otherArgs[1:]
	}

	return false, nil
}

func (cmd *Cmd) parseTestArgs(args []string) (done bool, _ error) {
	f := &flag.FlagSet{}
	work := f.Bool("work", false, "print workdir and don't cleanup on exit")
	dirs := f.String("dirs", "", "comma-separated list of directories")
	files := f.String("files", "", "comma-separated list of files to avoid annotating big directories")
	run := f.String("run", "", "run test")
	verbose := f.Bool("v", false, "verbose")

	if err := f.Parse(args); err != nil {
		if err == flag.ErrHelp {
			f.SetOutput(cmd.Stderr)
			f.PrintDefaults()
			return true, nil
		}
		return true, err
	}

	cmd.flags.work = *work
	cmd.flags.dirs = splitCommaList(*dirs)
	cmd.flags.files = splitCommaList(*files)
	cmd.flags.otherArgs = f.Args()

	// set test run flag at other flags to pass to the test exec
	if *run != "" {
		a := []string{"-test.run", *run}
		cmd.flags.runArgs = append(a, cmd.flags.runArgs...)
	}

	// verbose
	if *verbose {
		a := []string{"-test.v"}
		cmd.flags.runArgs = append(a, cmd.flags.runArgs...)
	}

	return false, nil
}

func (cmd *Cmd) parseBuildArgs(args []string) (done bool, _ error) {
	f := &flag.FlagSet{}
	work := f.Bool("work", false, "print workdir and don't cleanup on exit")
	dirs := f.String("dirs", "", "comma-separated list of directories")
	files := f.String("files", "", "comma-separated list of files to avoid annotating big directories")
	addr := f.String("addr", "", "address to serve from, built into the binary")
	env := f.String("env", "", "set env variables for build, separated by comma (ex: \"GOOS=linux,...\"'")

	if err := f.Parse(args); err != nil {
		if err == flag.ErrHelp {
			f.SetOutput(cmd.Stderr)
			f.PrintDefaults()
			return true, nil
		}
		return true, err
	}

	cmd.flags.work = *work
	cmd.flags.dirs = splitCommaList(*dirs)
	cmd.flags.files = splitCommaList(*files)
	cmd.flags.address = *addr
	cmd.flags.env = strings.Split(*env, ",")
	cmd.flags.otherArgs = f.Args()
	if len(cmd.flags.otherArgs) > 0 {
		cmd.flags.filename = cmd.flags.otherArgs[0]
		cmd.flags.otherArgs = cmd.flags.otherArgs[1:]
	}

	return false, nil
}

func (cmd *Cmd) parseConnectArgs(args []string) (done bool, _ error) {
	f := &flag.FlagSet{}
	addr := f.String("addr", "", "address to connect to, built into the binary")

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

func cmdUsage() string {
	return `Usage:
	GoDebug <command> [arguments]
The commands are:
	run		build and run go program with debug data
	test		test packages compiled with debug data
	build 	build binary with debug data (allows remote debug)
	connect	connect to a binary built with debug data (allows remote debug)
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

func (cmd *Cmd) populateParentDirectories() (err error) {
	// don't populate annotated files
	noPop := map[string]bool{}
	cmd.ann.FSet.Iterate(func(f *token.File) bool {
		d := f.Name()
		noPop[d] = true
		return true
	})

	// populate parent directories
	vis := map[string]bool{}
	cmd.ann.FSet.Iterate(func(f *token.File) bool {
		d := filepath.Dir(f.Name())
		err = cmd.populateDir(d, vis, noPop)
		if err != nil {
			return false
		}
		return true
	})
	return err
}

func (cmd *Cmd) populateDir(dir string, vis, noPop map[string]bool) error {
	// don't populate an already visited dir
	if _, ok := vis[dir]; ok {
		return nil
	}
	vis[dir] = true

	// visit only up to srcdir
	srcDir, _ := goutil.ExtractSrcDir(dir)
	if len(srcDir) <= 1 || strings.Index(dir, srcDir) < 0 {
		return nil
	}

	// populate: copy go files
	filenames := dirGoFiles(dir)
	for _, f := range filenames {
		if _, ok := noPop[f]; ok {
			continue
		}
		fAtTmp := cmd.tmpDirBasedFilename(f)
		if err := copyFile(f, fAtTmp); err != nil {
			return err
		}
	}

	// visit parent dir
	pd := filepath.Dir(dir)
	return cmd.populateDir(pd, vis, noPop)
}

//------------

func copyFile(src, dst string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer to.Close()
	_, err = io.Copy(to, from)
	return err
}

//------------

func dirGoFiles(dir string) []string {
	var a []string
	fis, err := ioutil.ReadDir(dir)
	if err == nil {
		for _, fi := range fis {
			if filepath.Ext(fi.Name()) == ".go" {
				a = append(a, filepath.Join(dir, fi.Name()))
			}
		}
	}
	return a
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
	if !strings.HasPrefix(filename, "./") {
		return "./" + filename
	}
	return filename
}

//------------

func splitCommaList(val string) []string {
	a := strings.Split(val, ",")
	seen := make(map[string]bool)
	u := []string{}
	for _, s := range a {
		// don't add empty strings
		s := strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// don't add repeats
		if seen[s] {
			continue
		}
		seen[s] = true

		u = append(u, s)
	}
	return u
}

//------------

func setupServerNetAddr(addr string) {
	if addr != "" {
		debug.ServerNetwork = "tcp"
		debug.ServerAddress = addr
		return
	}

	// generate address: allows multiple editors to run debug sessions at the same time.

	seed := time.Now().UnixNano() + int64(os.Getpid())
	ra := rand.New(rand.NewSource(seed))
	r := ra.Intn(10000)

	switch runtime.GOOS {
	case "linux":
		debug.ServerNetwork = "unix"
		p := "editor_godebug.sock" + fmt.Sprintf("%v", r)
		debug.ServerAddress = filepath.Join(os.TempDir(), p)
	default:
		debug.ServerNetwork = "tcp"
		p := fmt.Sprintf("%v", 30071+r)
		debug.ServerAddress = "127.0.0.1:" + p
	}
}
