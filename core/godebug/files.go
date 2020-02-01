package godebug

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/osutil"
	"golang.org/x/tools/go/packages"
)

// Finds the set of files that need to be annotated/copied.
type Files struct {
	Dir string

	filenames       map[string]struct{} // filenames to solve
	progFilenames   map[string]struct{} // program filenames (loaded)
	progDirPkgPaths map[string]string   // prog dir pkgPath
	annFilenames    map[string]struct{} // to annotate
	copyFilenames   map[string]struct{} // to copy
	modFilenames    map[string]struct{} // go.mod's
	modMissings     map[string]struct{} // go.mod's to be created
	annTypes        map[string]AnnotationType

	fset      *token.FileSet
	noModules bool
	cache     struct {
		fullAstFile map[string]*ast.File
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
	files.cache.fullAstFile = map[string]*ast.File{}
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
		// always add to avoid confusion (run godebug and not show anything)
		files.Add(mainFilename) // direct add for annotation
	}

	// TODO: add only test files, not whole directory
	// add tests directory
	if tests && files.Dir != "" {
		files.Add(files.Dir) // direct add for annotation
	}

	// all program packages
	loadMode := 0 |
		packages.NeedDeps |
		packages.NeedImports |
		packages.NeedName | // name and pkgpath
		packages.NeedFiles |
		0

	parseFile := func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		return files.fullAstFile(filename)
	}

	pkgs, err := ProgramPackages(ctx, files.fset, loadMode, files.Dir, mainFilename, tests, env, parseFile)
	if err != nil {
		return err
	}
	files.populateProgFilenamesMap(pkgs)

	if err := files.addFilesWithDebugComment(ctx, pkgs); err != nil {
		return err
	}
	if err := files.solveFilenames(); err != nil {
		return err
	}
	if err := files.findGoMods(); err != nil {
		return err
	}

	files.findFilesToCopy(files.noModules)

	return nil
}

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
	// can't include these because they can't be annotated (will not be able to include a "replace" directive in go.mod)
	goRoot := os.Getenv("GOROOT")
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

func (files *Files) pkgPathDir(filename string) (string, bool) {
	if p, ok := files.progDirPkgPaths[filename]; ok {
		return p, true
	}
	d := filepath.Dir(filename)
	if p, ok := files.progDirPkgPaths[d]; ok {
		u := filepath.Join(p, filepath.Base(filename))
		return u, true
	}
	return "", false
}

//----------

func (files *Files) addFilesWithDebugComment(ctx context.Context, pkgs []*packages.Package) error {
	for filename := range files.progFilenames {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		astFile, err := files.fullAstFile(filename)
		if err != nil {
			return err
		}
		for _, cg := range astFile.Comments {
			for _, c := range cg.List {
				opt, err := AnnotationOptInComment(c)
				if err != nil {
					return PositionError(err, files.fset, c.Pos())
				}
				err = files.keepAnnotationOpt(filename, opt)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (files *Files) keepAnnotationOpt(filename string, opt *AnnotationOpt) error {
	switch opt.Type {
	case AnnotationTypeNone:
	case AnnotationTypeOff:
	case AnnotationTypeBlock:
		files.keepAnnFilename(filename, opt.Type)
	case AnnotationTypeFile:
		files.keepAnnFilename(filename, opt.Type)
	case AnnotationTypePackage:
		if opt.Opt == "" {
			files.keepAnnFilename(filename, opt.Type) // keep
			dir := filepath.Dir(filename)
			files.Add(dir) // add package dir
		} else {
			// filename is not added
			ok := false
			for dir, pkgPath := range files.progDirPkgPaths {
				if pkgPath == opt.Opt {
					ok = true
					files.Add(dir) // add pkg dir
				}
			}
			if !ok {
				err := fmt.Errorf("pkg to annotate not used: %q", opt.Opt)
				err = PositionError(err, files.fset, opt.Pos)
				return err
			}
		}
	case AnnotationTypeImport:
		// read next line, must be an import
	case AnnotationTypeModule:
		files.keepAnnFilename(filename, opt.Type)
		dir := filepath.Dir(filename)
		goMod, ok := goutil.FindGoMod(dir)
		if ok {
			dir2 := filepath.Dir(goMod) + string(filepath.Separator)
			// files under the gomod directory
			for f := range files.progFilenames {
				if strings.HasPrefix(f, dir2) {
					files.keepAnnFilename(f, opt.Type)
				}
			}
		}
	default:
		log.Printf("todo: keepAnnotationOpt: %v", opt.Type)
	}
	return nil
}

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

//----------

func (files *Files) solveFilenames() error {
	for fname := range files.filenames {
		fi, err := os.Stat(fname)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			// add all dir program files
			dir := fname
			for k := range files.progFilenames {
				// dir program file
				if osutil.FilepathHasDirPrefix(k, dir) {
					files.solveDirectoryFile(dir, k)
				}
			}
		}
		// must be a program file
		_, ok := files.progFilenames[fname]
		if ok {
			files.keepAnnFilename(fname, AnnotationTypeFile)
		}
	}
	return nil
}

func (files *Files) solveDirectoryFile(dir, filename string) {
	// in subdirectory
	rest := osutil.FilepathSplitAt(filename, len(dir))
	inSubDir := strings.Index(rest, string(filepath.Separator)) >= 0

	if !inSubDir {
		files.keepAnnFilename(filename, AnnotationTypeFile)
	}
}

//----------

func (files *Files) keepAnnFilename(filename string, typ AnnotationType) {
	files.annFilenames[filename] = struct{}{}
	t, ok := files.annTypes[filename]
	if !ok || typ > t { // update if higher
		files.annTypes[filename] = typ
	}
}

//----------

func (files *Files) findGoMods() error {
	// search (to be) annotated files dir for go.mod's
	seen := map[string]struct{}{}
	for f := range files.annFilenames {
		dir := filepath.Dir(f)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		u, ok := goutil.FindGoMod(dir)
		if ok {
			files.modFilenames[u] = struct{}{}
		} else {
			w := filepath.Join(dir, "go.mod")
			files.modMissings[w] = struct{}{}
		}
	}
	return nil
}

//----------

func (files *Files) fullAstFile(filename string) (*ast.File, error) {
	astFile, ok := files.cache.fullAstFile[filename]
	if ok {
		return astFile, nil
	}
	mode := parser.ParseComments // full mode
	astFile, err := parser.ParseFile(files.fset, filename, nil, mode)
	if err != nil {
		return nil, err
	}
	files.cache.fullAstFile[filename] = astFile
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

type AnnotationType int

const (
	// Order matters, last is the bigger set
	AnnotationTypeNone AnnotationType = iota
	AnnotationTypeOff
	AnnotationTypeBlock
	AnnotationTypeFile
	AnnotationTypeImport  // annotates set of files (reads next src line)
	AnnotationTypePackage // annotates set of files
	AnnotationTypeModule  // annotates set of packages
)

//----------

type AnnotationOpt struct {
	Type AnnotationType
	Opt  string
	Pos  token.Pos
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

//----------

func ProgramPackages(
	ctx context.Context,
	fset *token.FileSet,
	mode packages.LoadMode,
	dir, filename string,
	tests bool,
	env []string,
	parseFile func(
		fset *token.FileSet,
		filename string, src []byte,
	) (*ast.File, error),
) ([]*packages.Package, error) {

	cfg := &packages.Config{
		Context:   ctx,
		Fset:      fset,
		Tests:     tests,
		Dir:       dir,
		Mode:      mode,
		Env:       env,
		ParseFile: parseFile,
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

func AnnotationOptInComment(c *ast.Comment) (*AnnotationOpt, error) {
	typ, opt, err := AnnotationTypeInString(c.Text)
	if err != nil {
		return nil, err
	}
	return &AnnotationOpt{typ, opt, c.Pos()}, nil
}

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

	// ensure early error if opt is set
	if hasOpt {
		switch at {
		case AnnotationTypePackage:
		default:
			return at, opt, fmt.Errorf("godebug: unexpected annotate string: %v", opt)
		}
	}

	return at, opt, nil
}

//----------

func PositionError(err error, fset *token.FileSet, pos token.Pos) error {
	p := fset.Position(pos)
	return fmt.Errorf("%v: %w", p, err)
}
