package godebug

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
	"golang.org/x/tools/go/packages"
)

type Files struct {
	Dir string

	fset      *token.FileSet
	filenames map[string]struct{}
	notes     map[string]*FileNote
}

func NewFiles(fset *token.FileSet) *Files {
	spew.Config.DisableMethods = true

	files := &Files{fset: fset}
	files.filenames = map[string]struct{}{}
	files.notes = map[string]*FileNote{}

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

func (files *Files) Do(ctx context.Context, mainFilename *string, tests bool) ([]*FileNote, error) {
	debug.AnnotateFile()
	if !tests && *mainFilename != "" {
		*mainFilename = files.absFilename(*mainFilename)

		// always add to avoid confusion (run godebug and not show anything)
		files.Add(*mainFilename) // direct add for annotation
	}

	if tests && files.Dir != "" {
		files.Add(files.Dir) // direct add for annotation
	}

	// all program packages
	loadMode := 0 |
		packages.NeedDeps |
		packages.NeedImports |
		packages.NeedName | // name and pkgpath
		packages.NeedFiles
	//tmpFset := token.NewFileSet()
	pkgs, err := ProgramPackages(ctx, loadMode, files.Dir, *mainFilename, tests, files.fset)
	if err != nil {
		return nil, err
	}

	// filenames that import the debug pkg
	filenamesImp, err := files.filenamesImportingDebugPkg(pkgs)
	if err != nil {
		return nil, err
	}

	// all filenames
	allFilenames := map[string]struct{}{}
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, fname := range pkgGoFiles(pkg) {
			allFilenames[fname] = struct{}{}
		}
	})

	// type of annotation in the files that import the debug pkg
	mode := parser.ParseComments // full mode
	for _, filename := range filenamesImp {
		astFile, err := parser.ParseFile(files.fset, filename, nil, mode)
		if err != nil {
			return nil, err
		}
		typ, ok := AstFileAnnotationType(astFile)
		if !ok {
			continue
		}
		if typ >= AnnotationTypeBlock {
			note := &FileNote{Filename: filename, Type: typ, AstFile: astFile}
			note.DebugSrc = "src annotated"
			if err := files.addFileNote(note); err != nil {
				return nil, err
			}
		}
		// add directory containing package
		if typ == AnnotationTypePackage {
			dir := filepath.Dir(filename)
			err := files.addDir(ctx, dir, tests, allFilenames)
			if err != nil {
				return nil, err
			}
		}
	}

	// handle added filenames (that can include directories)
	for fname, _ := range files.filenames {
		fi, err := os.Stat(fname)
		if err != nil {
			return nil, err
		}
		// directories
		if fi.IsDir() {
			err := files.addDir(ctx, fname, tests, allFilenames)
			if err != nil {
				return nil, err
			}
		} else { // regular files
			// must exist in all filenames that belong to the program
			if _, ok := allFilenames[fname]; !ok {
				continue
			}

			note := &FileNote{Filename: fname, Type: AnnotationTypeFile}
			note.DebugSrc = "add file (direct)"
			if err := files.addFileNote(note); err != nil {
				return nil, err
			}
		}
	}

	// find parent directories that need to be populated to satisfy hierarchy since golang will set parent directories (of working dir) as empty pkgs if not populated
	// TODO: this might not be needed with go modules?
	if err := PopulateParentPkgDirs(ctx, files, tests); err != nil {
		return nil, err
	}

	// group file notes
	u := []string{}
	for k, _ := range files.notes {
		u = append(u, k)
	}
	sort.Strings(u)
	notes := []*FileNote{}
	for _, k := range u {
		note := files.notes[k]
		notes = append(notes, note)
	}

	return notes, nil
}

//----------

func (files *Files) addFileNote(note *FileNote) error {
	n, ok := files.notes[note.Filename]

	// don't add if it has the same or inferior type
	if ok && note.Type <= n.Type {
		return nil
	}

	files.notes[note.Filename] = note

	// add full astfile on demand
	if !note.JustCopy && note.AstFile == nil {
		mode := parser.ParseComments // full mode
		astFile, err := parser.ParseFile(files.fset, note.Filename, nil, mode)
		if err != nil {
			return err
		}
		note.AstFile = astFile
	}
	return nil
}

func (files *Files) addDir(ctx context.Context, dir string, tests bool, allFilenames map[string]struct{}) error {
	fnames, err := PkgFilenames(ctx, dir, tests)
	if err != nil {
		return err
	}
	for _, fname := range fnames {
		// must exist in all filenames that belong to the program
		if _, ok := allFilenames[fname]; !ok {
			continue
		}

		note := &FileNote{Filename: fname, Type: AnnotationTypeFile}
		note.DebugSrc = "add dir (direct or pkg annotation)"
		if err := files.addFileNote(note); err != nil {
			return err
		}
	}
	return nil
}

//----------

func (files *Files) filenamesImportingDebugPkg(pkgs []*packages.Package) ([]string, error) {
	debugPkgPath := "github.com/jmigpin/editor/core/godebug/debug"

	pkgsImp := PackagesImportingPkgPath(pkgs, debugPkgPath)

	mode := parser.ImportsOnly // fast mode

	u := []string{}
	for _, pkg := range pkgsImp {
		for _, filename := range pkgGoFiles(pkg) {
			astFile, err := parser.ParseFile(files.fset, filename, nil, mode)
			if err != nil {
				return nil, err
			}
			if FileImportsPkgPath(astFile, debugPkgPath) {
				u = append(u, filename)
			}
		}
	}
	return u, nil
}

//----------
//----------
//----------

type FileNote struct {
	Filename string
	JustCopy bool // true: ignore type and astfile, just copy (no annotation)
	Type     AnnotationType
	AstFile  *ast.File

	DebugSrc string
}

//----------
//----------
//----------

func AstFileAnnotationType(file *ast.File) (AnnotationType, bool) {
	stop := false
	typ := AnnotationTypeNone
	var vis VisitorFn
	vis = func(node ast.Node) ast.Visitor {
		if stop {
			return nil
		}
		if ce, ok := node.(*ast.CallExpr); ok {
			if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
				if id, ok := se.X.(*ast.Ident); ok {
					if id.Name == "debug" {
						u := AnnotationTypeNone
						switch se.Sel.Name {
						case "NoAnnotations":
							u = AnnotationTypeNoAnnotations
						case "AnnotateBlock":
							u = AnnotationTypeBlock
						case "AnnotateFile":
							u = AnnotationTypeFile
						case "AnnotatePackage":
							u = AnnotationTypePackage
							stop = true // no higher level
						}
						if typ < u {
							typ = u
						}
					}
				}
			}
		}
		return vis
	}
	ast.Walk(vis, file)
	return typ, typ != AnnotationTypeNone
}

//----------

type AnnotationType int

const (
	AnnotationTypeNone AnnotationType = iota
	AnnotationTypeNoAnnotations
	AnnotationTypeBlock
	AnnotationTypeFile
	AnnotationTypePackage // last to be able to stop early
)

//----------

type VisitorFn func(ast.Node) ast.Visitor

func (x VisitorFn) Visit(node ast.Node) ast.Visitor {
	return x(node)
}

//----------
//----------
//----------

func pkgGoFiles(pkg *packages.Package) []string {
	return pkg.GoFiles
	//return pkg.CompiledGoFiles
}

//----------

func PkgFilenames(ctx context.Context, dir string, tests bool) ([]string, error) {
	cfg := &packages.Config{
		Context: ctx,
		Dir:     dir,
		Mode:    packages.NeedFiles,
		Tests:   tests,
	}
	pkgs, err := packages.Load(cfg, "")
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, pkg := range pkgs {
		files = append(files, pkgGoFiles(pkg)...)
	}
	return files, nil
}

//----------

func ProgramPackages(ctx context.Context, mode packages.LoadMode, dir, filename string, tests bool, fset *token.FileSet) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Context: ctx,
		Fset:    fset,
		Tests:   tests,
		Dir:     dir,
		Mode:    mode,
	}
	pattern := ""
	if filename != "" {
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

func PackagesImportingPkgPath(pkgs []*packages.Package, pkgPath string) []*packages.Package {
	pkgsImporting := []*packages.Package{}
	visited := map[*packages.Package]bool{}
	var visit func(parent, pkg *packages.Package)
	visit = func(parent, pkg *packages.Package) {
		// don't mark pkgPath as visited or it won't work
		if pkg.PkgPath != pkgPath {
			// mark visited
			if visited[pkg] {
				return
			}
			visited[pkg] = true
		}
		if parent != nil && pkg.PkgPath == pkgPath {
			pkgsImporting = append(pkgsImporting, parent)
		}
		for _, pkg2 := range pkg.Imports {
			visit(pkg, pkg2)
		}
	}
	for _, pkg := range pkgs {
		visit(nil, pkg)
	}
	return pkgsImporting
}

//----------

func FileImportsPkgPath(file *ast.File, pkgPath string) bool {
	for _, decl := range file.Decls {
		switch t := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range t.Specs {
				switch t2 := spec.(type) {
				case *ast.ImportSpec:
					s := t2.Path.Value
					if len(s) > 2 {
						s = s[1 : len(s)-1]
						if s == pkgPath {
							return true
						}
					}

				}
			}
		}
	}
	return false
}

//----------

func AstFileFilename(astFile *ast.File, fset *token.FileSet) (string, error) {
	if astFile == nil {
		panic("!")
	}
	tfile := fset.File(astFile.Package)
	if tfile == nil {
		return "", fmt.Errorf("not found")
	}
	return tfile.Name(), nil
}

//----------

func PopulateParentPkgDirs(ctx context.Context, files *Files, tests bool) error {
	// don't copy annotated files
	noCopy := map[string]struct{}{}
	for _, v := range files.notes {
		if v.Type > 0 {
			noCopy[v.Filename] = struct{}{}
		}
	}
	// populate parent directories
	vis := map[string]struct{}{}
	for _, v := range files.notes {
		if v.Type > 0 {
			dir := filepath.Dir(v.Filename)
			err := populateDir(ctx, files, tests, dir, vis, noCopy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func populateDir(ctx context.Context, files *Files, tests bool, dir string, vis map[string]struct{}, noCopy map[string]struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// don't populate an already visited dir
	if _, ok := vis[dir]; ok {
		return nil
	}
	vis[dir] = struct{}{}

	// visit only up to srcdir
	// TODO: based this somehow on packages.*?
	srcDir, _ := goutil.ExtractSrcDir(dir)
	if len(srcDir) <= 1 || strings.Index(dir, srcDir) < 0 {
		return nil
	}

	// filenote to populate dir (files that need to be copied)
	filenames, err := PkgFilenames(ctx, dir, tests)
	if err != nil {
		return err
	}
	for _, fname := range filenames {
		if _, ok := noCopy[fname]; ok {
			continue
		}
		note := &FileNote{Filename: fname, JustCopy: true}
		note.DebugSrc = "populate dir (just copy)"
		files.addFileNote(note)
	}

	// visit parent dir
	pdir := filepath.Dir(dir)
	return populateDir(ctx, files, tests, pdir, vis, noCopy)
}

//----------
