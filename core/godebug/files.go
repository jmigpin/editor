package godebug

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/osutil"

	"golang.org/x/mod/modfile"

	"golang.org/x/tools/go/packages"
)

//----------

// issues with the editorPkg
// 1: annotated
// 2: not annotated but used (ex: some func in mathutil)
// 3: not annotated and not being used due to only accessing the debug pkg (ex: access a debug.* struct)

//----------

// need to use the same location as imported in the client (gob decoder), as well as for detecting if self debugging to avoid including these for annotation
const editorPkgPath = "github.com/jmigpin/editor"
const partialDebugPkgPath = "core/godebug/debug"
const debugPkgPath = editorPkgPath + "/" + partialDebugPkgPath

//----------

// Finds the set of files that need to be annotated/copied.
type Files struct {
	dir        string
	testMode   bool
	gopathMode bool
	filenames  map[string]struct{} // filenames to solve

	files    map[string]File
	srcFiles map[string]*SrcFile
	modFiles map[string]*ModFile

	nodeAnnTypes map[ast.Node]AnnotationType

	fset   *token.FileSet
	stderr io.Writer
	cache  struct {
		sync.RWMutex
		srcs        map[string][]byte // cleared at the end
		fullAstFile map[string]*ast.File
	}
}

func NewFiles(fset *token.FileSet, dir string, testMode bool, gopathMode bool, stderr io.Writer) *Files {
	files := &Files{fset: fset, dir: dir, testMode: testMode, gopathMode: gopathMode, stderr: stderr}
	files.filenames = map[string]struct{}{}
	files.files = map[string]File{}
	files.srcFiles = map[string]*SrcFile{}
	files.modFiles = map[string]*ModFile{}
	files.nodeAnnTypes = map[ast.Node]AnnotationType{}
	files.cache.srcs = map[string][]byte{}
	files.cache.fullAstFile = map[string]*ast.File{}
	return files
}

//----------

// Add filenames to be solved (can be directories).
func (files *Files) Add(filenames ...string) {
	for _, filename := range filenames {
		filename = files.absFilename(filename)
		files.filenames[filename] = struct{}{}
	}
}

func (files *Files) absFilename(filename string) string {
	if !filepath.IsAbs(filename) {
		u := filepath.Join(files.dir, filename)
		v, err := filepath.Abs(u)
		if err == nil {
			filename = v
		}
	}
	return filename
}

//----------

func (files *Files) Do(ctx context.Context, filenames []string, env []string) error {
	// all program packages
	pkgs, err := files.progPackages(ctx, filenames, env)
	if err != nil {
		return err
	}

	if err := files.populateFilesMap(ctx, pkgs); err != nil {
		return err
	}

	// find main to be added for annotation
	if err := files.solveMainFunc(ctx); err != nil {
		return err
	}
	// find files to be added through src code comments
	if err := files.solveCommentedFiles(ctx); err != nil {
		return err
	}
	// solve files/directories to add for annotation
	if err := files.solveGivenFilenames(); err != nil {
		return err
	}
	// solve other implicitly needed files
	if err := files.solveDependencyFiles(); err != nil {
		return err
	}

	// done at the end since the inserted files will not have ASTs
	if err := files.setupDebugPkgFiles(); err != nil {
		return err
	}

	if err := files.doAnnFilesHashes(); err != nil {
		return err
	}
	return files.cleanupForGc()
}

//----------

func (files *Files) populateFilesMap(ctx context.Context, pkgs []*packages.Package) error {
	// ignore filepaths inside GOROOT
	goRoot := filepath.Clean(goutil.GoRoot()) + string(filepath.Separator)

	err2 := (error)(nil)
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		// returns if imports should be visited

		// early stop
		if err := ctx.Err(); err != nil {
			err2 = err
			return false
		}

		//// can't annotate debugPkg, must use the editor injected version
		//if pkg.PkgPath == debugPkgPath {
		//	f2 := files.getModFileFromPkg(pkg, fname)
		//	f2.action = FACopy
		//	continue
		//}

		for _, fname := range pkg.GoFiles {
			// already added
			_, ok := files.files[fname]
			if ok {
				continue
			}
			// skip goroot filepaths (can't annotate for compile)
			if osutil.FilepathHasDirPrefix(fname, goRoot) {
				return false
			}
			// skip generated files (tests) without ".go" suffix
			if !strings.HasSuffix(fname, ".go") {
				continue
			}

			_ = newSrcFileFromPkg(fname, pkg, files)
		}
		return true // visit imports
	}, nil)
	return err2
}

func (files *Files) getModFileFromPkg(pkg *packages.Package, filenameHelper string) *ModFile {
	fname1 := pkg.Module.GoMod // go.mod filepath
	create := false
	if fname1 == "" {
		// possibly:
		// pkg.Module.Main = true
		// pkg.Module.Path = "command-line-argments"
		// pkg.Module.Version = ""
		// pkg.Module.Dir = ""
		//dir := filepath.Dir(filenameHelper) // can differ from current dir
		//dir := files.dir // can differ from main file location (TODO)
		dir := files.dir
		fname1 = filepath.Join(dir, "go.mod")
		create = true
	}

	// already exists
	if f, ok := files.modFiles[fname1]; ok {
		return f
	}

	// create
	f := newModFile(fname1, pkg.Module.Path, pkg.Module.Version, pkg.Module.Dir, files)
	f.main = pkg.Module.Main
	if create {
		f.action = FACreate
		f.actionCreateBytes = []byte(fmt.Sprintf("module %v\n", pkg.Module.Path))
	}
	return f
}

//----------

func (files *Files) solveMainFunc(ctx context.Context) error {
	name := "main"
	if files.testMode {
		name = "TestMain"
	}
	fd, f1, ok, err := files.lookupMainFuncDecl(ctx, name)
	if err != nil {
		return err
	}
	if ok {
		f1.mainFuncDecl = fd
		files.addAnnFile(f1, AnnotationTypeFile)
		return nil
	}
	if files.testMode {
		return files.createPossibleTestMains()
	}
	return errors.New("main function not found")
}

func (files *Files) createPossibleTestMains() error {
	// add *_test.go files for annotation (done here before creating new files)
	for _, f := range files.srcFiles {
		if f.hasTestFilename() {
			files.addAnnFile(f, AnnotationTypeFile)
		}
	}

	// find all *_test.go files and create a testmain in each directory
	count := 0
	seen := map[string]bool{}
	for _, f := range files.srcFiles {
		if f.hasTestFilename() {
			// one per dir (this case should not happen, playing safe)
			dir := filepath.Dir(f.filename)
			if seen[dir] {
				continue
			}
			seen[dir] = true

			if err := files.createTestMain(dir, f); err != nil {
				return err
			}
			count++
		}
	}
	if count == 0 {
		return fmt.Errorf("no testmain was created")
	}
	return nil
}

func (files *Files) createTestMain(dir string, f1 *SrcFile) error {
	fname := filepath.Join(dir, "godebug_testmain_test.go")
	f2 := newSrcFile(fname, f1.pkgPath, files)

	// parse ast and keep mainfuncdecl
	src := files.testMainSrc(f1.pkgName)
	astFile, err := files.fullAstFile2(f2.filename, src)
	if err != nil {
		return err
	}
	if fd, ok := lookupFuncDeclWithBody(astFile, "TestMain"); ok {
		f2.mainFuncDecl = fd
		f2.action = FAWrite // not annotated
	}
	return nil
}

func (files *Files) testMainSrc(pkgName string) []byte {
	return []byte(`
		package ` + pkgName + `
		import "os"
		import "testing"
		func TestMain(m *testing.M) {
			os.Exit(m.Run())
		}
	`)
}

func (files *Files) lookupMainFuncDecl(ctx context.Context, name string) (*ast.FuncDecl, *SrcFile, bool, error) {
	for _, f := range files.srcFiles {
		// early stop
		if err := ctx.Err(); err != nil {
			return nil, nil, false, err
		}

		// performance
		if f.modf != nil && !f.modf.main {
			continue
		}

		astFile, err := files.fullAstFile(f.filename)
		if err != nil {
			return nil, nil, false, err
		}
		if fd, ok := lookupFuncDeclWithBody(astFile, name); ok {
			return fd, f, true, nil
		}
	}
	return nil, nil, false, nil
}

//----------

func (files *Files) solveGivenFilenames() error {
	typ := AnnotationTypeFile
	for fname := range files.filenames {
		// don't stat the file if already added
		if files.addedAnnFilename(fname, typ) {
			continue
		}

		fi, err := os.Stat(fname)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			dir := fname
			if err := files.addAnnDir(dir, typ); err != nil {
				return err
			}
		} else {
			files.addAnnFilename(fname, typ)
		}
	}
	return nil
}

//----------

func (files *Files) solveCommentedFiles(ctx context.Context) error {
	for _, f := range files.srcFiles {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := files.solveCommentedFile(f); err != nil {
			return err
		}
	}
	return nil
}

func (files *Files) solveCommentedFile(f *SrcFile) error {
	astFile, err := files.fullAstFile(f.filename)
	if err != nil {
		return err
	}
	return files.solveCommentedFile2(f, astFile)
}

func (files *Files) solveCommentedFile2(f *SrcFile, astFile *ast.File) error {
	// find comments that have "//godebug" directives
	opts := []*AnnotationOpt{}
	for _, cg := range astFile.Comments {
		for _, c := range cg.List {
			opt, err := AnnotationOptInComment(c)
			if err != nil {
				return err
			}
			opts = append(opts, opt)
		}
	}
	// find annotationOpt associated nodes
	optm := annOptNodesMap(files.fset, astFile, opts)
	// add
	for _, opt := range opts {
		node, ok := optm[opt]
		if ok {
			opt.Node = node
			// keep for annotation phase
			files.nodeAnnTypes[node] = opt.Type
		}
		// node can be nil
		if err := files.addAnnOpt(f, opt); err != nil {
			err = files.errorPos(opt.Comment, err)
			files.warnErr(err)
			//return err
		}
	}
	return nil
}

func (files *Files) addAnnOpt(f *SrcFile, opt *AnnotationOpt) error {
	switch opt.Type {
	case AnnotationTypeNone:
		return nil
	case AnnotationTypeOff:
		return nil
	case AnnotationTypeBlock:
		files.addAnnFile(f, opt.Type)
		return nil
	case AnnotationTypeFile:
		filename := f.filename
		if opt.Opt != "" {
			fname := opt.Opt
			if !filepath.IsAbs(fname) {
				d := filepath.Dir(filename)
				fname = filepath.Join(d, fname)
			}
			_, ok := files.files[fname]
			if !ok {
				err := fmt.Errorf("file not found in loaded program (or is a stdlib file): %v", opt.Opt)
				return err
			}
			filename = fname
		}
		return files.addAnnFilename(filename, opt.Type)
	case AnnotationTypeImport:
		return files.addAnnTypeImport(opt)
	case AnnotationTypePackage:
		pkgPath := f.pkgPath
		if opt.Opt != "" {
			pkgPath = opt.Opt
		}
		return files.addPkgPath(pkgPath, opt.Type)
	case AnnotationTypeModule:
		f2 := f.modf
		if opt.Opt != "" { // package path
			// get a filename that belongs to the pkg
			_, f3, err := files.pkgPathDir(opt.Opt)
			if err != nil {
				return err
			}
			f2 = f3.modf
		}
		if f2 == nil {
			return fmt.Errorf("missing mod file: %v", f2.filename)
		}
		files.addModule(f2, opt.Type)
		return nil
	default:
		return fmt.Errorf("todo: handleAnnOpt: %v", opt.Type)
	}
}

func (files *Files) addAnnTypeImport(opt *AnnotationOpt) error {
	path, err := files.annTypeImportPath(opt)
	if err != nil {
		return err
	}
	return files.addPkgPath(path, opt.Type)
}

func (files *Files) annTypeImportPath(opt *AnnotationOpt) (string, error) {
	n := opt.Node
	if n == nil {
		return "", fmt.Errorf("missing import node")
	}
	if gd, ok := n.(*ast.GenDecl); ok {
		if len(gd.Specs) > 0 {
			is, ok := gd.Specs[0].(*ast.ImportSpec)
			if ok {
				n = is
			}
		}
	}
	is, ok := n.(*ast.ImportSpec)
	if !ok {
		return "", fmt.Errorf("not at an import spec")
	}
	return strconv.Unquote(is.Path.Value)
}

func (files *Files) addPkgPath(pkgPath string, typ AnnotationType) error {
	dir, _, err := files.pkgPathDir(pkgPath)
	if err != nil {
		return err
	}
	return files.addAnnDir(dir, typ)
}

func (files *Files) addModule(modF *ModFile, typ AnnotationType) {
	for _, f2 := range files.srcFiles {
		if f2.modf == modF {
			files.addAnnFile(f2, typ)
		}
	}
}

//----------

func (files *Files) addAnnFile(f *SrcFile, typ AnnotationType) {
	if typ > f.annType {
		f.action = FAAnnotate
		f.annType = typ
	}
}

func (files *Files) addAnnFilename(filename string, typ AnnotationType) error {
	f, ok := files.srcFiles[filename]
	if !ok {
		return fmt.Errorf("not part of the loaded program: %v", filename)
	}
	files.addAnnFile(f, typ)
	return nil
}

func (files *Files) addedAnnFilename(filename string, typ AnnotationType) bool {
	f, ok := files.srcFiles[filename]
	return ok && f.action == FAAnnotate && f.annType >= typ
}

func (files *Files) addAnnDir(dir string, typ AnnotationType) error {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		u := filepath.Join(dir, fi.Name())
		// add only if part of the loaded program (ignore others, ex: src files with build tags)
		if f, ok := files.srcFiles[u]; ok {
			files.addAnnFile(f, typ)
		}
	}
	return nil
}

//----------

func (files *Files) setupDebugPkgFiles() error {
	dir, ok := files.getLoadedEditorPkgDir()
	if ok {
		files.setupDebugPkgFilesAt(dir, false)
		return nil
	}

	dir2 := filepath.FromSlash(editorPkgPath)
	files.setupDebugPkgFilesAt(dir2, true)
	return nil
}

func (files *Files) getLoadedEditorPkgDir() (string, bool) {
	// remove all debugPkg version files being debugged
	for _, f := range files.srcFiles {
		if f.pkgPath == debugPkgPath {
			files.unregisterFile(f)
		}
	}

	// find root dir of the used editorPkg, and add the rest of the debugPkg path
	dir := ""
	for _, f := range files.modFiles {
		if f.path == editorPkgPath {
			dir = f.modDir
			break
		}
	}

	ok := dir != ""
	return dir, ok
}

func (files *Files) setupDebugPkgFilesAt(editorModDir string, needMod bool) {
	// debug pkg config.go (content inserted later after annotations)
	configFp := &FilePack{Name: files.debugPkgConfigName(), Data: ""}

	// debug pkg files
	w := append(DebugFilePacks(), configFp)
	dir2 := filepath.Join(editorModDir, partialDebugPkgPath)
	for _, fp := range w {
		fname1 := filepath.Join(dir2, fp.Name)
		f := newSrcFile(fname1, debugPkgPath, files)
		f.action = FACreate
		f.actionCreateBytes = []byte(fp.Data)
	}

	// editor pkg go.mod
	if needMod && !files.gopathMode {
		fname2 := filepath.Join(editorModDir, "go.mod")
		f2 := newModFile(fname2, editorPkgPath, "v0.0.0-godebug", editorModDir, files)
		f2.action = FACreate
		f2.actionCreateBytes = []byte(fmt.Sprintf("module %v\n", editorPkgPath))
	}
}

func (files *Files) debugPkgConfigName() string {
	return "config.go"
}

//----------

func (files *Files) setDebugConfigContent(cmd *Cmd) error {
	acceptOnlyFirstClient := cmd.flags.mode.run || cmd.flags.mode.test
	afdEntries := cmd.annset.buildDebugConfigEntries()
	src := files.debugConfigSrc(cmd.start.network, cmd.start.address, cmd.flags.syncSend, acceptOnlyFirstClient, afdEntries)
	return files.setDebugConfigContentSrc(src)
}

func (files *Files) setDebugConfigContentSrc(src []byte) error {
	for _, f := range files.srcFiles {
		if f.pkgPath == debugPkgPath {
			fname := filepath.Base(f.filename)
			if fname == files.debugPkgConfigName() {
				f.actionCreateBytes = src
				return nil
			}
		}
	}
	return fmt.Errorf("config srcfile not found")
}

func (files *Files) debugConfigSrc(serverNetwork, serverAddr string, syncSend, acceptOnlyFirstClient bool, afdEntries string) []byte {
	src := `package debug
func init(){
	ServerNetwork = "` + serverNetwork + `"
	ServerAddress = "` + serverAddr + `"
	SyncSend = ` + strconv.FormatBool(syncSend) + `
	AcceptOnlyFirstClient = ` + strconv.FormatBool(acceptOnlyFirstClient) + `
	AnnotatorFilesData = []*AnnotatorFileData{
		` + afdEntries + `
	}
	StartServer()
}`
	return []byte(src)
}

//----------

func (files *Files) solveDependencyFiles() error {
	if files.gopathMode {
		return files.solveFilesGopathMode()
	}

	// set other module files of annotated
	seen := map[*ModFile]bool{}
	for _, f := range files.srcFiles {
		if f.modf != nil {
			// editorPkg special handling
			handle := f.modf.path == editorPkgPath

			if f.action == FAAnnotate || handle {
				// handle once
				if seen[f.modf] {
					continue
				}
				seen[f.modf] = true

				if err := files.setOtherModuleFilesAction(f.modf); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (files *Files) setOtherModuleFilesAction(modf *ModFile) error {
	// go.mod
	if modf.action == FANone {
		modf.action = FACopy
	}
	modf.setupGoSum()

	// copy all module files that used by the loaded program
	for _, f := range files.srcFiles {
		if f.modf == modf {
			if f.action == FANone {
				// can't copy: need src line references in case of panic
				f.action = FAWrite
			}
		}
	}
	return nil
}

func (files *Files) solveFilesGopathMode() error {
	// use parent directories of annotated files
	for _, f1 := range files.srcFiles {
		if f1.action == FAAnnotate {
			p1 := f1.pkgPath
			for _, f2 := range files.srcFiles {
				if f2.action == FANone {
					p2 := f2.pkgPath + "/"
					if strings.HasPrefix(p1, p2) {
						f2.action = FAWrite
					}
				}
			}
		}
	}
	return nil
}

//----------

func (files *Files) fullAstFile(filename string) (*ast.File, error) {
	return files.fullAstFile2(filename, nil)
}
func (files *Files) fullAstFile2(filename string, src []byte) (*ast.File, error) {
	files.cache.RLock()
	if astFile, ok := files.cache.fullAstFile[filename]; ok {
		files.cache.RUnlock()
		return astFile, nil
	}
	files.cache.RUnlock()

	// read src if not provided (need to be kept for later)
	if src == nil {
		src2, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		src = src2
	}

	// parse ast
	mode := parser.ParseComments // full mode
	astFile, err := parser.ParseFile(files.fset, filename, src, mode)
	if err != nil {
		return nil, err
	}

	files.cache.Lock()
	defer files.cache.Unlock()

	files.cache.fullAstFile[filename] = astFile

	//if _, ok := files.cache.srcs[filename]; !ok {
	//	if f2, ok := files.srcFiles[filename]; ok {
	//		spew.Dump(f2)
	//	}
	//}

	files.cache.srcs[filename] = src // keep for hash computations

	return astFile, nil
}

//----------

func (files *Files) doAnnFilesHashes() error {
	for _, f := range files.filesToAnnotate() {
		src, ok := files.cache.srcs[f.filename]
		if !ok {
			return fmt.Errorf("missing src: %v", src)
		}
		afd := &AnnFileData{
			FileSize: len(src),
			FileHash: sourceHash(src),
		}
		f.annFileData = afd
	}
	return nil
}

// cleanup (allow garbage collection)
func (files *Files) cleanupForGc() error {
	// clean all srcs (hash calcs already done)
	files.cache.srcs = nil

	// cleanup ASTs
	for k := range files.cache.fullAstFile {
		f, ok := files.srcFiles[k]
		if !ok || f.action == FANone {
			delete(files.cache.fullAstFile, k)
		}
	}
	return nil
}

//----------

func (files *Files) pkgPathDir(pkgPath string) (string, *SrcFile, error) {
	for _, f := range files.srcFiles {
		if f.pkgPath == pkgPath {
			return filepath.Dir(f.filename), f, nil
		}
	}
	return "", nil, fmt.Errorf("pkg path not found in loaded program (or is a stdlib pkg): %v", pkgPath)
}

//----------

func (files *Files) NodeAnnType(n ast.Node) AnnotationType {
	at, ok := files.nodeAnnTypes[n]
	if ok {
		return at
	}
	return AnnotationTypeNone
}

//----------

func (files *Files) progPackages(ctx context.Context, filenames []string, env []string) ([]*packages.Package, error) {
	loadMode := 0 |
		packages.NeedDeps |
		packages.NeedImports |
		packages.NeedName | // name and pkgpath
		packages.NeedFiles |
		packages.NeedTypes |
		packages.NeedModule |
		packages.NeedTypesInfo | // access to pkgs.TypesInfo.*
		//packages.NeedSyntax | // ast.File, (commented, using custom cache)
		0

	parseFile := func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		return files.fullAstFile2(filename, src)
	}

	pkgs, err := programPackages(ctx, files.fset, loadMode, files.dir, filenames, files.testMode, env, parseFile, files.stderr)

	// programpackages parses files concurrently, on ctx cancel it concats useless repeated errors, get just one ctx error
	if err2 := ctx.Err(); err2 != nil {
		return nil, err2
	}

	return pkgs, err
}

//----------

func (files *Files) warnErr(err error) {
	if files.stderr != nil {
		fmt.Fprintf(files.stderr, "# warning: %v\n", err)
	}
}

//----------

func (files *Files) filesToAnnotate() []*SrcFile {
	r := []*SrcFile{}
	for _, f := range files.srcFiles {
		if f.action == FAAnnotate {
			r = append(r, f)
		}
	}
	return r
}

//----------

func (files *Files) verbose(cmd *Cmd) {
	files.printFiles2(cmd.Printf)
}

func (files *Files) printFiles1() {
	files.printFiles2(fmt.Printf)
}

func (files *Files) printFiles2(ps0 func(format string, a ...interface{}) (int, error)) {
	ps0("===files-actions===\n")
	seen := map[string]bool{}
	o := []string{}
	ps0("*files to annotate:\n")
	for _, f := range files.files {
		if f.Action() == FAAnnotate {
			seen[f.Filename()] = true
			o = append(o, f.Filename())
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files to create:\n")
	for _, f := range files.files {
		if f.Action() == FACreate {
			seen[f.Filename()] = true
			o = append(o, f.Filename())
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files to copy:\n")
	for _, f := range files.files {
		if f.Action() == FACopy {
			seen[f.Filename()] = true
			o = append(o, f.Filename())
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files to write:\n")
	for _, f := range files.files {
		if f.Action() == FAWrite {
			seen[f.Filename()] = true
			o = append(o, f.Filename())
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files (other):\n")
	for _, f := range files.files {
		if !seen[f.Filename()] {
			o = append(o, f.Filename())
		}
	}
	files.sortedFilesStr(o, ps0)
}

func (files *Files) sortedFilesStr(o []string, ps0 func(format string, a ...interface{}) (int, error)) {
	u := []string{}
	for _, s := range o {
		f := files.files[s]
		//fn := f.filename
		fn := f.shortFilename()
		if f2, ok := f.(*SrcFile); ok {
			if f2.mainFuncDecl != nil {
				fn += " (mainfunc)"
			}
			if f2.modf != nil && f2.modf.main {
				fn += " (mainmod)"
			}
		}
		u = append(u, fn)
	}
	sort.Strings(u)
	for _, s := range u {
		ps0("\t%v\n", s)
	}
}

//----------

func (files *Files) writeToTmpDir(cmd *Cmd) error {
	for _, f1 := range files.files {
		dest := cmd.tmpDirBasedFilename(f1.destFilename())
		switch f2 := f1.(type) {
		case *SrcFile:
			switch f2.action {
			case FANone:
			case FAWrite:
				astFile, err := files.fullAstFile(f1.Filename())
				if err != nil {
					return err
				}
				if err := cmd.mkdirAllWriteAstFile(dest, astFile); err != nil {
					return err
				}
			case FACreate:
				if err := mkdirAllWriteFile(dest, f2.actionCreateBytes); err != nil {
					return err
				}
			default:
				panic("todo")
			}
		case *ModFile:
			switch f2.action {
			case FANone:
			case FACopy:
				if err := mkdirAllCopyFileSync(f1.Filename(), dest); err != nil {
					return err
				}
			case FACreate:
				if err := mkdirAllWriteFile(dest, f2.actionCreateBytes); err != nil {
					return err
				}
			case FAWrite:
				m, err := f2.modFile()
				if err != nil {
					return err
				}
				m.SortBlocks()
				m.Cleanup()
				b1, err := m.Format()
				if err != nil {
					return err
				}
				cmd.Vprintf("===go.mod===\n")
				cmd.Vprintf("%v\n%v", dest, string(b1))
				if err := mkdirAllWriteFile(dest, b1); err != nil {
					return err
				}
			default:
				panic("todo")
			}
		case *SumFile:
			switch f2.action {
			case FANone:
			case FACopy:
				if err := mkdirAllCopyFileSync(f1.Filename(), dest); err != nil {
					return err
				}
			default:
				panic("todo")
			}
		default:
			panic("todo")
		}
	}
	return nil
}

//----------

func (files *Files) registerFile(f File) {
	fname := f.Filename()
	// general map
	files.files[fname] = f
	// specific maps
	switch f2 := f.(type) {
	case *SrcFile:
		files.srcFiles[fname] = f2
	case *ModFile:
		files.modFiles[fname] = f2
	default:
		// other files (ex: sumFile)
	}
}

func (files *Files) unregisterFile(f File) {
	fname := f.Filename()
	delete(files.files, fname)
	delete(files.srcFiles, fname)
	delete(files.modFiles, fname)
}

//----------

func (files *Files) errorPos(node ast.Node, err error) error {
	return errorPos(err, files.fset, node.Pos())
}

//----------
//----------
//----------

type AnnFileData struct {
	FileSize int
	FileHash []byte
}

//----------

type AnnotationType int

const (
	// Order matters, last is the bigger set
	AnnotationTypeNone AnnotationType = iota
	AnnotationTypeOff
	AnnotationTypeBlock
	AnnotationTypeFile
	AnnotationTypeImport  // annotates set of files (importspec)
	AnnotationTypePackage // annotates set of files
	AnnotationTypeModule  // annotates set of packages
)

func AnnotationTypeInString(s string) (AnnotationType, string, error) {
	prefix := "//godebug:"
	if !strings.HasPrefix(s, prefix) {
		return AnnotationTypeNone, "", nil
	}

	// type and optional rest of the string
	typ := s[len(prefix):]
	opt, hasOpt := "", false
	i := strings.Index(typ, ":")
	if i >= 0 {
		hasOpt = true
		typ, opt = typ[:i], typ[i+1:]
	} else {
		// allow some space at the end (ex: comments)
		i := strings.Index(typ, " ")
		if i >= 0 {
			typ = typ[:i]
		}
	}
	typ = strings.TrimSpace(typ)
	opt = strings.TrimSpace(opt)

	var at AnnotationType
	switch typ {
	case "annotateoff":
		at = AnnotationTypeOff
	case "annotateblock":
		at = AnnotationTypeBlock
	case "annotatefile":
		at = AnnotationTypeFile
	case "annotatepackage":
		at = AnnotationTypePackage
	case "annotateimport":
		at = AnnotationTypeImport
	case "annotatemodule":
		at = AnnotationTypeModule
	default:
		err := fmt.Errorf("godebug: unexpected annotate type: %q", typ)
		return AnnotationTypeNone, "", err
	}

	// ensure early error if opt is set on annotations not expecting it
	if hasOpt {
		switch at {
		case AnnotationTypeFile:
		case AnnotationTypePackage:
		case AnnotationTypeModule:
		default:
			return at, opt, fmt.Errorf("godebug: unexpected annotate string: %v", opt)
		}
	}

	return at, opt, nil
}

//----------

type AnnotationOpt struct {
	Type    AnnotationType
	Opt     string
	Comment *ast.Comment
	Node    ast.Node // node associated to comment (can be nil)
}

func AnnotationOptInComment(c *ast.Comment) (*AnnotationOpt, error) {
	typ, opt, err := AnnotationTypeInString(c.Text)
	if err != nil {
		return nil, err
	}
	return &AnnotationOpt{Type: typ, Opt: opt, Comment: c}, nil
}

//----------

type File interface {
	Filename() string
	destFilename() string
	shortFilename() string
	Action() FileAction
	hasAction() bool
}

//----------

type SrcFile struct {
	fileCommon
	annType      AnnotationType // how to annotate
	annFileData  *AnnFileData
	mainFuncDecl *ast.FuncDecl // can be nil
	modf         *ModFile      // can be nil

	pkgPath   string
	typesInfo *types.Info // can be nil

	// can be "main" for a pkgPath of "_/tmp/...".
	// useful to create testmain code with the line "package <name>"
	pkgName string
}

func newSrcFile(filename string, pkgPath string, files *Files) *SrcFile {
	fc := makeFileCommon(filename, files)
	f := &SrcFile{fileCommon: fc, pkgPath: pkgPath}
	files.registerFile(f)
	return f
}

func newSrcFileFromPkg(filename string, pkg *packages.Package, files *Files) *SrcFile {
	f := newSrcFile(filename, pkg.PkgPath, files)
	f.pkgName = pkg.Name
	f.typesInfo = pkg.TypesInfo
	if !files.gopathMode { // pkg.Module!=nil
		f.modf = files.getModFileFromPkg(pkg, f.filename)
	}
	return f
}

func (f *SrcFile) astIdentObj(id *ast.Ident) (types.Object, bool) {
	if f.typesInfo != nil {
		u := f.typesInfo.ObjectOf(id)
		if u != nil {
			return u, true
		}
	}
	return nil, false
}

func (f *SrcFile) astExprType(e ast.Expr) (types.TypeAndValue, bool) {
	if f.typesInfo != nil {
		t, ok := f.typesInfo.Types[e]
		return t, ok
	}
	return types.TypeAndValue{}, false
}

//----------

type ModFile struct {
	fileCommon
	main    bool
	path    string // module path
	version string
	modDir  string // can differ from filename dir (cache locations)
	cache   struct {
		modFile *modfile.File
	}
}

func newModFile(filename, path, version string, modDir string, files *Files) *ModFile {
	fc := makeFileCommon(filename, files)
	f := &ModFile{fileCommon: fc, path: path, version: version}
	files.registerFile(f)

	f.modDir = modDir
	if f.modDir == "" {
		f.modDir = filepath.Dir(filename)
	}

	// fix file name not being "go.mod" (cache locations)
	// fix target dir (cache locations)
	f.destFilename2 = filepath.Join(f.modDir, "go.mod")

	return f
}

func (f *ModFile) setupGoSum() {
	gosum := "go.sum"

	//  go.mod
	dir1 := filepath.Dir(f.filename)
	fname1 := filepath.Join(dir1, gosum)

	// must exist
	if _, err := os.Stat(fname1); err != nil {
		if f.modDir == "" {
			return
		}
		fname2 := filepath.Join(f.modDir, gosum)
		if _, err := os.Stat(fname2); err != nil {
			return
		}
		fname1 = fname2
	}

	// go.sum dest
	dir2 := filepath.Dir(f.destFilename())
	fname2 := filepath.Join(dir2, gosum)

	f3 := newSumFile(fname1, f.files)
	f3.action = FACopy
	f3.destFilename2 = fname2
}

func (f *ModFile) modFile() (*modfile.File, error) {
	if f.cache.modFile != nil {
		return f.cache.modFile, nil
	}
	// read src
	src := []byte{}
	if f.action == FACreate {
		if f.actionCreateBytes == nil {
			return nil, errors.New("missing go.mod content (create)")
		}
		src = f.actionCreateBytes
	} else {
		b, err := ioutil.ReadFile(f.filename)
		if err != nil {
			return nil, err
		}
		src = b
	}

	u, err := modfile.Parse(f.destFilename(), src, nil)
	if err != nil {
		return nil, err
	}
	f.cache.modFile = u
	return u, nil
}

//----------

type SumFile struct {
	fileCommon
}

func newSumFile(filename string, files *Files) *SumFile {
	fc := makeFileCommon(filename, files)
	f := &SumFile{fileCommon: fc}
	files.registerFile(f)
	return f
}

//----------

type fileCommon struct {
	filename          string
	destFilename2     string
	action            FileAction
	actionCreateBytes []byte // action = FACreate
	files             *Files
}

func makeFileCommon(filename string, files *Files) fileCommon {
	return fileCommon{filename: filename, files: files}
}

func (f *fileCommon) Filename() string {
	return f.filename
}

func (f *fileCommon) shortFilename() string {
	// cut parent dirs to show just the first parent
	fn := f.filename
	if len(fn) >= 1 {
		d1 := filepath.Dir(fn)
		d2 := filepath.Dir(d1)
		if d2 != "." { // empty dir case
			fn = f.filename[len(d2)+len(string(filepath.Separator)):]
		}
	}
	return fn
}

func (f *fileCommon) hasTestFilename() bool {
	return strings.HasSuffix(f.filename, "_test.go")
}

func (f *fileCommon) destFilename() string {
	if f.destFilename2 != "" {
		return f.destFilename2
	}
	return f.filename
}

func (f *fileCommon) Action() FileAction {
	return f.action
}

func (f *fileCommon) hasAction() bool {
	return f.action != FANone
}

//----------

type FileAction int

const (
	FANone FileAction = iota
	FACopy            // should not be used in fileSrc (need src line references)
	FACreate
	FAAnnotate
	FAWrite // output from struct (ast, modfile struct)
)

//----------
//----------
//----------

func programPackages(
	ctx context.Context,
	fset *token.FileSet,
	mode packages.LoadMode,
	dir string,
	filenames []string,
	tests bool,
	env []string,
	parseFile func(fset *token.FileSet, filename string, src []byte) (*ast.File, error),
	stderr io.Writer,
) ([]*packages.Package, error) {
	// faster startup
	env = append(env, "GOPACKAGESDRIVER=off")

	cfg := &packages.Config{
		Context:    ctx,
		Fset:       fset,
		Tests:      tests,
		Dir:        dir,
		Mode:       mode,
		Env:        env,
		BuildFlags: godebugBuildFlags(env),
		ParseFile:  parseFile,
		Logf: func(format string, args ...interface{}) {
			if stderr != nil {
				// TODO: too much output, output on flag only
				//s := fmt.Sprintf(format, args...)
				//fmt.Fprintf(stderr, "# packages: %v\n", s)
			}
		},
	}

	// build file patterns
	patterns := []string{}
	for _, f := range filenames {
		//p := "file=" + f // not working for multiple files
		p := f
		patterns = append(patterns, p)
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}

	// join errors into one error: golang.org/x/tools/go/packages/visit.go:46
	errStrs := []string{}
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			errStrs = append(errStrs, err.Error())
		}
	})
	if len(errStrs) > 0 {
		return nil, errors.New(strings.Join(errStrs, "\n"))
	}

	return pkgs, nil
}

//----------

func errorPos(err error, fset *token.FileSet, pos token.Pos) error {
	p := fset.Position(pos)
	return fmt.Errorf("%v: %w", p, err)
}

//----------

func sourceHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}

//----------

func annOptNodesMap(fset *token.FileSet, astFile *ast.File, opts []*AnnotationOpt) map[*AnnotationOpt]ast.Node {
	// wrap comments in commentgroups to use ast.NewCommentMap
	cgs := []*ast.CommentGroup{}
	cmap := map[*ast.CommentGroup]*AnnotationOpt{}
	for _, opt := range opts {
		cg := &ast.CommentGroup{List: []*ast.Comment{opt.Comment}}
		cgs = append(cgs, cg)
		cmap[cg] = opt
	}
	// map nodes to comments
	nmap := ast.NewCommentMap(fset, astFile, cgs)
	optm := map[*AnnotationOpt]ast.Node{}
	for n, cgs := range nmap {
		cg := cgs[0]
		opt, ok := cmap[cg]
		if ok {
			optm[opt] = n
		}
	}
	return optm
}

// alternative to annOptNodesMap
//func findCommentNode(top ast.Node, c *ast.Comment) (ast.Node, error) {
//	//fmt.Printf("comment pos: %v\n", c.Pos())
//	found := (ast.Node)(nil)
//	state := "search"
//	ast.Inspect(top, func(n ast.Node) bool {
//		//fmt.Printf("node: %T", n)
//		//if n != nil {
//		//	fmt.Printf(" %v", n.Pos())
//		//}
//		//fmt.Printf(" st=%v", state)
//		//fmt.Printf("\n")

//		switch state {
//		case "search":
//			if n == c {
//				state = "exit_comment"
//			}
//		case "exit_comment":
//			if n == nil {
//				state = "exit_comment_group"
//			} else {
//				state = "fail"
//			}
//		case "exit_comment_group":
//			if n == nil {
//				state = "out_of_comment_group"
//			} else {
//				state = "fail"
//			}
//		case "out_of_comment_group":
//			if n == nil {
//				state = "fail"
//			} else {
//				state = "done"
//				found = n
//			}
//		case "fail", "done":
//			return false
//		}
//		return true
//	})
//	if found != nil {
//		//fmt.Printf("FOUND %T\n", found)
//		return found, nil
//	}
//	return nil, fmt.Errorf("comment node not found")
//}

//----------
