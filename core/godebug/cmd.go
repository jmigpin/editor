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
	"github.com/jmigpin/editor/util/osexecutil"
)

type Cmd struct {
	Client *Client

	Dir    string // "" will use current dir
	Stdout io.Writer
	Stderr io.Writer

	args    []string
	mainSrc interface{}
	ann     *Annotator

	tmpDir       string
	tmpBuiltFile string // file built and exec'd
	tmpGoPath    bool

	start struct {
		cancel    context.CancelFunc
		waitg     sync.WaitGroup
		serverErr error
	}

	work bool // don't cleanup at the end
}

func NewCmd(args []string, mainSrc interface{}) *Cmd {
	cmd := &Cmd{
		args:    args,
		mainSrc: mainSrc,
		ann:     NewAnnotator(),
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
	return cmd
}

//------------

func (cmd *Cmd) Start(ctx context.Context) error {
	// parse arguments
	pflags, pargs, err := cmd.parseArgs()
	if err != nil {
		return err
	}

	// tmp dir for annotated files
	tmpDir, err := ioutil.TempDir(os.TempDir(), "godebug")
	if err != nil {
		return err
	}
	cmd.tmpDir = tmpDir

	// print tmp dir if work flag is present
	work := flagGet(pflags, "work").(bool)
	if work {
		cmd.work = true
		fmt.Fprintf(cmd.Stdout, "work: %v\n", cmd.tmpDir)
	}

	// mode
	mode := flagGet(pflags, "mode").(string)
	switch mode {
	case "run", "test":
		testMode := mode == "test"
		filename, err := cmd.initMode(ctx, pflags, testMode)
		if err != nil {
			return err
		}
		return cmd.startServerClient(ctx, filename, pargs)
	default:
		return fmt.Errorf("unexpected mode: %v", mode)
	}

}

//------------

func (cmd *Cmd) Wait() error {
	cmd.start.waitg.Wait()
	cmd.start.cancel() // ensure resources are cleared
	return cmd.start.serverErr
}

//------------

func (cmd *Cmd) initMode(ctx context.Context, flags *flag.FlagSet, testMode bool) (string, error) {
	// filename
	var filename string
	if testMode {
		filename = filepath.Join(cmd.getDir(), "pkgtest")
	} else {
		filename = flagGet(flags, "run.filename").(string)
	}

	filenameAtTmp := cmd.tmpDirBasedFilename(filename)

	// pre-build without annotations for better errors (result is ignored)
	if cmd.mainSrc == nil {
		if testMode {
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

	// annotate
	if !testMode {
		if err := cmd.annotateFile(filename, cmd.mainSrc); err != nil {
			return "", err
		}
	}
	if err := cmd.annotateDirs(ctx, flags); err != nil {
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
	if testMode {
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

	// build
	cmd.setupTmpGoPath()
	if testMode {
		return cmd.buildTest(ctx, filenameAtTmp)
	} else {
		return cmd.build(ctx, filenameAtTmp)
	}
}

//------------

func (cmd *Cmd) getDir() string {
	if cmd.Dir == "" {
		if d, err := os.Getwd(); err == nil {
			return d
		}
	}
	return cmd.Dir
}

//------------

func (cmd *Cmd) setupTmpGoPath() {
	// TODO: copy all packages to tmp dir?
	// TODO: reuse tmp dir - check modtime
	// add  tmpdir to gopath to use the files written to tmpdir
	gopath := os.Getenv("GOPATH")
	u := strings.Join([]string{cmd.tmpDir, gopath}, ":")
	os.Setenv("GOPATH", u)

	// flag to cleanup at the end
	cmd.tmpGoPath = true
}

//------------

func (cmd *Cmd) startServerClient(ctx context.Context, filenameOut string, args []string) error {
	// move filenameout to working dir
	filenameWork := filepath.Join(cmd.getDir(), filepath.Base(filenameOut))
	if err := os.Rename(filenameOut, filenameWork); err != nil {
		return err
	}

	// keep moved filename that will run in working dir for later cleanup
	cmd.tmpBuiltFile = filenameWork

	// server/client context to cancel the other when one of them ends
	ctx2, cancel := context.WithCancel(ctx)
	cmd.start.cancel = cancel

	// start server
	filenameWork2 := normalizeFilenameForExec(filenameWork)
	args = append([]string{filenameWork2}, args...)
	serverCmd, err := cmd.startCmd(ctx2, cmd.getDir(), args)
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
	_, rest := gosource.ExtractSrcDir(filename)
	return filepath.Join(cmd.tmpDir, "src", rest)
}

//------------

func (cmd *Cmd) Cleanup() {
	// always cleanup gopath
	if cmd.tmpGoPath {
		gopath := os.Getenv("GOPATH")
		s := cmd.tmpDir + ":"
		if strings.HasPrefix(gopath, s) {
			os.Setenv("GOPATH", gopath[len(s):])
		}
	}

	// don't cleanup
	if cmd.work {
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
		"-tags", "godebug",
		"-o", filenameOut,
		filename,
	}
	dir := filepath.Dir(filenameOut)
	err := cmd.runCmd(ctx, dir, args)
	return filenameOut, err
}

func (cmd *Cmd) buildTest(ctx context.Context, filename string) (string, error) {
	filenameOut := replaceExt(filename, "_godebug_test")
	args := []string{
		"go", "test",
		"-tags", "godebug",
		"-c", // compile binary but don't run
		// "-toolexec", "", // don't run asm // TODO: faster dummy pre-builts?
		"-o", filenameOut,
	}
	dir := filepath.Dir(filenameOut)
	err := cmd.runCmd(ctx, dir, args)
	return filenameOut, err
}

//------------

func (cmd *Cmd) runCmd(ctx context.Context, dir string, args []string) error {
	// ctx with early cancel for startcmd to clear inner goroutine resource
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	ecmd, err := cmd.startCmd(ctx2, dir, args)
	if err != nil {
		return err
	}
	return ecmd.Wait()
}

func (cmd *Cmd) startCmd(ctx context.Context, dir string, args []string) (*exec.Cmd, error) {
	ecmd := exec.CommandContext(ctx, args[0], args[1:]...)
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

func (cmd *Cmd) annotateDirs(ctx context.Context, flags *flag.FlagSet) error {
	dirs := flagGet(flags, "dirs").([]string)
	for _, d := range dirs {
		if err := cmd.annotateDir(d); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Cmd) annotateDir(dir string) error {
	// if dir is not absolute, check if exists in cmd.dir
	if !filepath.IsAbs(dir) {
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
	astFile, err := cmd.ann.ParseAnnotate(filename, src)
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

func (cmd *Cmd) parseArgs() (*flag.FlagSet, []string, error) {
	if len(cmd.args) == 0 {
		return nil, nil, fmt.Errorf("expecting first arg: {run,test}")
	}

	// this flagset is not parsed but only used to keep track of the flags
	flags1 := &flag.FlagSet{}
	_ = flags1.String("mode", "", "") // run, test
	_ = flags1.String("run.filename", "", "")
	_ = flags1.Bool("work", false, "")
	var df stringsFlag
	flags1.Var(&df, "dirs", "")

	// flagset that gets parsed
	flags2 := &flag.FlagSet{}

	// common flags for all modes
	_ = flags2.Bool("work", false, "print workdir and don't cleanup on exit")
	_ = flags2.String("dirs", "", "comma-separated list of directories")

	// mode flags
	mode := cmd.args[0]
	flags1.Set("mode", mode)
	switch mode {
	case "run":
	case "test":
		_ = flags2.String("run", "", "regexp to select test to run")
		_ = flags2.Bool("v", false, "verbose output")
	default:
		return nil, nil, fmt.Errorf("unexpected mode {run,test}: %v", mode)
	}

	// parse without mode argument
	if err := flags2.Parse(cmd.args[1:]); err != nil {
		return nil, nil, err
	}

	// process flags2 into flags1

	flags1.Set("work", fmt.Sprintf("%v", flagGet(flags2, "work").(bool)))

	// dirs
	dirs := flagGet(flags2, "dirs").(string)
	// test.dirs: auto include workingdir in test mode
	if mode == "test" {
		dirs += "," + cmd.getDir()
	}
	if err := flags1.Set("dirs", dirs); err != nil {
		return nil, nil, err
	}

	otherArgs := flags2.Args()

	// run.filename
	if mode == "run" {
		if len(otherArgs) > 0 {
			filename := otherArgs[0]
			otherArgs = otherArgs[1:]

			if cmd.mainSrc == nil {
				// base on workingdir
				if !filepath.IsAbs(filename) {
					filename = filepath.Join(cmd.getDir(), filename)
				}
			}

			flags1.Set("run.filename", filename)
		}
	}

	if mode == "test" {
		// test.run: set test run flag at other flags to pass to the test exec
		s := flagGet(flags2, "run").(string)
		if s != "" {
			otherArgs = append([]string{"-test.run", s}, otherArgs...)
		}

		// test.v
		v := flagGet(flags2, "v").(bool)
		if v {
			otherArgs = append([]string{"-test.v"}, otherArgs...)
		}
	}

	return flags1, otherArgs, nil
}

//------------

func (cmd *Cmd) populateParentDirectories() (err error) {
	fmt.Println("pop par dirs")
	dirs := map[string]bool{}
	cmd.ann.FSet.Iterate(func(f *token.File) bool {
		//fmt.Printf("-> %v\n", f.Name())

		d := filepath.Dir(f.Name())
		if _, ok := dirs[d]; !ok { // visit each dir only once
			dirs[d] = true

			pd := filepath.Dir(d) // parent dir
			err = cmd.populateDir(pd)
			if err != nil {
				return false
			}
		}
		return true
	})
	return err
}

func (cmd *Cmd) populateDir(dir string) error {
	// visit only up to srcdir
	srcDir, _ := gosource.ExtractSrcDir(dir)
	if len(srcDir) <= 1 || strings.Index(dir, srcDir) < 0 {
		return nil
	}

	// copy go files
	//fmt.Printf("pop dir %v\n", dir)
	filenames := dirGoFiles(dir)
	for _, f := range filenames {
		fAtTmp := cmd.tmpDirBasedFilename(f)
		if err := copyFile(f, fAtTmp); err != nil {
			return err
		}
	}

	// visit parent dir
	pd := filepath.Dir(dir)
	return cmd.populateDir(pd)
}

//------------

func copyFile(src, dst string) error {
	//fmt.Printf("copied %v -> %v\n", src, dst)
	//fmt.Printf("copying %v\n", filepath.Base(dst))

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

func flagGet(flags *flag.FlagSet, name string) interface{} {
	f := flags.Lookup(name)
	return f.Value.(flag.Getter).Get()
}

//------------

type stringsFlag []string

func (f *stringsFlag) String() string {
	return fmt.Sprint(*f)
}
func (f *stringsFlag) Get() interface{} {
	var u []string = *f
	return u
}
func (f *stringsFlag) Set(value string) error {
	if len(*f) > 0 {
		return fmt.Errorf("flag already set: newvalue=%v", value)
	}

	// split into slice
	a := strings.Split(value, ",")
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
	*f = u

	return nil
}
