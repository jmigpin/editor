package godebug

import (
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/core/gosource"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/osexecutil"
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
			run  bool
			test bool
		}
		run struct {
			filename string
		}
		test struct {
		}
		work      bool
		dirs      []string
		files     []string
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

	// run
	switch {
	case cmd.flags.mode.run, cmd.flags.mode.test:
		filename, err := cmd.initMode(ctx)
		if err != nil {
			return true, err
		}
		err = cmd.startServerClient(ctx, filename)
		return false, err
	default:
		panic("!")
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

func (cmd *Cmd) initMode(ctx context.Context) (string, error) {
	// filename
	var filename string

	if cmd.flags.mode.test {
		filename = filepath.Join(cmd.Dir, "pkgtest")
	} else {
		filename = cmd.flags.run.filename
		if filename == "" {
			return "", fmt.Errorf("missing filename arg")
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
				return "", err
			}
			os.Remove(fout)
		} else {
			fout, err := cmd.build(ctx, filename)
			if err != nil {
				return "", err
			}
			os.Remove(fout)
		}
	}

	// annotate: main file
	if !cmd.flags.mode.test {
		if err := cmd.annotateFile(filename, cmd.mainSrc); err != nil {
			return "", err
		}
	}

	// annotate: files option
	for _, filename := range cmd.flags.files {
		if err := cmd.annotateFile(filename, nil); err != nil {
			return "", err
		}
	}

	// annotate: auto include working dir in test mode
	if cmd.flags.mode.test {
		cmd.flags.dirs = append(cmd.flags.dirs, cmd.Dir)
	}

	// annotate: directories
	if err := cmd.annotateDirs(ctx); err != nil {
		return "", err
	}

	// write config file after annotations
	if err := cmd.writeConfigFileToTmpDir(); err != nil {
		return "", err
	}

	// populate missing go files in parent directories
	if err := cmd.populateParentDirectories(); err != nil {
		return "", err
	}

	// main/testmain exit
	if cmd.flags.mode.test {
		if !cmd.ann.InsertedExitIn.TestMain {
			if err := cmd.writeTestMainFilesToTmpDir(); err != nil {
				return "", err
			}
		}
	} else {
		if !cmd.ann.InsertedExitIn.Main {
			return "", fmt.Errorf("have not inserted debug exit in main()")
		}
	}

	filenameAtTmp := cmd.tmpDirBasedFilename(filename)

	// create parent dirs
	if err := os.MkdirAll(filepath.Dir(filenameAtTmp), 0755); err != nil {
		return "", err
	}

	// build
	if cmd.flags.mode.test {
		return cmd.buildTest(ctx, filenameAtTmp)
	} else {
		return cmd.build(ctx, filenameAtTmp)
	}
}

//------------

func (cmd *Cmd) startServerClient(ctx context.Context, filenameOut string) error {
	// move filenameout to working dir
	filenameWork := filepath.Join(cmd.Dir, filepath.Base(filenameOut))
	if err := os.Rename(filenameOut, filenameWork); err != nil {
		return err
	}

	// keep moved filename that will run in working dir for later cleanup
	cmd.tmpBuiltFile = filenameWork

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
	serverCmd, err := cmd.startCmd(ctx2, cmd.Dir, args, nil)
	if err != nil {
		cmd.start.cancel() // cmd.Wait() won't be called, need to clear resources
		return err
	}

	// output cmd pid
	fmt.Fprintf(serverCmd.Stdout, "# pid %d\n", serverCmd.Process.Pid)

	// start client (blocking connect)
	client, err := NewClient(ctx2)
	if err != nil {
		cmd.start.cancel() // cmd.Wait() won't be called, need to clear resources
		return err
	}
	cmd.Client = client

	// NOTE: from this point, cmd.Wait() clears resources from cmd.start.cancel

	// server done
	cmd.start.waitg.Add(1)
	go func() {
		defer cmd.start.waitg.Done()
		cmd.start.serverErr = serverCmd.Wait() // wait for server to finish
	}()

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
	gopath := []string{}

	// add tmpdir to gopath to allow the compiler to give priority to the annotated files
	gopath = append(gopath, cmd.tmpDir)

	// add already defined gopath
	gopath = append(gopath, goutil.GoPath()...)

	// build env
	s := "GOPATH=" + strings.Join(gopath, ":")
	env := os.Environ()
	env = append(env, s)
	return env
}

//------------

func (cmd *Cmd) Cleanup() {
	// don't cleanup
	if cmd.flags.work {
		return
	}

	if cmd.tmpDir != "" {
		defer func() { cmd.tmpDir = "" }()
		_ = os.RemoveAll(cmd.tmpDir)
	}
	if cmd.tmpBuiltFile != "" {
		defer func() { cmd.tmpBuiltFile = "" }()
		_ = os.Remove(cmd.tmpBuiltFile)
	}
}

//------------

func (cmd *Cmd) build(ctx context.Context, filename string) (string, error) {
	filenameOut := replaceExt(filename, "_godebug")
	args := []string{
		"go", "build",
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
		"go", "test",
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
	cargs := []string{"sh", "-c", strings.Join(args, " ")}
	ecmd := exec.CommandContext(ctx, cargs[0], cargs[1:]...)

	ecmd.Env = env
	ecmd.Dir = dir
	ecmd.Stdout = cmd.Stdout
	ecmd.Stderr = cmd.Stderr
	osexecutil.SetupExecCmdSysProcAttr(ecmd)

	if err := ecmd.Start(); err != nil {
		return nil, err
	}

	// ensure kill to child processes on context cancel
	// the ctx must be cancelable, otherwise it might kill the process on start
	go func() {
		select {
		case <-ctx.Done():
			_ = osexecutil.KillExecCmd(ecmd)
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
	dir2, _, names, err := gosource.PkgFilenames(dir, true)
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

	defer logger.Printf("write astfile to tmpdir: %v", destFilename)

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
		cmd.flags.run.filename = cmd.flags.otherArgs[0]
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

func cmdUsage() string {
	return `Usage:
	GoDebug <command> [arguments]
The commands are:
	run		compile and run go program with debugging data
	test		test packages compiled with debugging data
Examples:
	GoDebug run main.go
	GoDebug run main.go -- progArg1 progArg2
	GoDebug run --help
	GoDebug run -dirs=./pkg1,./pkg2 main.go
	GoDebug test
	GoDebug test -run mytest
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
