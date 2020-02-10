package godebug

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/osutil"

	"golang.org/x/tools/go/packages"
)

// Finds the set of files that need to be annotated/copied.
type Files struct {
	Dir string
	//TmpDir string

	filenames       map[string]struct{}       // filenames to solve
	progFilenames   map[string]struct{}       // program filenames (loaded)
	progDirPkgPaths map[string]string         // [dir] pkgPath
	annFilenames    map[string]struct{}       // to annotate
	copyFilenames   map[string]struct{}       // to copy
	modFilenames    map[string]struct{}       // go.mod's
	modMissings     map[string]struct{}       // go.mod's to be created
	annTypes        map[string]AnnotationType // [filename]
	annFileData     map[string]*AnnFileData   // [filename] hash/filesize
	nodeAnnTypes    map[ast.Node]AnnotationType

	fset      *token.FileSet
	noModules bool
	cache     struct {
		sync.RWMutex
		fullAstFile map[string]*ast.File
		srcs        map[string][]byte // cleared at the end
	}
}

func NewFiles(fset *token.FileSet, noModules bool) *Files {
	//spew.Config.DisableMethods = true

	files := &Files{fset: fset, noModules: noModules}
	files.filenames = map[string]struct{}{}
	files.progFilenames = map[string]struct{}{}
	files.progDirPkgPaths = map[string]string{}
	files.annFilenames = map[string]struct{}{}
	files.copyFilenames = map[string]struct{}{}
	files.modFilenames = map[string]struct{}{}
	files.modMissings = map[string]struct{}{}
	files.annTypes = map[string]AnnotationType{}
	files.annFileData = map[string]*AnnFileData{}
	files.nodeAnnTypes = map[ast.Node]AnnotationType{}
	files.cache.fullAstFile = map[string]*ast.File{}
	files.cache.srcs = map[string][]byte{}
	return files
}

//----------

// Add filenames (including directories).
func (files *Files) Add(filenames ...string) {
	for _, filename := range filenames {
		filename = files.absFilename(filename)
		files.filenames[filename] = struct{}{}
	}
}

func (files *Files) absFilename(filename string) string {
	if !filepath.IsAbs(filename) {
		u := filepath.Join(files.Dir, filename)
		v, err := filepath.Abs(u)
		if err == nil {
			filename = v
		}
	}
	return filename
}

//----------

func (files *Files) Do(ctx context.Context, mainFilename string, tests bool, env []string) error {
	if !filepath.IsAbs(mainFilename) {
		return fmt.Errorf("filename not absolute: %v", mainFilename)
	}

	// add mainfilename
	if !tests && mainFilename != "" {
		// add to avoid confusion (run godebug and not show anything)
		files.Add(mainFilename)
	}

	// add tests directory
	if tests {
		//files.Add(files.Dir) // auto add the full directory (useful)

		// add only test files (can always add an annotatepackage on the file)
		fis, err := ioutil.ReadDir(files.Dir)
		if err != nil {
			return err
		}
		for _, fi := range fis {
			w := filepath.Join(files.Dir, fi.Name())
			if strings.HasSuffix(w, "_test.go") {
				files.Add(w)
			}
		}
	}

	// all program packages
	pkgs, err := files.progPackages(ctx, mainFilename, tests, env)
	if err != nil {
		return err
	}

	files.populateProgFilenamesMap(pkgs)
	if err := files.addCommentedFiles(ctx); err != nil {
		return err
	}
	if err := files.solveFilenames(); err != nil {
		return err
	}
	files.findGoMods()
	files.findFilesToCopy(files.noModules)
	if err := files.doAnnFilesHashes(); err != nil {
		return err
	}
	//if err := files.reviewTmpDirCache(); err != nil {
	//	return err
	//}
	return nil
}

//----------

func (files *Files) verbose(cmd *Cmd) {
	cmd.Printf("program:\n")
	files.verboseMap(cmd, files.progFilenames)
	cmd.Printf("annotate:\n")
	files.verboseMap(cmd, files.annFilenames)
	cmd.Printf("copy:\n")
	files.verboseMap(cmd, files.copyFilenames)
	cmd.Printf("modules:\n")
	files.verboseMap(cmd, files.modFilenames)
	cmd.Printf("modules missing:\n")
	files.verboseMap(cmd, files.modMissings)
}
func (files *Files) verboseMap(cmd *Cmd, m map[string]struct{}) {
	sl := []string{}
	for k := range m {
		// shorten k
		sep := string(filepath.Separator)
		j := strings.LastIndex(k, sep)
		if j > 0 {
			j = strings.LastIndex(k[:j], sep)
			if j > 0 {
				k = k[j:]
				if len(k) > 0 && k[0] == sep[0] {
					k = k[1:]
				}
			}
		}

		sl = append(sl, k)
	}
	sort.Strings(sl)
	for _, s := range sl {
		cmd.Printf("\t%v\n", s)
	}
}

//----------

func (files *Files) populateProgFilenamesMap(pkgs []*packages.Package) {
	// There is no "std" module for the std library, "replace" won't work
	goRoot := os.Getenv("GOROOT")
	goRoot = filepath.Clean(goRoot) + string(filepath.Separator)

	// don't annotate a possible second duplicate module (self debug)
	godebugPkgs := []string{GodebugconfigPkgPath, DebugPkgPath}

	// all filenames in the program (except goroot)
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
	loop1:
		for _, fname := range pkg.GoFiles {
			// skip filepaths
			if osutil.FilepathHasDirPrefix(fname, goRoot) {
				continue
			}
			// skip pkgs
			for _, p := range godebugPkgs {
				if p == pkg.PkgPath {
					continue loop1
				}
			}

			files.progFilenames[fname] = struct{}{}

			// map pkg path
			dir := filepath.Dir(fname)
			files.progDirPkgPaths[dir] = pkg.PkgPath
		}
	})
}

//----------

func (files *Files) addCommentedFiles(ctx context.Context) error {
	for filename := range files.progFilenames {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := files.addCommentedFile(filename); err != nil {
			return err
		}
	}
	return nil
}

func (files *Files) addCommentedFile(filename string) error {
	astFile, err := files.fullAstFile(filename)
	if err != nil {
		return err
	}
	return files.addCommentedFile2(filename, astFile)
}

func (files *Files) addCommentedFile2(filename string, astFile *ast.File) error {
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
				return err
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
				return err
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
			_, ok := files.progFilenames[f]
			if !ok {
				return false, fmt.Errorf("file not found in loaded program: %v", opt.Opt)
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
		dir := filepath.Dir(filename)
		if opt.Opt != "" {
			dir2, ok := files.pkgPathDir(opt.Opt)
			if !ok {
				return false, fmt.Errorf("pkg path dir not found in loaded program: %v", opt.Opt)
			}
			dir = dir2
		}
		goMod, _ := files.findGoModOrMissing(dir)
		// files under the gomod directory
		dir2 := filepath.Dir(goMod) + string(filepath.Separator)
		for f := range files.progFilenames {
			if strings.HasPrefix(f, dir2) {
				files.addAnnFilename(f, opt.Type)
			}
		}
		return true, nil
	default:
		err := fmt.Errorf("todo: keepAnnotationOpt: %v", opt.Type)
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
	dir, ok := files.pkgPathDir(pkgPath)
	if !ok {
		return fmt.Errorf("pkg to annotate not found in loaded program: %q", pkgPath)
	}
	return files.addAnnDir(dir, typ)
}

//----------

func (files *Files) solveFilenames() error {
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
	// must be a program file
	_, ok := files.progFilenames[filename]
	if !ok {
		return
	}

	files.annFilenames[filename] = struct{}{}

	t, ok := files.annTypes[filename]
	if !ok || typ > t { // update if higher
		files.annTypes[filename] = typ
	}
}

func (files *Files) addedAnnFilename(filename string, typ AnnotationType) bool {
	t, ok := files.annTypes[filename]
	if !ok {
		return false
	}
	return typ <= t
}

func (files *Files) addAnnDir(dir string, typ AnnotationType) error {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		u := filepath.Join(dir, fi.Name())
		files.addAnnFilename(u, typ)
	}
	return nil
}

//----------

func (files *Files) findGoMods() {
	// search annotated files dir for go.mod's
	seen := map[string]struct{}{}
	for f := range files.annFilenames {
		dir := filepath.Dir(f)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		u, found := files.findGoModOrMissing(dir)
		if found {
			files.modFilenames[u] = struct{}{}
		} else {
			files.modMissings[u] = struct{}{}
		}
	}
}

func (files *Files) findGoModOrMissing(dir string) (_ string, found bool) {
	u, ok := goutil.FindGoMod(dir)
	if ok {
		return u, true
	}
	// find where the go.mod location should be (lowest common denominator)
	for u := range files.progFilenames {
		dir2 := filepath.Dir(u) + string(filepath.Separator)
		if strings.HasPrefix(dir, dir2) {
			dir = dir2
		}
	}
	f := filepath.Join(dir, "go.mod")
	return f, false
}

func (files *Files) modFilenamesAndMissing() map[string]struct{} {
	mods := map[string]struct{}{}
	w := []map[string]struct{}{files.modFilenames, files.modMissings}
	for _, m := range w {
		for k, v := range m {
			mods[k] = v
		}
	}
	return mods
}

//----------

func (files *Files) parseFileFn() func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	return func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		return files.fullAstFile2(filename, src)
	}
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

func (files *Files) findFilesToCopy(noModules bool) {
	if noModules {
		files.findParentDirsOfAnnotated()
	} else {
		files.findModulesFiles()
	}
}

func (files *Files) findModulesFiles() {
	// all mods, need to deal with directories of missing go.mod's
	for fn1 := range files.modFilenamesAndMissing() {
		dir1 := filepath.Dir(fn1)
		for fn2 := range files.progFilenames {
			if !files.canCopyOnly(fn2) {
				continue
			}
			// inside dir tree of a mod file
			if osutil.FilepathHasDirPrefix(fn2, dir1) {
				files.copyFilenames[fn2] = struct{}{}
			}
		}
	}
}

func (files *Files) findParentDirsOfAnnotated() {
	for fn1 := range files.annFilenames {
		for fn2 := range files.progFilenames {
			if !files.canCopyOnly(fn2) {
				continue
			}
			// parent directory of an annotated file
			dir2 := filepath.Dir(fn2)
			if osutil.FilepathHasDirPrefix(fn1, dir2) {
				files.copyFilenames[fn2] = struct{}{}
			}
		}
	}
}

func (files *Files) canCopyOnly(filename string) bool {
	// annotated files are not for copy-only
	_, ok := files.annFilenames[filename]
	if ok {
		return false
	}
	return true
}

//----------

func (files *Files) DebugPkgFilename(filename string) string {
	fp := filepath.FromSlash(DebugPkgPath)

	// cosmetic output, not necessary
	if !files.noModules {
		fp = filepath.FromSlash("godebug/debug")
	}

	return filepath.Join(fp, filename)
}

func (files *Files) GodebugconfigPkgFilename(filename string) string {
	fp := filepath.FromSlash(GodebugconfigPkgPath)

	// cosmetic output, not necessary
	if !files.noModules {
		fp = filepath.FromSlash("godebug/godebugconfig")
	}

	return filepath.Join(fp, filename)
}

//----------

func (files *Files) doAnnFilesHashes() error {
	// allow cache garbage collect at the end
	defer func() {
		files.cache.srcs = nil
	}()

	for f := range files.annFilenames {
		src, ok := files.cache.srcs[f]
		if !ok {
			return fmt.Errorf("missing src: %v", src)
		}
		afd := &AnnFileData{
			FileSize: len(src),
			FileHash: sourceHash(src),
		}
		files.annFileData[f] = afd
	}
	return nil
}

//----------

func (files *Files) pkgPathDir(pkgPath string) (string, bool) {
	for dir, pkgPath2 := range files.progDirPkgPaths {
		if pkgPath2 == pkgPath {
			return dir, true
		}
	}
	return "", false
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

func (files *Files) progPackages(ctx context.Context, filename string, tests bool, env []string) ([]*packages.Package, error) {
	loadMode := 0 |
		packages.NeedDeps |
		packages.NeedImports |
		packages.NeedName | // name and pkgpath
		packages.NeedFiles |
		// TODO: needed to have custom parsefile be used
		//packages.NeedSyntax |
		packages.NeedTypes |
		0
	pkgs, err := ProgramPackages(ctx, files.fset, loadMode, files.Dir, filename, tests, env, files.parseFileFn())
	// programpackages parses files concurrently, on ctx cancel it concats useless repeated errors, get just one ctx error
	if err2 := ctx.Err(); err2 != nil {
		return nil, err2
	}
	return pkgs, err
}

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
	dir, filename string,
	tests bool,
	env []string,
	parseFile func(fset *token.FileSet, filename string, src []byte) (*ast.File, error),
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
	}
	pattern := ""
	if !tests && filename != "" {
		pattern = "file=" + filename
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}

	// errors: join errors into one error (check packages.PrintErrors(pkgs))
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

//func AstFileAnnotationType(file *ast.File) AnnotationType {
//	stop := false
//	typ := AnnotationTypeNone
//	var vis VisitorFn
//	vis = func(node ast.Node) ast.Visitor {
//		if stop {
//			return nil
//		}
//		if ce, ok := node.(*ast.CallExpr); ok {
//			if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
//				if id, ok := se.X.(*ast.Ident); ok {
//					if id.Name == "debug" {
//						u := AnnotationTypeNone
//						switch se.Sel.Name {
//						//case "NoAnnotations":
//						//u = AnnotationTypeNoAnnotations
//						case "AnnotateBlock":
//							u = AnnotationTypeBlock
//						case "AnnotateFile":
//							u = AnnotationTypeFile
//						case "AnnotatePackage":
//							u = AnnotationTypePackage
//							stop = true // no higher level
//						}
//						if typ < u {
//							typ = u
//						}
//					}
//				}
//			}
//		}
//		return vis
//	}
//	ast.Walk(vis, file)
//	return typ
//}

//type VisitorFn func(ast.Node) ast.Visitor

//func (x VisitorFn) Visit(node ast.Node) ast.Visitor {
//	return x(node)
//}

//----------

//func PkgFilenames(ctx context.Context, dir string, tests bool) ([]string, error) {
//	cfg := &packages.Config{
//		Context: ctx,
//		Dir:     dir,
//		Mode:    packages.NeedFiles,
//		Tests:   tests,
//	}
//	pkgs, err := packages.Load(cfg, "")
//	if err != nil {
//		return nil, err
//	}
//	files := []string{}
//	for _, pkg := range pkgs {
//		files = append(files, pkg.GoFiles...)
//	}
//	return files, nil
//}

//func PackagesImportingPkgPath(pkgs []*packages.Package, pkgPath string) []*packages.Package {
//	pkgsImporting := []*packages.Package{}
//	visited := map[*packages.Package]bool{}
//	var visit func(parent, pkg *packages.Package)
//	visit = func(parent, pkg *packages.Package) {
//		// don't mark pkgPath as visited or it won't work
//		if pkg.PkgPath != pkgPath {
//			// mark visited
//			if visited[pkg] {
//				return
//			}
//			visited[pkg] = true
//		}
//		if parent != nil && pkg.PkgPath == pkgPath {
//			pkgsImporting = append(pkgsImporting, parent)
//		}
//		for _, pkg2 := range pkg.Imports {
//			visit(pkg, pkg2)
//		}
//	}
//	for _, pkg := range pkgs {
//		visit(nil, pkg)
//	}
//	return pkgsImporting
//}

//----------

//func FileImportsPkgPath(file *ast.File, pkgPath string) bool {
//	for _, decl := range file.Decls {
//		switch t := decl.(type) {
//		case *ast.GenDecl:
//			for _, spec := range t.Specs {
//				switch t2 := spec.(type) {
//				case *ast.ImportSpec:
//					s := t2.Path.Value
//					if len(s) > 2 {
//						s = s[1 : len(s)-1]
//						if s == pkgPath {
//							return true
//						}
//					}

//				}
//			}
//		}
//	}
//	return false
//}

//----------

//----------

//func (files *Files) addFilesImportingDebugPkg(pkgs []*packages.Package) error {
//	filenamesImp, err := files.filenamesImportingDebugPkg(pkgs)
//	if err != nil {
//		return err
//	}
//	// type of annotation in the files that import the debug pkg
//	for _, filename := range filenamesImp {
//		astFile, err := files.fullAstFile(filename)
//		if err != nil {
//			return err
//		}
//		typ := AstFileAnnotationType(astFile)
//		if typ.Annotated() {
//			files.keepAnnFilename(filename, typ)
//			// add directory containing package
//			if typ == AnnotationTypePackage {
//				dir := filepath.Dir(filename)
//				files.Add(dir)
//			}
//		}
//	}
//	return nil
//}

//func (files *Files) filenamesImportingDebugPkg(pkgs []*packages.Package) ([]string, error) {
//	debugPkgPath := "github.com/jmigpin/editor/core/godebug/debug"
//	pkgsImp := PackagesImportingPkgPath(pkgs, debugPkgPath)
//	mode := parser.ImportsOnly // fast mode
//	u := []string{}
//	for _, pkg := range pkgsImp {
//		for _, filename := range pkg.GoFiles {
//			astFile, err := parser.ParseFile(files.fset, filename, nil, mode)
//			if err != nil {
//				return nil, err
//			}
//			if FileImportsPkgPath(astFile, debugPkgPath) {
//				u = append(u, filename)
//			}
//		}
//	}
//	return u, nil
//}
