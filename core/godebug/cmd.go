package godebug

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/astut"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/pathutil"
	"golang.org/x/mod/modfile"
)

// The godebug/debug pkg is writen to a tmp dir and used with the pkg path "godebugconfig/debug" to avoid clashes when self debugging. A config.go file is added with the annotation data. The godebug/debug pkg is included in the editor binary via //go:embed directive.
var debugPkgPath = "godebugconfig/debug"

//----------

type Cmd struct {
	Dir string // running directory

	flags      flags
	gopathMode bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	tmpDir           string
	tmpBuiltFile     string // godebug file built
	tmpGoModFilename string

	mainFuncFilename string // set at annotation time

	fset   *token.FileSet
	env    []string // set at start
	annset *AnnotatorSet

	debugPkgDir      string
	alternativeGoMod string
	overlayFilename  string
	overlay          map[string]string // orig->new

	Client *Client
	start  struct {
		network    string
		address    string
		cancel     context.CancelFunc
		serverWait func() error // annotated program; can be nil
		filesData  *debug.FilesDataMsg
	}
}

func NewCmd() *Cmd {
	cmd := &Cmd{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	cmd.fset = token.NewFileSet()
	cmd.annset = NewAnnotatorSet(cmd.fset)
	return cmd
}

//------------

func (cmd *Cmd) printf(f string, a ...interface{}) (int, error) {
	return fmt.Fprintf(cmd.Stdout, "# "+f, a...)
}
func (cmd *Cmd) logf(f string, a ...interface{}) (int, error) {
	if cmd.flags.verbose {
		f = strings.TrimRight(f, "\n") + "\n" // ensure one newline
		return cmd.printf(f, a...)
	}
	return 0, nil
}
func (cmd *Cmd) Error(err error) {
	cmd.printf("error: %v\n", err)
}

//------------

func (cmd *Cmd) Start(ctx context.Context, args []string) (bool, error) {
	if err := cmd.start2(ctx, args); err != nil {
		return true, err
	}
	if cmd.flags.mode.build {
		return true, nil
	}
	return false, nil
}
func (cmd *Cmd) start2(ctx context.Context, args []string) error {
	defer cmd.cleanupAfterStart()

	cmd.logf("dir=%v\n", cmd.Dir)
	cmd.logf("testmode=%v\n", cmd.flags.mode.test)

	if err := cmd.neededGoVersion(); err != nil {
		return err
	}

	// use absolute dir
	dir0, err := filepath.Abs(cmd.Dir)
	if err != nil {
		return err
	}
	cmd.Dir = dir0

	// read flags
	cmd.flags.stderr = cmd.Stderr
	if err := cmd.flags.parseArgs(args); err != nil {
		return err
	}

	// setup environment
	cmd.env = goutil.OsAndGoEnv(cmd.Dir)
	cmd.env = osutil.SetEnvs(cmd.env, cmd.flags.env)

	if err := cmd.detectGopathMode(cmd.env); err != nil {
		return err
	}

	// REVIEW
	// depends on: gopathMode, tmpDir
	//cmd.env = cmd.setGoPathEnv(cmd.env)

	// depends on cmd.flags.work
	if err := cmd.setupTmpDir(); err != nil {
		return err
	}

	if err := cmd.setupNetworkAddress(); err != nil {
		return err
	}

	m := &cmd.flags.mode
	if m.run || m.test || m.build {
		if err := cmd.build(ctx); err != nil {
			return err
		}
	}

	switch {
	case m.build:
		// inform the address used in the binary
		cmd.printf("build: %v (builtin address: %v, %v)\n", cmd.tmpBuiltFile, cmd.start.network, cmd.start.address)
		return nil
	case m.run || m.test:
		return cmd.startServerClient(ctx)
	case m.connect:
		return cmd.startClient(ctx)
	default:
		panic(fmt.Sprintf("unhandled mode: %v", m))
	}
}

//----------

func (cmd *Cmd) build(ctx context.Context) error {
	fa := NewFilesToAnnotate(cmd)
	if err := fa.find(ctx); err != nil {
		return err
	}
	if err := cmd.annotateFiles2(ctx, fa); err != nil {
		return err
	}
	if err := cmd.buildDebugPkg(ctx, fa); err != nil {
		return err
	}
	if !cmd.gopathMode {
		if err := cmd.buildAlternativeGoMod(ctx, fa); err != nil {
			return err
		}
	}
	if err := cmd.buildOverlayFile(ctx); err != nil {
		return err
	}

	// DEBUG
	//cmd.printAnnotatedFilesAsts(fa)

	if err := cmd.buildBinary(ctx, fa); err != nil {
		// auto-set work flag to avoid cleanup; allows clicking on failing work files locations
		cmd.flags.work = true

		return err
	}
	return nil
}

func (cmd *Cmd) buildBinary(ctx context.Context, fa *FilesToAnnotate) error {
	outFilename, err := cmd.buildOutFilename(fa)
	if err != nil {
		return err
	}
	cmd.tmpBuiltFile = outFilename

	// build args
	a := []string{osutil.ExecName("go")}
	if cmd.flags.mode.test {
		a = append(a, "test")
		a = append(a, "-c") // compile binary but don't run
		//a = append(a, "-vet=off")
	} else {
		a = append(a, "build")
	}
	if cmd.alternativeGoMod != "" {
		a = append(a, "-modfile="+cmd.alternativeGoMod)
	}
	a = append(a, "-overlay="+cmd.overlayFilename)
	a = append(a, "-o="+cmd.tmpBuiltFile)
	a = append(a, cmd.buildArgs()...)
	a = append(a, cmd.flags.unnamedArgs...)

	cmd.logf("build binary: %v\n", a)
	ec := cmd.newCmdI(ctx, a)
	if err := ec.Start(); err != nil {
		return err
	}
	return ec.Wait()
}

//------------

// DEBUG
func (cmd *Cmd) printAnnotatedFilesAsts(fa *FilesToAnnotate) {
	for orig := range cmd.overlay {
		astFile, ok := fa.filesAsts[orig]
		if ok {
			astut.PrintNode(cmd.annset.fset, astFile)
		}
	}
}

//------------

func (cmd *Cmd) startServerClient(ctx context.Context) error {
	// server/client context to cancel the other when one of them ends
	ctx2, cancel := context.WithCancel(ctx)
	cmd.start.cancel = cancel

	if err := cmd.startServer(ctx2); err != nil {
		return err
	}
	return cmd.startClient(ctx2)
}
func (cmd *Cmd) cancelStart() {
	if cmd.start.cancel != nil {
		cmd.start.cancel()
	}
}
func (cmd *Cmd) startServer(ctx context.Context) error {
	return cmd.runBinary(ctx)
}
func (cmd *Cmd) runBinary(ctx context.Context) error {
	// args of the built binary to run (annotated program)
	args := []string{}
	if cmd.flags.toolExec != "" {
		args = append(args, cmd.flags.toolExec)
	}
	args = append(args, cmd.tmpBuiltFile)
	args = append(args, cmd.flags.binaryArgs...)

	// callback func to print process id and args
	cb := func(cmdi osutil.CmdI) {
		cmd.printf("pid %d: %v\n", cmdi.Cmd().Process.Pid, args)
	}

	// run the annotated program
	ci := cmd.newCmdI(ctx, args)
	ci = osutil.NewCallbackOnStartCmd(ci, cb)
	if err := ci.Start(); err != nil {
		cmd.cancelStart()
		return err
	}

	//waitErr := error(nil)
	//wg := sync.WaitGroup{}
	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	waitErr = ec.Wait()
	//}()

	//log.Println("server started")
	cmd.start.serverWait = func() error {
		//defer log.Println("server wait done")
		//wg.Wait()
		//return waitErr

		return ci.Wait()
	}
	return nil
}
func (cmd *Cmd) startClient(ctx context.Context) error {
	// blocks until connected
	client, err := NewClient(ctx, cmd.start.network, cmd.start.address)
	if err != nil {
		//log.Println("client ended")
		cmd.cancelStart()
		if cmd.start.serverWait != nil {
			cmd.start.serverWait()
		}
		return err
	}
	cmd.Client = client

	// set deadline for the starting protocol
	deadline := time.Now().Add(8 * time.Second)
	cmd.Client.Conn.SetWriteDeadline(deadline)
	defer cmd.Client.Conn.SetWriteDeadline(time.Time{}) // clear

	// starting protocol
	if err := cmd.requestFilesData(); err != nil {
		return err
	}
	// wait for filesdata
	msg, ok := <-cmd.Client.Messages
	if !ok {
		return fmt.Errorf("clients msgs chan closed")
	}
	if fd, ok := msg.(*debug.FilesDataMsg); !ok {
		return fmt.Errorf("unexpected msg: %#v", msg)
	} else {
		cmd.start.filesData = fd
	}
	// request start
	if err := cmd.requestStart(); err != nil {
		return err
	}
	return nil
}

func (cmd *Cmd) Wait() error {
	defer cmd.cleanupAfterWait()
	defer cmd.cancelStart()
	err := error(nil)
	if cmd.start.serverWait != nil { // might be nil (ex: connect mode)
		err = cmd.start.serverWait()
	}
	if cmd.Client != nil { // might be nil (ex: server failed to start)
		cmd.Client.Wait()
	}
	return err
}

//------------

func (cmd *Cmd) Messages() chan interface{} {
	return cmd.Client.Messages
}
func (cmd *Cmd) FilesData() *debug.FilesDataMsg {
	return cmd.start.filesData
}

//------------

func (cmd *Cmd) requestFilesData() error {
	msg := &debug.ReqFilesDataMsg{}
	encoded, err := debug.EncodeMessage(msg)
	if err != nil {
		return err
	}
	_, err = cmd.Client.Conn.Write(encoded)
	return err
}

func (cmd *Cmd) requestStart() error {
	msg := &debug.ReqStartMsg{}
	encoded, err := debug.EncodeMessage(msg)
	if err != nil {
		return err
	}
	_, err = cmd.Client.Conn.Write(encoded)
	return err
}

//------------

//func (cmd *Cmd) tmpDirBasedFilename(filename string) string {
//	// remove volume name
//	v := filepath.VolumeName(filename)
//	if len(v) > 0 {
//		filename = filename[len(v):]
//	}

//	if cmd.gopathMode {
//		// trim filename when inside a src dir
//		rhs := trimAtFirstSrcDir(filename)
//		return filepath.Join(cmd.tmpDir, "src", rhs)
//	}

//	// just replicate on tmp dir
//	return filepath.Join(cmd.tmpDir, filename)
//}

//------------

//func (cmd *Cmd) setGoPathEnv(env []string) []string {
//	// after cmd.flags.env such that this result won't be overriden

//	s := cmd.fullGoPathStr(env)
//	return osutil.SetEnv(env, "GOPATH", s)
//}

//func (cmd *Cmd) fullGoPathStr(env []string) string {
//	u := []string{} // first has priority, use new slice

//	// add tmpdir for priority to the annotated files
//	if cmd.gopathMode {
//		u = append(u, cmd.tmpDir)
//	}

//	if s := osutil.GetEnv(cmd.flags.env, "GOPATH"); s != "" {
//		u = append(u, s)
//	}

//	// always include default gopath last (includes entry that might not be defined anywhere, needs to be set)
//	u = append(u, goutil.GetGoPath(env)...)

//	return goutil.JoinPathLists(u...)
//}

func (cmd *Cmd) addToGopathStart(dir string) {
	varName := "GOPATH"
	v := osutil.GetEnv(cmd.env, varName)
	sep := ""
	if v != "" {
		sep = string(os.PathListSeparator)
	}
	v2 := dir + sep + v
	cmd.env = osutil.SetEnv(cmd.env, varName, v2)
}

//------------

func (cmd *Cmd) detectGopathMode(env []string) error {
	modsMode, err := cmd.detectModulesMode(env)
	if err != nil {
		return err
	}
	cmd.gopathMode = !modsMode
	cmd.logf("gopathmode=%v\n", cmd.gopathMode)
	return nil
}
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
func (cmd *Cmd) neededGoVersion() error {
	// need go version that supports overlays
	v, err := goutil.GoVersion()
	if err != nil {
		return err
	}
	if parseutil.VersionLessThan(v, "1.16") {
		return fmt.Errorf("need go version >=1.16 that supports -overlay flag")
	}
	return nil
}

//------------

func (cmd *Cmd) cleanupAfterStart() {
	// always remove (written in src dir)
	if cmd.tmpGoModFilename != "" {
		_ = os.Remove(cmd.tmpGoModFilename) // best effort
	}
	// remove dirs
	if cmd.tmpDir != "" && !cmd.flags.work {
		_ = os.RemoveAll(cmd.tmpDir) // best effort
	}
}

func (cmd *Cmd) cleanupAfterWait() {
	// cleanup unix socket in case of bad stop
	if cmd.start.network == "unix" {
		_ = os.Remove(cmd.start.address) // best effort
	}

	if cmd.tmpBuiltFile != "" && !cmd.flags.mode.build {
		_ = os.Remove(cmd.tmpBuiltFile) // best effort
	}
}

//------------

func (cmd *Cmd) mkdirAllWriteAstFile(filename string, astFile *ast.File) error {
	buf := &bytes.Buffer{}

	pcfg := &printer.Config{Tabwidth: 4}

	// by default, don't print with sourcepos since it will only confuse the user. If the original code doesn't compile, the load packages should fail early before getting to output any ast file.
	if cmd.flags.srcLines {
		pcfg.Mode = printer.SourcePos
	}

	if err := pcfg.Fprint(buf, cmd.fset, astFile); err != nil {
		return err
	}
	return mkdirAllWriteFile(filename, buf.Bytes())
}

//------------

func (cmd *Cmd) setupTmpDir() error {
	fixedDir := filepath.Join(os.TempDir(), "editor_godebug")
	if err := iout.MkdirAll(fixedDir); err != nil {
		return err
	}
	dir, err := ioutil.TempDir(fixedDir, "work*")
	if err != nil {
		return err
	}
	cmd.tmpDir = dir

	// print tmp dir if work flag is present
	if cmd.flags.work {
		cmd.printf("tmpDir: %v\n", cmd.tmpDir)
	}
	return nil
}

//------------

func (cmd *Cmd) buildArgs() []string {
	u := []string{}
	u = append(u, envGodebugBuildFlags(cmd.env)...)
	u = append(u, cmd.flags.unknownArgs...)
	return u
}

//------------

func (cmd *Cmd) setupNetworkAddress() error {
	// can't consider using stdin/out since the program could use it

	if cmd.flags.address != "" {
		cmd.start.network = "tcp"
		cmd.start.address = cmd.flags.address
		return nil
	}

	// OS target to choose how to connect
	goOs := osutil.GetEnv(cmd.env, "GOOS")
	if goOs == "" {
		goOs = runtime.GOOS
	}

	switch goOs {
	case "linux":
		cmd.start.network = "unix"
		cmd.start.address = filepath.Join(cmd.tmpDir, "godebug.sock")
	default:
		port, err := osutil.GetFreeTcpPort()
		if err != nil {
			return err
		}
		cmd.start.network = "tcp"
		cmd.start.address = fmt.Sprintf("127.0.0.1:%v", port)
	}
	return nil
}

//------------

func (cmd *Cmd) annotateFiles2(ctx context.Context, fa *FilesToAnnotate) error {
	// annotate files
	handledMain := false
	cmd.overlay = map[string]string{}
	mainName := mainFuncName(fa.cmd.flags.mode.test)
	for filename := range fa.toAnnotate {
		astFile, ok := fa.filesAsts[filename]
		if !ok {
			return fmt.Errorf("missing ast file: %v", filename)
		}

		// annotate
		ti := (*types.Info)(nil)
		pkg, ok := fa.filesPkgs[filename]
		if ok {
			ti = pkg.TypesInfo
		}
		if err := cmd.annset.AnnotateAstFile(astFile, ti, fa.nodeAnnTypes); err != nil {
			return err
		}

		// setup main ast with debug.exitserver
		if fd, ok := findFuncDeclWithBody(astFile, mainName); ok {
			handledMain = true
			cmd.mainFuncFilename = filename
			cmd.annset.setupDebugExitInFuncDecl(fd, astFile)
		}
	}

	if !handledMain {
		if !cmd.flags.mode.test {
			return fmt.Errorf("main func not handled")
		}
		// insert testmains in "*_test.go" files
		seen := map[string]bool{}
		for filename := range fa.toAnnotate {
			if !strings.HasSuffix(filename, "_test.go") {
				continue
			}

			// one testmain per dir
			dir := filepath.Dir(filename)
			if seen[dir] {
				continue
			}
			seen[dir] = true

			astFile, ok := fa.filesAsts[filename]
			if !ok {
				continue
			}
			if err := cmd.annset.insertTestMain(astFile); err != nil {
				return err
			}

			// use dir of the first file // TODO: just use current dir?
			if cmd.mainFuncFilename == "" {
				dir := filepath.Dir(filename)
				cmd.mainFuncFilename = filepath.Join(dir, "testmain")
			}
		}
	}

	for filename := range fa.toAnnotate {
		astFile, ok := fa.filesAsts[filename]
		if !ok {
			return fmt.Errorf("missing ast file: %v", filename)
		}

		// encode filename for a flat map
		ext := filepath.Ext(filename)
		base := filepath.Base(filename)
		base = base[:len(base)-len(ext)]
		hash := hashStringN(filename, 12)
		name := fmt.Sprintf("%s_%s%s", base, hash, ext)

		// write annotated files and keep in map for overlay
		filename2 := filepath.Join(cmd.tmpDir, "annotated", name)
		if err := cmd.mkdirAllWriteAstFile(filename2, astFile); err != nil {
			return err
		}

		cmd.overlay[filename] = filename2
	}

	return nil
}

//------------

func (cmd *Cmd) buildOverlayFile(ctx context.Context) error {
	// build entries
	w := []string{}
	for src, dest := range cmd.overlay {
		w = append(w, fmt.Sprintf("%q:%q", src, dest))
	}
	// write overlay file
	src := []byte(fmt.Sprintf("{%q:{%s}}", "Replace", strings.Join(w, ",")))
	cmd.overlayFilename = filepath.Join(cmd.tmpDir, "annotated_overlay.json")
	return mkdirAllWriteFile(cmd.overlayFilename, src)
}

//------------

func (cmd *Cmd) buildDebugPkg(ctx context.Context, fa *FilesToAnnotate) error {
	//// detect if the editor debug pkg is used
	//cmd.selfMode = false
	//selfDebugPkgDir := ""
	//for pkgPath, pkg := range fa.pathsPkgs {
	//	if strings.HasPrefix(pkgPath, editorPkgPath+"/") {
	//		// find dir
	//		f := pkg.GoFiles[0]
	//		k := strings.Index(f, editorPkgPath)
	//		if k >= 0 {
	//			cmd.selfMode = true
	//			selfDebugPkgDir = filepath.Join(f[:k], debugPkgPath)
	//			break
	//		}
	//	}
	//}

	//// setup current files to be empty by default (attempt to discard if debugging an old version of the editor)
	//if cmd.selfMode {
	//	fis, err := ioutil.ReadDir(selfDebugPkgDir)
	//	if err == nil {
	//		for _, fi := range fis {
	//			filename := fsutil.JoinPath(selfDebugPkgDir, fi.Name())
	//			cmd.overlay[filename] = ""
	//		}
	//	}
	//}

	// target dir
	cmd.debugPkgDir = filepath.Join(cmd.tmpDir, "debugpkg")
	if cmd.gopathMode {
		cmd.addToGopathStart(cmd.debugPkgDir)
		cmd.debugPkgDir = filepath.Join(cmd.debugPkgDir, "src/"+debugPkgPath)
	}

	// util to add file to debug pkg dir
	writeFile := func(name string, src []byte) error {
		filename2 := filepath.Join(cmd.debugPkgDir, name)

		//if cmd.selfMode {
		//	filename3 := filepath.Join(selfDebugPkgDir, name)
		//	//println("overlay", filename3, filename2)
		//	cmd.overlay[filename3] = filename2
		//}

		return mkdirAllWriteFile(filename2, src)
	}

	// local src pkg dir where the debug pkg is located (io/fs)
	srcDir := "debug"
	des, err := debugPkgFs.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, de := range des {
		// must use path.join since dealing with embedFs
		filename1 := path.Join(srcDir, de.Name())
		if strings.HasSuffix(filename1, "_test.go") {
			continue
		}
		src, err := debugPkgFs.ReadFile(filename1)
		if err != nil {
			return err
		}
		if err := writeFile(de.Name(), src); err != nil {
			return err
		}
	}

	// dynamically create go.mod since go:embed doesn't allow it
	if !cmd.gopathMode {
		src3 := []byte(fmt.Sprintf("module %s\n", debugPkgPath))
		if err := writeFile("go.mod", src3); err != nil {
			return err
		}
	}

	// init() functions declared across multiple files in a package are processed in alphabetical order of the file name. Use name starting with "a" to setup config vars as early as possible.
	configFilename := "aaaconfig.go"

	// build config file
	src4 := cmd.annset.BuildConfigSrc(cmd.start.network, cmd.start.address, &cmd.flags)
	if err := writeFile(configFilename, src4); err != nil {
		return err
	}

	return nil
}

//------------

func (cmd *Cmd) buildAlternativeGoMod(ctx context.Context, fa *FilesToAnnotate) error {
	filename, ok := fa.GoModFilename()
	if !ok {
		//return fmt.Errorf("missing go.mod")

		// in the case of a simple main.go without any go.mod (but in modules mode), it needs to create an artificial go.mod in order to reference the debug pkg that is located in the tmp dir

		// TODO: last resort, having to create files in the src dir is to be avoided -- needs review

		// create temporary go.mod in src dir based on main file
		if cmd.mainFuncFilename == "" {
			return fmt.Errorf("missing main func filename")
		}
		dir := filepath.Dir(cmd.mainFuncFilename)
		fname2 := filepath.Join(dir, "go.mod")
		// must not exist
		if _, err := os.Stat(fname2); !os.IsNotExist(err) {
			return fmt.Errorf("file should not exist because gomodfilename didn't found it: %v", fname2)
		}
		// create
		src := []byte("module main\n")
		if err := mkdirAllWriteFile(fname2, src); err != nil {
			return err
		}
		cmd.tmpGoModFilename = fname2
		cmd.logf("tmpgomodfilename: %v\n", cmd.tmpGoModFilename)
		filename = fname2
	}

	// build based on current go.mod
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read mod file: %w", err)
	}
	mf, err := modfile.ParseLax(filename, src, nil)
	if err != nil {
		return err
	}

	if cmd.flags.usePkgLinks {
		if err := cmd.buildPkgLinks(mf, fa); err != nil {
			return err
		}
	}

	// include debug pkg require/replace lines
	mf.AddNewRequire(debugPkgPath, "v0.0.0", false)
	mf.AddReplace(debugPkgPath, "", cmd.debugPkgDir, "")

	src2, err := mf.Format()
	if err != nil {
		return err
	}
	cmd.alternativeGoMod = filepath.Join(cmd.tmpDir, "alternative.mod")
	if err := mkdirAllWriteFile(cmd.alternativeGoMod, src2); err != nil {
		return err
	}

	// REVIEW: commented: using overlay for go.mod, so the original go.sum should be used(?)
	//// copy as well go.sum or it will fail, just need a best effort
	//dir := filepath.Dir(filename)
	//gosum := filepath.Join(dir, "go.sum")
	//gosumDst := pathutil.ReplaceExt(cmd.alternativeGoMod, ".sum")
	//_ = copyFile(gosum, gosumDst)

	cmd.overlay[filename] = cmd.alternativeGoMod
	////cmd.overlay[gosum] = gosumDst
	cmd.alternativeGoMod = "" // disable (using overlay)

	return nil
}

func (cmd *Cmd) buildPkgLinks(mf *modfile.File, fa *FilesToAnnotate) error {

	// old pkgs don't have a go.mod file, and reside in another location. After go1.19, the compiler is not detecting the existence of inserted pkgs in their special/generated go.mod files. There is a workaround to have the pkg symlinked in a tmpdir, and then use the overlay file with a reference to that symlink.

	linksDir := filepath.Join(cmd.tmpDir, "pkglinks")
	if err := iout.MkdirAll(linksDir); err != nil {
		return err
	}

	linkFilename := func(dir string) string {
		hash := hashStringN(dir, 10)
		base := filepath.Base(dir)
		name := fmt.Sprintf("%s-%s", base, hash)
		return filepath.Join(linksDir, name)
	}

	seen := map[string]bool{}             // seen module
	for filename := range fa.toAnnotate { // all annotated files
		fpkg, ok := fa.filesPkgs[filename]
		if !ok {
			continue
		}

		// module of an annotated file
		afMod := pkgMod(fpkg)
		if afMod == nil || afMod.Dir == "" {
			continue
		}

		// visited already
		if seen[afMod.Dir] {
			continue
		}
		seen[afMod.Dir] = true

		goModIsGenerated := filepath.Dir(afMod.GoMod) != afMod.Dir
		if !goModIsGenerated {
			continue
		}

		// pkg with annotated files without a src go.mod (it was possibly generated)

		// make a link to the package module and use that link dir to bypass the erroneous behaviour introduced by go1.19.x
		ldir := linkFilename(afMod.Dir)
		if err := os.Symlink(afMod.Dir, ldir); err != nil {
			return fmt.Errorf("builddirlinks: %w", err)
		}

		// add replace directive to go.mod
		mf.AddReplace(afMod.Path, "", ldir, "")

		// replace all references in the overlay map to the created link dir
		for oldf, newf := range cmd.overlay {
			dir2 := afMod.Dir + string(filepath.Separator)
			if strings.HasPrefix(oldf, dir2) {
				rest := oldf[len(dir2):]
				name2 := filepath.Join(ldir, rest)
				cmd.overlay[name2] = newf
				delete(cmd.overlay, oldf)
			}
		}

		// add module go.mod in overlay (ex: case of module without a go.mod, but with a built go.mod in a cache dir)
		dir2 := filepath.Dir(afMod.GoMod)
		if dir2 != afMod.Dir {
			filename := filepath.Join(ldir, "go.mod")
			cmd.overlay[filename] = afMod.GoMod
		}
	}
	return nil
}

//------------

func (cmd *Cmd) buildOutFilename(fa *FilesToAnnotate) (string, error) {
	if cmd.flags.outFilename != "" {
		return cmd.flags.outFilename, nil
	}

	if cmd.mainFuncFilename == "" {
		return "", fmt.Errorf("missing main filename")
	}

	// commented: output to tmp dir
	//fname := filepath.Base(cmd.mainFuncFilename)
	//fname = fsutil.JoinPath(cmd.tmpDir, fname)

	// output to main file dir
	fname := cmd.mainFuncFilename
	fname = pathutil.ReplaceExt(fname, "_godebug") // don't use ".godebug", not a file type

	fname = osutil.ExecName(fname)
	return fname, nil
}

//------------

func (cmd *Cmd) newCmdI(ctx context.Context, args []string) osutil.CmdI {
	ec := exec.CommandContext(ctx, args[0], args[1:]...)
	ec.Stdin = cmd.Stdin
	ec.Stdout = cmd.Stdout
	ec.Stderr = cmd.Stderr
	ec.Dir = cmd.Dir
	ec.Env = cmd.env

	ci := osutil.NewCmdI(ec)
	ci = osutil.NewSetSidCmd(ctx, ci)
	ci = osutil.NewShellCmd(ci)
	return ci
}

//------------
//------------
//------------

//go:embed debug/*
var debugPkgFs embed.FS

//------------

func writeFile(filename string, src []byte) error {
	return os.WriteFile(filename, src, 0640)
}
func mkdirAllWriteFile(filename string, src []byte) error {
	return iout.MkdirAllWriteFile(filename, src, 0640)
}

func mkdirAllCopyFile(src, dst string) error {
	return iout.MkdirAllCopyFile(src, dst, 0640)
}
func mkdirAllCopyFileSync(src, dst string) error {
	return iout.MkdirAllCopyFileSync(src, dst, 0640)
}

func copyFile(src, dst string) error {
	return iout.CopyFile(src, dst, 0640)
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

//func trimAtFirstSrcDir(filename string) string {
//	v := filename
//	w := []string{}
//	for {
//		base := filepath.Base(v)
//		if base == "src" {
//			return filepath.Join(w...) // trimmed
//		}
//		w = append([]string{base}, w...)
//		oldv := v
//		v = filepath.Dir(v)
//		isRoot := oldv == v
//		if isRoot {
//			break
//		}
//	}
//	return filename
//}

//----------

// TODO: remove once env vars supported in editor
func envGodebugBuildFlags(env []string) []string {
	bfs := osutil.GetEnv(env, "GODEBUG_BUILD_FLAGS")
	if len(bfs) == 0 {
		return nil
	}
	return strings.Split(bfs, ",")
}
