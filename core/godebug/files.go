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
// 1: editorPkg is annotated
// 	- can't point to original, need to copy to avoid duplicating debugPkg
// 2: editorPkg is not annotated but used (ex: some func in mathutil)
// 	- can't point to original, need to copy to avoid duplicating debugPkg
// 3: editorPkg is not annotated and is not being used due to only accessing the debug pkg (ex: access a debug.* struct)
// 	- need to drop requires, not used

//----------

// need to use the same location as imported in the client (gob decoder), as well as for detecting if self debugging to avoid including these for annotation
const editorPkgPath = "github.com/jmigpin/editor"
const godebugPkgPath = editorPkgPath + "/core/godebug"
const DebugPkgPath = godebugPkgPath + "/debug"
const GodebugconfigPkgPath = godebugPkgPath + "/godebugconfig"

//----------

// Finds the set of files that need to be annotated/copied.
type Files struct {
	dir        string
	testMode   bool
	gopathMode bool
	filenames  map[string]struct{} // filenames to solve

	files        map[string]*File
	pkgPath      map[string]*File
	nodeAnnTypes map[ast.Node]AnnotationType
	pkgs         []*packages.Package

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
	files.files = map[string]*File{}
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
	if !filepath.IsAbs(files.dir) {
		return fmt.Errorf("files.dir is not absolute")
	}

	// all program packages
	pkgs, err := files.progPackages(ctx, filenames, env)
	if err != nil {
		return err
	}
	files.pkgs = pkgs

	files.populateFilesMap(pkgs)

	// find main to be added for annotation
	if err := files.findMain(ctx); err != nil {
		return err
	}
	// find files to be added through src code comments
	if err := files.findCommentedFiles(ctx); err != nil {
		return err
	}
	// solve files/directories to add for annotation
	if err := files.solveGivenFilenames(); err != nil {
		return err
	}

	files.setupDebugPkgFiles()

	if err := files.solveFiles(); err != nil {
		return err
	}

	if err := files.doAnnFilesHashes(); err != nil {
		return err
	}

	//if err := files.reviewTmpDirCache(); err != nil {
	//	return err
	//}

	return nil
}

//----------

func (files *Files) populateFilesMap(pkgs []*packages.Package) {
	// ignore filepaths inside GOROOT
	goRoot := filepath.Clean(goutil.GoRoot()) + string(filepath.Separator)

	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		// returns if imports should be visited

		//fmt.Printf("visiting pkg: %v\n", pkg.PkgPath)

		// can't annotate debugPkg due to pkgPath duplication
		if pkg.PkgPath == DebugPkgPath {
			return false
		}
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
			// construct file
			f := files.NewFile(fname, FTSrc, pkg)
			// construct go.mod if any
			if pkg.Module != nil {
				f.moduleGoMod = files.pkgGoModFile(pkg, f.filename)
			}
		}
		return true // visit imports
	}, nil)
}

func (files *Files) pkgGoModFile(pkg *packages.Package, filename string) *File {
	// note: the go.mod location can be far away in a cache dir and be named *.mod
	fname2 := pkg.Module.GoMod

	needCreate := false
	if fname2 == "" {
		needCreate = true
		//dir := pkg.Module.Dir // can be ""
		//dir := files.Dir // can differ from main file location
		dir := filepath.Dir(filename)
		fname2 = filepath.Join(dir, "go.mod")
	}

	f, ok := files.files[fname2]
	if !ok {
		// construct file
		f = files.NewFile(fname2, FTMod, pkg)
		if needCreate {
			f.action = FACreate
			f.createContent = "module " + pkg.Module.Path + "\n"
		}

		// fix destination filename because go.mod could be in a cache dir (without the src files) and with a *.mod name
		if pkg.Module.Dir != "" {
			f.destFilename2 = filepath.Join(pkg.Module.Dir, "go.mod")
		}
	}
	return f
}

//----------

func (files *Files) findMain(ctx context.Context) error {
	name := "main"
	if files.testMode {
		name = "TestMain"
	}

	for _, f := range files.files {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		if f.canHaveMain(files.testMode) {
			if files.testMode {
				// auto add for annotation *_test.go
				files.Add(f.filename)
			}

			astFile, err := files.fullAstFile(f.filename)
			if err != nil {
				return err
			}
			// search for the main function
			obj := astFile.Scope.Lookup(name)
			if obj != nil && obj.Kind == ast.Fun {
				fd, ok := obj.Decl.(*ast.FuncDecl)
				if ok && fd.Body != nil {
					f.hasMainFunc = true

					// add for annotation
					// ex: main.go, testmain_test.go
					files.Add(f.filename)

					return nil
				}
			}
		}
	}

	// create testmain_test.go since it was not found
	if files.testMode {
		seen := map[string]bool{}
		for _, f := range files.files {
			if f.canHaveMain(files.testMode) {
				dir := filepath.Dir(f.filename)

				// one per dir (this case should not happen, playing safe)
				if seen[dir] {
					continue
				}
				seen[dir] = true

				fname := filepath.Join(dir, "godebug_testmain_test.go")
				f2 := files.NewFile(fname, FTSrc, nil)
				f2.action = FACreate
				f2.createContent = files.testMainSrc(f.pkgName)
				f2.hasMainFunc = true
			}
		}
		return nil
	}

	return errors.New("main function not found")
}

func (files *Files) testMainSrc(pkgName string) string {
	return `
		package ` + pkgName + `
		import debugPkg "` + DebugPkgPath + `"
		import "testing"
		import "os"
		func TestMain(m *testing.M) {
			var code int
			defer func(){ os.Exit(code) }()
			defer debugPkg.ExitServer()
			code = m.Run()
		}
	`
}

//----------

func (files *Files) findCommentedFiles(ctx context.Context) error {
	for _, f := range files.files {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		if f.typ != FTSrc || f.action == FACreate {
			continue
		}
		if err := files.findCommentedFile(f.filename); err != nil {
			return err
		}
	}
	return nil
}

func (files *Files) findCommentedFile(filename string) error {
	astFile, err := files.fullAstFile(filename)
	if err != nil {
		return err
	}
	return files.findCommentedFile2(filename, astFile)
}

func (files *Files) findCommentedFile2(filename string, astFile *ast.File) error {
	opts := []*AnnotationOpt{}
	for _, cg := range astFile.Comments {
		for _, c := range cg.List {
			opt, err := AnnotationOptInComment(c, files.fset)
			if err != nil {
				return err
			}
			ok, err := files.handleAnnOpt(filename, opt)
			if err != nil {
				err = positionError(err, files.fset, c.Pos())
				//return err
				files.warnErr(err)
			}
			if ok {
				opts = append(opts, opt)
			}
		}
	}
	// map for the annotator to know nodes annotations
	mn := annOptCommentsNodesMap(files.fset, astFile, opts)
	for n, opt := range mn {
		files.nodeAnnTypes[n] = opt.Type
		// handle annotateimport now that the node is known
		if opt.Type == AnnotationTypeImport {
			err := files.handleAnnTypeImport(n, opt)
			if err != nil {
				err = positionError(err, files.fset, opt.Comment.Pos())
				//return err
				files.warnErr(err)
			}
		}
	}
	return nil
}

func (files *Files) handleAnnOpt(filename string, opt *AnnotationOpt) (bool, error) {
	switch opt.Type {
	case AnnotationTypeNone:
		return false, nil
	case AnnotationTypeOff:
		return true, nil
	case AnnotationTypeBlock:
		files.addAnnFilename(filename, opt.Type)
		return true, nil
	case AnnotationTypeFile:
		if opt.Opt != "" {
			f := opt.Opt
			if !filepath.IsAbs(f) {
				d := filepath.Dir(filename)
				f = filepath.Join(d, f)
			}
			_, ok := files.files[f]
			if !ok {
				return false, fmt.Errorf("file not found in loaded program (or stdlib file): %v", opt.Opt)
			}
			filename = f
		}
		files.addAnnFilename(filename, opt.Type)
		return true, nil
	case AnnotationTypeImport:
		return true, nil
	case AnnotationTypePackage:
		// add external pkg
		if opt.Opt != "" {
			if err := files.addPkgPath(opt.Opt, opt.Type); err != nil {
				return false, err
			}
			return true, nil
		}
		// add current pkg
		files.addAnnFilename(filename, opt.Type)
		dir := filepath.Dir(filename)
		if err := files.addAnnDir(dir, opt.Type); err != nil {
			return false, err
		}
		return true, nil
	case AnnotationTypeModule:
		if opt.Opt != "" { // package path
			_, filename2, err := files.pkgPathDir(opt.Opt)
			if err != nil {
				return false, err
			}
			filename = filename2 // a filename that belongs to the pkg
		}
		// packages files
		f, ok := files.files[filename]
		if ok {
			if f.moduleGoMod != nil {
				for _, f2 := range files.files {
					if f2.moduleGoMod == f.moduleGoMod {
						files.addAnnFilename(f2.filename, opt.Type)
					}
				}
			}
		}

		//goMod, _ := files.findGoModOrMissing(dir)
		//// files under the gomod directory
		//dir2 := filepath.Dir(goMod) + string(filepath.Separator)
		//for _, f := range files.files {
		//	if f.typ==FTGo && strings.HasPrefix(f.filename, dir2) {
		//		files.addAnnFilename(f.filename, opt.Type)
		//	}
		//}
		return true, nil
	default:
		err := fmt.Errorf("todo: handleAnnOpt: %v", opt.Type)
		return false, err
	}
}

func (files *Files) handleAnnTypeImport(n ast.Node, opt *AnnotationOpt) error {
	path, err := files.annTypeImportPath(n, opt)
	if err != nil {
		return err
	}
	return files.addPkgPath(path, opt.Type)
}

func (files *Files) annTypeImportPath(n ast.Node, opt *AnnotationOpt) (string, error) {
	if gd, ok := n.(*ast.GenDecl); ok {
		if len(gd.Specs) > 0 {
			is, ok := gd.Specs[0].(*ast.ImportSpec)
			if ok {
				n = is
			}
		}
	}
	if is, ok := n.(*ast.ImportSpec); ok {
		return strconv.Unquote(is.Path.Value)
	}
	return "", fmt.Errorf("not at an import spec")
}

func (files *Files) addPkgPath(pkgPath string, typ AnnotationType) error {
	dir, _, err := files.pkgPathDir(pkgPath)
	if err != nil {
		return err
	}
	return files.addAnnDir(dir, typ)
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

func (files *Files) addAnnFilename(filename string, typ AnnotationType) {
	f, ok := files.files[filename]
	if ok && f.typ == FTSrc {
		if typ > f.annType {
			f.action = FAAnnotate
			f.annType = typ
		}
		return
	}

	err := errors.New("not part of the program: " + filename)
	files.warnErr(err)
}

func (files *Files) addedAnnFilename(filename string, typ AnnotationType) bool {
	f, ok := files.files[filename]
	return ok && f.action == FAAnnotate && f.annType <= typ
}

func (files *Files) addAnnDir(dir string, typ AnnotationType) error {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		u := filepath.Join(dir, fi.Name())
		// add for annotation if part of the program
		f, ok := files.files[u]
		if ok && f.typ == FTSrc {
			files.addAnnFilename(u, typ)
		}
	}
	return nil
}

//----------

func (files *Files) setupDebugPkgFiles() {
	{
		// debug pkg: go.mod
		goMod := (*File)(nil)
		if !files.gopathMode {
			fname1 := files.DebugPkgFilename("go.mod")
			f := files.NewFile(fname1, FTMod, nil)
			f.typ = FTMod
			f.action = FACreate
			f.pkgPath = DebugPkgPath
			f.modulePath = f.pkgPath
			f.createContent = fmt.Sprintf("module %v\n", DebugPkgPath)
			goMod = f
		}
		// debug pkg: various files
		for _, fp := range DebugFilePacks() {
			fname1 := files.DebugPkgFilename(fp.Name)
			f := files.NewFile(fname1, FTSrc, nil)
			f.action = FACreate
			f.createContent = fp.Data
			f.pkgPath = DebugPkgPath
			f.modulePath = f.pkgPath
			f.moduleGoMod = goMod
		}
	}
	{
		// godebugconfig pkg: go.mod
		goMod := (*File)(nil)
		if !files.gopathMode {
			fname1 := files.GodebugconfigPkgFilename("go.mod")
			f := files.NewFile(fname1, FTMod, nil)
			f.action = FACreate
			f.pkgPath = GodebugconfigPkgPath
			f.modulePath = f.pkgPath
			f.createContent = fmt.Sprintf("module %v\n", GodebugconfigPkgPath)
			goMod = f
		}
		// godebugconfig pkg: config.go (content inserted later)
		fname1 := files.godebugconfigFilename()
		f := files.NewFile(fname1, FTSrc, nil)
		f.action = FACreate // content created after annotations
		f.pkgPath = GodebugconfigPkgPath
		f.modulePath = f.pkgPath
		f.moduleGoMod = goMod
	}
}

func (files *Files) godebugconfigFilename() string {
	return files.GodebugconfigPkgFilename("config.go")
}

func (files *Files) setGodebugconfigContent(src string) error {
	fname1 := files.godebugconfigFilename()
	for _, f := range files.files {
		if f.filename == fname1 {
			f.createContent = src
			return nil
		}
	}
	return errors.New("config file not found to set content: " + fname1)
}

func (files *Files) DebugPkgFilename(filename string) string {
	fp := filepath.FromSlash(DebugPkgPath)
	return filepath.Join(fp, filename)
}

func (files *Files) GodebugconfigPkgFilename(filename string) string {
	fp := filepath.FromSlash(GodebugconfigPkgPath)
	return filepath.Join(fp, filename)
}

//func (files *Files) GodebugDestFilename(filename string) string {
//	// simplify destination path to be more visible in work dir (probably not a good thing)

//	//fp := filepath.FromSlash("godebug")
//	//return filepath.Join(fp, filename)
//	dir, fname := filepath.Split(filename)
//	dir2 := filepath.Base(dir)
//	dir3 := filepath.Join("godebug", dir2)
//	return filepath.Join(dir3, fname)
//}

//----------

func (files *Files) solveFiles() error {
	for _, f := range files.files {
		if f.typ == FTSrc {
			if f.action == FAAnnotate {
				if err := files.solveFilesDueToAnnotation(f); err != nil {
					return err
				}
			}
			if f.action == FANone && f.modulePath == editorPkgPath {
				if err := files.solveFilesDueToEditorPkg(f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (files *Files) solveFilesDueToAnnotation(af *File) error {
	return files.setOtherModuleFilesForCopy(af.moduleGoMod, true)
}

func (files *Files) solveFilesDueToEditorPkg(f *File) error {
	return files.setOtherModuleFilesForCopy(f.moduleGoMod, false)
}

func (files *Files) setOtherModuleFilesForCopy(modF *File, needDebugPkgs bool) error {
	for _, f := range files.files {
		if f.action == FANone {
			if f.moduleGoMod == modF {
				f.action = FACopy
			}
		}
		// go.mod
		if f.typ == FTMod && f == modF {
			if f.action == FANone {
				f.action = FACopy
			}
			f.needDebugMods = needDebugPkgs
			files.tryToSetupGoSumForCopy(f)
		}
	}
	return nil
}

//----------

func (files *Files) parseFile(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	return files.fullAstFile2(filename, src)
}

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
	files.cache.srcs[filename] = src // keep for hash computations

	return astFile, nil
}

//----------

func (files *Files) doAnnFilesHashes() error {
	// allow cache garbage collection at the end
	defer func() {
		files.cache.srcs = nil
	}()

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

//----------

func (files *Files) pkgPathDir(pkgPath string) (string, string, error) {
	for _, f := range files.files {
		if f.typ == FTSrc && f.pkgPath == pkgPath {
			return filepath.Dir(f.filename), f.filename, nil
		}
	}
	err := fmt.Errorf("pkg path not found in loaded program (or stdlib pkg): %v", pkgPath)
	return "", "", err
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
	pkgs, err := ProgramPackages(ctx, files.fset, loadMode, files.dir, filenames, files.testMode, env, files.parseFile, files.stderr)

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

func (files *Files) filesToAnnotate() []*File {
	r := []*File{}
	for _, f := range files.files {
		if f.action == FAAnnotate {
			r = append(r, f)
		}
	}
	return r
}

func (files *Files) filesToInsertMain() []*File {
	r := []*File{}
	for _, f := range files.files {
		if f.hasMainFunc && f.action != FACreate {
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
	seen := map[string]bool{}
	o := []string{}
	ps0("*files to annotate:\n")
	for _, f := range files.files {
		if f.action == FAAnnotate {
			seen[f.filename] = true
			o = append(o, f.filename)
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files to writeast:\n")
	for _, f := range files.files {
		if f.action == FAWriteAst {
			seen[f.filename] = true
			o = append(o, f.filename)
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files to copy:\n")
	for _, f := range files.files {
		if f.action == FACopy {
			seen[f.filename] = true
			o = append(o, f.filename)
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files to create:\n")
	for _, f := range files.files {
		if f.action == FACreate {
			seen[f.filename] = true
			o = append(o, f.filename)
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files to writemod:\n")
	for _, f := range files.files {
		if f.action == FAWriteMod {
			seen[f.filename] = true
			o = append(o, f.filename)
		}
	}
	files.sortedFilesStr(o, ps0)

	o = []string{}
	ps0("*files (other):\n")
	for _, f := range files.files {
		if !seen[f.filename] {
			o = append(o, f.filename)
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
		if f.hasMainFunc {
			fn += " (main)"
		}
		if f.mainModule {
			fn += " (main mod)"
		}
		if f.typ == FTSrc {
			fn += fmt.Sprintf(" (%v)", f.annType)
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
	for _, f := range files.files {
		switch f.action {
		case FACopy:
			dest := cmd.tmpDirBasedFilename(f.destFilename())
			if err := mkdirAllCopyFileSync(f.filename, dest); err != nil {
				return err
			}

		case FACreate:
			dest := cmd.tmpDirBasedFilename(f.destFilename())
			if err := mkdirAllWriteFile(dest, []byte(f.createContent)); err != nil {
				return err
			}

		case FAWriteAst:
			astFile, err := files.fullAstFile(f.filename)
			if err != nil {
				return err
			}
			dest := cmd.tmpDirBasedFilename(f.destFilename())
			if err := cmd.mkdirAllWriteAstFile(dest, astFile); err != nil {
				return err
			}

		case FAWriteMod:
			m, err := f.modFile()
			if err != nil {
				return err
			}
			m.SortBlocks()
			m.Cleanup()
			b1, err := m.Format()
			if err != nil {
				return err
			}
			dest := cmd.tmpDirBasedFilename(f.destFilename())
			//cmd.Vprintf("===go.mod===\n")
			//cmd.Vprintf("%v\n%v", dest, string(b1))
			if err := mkdirAllWriteFile(dest, b1); err != nil {
				return err
			}
		}
	}
	return nil
}

//----------

func (files *Files) tryToSetupGoSumForCopy(modF *File) {
	// do once
	if modF.triedGoSum {
		return
	}
	modF.triedGoSum = true

	goSum := "go.sum"

	// next to go.mod
	dir1 := filepath.Dir(modF.filename)
	fname1 := filepath.Join(dir1, goSum)

	// next to go.mod dest
	dir2 := filepath.Dir(modF.destFilename())
	fname2 := filepath.Join(dir2, goSum)

	// must exist
	if _, err := os.Stat(fname1); err != nil {
		if _, err := os.Stat(fname2); err != nil {
			return
		}
		fname1 = fname2
	}

	f3 := files.NewFile(fname1, FTSum, nil)
	f3.action = FACopy
	f3.destFilename2 = fname2
}

//----------

func (files *Files) NewFile(filename string, typ FileType, pkg *packages.Package) *File {
	f := &File{files: files, filename: filename}
	f.typ = typ
	if pkg != nil {
		f.pkg = pkg
		f.pkgPath = pkg.PkgPath
		f.pkgName = pkg.Name
		if pkg.Module != nil {
			f.mainModule = pkg.Module.Main
			f.modulePath = pkg.Module.Path
			f.moduleVersion = pkg.Module.Version
		}
	}
	files.files[f.filename] = f // keep it in map
	return f
}

//----------

//func (files *Files) astIdentObj(id *ast.Ident) (types.Object, bool) {
//	found := false
//	o := (types.Object)(nil)
//	packages.Visit(files.pkgs, func(pkg *packages.Package) bool {
//		if found {
//			return false // don't visit imports
//		}
//		u := pkg.TypesInfo.ObjectOf(id)
//		if u != nil {
//			found = true
//			o = u
//			return false // don't visit imports
//		}
//		return true // visit imports
//	}, nil)
//	return o, found
//}

//----------

//func (files *Files) reviewTmpDirCache() error {
//	// TODO: in one rune a package was annotated, but in another run, the package is not required to be annotated, but the annotated files are in the cache and going to be used
//	// TODO: in progfilenames and annotated, if in cache could have been just copied?
//	// TODO: in progfilenames and not annotated, if in cache could have been annotated?

//	if files.TmpDir == "" {
//		return nil
//	}

//	fileAtTmp := func(f string) string {
//		return filepath.Join(files.TmpDir, f)
//	}

//	// cache version (allows to invalidate the cache for update reasons)
//	version := "1"
//	versionFile := fileAtTmp("version")
//	cacheValid, create := false, false
//	b, err := ioutil.ReadFile(versionFile)
//	if os.IsNotExist(err) {
//		create = true
//	} else {
//		v := string(b)
//		if v == version {
//			cacheValid = true
//		}
//	}
//	if !cacheValid {
//		os.RemoveAll(files.TmpDir)
//	}
//	if create {
//		if err := iout.MkdirAllWriteFile(versionFile, []byte(version), 0660); err != nil {
//			return err
//		}
//	}

//	// merge maps keys (filenames to check)
//	m := map[string]struct{}{}
//	for k := range files.annFilenames {
//		m[k] = struct{}{}
//	}
//	for k := range files.copyFilenames {
//		m[k] = struct{}{}
//	}
//	for k := range files.modFilenames {
//		m[k] = struct{}{}
//	}
//	for k := range files.modMissings {
//		m[k] = struct{}{}
//	}

//	// compare cache files with original files
//	for f1 := range m {
//		// cached
//		f2 := fileAtTmp(f1)
//		fi2, err := os.Stat(f2)
//		if err != nil {
//			continue // cached file missing
//		}
//		// original
//		fi1, err := os.Stat(f1)
//		if err != nil {
//			// original file missing (ex: a go.mod)
//			_ = os.Remove(f2) // remove from cache
//			continue
//		}

//		cacheIsOld := fi2.ModTime().Before(fi1.ModTime())
//		if cacheIsOld {
//			_ = os.Remove(f2)
//			continue
//		}

//		delete(files.copyFilenames, f1)
//		delete(files.annFilenames, f1)
//		//fmt.Printf("using cached: %v\n", f1)
//	}
//	return nil
//}

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
	s2 := s[len(prefix):]
	typ, opt, hasOpt := s2, "", false
	i := strings.Index(typ, ":")
	if i >= 0 {
		hasOpt = true
		typ, opt = typ[:i], typ[i+1:]
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
		err := fmt.Errorf("godebug: unexpected annotate type: %q", s2)
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
}

func AnnotationOptInComment(c *ast.Comment, fset *token.FileSet) (*AnnotationOpt, error) {
	typ, opt, err := AnnotationTypeInString(c.Text)
	if err != nil {
		return nil, err
	}
	return &AnnotationOpt{typ, opt, c}, nil
}

//----------

func ProgramPackages(
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
		p := "file=" + f
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

func positionError(err error, fset *token.FileSet, pos token.Pos) error {
	p := fset.Position(pos)
	return fmt.Errorf("%v: %w", p, err)
}

//----------

func annOptCommentsNodesMap(fset *token.FileSet, astFile *ast.File, opts []*AnnotationOpt) map[ast.Node]*AnnotationOpt {
	// wrap comments in commentgroups
	cgs := []*ast.CommentGroup{}
	m := map[*ast.CommentGroup]*AnnotationOpt{}
	for _, opt := range opts {
		cg := &ast.CommentGroup{List: []*ast.Comment{opt.Comment}}
		cgs = append(cgs, cg)
		m[cg] = opt
	}
	// map nodes to comments
	cmap := ast.NewCommentMap(fset, astFile, cgs)
	mn := map[ast.Node]*AnnotationOpt{}
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch n.(type) {
		case nil, *ast.CommentGroup, *ast.Comment:
			return false
		}
		cgs, ok := cmap[n]
		if ok {
			mn[n] = m[cgs[0]]
		}
		return true
	})
	return mn
}

//----------

func sourceHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}

//----------

type File struct {
	filename string
	action   FileAction
	typ      FileType

	// type: src
	annType     AnnotationType // annotate=true
	moduleGoMod *File          // goFile=true
	triedGoSum  bool
	pkgPath     string
	pkgName     string
	hasMainFunc bool
	annFileData *AnnFileData

	// type: mod
	mainModule    bool // goMod=true
	needDebugMods bool

	// common
	createContent string
	destFilename2 string
	modulePath    string // goMod or goFile
	moduleVersion string

	// others
	files *Files
	pkg   *packages.Package
	cache struct {
		modFile *modfile.File
	}
}

//----------

func (f *File) used() bool {
	return f.action != FANone
}

func (f *File) canHaveMain(testMode bool) bool {
	if f.typ == FTSrc {
		if testMode {
			isTest := strings.HasSuffix(f.filename, "_test.go")
			if !isTest {
				return false
			}
		}
		if f.moduleGoMod == nil {
			return true
		} else {
			if f.moduleGoMod.mainModule {
				return true
			}
		}
	}
	return false
}

//----------

//func (f *File) fullAstFile() (*ast.File, error) {
//	return f.files.fullAstFile(f.filename)
//}

func (f *File) destFilename() string {
	if f.destFilename2 != "" {
		return f.destFilename2
	}
	return f.filename
}

func (f *File) shortFilename() string {
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

//func (f *File) splitFilename() (string, string, string, string) {
//	// split: dir, modpath, pkgpath, filename
//	// modulepath: example.com/pkg1
//	// pkgpath: example.com/pkg1
//	// pkgpath: example.com/pkg1/other/pkg2

//	partialPkgPath := ""
//	if strings.HasPrefix(f.pkgPath, f.modulePath) {
//		u := f.pkgPath[len(f.modulePath):]
//		partialPkgPath = filepath.FromSlash(u)
//	}

//	dir, fname := filepath.Split(f.filename)
//	dir = filepath.Clean(dir)
//	if strings.HasSuffix(dir, partialPkgPath) {
//		dir = dir[:len(dir)-len(partialPkgPath)]
//		dir = filepath.Clean(dir)
//	}

//	modPath := filepath.FromSlash(f.modulePath)
//	if strings.HasPrefix(dir, modPath) {
//		dir = dir[:len(dir)-len(modPath)]
//		dir = filepath.Clean(dir)
//	}

//	return dir, modPath, partialPkgPath, fname
//}

func (f *File) modFile() (*modfile.File, error) {
	if f.cache.modFile != nil {
		return f.cache.modFile, nil
	}
	if f.typ != FTMod {
		return nil, errors.New("not a go.mod")
	}
	// read src
	src := []byte{}
	if f.action == FACreate {
		if f.createContent != "" {
			src = []byte(f.createContent)
		} else {
			return nil, errors.New("missing go.mod content (create)")
		}
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

func (f *File) astIdentObj(id *ast.Ident) (types.Object, bool) {
	isTesting := f.pkg == nil
	if isTesting {
		return nil, false
	}

	u := f.pkg.TypesInfo.ObjectOf(id)
	if u != nil {
		return u, true
	}
	return nil, false
}

//----------

type FileAction int

const (
	FANone FileAction = iota
	FACopy
	FACreate
	FAAnnotate
	FAWriteAst
	FAWriteMod
)

type FileType int

const (
	FTSrc FileType = iota
	FTMod
	FTSum
)
