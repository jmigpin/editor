package godebug

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/astut"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/mathutil"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/pathutil"
	"golang.org/x/mod/modfile"
)

//go:embed debug/*
var debugPkgFs embed.FS

//----------

type Cmd struct {
	Dir string // running directory

	CmdLineMode bool
	Testing     bool // not the same as flags.mode.test

	flags      Flags
	gopathMode bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	tmpDir           string
	tmpBuiltFile     string // godebug file built
	tmpGoModFilename string

	mainFuncFilename string // set at annotation time

	env []string // set at start

	fa     *FilesToAnnotate
	fset   *token.FileSet // TODO: reset for gc
	annset *AnnotatorSet  // TODO: reset for gc

	debugPkgDir      string
	alternativeGoMod string
	overlayFilename  string
	overlay          map[string]string // orig->new

	start struct {
		proto         debug.Proto // check ProtoRead()
		execSideCmd   osutil.CmdI
		cleanupCancel context.CancelFunc
	}
}

func NewCmd() *Cmd {
	cmd := &Cmd{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	cmd.fa = NewFilesToAnnotate(cmd)
	cmd.fset = token.NewFileSet()
	cmd.annset = NewAnnotatorSet(cmd.fset)
	return cmd
}

//------------

func (cmd *Cmd) printf(f string, a ...any) (int, error) {
	return fmt.Fprintf(cmd.Stderr, "# "+f, a...)
}
func (cmd *Cmd) logf(f string, a ...any) (int, error) {
	if cmd.flags.verbose {
		f = strings.TrimRight(f, "\n") + "\n" // ensure one newline
		return cmd.printf(f, a...)
	}
	return 0, nil
}

//------------

// allow direct external access to this feature (ex: godebug -h)
func (cmd *Cmd) ParseFlagsOnce(args []string) error {
	cmd.flags.stderr = cmd.Stderr
	return cmd.flags.parseArgsOnce(args)
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

	// parse flags
	if err := cmd.ParseFlagsOnce(args); err != nil {
		return err
	}

	if cmd.CmdLineMode {
		if cmd.flags.mode.run || cmd.flags.mode.test {
			return fmt.Errorf("mode not available in cmd line: %q", args[0])
		}
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
		cmd.printBuildInfo()
		return nil
	case m.run || m.test:
		return cmd.start3(ctx)
	case m.connect:
		return cmd.startEditorSide(ctx)
	default:
		panic(fmt.Sprintf("unhandled mode: %v", m))
	}
}

func (cmd *Cmd) start3(ctx context.Context) error {
	ctx2, cancel := context.WithCancel(ctx)
	if err := cmd.start4(ctx2); err != nil {
		cancel() // possibly cancel the started exec
		return err
	}
	// keep to clear resources later
	cmd.start.cleanupCancel = cancel
	return nil
}
func (cmd *Cmd) start4(ctx context.Context) error {
	// start exec cmd first since the editor side will block
	if cmd.flags.startExec {
		if err := cmd.startExecSide(ctx); err != nil {
			return err
		}
	} else {
		cmd.printBuildInfo()
		cmd.printf("waiting for connect (exec side not started)\n")
	}

	return cmd.startEditorSide(ctx) // blocks
}

//----------

func (cmd *Cmd) startEditorSide(ctx context.Context) error {
	// use timeout: ex: exec side started but exited early with a crash
	if !cmd.flags.mode.connect || cmd.Testing {
		timeout := 30 * time.Second
		if cmd.Testing {
			timeout = 500 * time.Millisecond
		}
		ctx = context.WithValue(ctx, "connectTimeout", timeout)
	}

	addr := debug.NewAddrI(cmd.flags.network, cmd.flags.address)

	logw := io.Writer(nil)
	if !cmd.flags.noDebugMsg {
		logw = debug.NewPrefixWriter(cmd.Stderr, "# godebug.editor: ")
	}

	peds := &debug.ProtoEditorSide{}
	//peds.Logger = debug.Logger{"peds: ", stdout} // DEBUG: lots of output

	p, err := debug.NewProto(ctx, addr, peds, cmd.flags.editorIsServer, cmd.flags.continueServing, logw)
	if err != nil {
		return err
	}
	cmd.start.proto = p
	return nil
}

//------------

func (cmd *Cmd) startExecSide(ctx context.Context) error {
	// args of the built binary to run (annotated program)
	args := []string{}
	if cmd.flags.toolExec != "" {
		args = append(args, cmd.flags.toolExec)
	}
	args = append(args, cmd.tmpBuiltFile)
	args = append(args, cmd.flags.execArgs...)

	// callback func to print process id and args
	cb := func(cmdi osutil.CmdI) {
		cmd.printf("pid %d: %v\n", cmdi.Cmd().Process.Pid, args)
	}

	// run the annotated program
	ci := cmd.newCmdI(ctx, args)
	ci = osutil.NewPausedWritersCmd(ci, cb)
	if err := ci.Start(); err != nil {
		return err
	}
	cmd.start.execSideCmd = ci
	return nil
}

//------------

func (cmd *Cmd) Wait() error {
	w := []error{}

	if cmd.start.execSideCmd != nil {
		err := cmd.start.execSideCmd.Wait()
		w = append(w, err)
	}

	if cmd.start.proto != nil {
		if err := cmd.start.proto.CloseOrWait(); err != nil {
			w = append(w, err)
		}
	}

	cmd.cleanupAfterWait()
	return errors.Join(w...)
}

//------------

func (cmd *Cmd) ProtoRead() (any, error) {
	v := (any)(nil)
	err := cmd.start.proto.Read(&v)
	return v, err
}

//----------

func (cmd *Cmd) build(ctx context.Context) error {
	if err := cmd.fa.find(ctx); err != nil {
		return err
	}
	if err := cmd.annotateFiles2(ctx); err != nil {
		return err
	}
	if err := cmd.buildDebugPkg(ctx); err != nil {
		return err
	}
	if err := cmd.buildAlternativeGoMod(ctx); err != nil {
		return err
	}
	if err := cmd.buildOverlayFile(ctx); err != nil {
		return err
	}

	// DEBUG
	//cmd.printAnnotatedFilesAsts(cmd.fa)

	if err := cmd.build2(ctx); err != nil {
		// auto-set work flag to avoid cleanup; allows clicking on failing work files locations
		cmd.flags.work = true

		return err
	}
	return nil
}
func (cmd *Cmd) build2(ctx context.Context) error {
	outFilename, err := cmd.buildOutFilename(cmd.fa)
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
	// allow garbage collect
	cmd.fa = nil
	cmd.fset = nil
	cmd.annset = nil

	// always remove (written in src dir)
	if cmd.tmpGoModFilename != "" {
		_ = os.Remove(cmd.tmpGoModFilename) // best effort
	}
}

func (cmd *Cmd) cleanupAfterWait() {
	if cmd.start.cleanupCancel != nil {
		cmd.start.cleanupCancel()
	}

	// remove dirs (can/used-to be done at "afterstart")
	if cmd.tmpDir != "" && !cmd.flags.work {
		_ = os.RemoveAll(cmd.tmpDir) // best effort
	}

	// cleanup unix socket in case of bad stop
	if cmd.flags.network == "unix" {
		_ = os.Remove(cmd.flags.address) // best effort
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

func (cmd *Cmd) editorRootTmpDir() string {
	fixedDir := filepath.Join(os.TempDir(), "editor_godebug")
	_ = iout.MkdirAll(fixedDir) // best effort
	return fixedDir
}

func (cmd *Cmd) setupTmpDir() error {
	fixedDir := cmd.editorRootTmpDir()
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

	// TODO: to be removed once the std lib has a websocket impl.
	// add tag to existing tags
	added := false
	tag1 := "editorDebugExecSide"
	for i, e := range u {
		k, _, ok := strings.Cut(e, "=")
		if ok && k == "-tags" {
			u[i] += "," + tag1
			added = true
			break
		}
	}
	if !added {
		u = append(u, "-tags="+tag1)
	}
	//goutil.Logf("%v", u)

	return u
}

//------------

func (cmd *Cmd) setupNetworkAddress() error {
	// NOTE: for communication, can't consider using stdin/out since the program could use it

	cmd.improveNetwork()

	// auto fill empty address
	if cmd.flags.address == "" {
		switch cmd.flags.network {
		case "tcp", "ws", "auto":
			port, err := osutil.GetFreeTcpPort()
			if err != nil {
				return err
			}
			cmd.flags.address = fmt.Sprintf("127.0.0.1:%v", port)
		case "unix":
			// create file outside of tmpdir but inside the editor root tmp dir, otherwise the socket file will get deleted after "start"
			cmd.flags.address = filepath.Join(cmd.editorRootTmpDir(), "godebug.sock"+mathutil.GenDigitsStr(5))
		default:
			return fmt.Errorf("unexpected network: %q", cmd.flags.network)
		}
	}
	return nil
}
func (cmd *Cmd) improveNetwork() {
	// OS target to choose how to connect
	goOs := osutil.GetEnv(cmd.env, "GOOS")
	if goOs == "" {
		goOs = runtime.GOOS
	}
	switch goOs {
	case "linux":
		if cmd.flags.network == "tcp" && cmd.flags.address == "" {
			cmd.flags.network = "unix"
		}
	}
}

//------------

func (cmd *Cmd) annotateFiles2(ctx context.Context) error {
	// annotate files
	handledMain := false
	cmd.overlay = map[string]string{}
	for filename := range cmd.fa.toAnnotate {
		astFile, ok := cmd.fa.filesAsts[filename]
		if !ok {
			return fmt.Errorf("missing ast file: %v", filename)
		}

		// annotate
		ti := (*types.Info)(nil)
		pkg, ok := cmd.fa.filesPkgs[filename]
		if ok {
			ti = pkg.TypesInfo
		}
		ann, err := cmd.annset.AnnotateAstFile(astFile, ti, cmd.fa.nodeAnnTypes, cmd.flags.mode.test)
		if err != nil {
			return err
		}

		if ann.hasMainFunc {
			handledMain = true
			cmd.mainFuncFilename = filename
		}
	}

	if !handledMain {
		if !cmd.flags.mode.test {
			return fmt.Errorf("main func not handled")
		}
		// insert testmains in "*_test.go" files
		seen := map[string]bool{}
		for filename := range cmd.fa.toAnnotate {
			if !strings.HasSuffix(filename, "_test.go") {
				continue
			}

			// one testmain per dir
			dir := filepath.Dir(filename)
			if seen[dir] {
				continue
			}
			seen[dir] = true

			astFile, ok := cmd.fa.filesAsts[filename]
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

	for filename := range cmd.fa.toAnnotate {
		astFile, ok := cmd.fa.filesAsts[filename]
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

func (cmd *Cmd) buildDebugPkg(ctx context.Context) error {
	// target dir
	cmd.debugPkgDir = filepath.Join(cmd.tmpDir, "debugpkg")
	if cmd.gopathMode {
		cmd.addToGopathStart(cmd.debugPkgDir)
		cmd.debugPkgDir = filepath.Join(cmd.debugPkgDir, "src/"+cmd.annset.dopt.PkgPath)
	}

	// util to add file to debug pkg dir
	writeFile := func(name string, src []byte) error {
		filename2 := filepath.Join(cmd.debugPkgDir, name)
		return mkdirAllWriteFile(filename2, src)
	}

	// local src pkg dir where the debug pkg is located (io/fs)
	srcDir := "debug"
	des, err := debugPkgFs.ReadDir(srcDir)
	if err != nil {
		return err
	}
	// dynamically map some filenames due to "go:embed" // ex: go.mod
	m := map[string]string{
		//"gomod.txt": "go.mod",
		//"gosum.txt": "go.sum",
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

		name := de.Name()
		if s, ok := m[name]; ok {
			name = s
		}
		if err := writeFile(name, src); err != nil {
			return err
		}
	}

	// dynamically create go.mod since go:embed doesn't allow it
	if !cmd.gopathMode {
		src3 := goModuleSrc(cmd.annset.dopt.PkgPath)
		if err := writeFile("go.mod", []byte(src3)); err != nil {
			return err
		}
	}

	// init() functions declared across multiple files in a package are processed in alphabetical order of the file name. Use name starting with "a" to setup config vars as early as possible.
	configFilename := "aaaconfig.go"

	// build config file
	src4 := cmd.buildConfigSrc()
	if err := writeFile(configFilename, src4); err != nil {
		return err
	}

	return nil
}
func (cmd *Cmd) buildConfigSrc() []byte {
	fl := &cmd.flags
	bcce := cmd.annset.buildConfigAfdEntries()

	fb := strconv.FormatBool

	src := `package debug
func init(){
	exso.testing = ` + fb(cmd.Testing) + `
	exso.onExecSide = true
	exso.addr = NewAddrI("` + fl.network + `","` + fl.address + `")
	exso.isServer = ` + fb(!fl.editorIsServer) + `
	exso.continueServing = ` + fb(fl.continueServing) + `
	exso.noDebugMsg = ` + fb(fl.noDebugMsg) + `
	exso.srcLines = ` + fb(fl.srcLines) + `
	exso.syncSend = ` + fb(fl.syncSend) + `
	exso.stringifyBytesRunes = ` + fb(fl.stringifyBytesRunes) + `
	exso.filesData = []*AnnotatorFileData{` + bcce + `}
}
`
	return []byte(src)
}

//------------

func (cmd *Cmd) buildAlternativeGoMod(ctx context.Context) error {
	if cmd.gopathMode {
		return nil
	}

	filename, ok := cmd.fa.GoModFilename()
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
		src := goModuleSrc("main")
		if err := mkdirAllWriteFile(fname2, []byte(src)); err != nil {
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
		if err := cmd.buildPkgLinks(mf); err != nil {
			return err
		}
	}

	// include debug pkg require/replace lines
	mf.AddNewRequire(cmd.annset.dopt.PkgPath, "v0.0.0", false)
	mf.AddReplace(cmd.annset.dopt.PkgPath, "", cmd.debugPkgDir, "")

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

func (cmd *Cmd) buildPkgLinks(mf *modfile.File) error {

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

	seen := map[string]bool{}                 // seen module
	for filename := range cmd.fa.toAnnotate { // all annotated files
		fpkg, ok := cmd.fa.filesPkgs[filename]
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
	ci := osutil.NewCmdI2(ctx, args...)
	ec := ci.Cmd()
	ec.Stdin = cmd.Stdin
	ec.Stdout = cmd.Stdout
	ec.Stderr = cmd.Stderr
	ec.Dir = cmd.Dir
	ec.Env = cmd.env
	return ci
}

//------------

func (cmd *Cmd) printBuildInfo() {
	info := []string{}

	info = append(info, fmt.Sprintf("addr=(%v, %v)", cmd.flags.network, cmd.flags.address))

	info = append(info, fmt.Sprintf("editorIsServer=%v", cmd.flags.editorIsServer))

	cmd.printf("build: %v (builtin: %s)\n", cmd.tmpBuiltFile, strings.Join(info, ", "))
}

//------------
//------------
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

//----------

// TODO: remove once env vars supported in editor
func envGodebugBuildFlags(env []string) []string {
	bfs := osutil.GetEnv(env, "GODEBUG_BUILD_FLAGS")
	if len(bfs) == 0 {
		return nil
	}
	return strings.Split(bfs, ",")
}

//----------

func goModuleSrc(name string) string {
	// go 1.16 needed to support -overlay flag
	// go 1.18 needed to support "any"
	// go 1.22 needed to support "range int"

	//return fmt.Sprintf("module %s\ngo 1.18\n", name)

	v := "1.18"

	//v := "1.22" // TODO: fails with "go: updates to go.mod needed; to update it: go mod tidy"

	//if v2, err := goutil.GoVersion(); err == nil {
	//	v3 := "1.22"
	//	v2o := parseutil.VersionOrdinal(v2)
	//	v3o := parseutil.VersionOrdinal(v3)
	//	if v3o < v2o {
	//		v = v3
	//	}
	//}

	return fmt.Sprintf("module %s\ngo %v\n", name, v)
}
