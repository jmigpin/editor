package godebug

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/util/mathutil"
	"golang.org/x/tools/go/packages"
)

type FilesToAnnotate struct {
	cmd *Cmd

	pathsPkgs map[string]*packages.Package // map[pkgPath]
	filesPkgs map[string]*packages.Package // map[filename]
	filesAsts map[string]*ast.File         // map[filename]

	toAnnotate   map[string]AnnotationType   // map[filename]
	nodeAnnTypes map[ast.Node]AnnotationType // map[*ast.File and inner ast.Node's, check how a file is added for annotation]

	loadPkgs []*packages.Package
}

func NewFilesToAnnotate(cmd *Cmd) *FilesToAnnotate {
	fa := &FilesToAnnotate{cmd: cmd}
	fa.pathsPkgs = map[string]*packages.Package{}
	fa.filesPkgs = map[string]*packages.Package{}
	fa.filesAsts = map[string]*ast.File{}
	fa.toAnnotate = map[string]AnnotationType{}
	fa.nodeAnnTypes = map[ast.Node]AnnotationType{}
	return fa
}

func (fa *FilesToAnnotate) find(ctx context.Context) error {
	pkgs, err := fa.loadPackages(ctx)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}
	fa.loadPkgs = pkgs

	if err := fa.initMaps(pkgs); err != nil {
		return err
	}
	if err := fa.addFromArgs(ctx); err != nil {
		return err
	}
	if err := fa.addFromMain(ctx); err != nil {
		return err
	}
	//if err := fa.addFromMainFuncDecl(ctx); err != nil {
	//	return err
	//}
	if err := fa.addFromSrcDirectives(ctx); err != nil {
		return err
	}

	if fa.cmd.flags.verbose {
		fa.cmd.printf("files to annotate:\n")
		for k, v := range fa.toAnnotate {
			fa.cmd.printf("\t%v: %v\n", k, v)
		}
	}

	return nil
}

func (fa *FilesToAnnotate) initMaps(pkgs []*packages.Package) error {
	if fa.cmd.flags.verbose {
		fa.cmd.printf("main pkgs: %v\n", len(pkgs))
		for _, pkg := range pkgs {
			fa.cmd.printf("\t%v (%v)\n", pkg.PkgPath, pkg.ID)
			for _, filename := range pkg.CompiledGoFiles {
				fa.cmd.printf("\t\t%v\n", filename)
			}
		}
	}

	for _, pkg := range pkgs {
		if err := fa.initMaps2(pkg); err != nil {
			return err
		}
	}
	return nil
}
func (fa *FilesToAnnotate) initMaps2(pkg *packages.Package) error {
	// don't handle runtime pkg (ex: has a file that contains a "main()" func and gets caught only "sometimes" when findind for the main func decl)
	//if pkg.PkgPath == "runtime" {
	//	return
	//}

	// map pkgpaths to pkgs
	pkg0, ok := fa.pathsPkgs[pkg.PkgPath]
	if ok {
		if len(pkg0.Syntax) < len(pkg.Syntax) {
			// ok, visit again and keep the new pkg
		} else {
			// DEBUG
			//if pkg != pkg0 {
			//	fmt.Println("PKG0---")
			//	spew.Dump(pkg0)
			//	spew.Dump(len(pkg0.Syntax))
			//	spew.Dump(pkg0.CompiledGoFiles)
			//	fmt.Println("PKG---")
			//	spew.Dump(pkg)
			//	spew.Dump(len(pkg.Syntax))
			//	spew.Dump(pkg.CompiledGoFiles)
			//	fmt.Println("---")
			//}

			return nil // already visited
		}
	}
	fa.pathsPkgs[pkg.PkgPath] = pkg

	if fa.cmd.flags.verbose {
		fa.cmd.printf("pkg: %v\n", pkg.PkgPath)
		//	for _, filename := range pkg.CompiledGoFiles {
		//		fa.cmd.printf("\tpkgfile: %v\n", filename)
		//	}
	}

	// map filenames to pkgs
	for _, filename := range pkg.CompiledGoFiles {
		fa.filesPkgs[filename] = pkg
	}

	// map filenames to asts
	for _, astFile := range pkg.Syntax {
		filename, err := nodeFilename(fa.cmd.fset, astFile)
		if err != nil {
			return err
		}
		fa.filesAsts[filename] = astFile
	}

	// visit imports recursively
	for _, pkg2 := range pkg.Imports {
		if err := fa.initMaps2(pkg2); err != nil {
			return err
		}
	}
	return nil
}

//----------

func (fa *FilesToAnnotate) addFromArgs(ctx context.Context) error {
	absFilePath := func(s string) string {
		if !filepath.IsAbs(s) {
			return filepath.Join(fa.cmd.Dir, s)
		}
		return s
	}

	// detect filenames in args (best effort)
	for _, arg := range fa.cmd.flags.unnamedArgs {
		if !strings.HasSuffix(arg, ".go") {
			continue
		}
		filename := arg
		filename = absFilePath(filename)
		if _, ok := fa.filesPkgs[filename]; !ok {
			continue
		}
		fa.addToAnnotate(filename, AnnotationTypeFile)
	}

	for _, path := range fa.cmd.flags.paths {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		// because full paths are needed to match in the map
		path = absFilePath(path)

		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			dir := path
			des, err := os.ReadDir(dir)
			if err != nil {
				return fmt.Errorf("read dir error: %w", err)
			}
			for _, de := range des {
				filename := filepath.Join(dir, de.Name())
				if _, ok := fa.filesPkgs[filename]; !ok {
					continue
				}
				fa.addToAnnotate(filename, AnnotationTypeFile)
			}
		} else {
			filename := path
			if _, ok := fa.filesPkgs[filename]; !ok {
				return fmt.Errorf("file not loaded: %v", filename)
			}
			fa.addToAnnotate(filename, AnnotationTypeFile)
		}
	}
	return nil
}

//----------

func (fa *FilesToAnnotate) addFromMain(ctx context.Context) error {
	for _, pkg := range fa.loadPkgs {
		for _, filename := range pkg.CompiledGoFiles {

			if fa.cmd.flags.mode.test {
				// bypass files without .go ext (avoids the generated main() test file)
				ext := filepath.Ext(filename)
				if ext != ".go" {
					continue
				}
			}

			fa.addToAnnotate(filename, AnnotationTypeFile)
		}
	}
	return nil
}

//----------

func (fa *FilesToAnnotate) addFromSrcDirectives(ctx context.Context) error {
	for filename, astFile := range fa.filesAsts {
		// early stop
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := fa.addFromSrcDirectivesFile(filename, astFile); err != nil {
			return err
		}
	}
	return nil
}
func (fa *FilesToAnnotate) addFromSrcDirectivesFile(filename string, astFile *ast.File) error {

	// get nodes with associated comments
	cns := commentsWithNodes(fa.cmd.fset, astFile, astFile.Comments)

	// find comments that have "//godebug" directives
	opts := []*AnnotationOpt{}
	for _, cns := range cns {
		opt, ok, err := annOptInComment(cns.Comment, cns.Node)
		if err != nil {
			// improve error
			err = positionError(fa.cmd.fset, cns.Comment.Pos(), err)
			return err
		}
		if ok {
			opts = append(opts, opt)
		}
	}

	// keep node map for annotation phase
	for _, opt := range opts {
		fa.nodeAnnTypes[opt.Node] = opt.Type
	}
	// add filenames to annotate from annotations
	for _, opt := range opts {
		if err := fa.addFromAnnOpt(opt); err != nil {
			// improve error
			err = positionError(fa.cmd.fset, opt.Comment.Pos(), err)
			return err
		}
	}
	return nil
}
func (fa *FilesToAnnotate) addFromAnnOpt(opt *AnnotationOpt) error {
	switch opt.Type {
	case AnnotationTypeNone:
		return nil
	case AnnotationTypeOff:
		return nil
	case AnnotationTypeBlock:
		return fa.addNodeFilename(opt.Node, opt.Type)
	case AnnotationTypeFile:
		if opt.Opt != "" {
			filename := opt.Opt

			// make it relative to current filename dir if not absolute
			if !filepath.IsAbs(filename) {
				u, err := nodeFilename(fa.cmd.fset, opt.Comment)
				if err != nil {
					return err
				}
				dir := filepath.Dir(u)
				filename = filepath.Join(dir, filename)
			}

			return fa.addFilename(filename, opt.Type)
		}

		return fa.addNodeFilename(opt.Node, opt.Type)
	case AnnotationTypeImport:
		path, err := nodeImportPath(opt.Node)
		if err != nil {
			return err
		}
		// TODO: pkg==pkgpath always?
		return fa.addPkgPath(path, opt.Type)
	case AnnotationTypePackage:
		if opt.Opt != "" {
			pkgPath := opt.Opt
			return fa.addPkgPath(pkgPath, opt.Type)
		}
		pkg, err := fa.nodePkg(opt.Node)
		if err != nil {
			return err
		}
		return fa.addPkg(pkg, opt.Type)
	case AnnotationTypeModule:
		if opt.Opt != "" {
			pkgPath := opt.Opt
			pkg, err := fa.pathPkg(pkgPath)
			if err != nil {
				return err
			}
			return fa.addModule(pkg, opt.Type)
		}
		pkg, err := fa.nodePkg(opt.Node)
		if err != nil {
			return err
		}
		return fa.addModule(pkg, opt.Type)
	default:
		return fmt.Errorf("todo: handleAnnOpt: %v", opt.Type)
	}
}
func (fa *FilesToAnnotate) addNodeFilename(node ast.Node, typ AnnotationType) error {
	filename, err := nodeFilename(fa.cmd.fset, node)
	if err != nil {
		return err
	}
	return fa.addFilename(filename, typ)
}
func (fa *FilesToAnnotate) addPkgPath(pkgPath string, typ AnnotationType) error {
	pkg, err := fa.pathPkg(pkgPath)
	if err != nil {
		return err
	}
	return fa.addPkg(pkg, typ)
}
func (fa *FilesToAnnotate) addPkg(pkg *packages.Package, typ AnnotationType) error {
	for _, filename := range pkg.CompiledGoFiles {
		if err := fa.addFilename(filename, typ); err != nil {
			return err
		}
	}
	return nil
}
func (fa *FilesToAnnotate) addModule(pkg *packages.Package, typ AnnotationType) error {
	if pkg.Module == nil {
		return fmt.Errorf("missing module in pkg: %v", pkg.Name)
	}
	// add pkgs that belong to module
	for _, pkg2 := range fa.filesPkgs {
		if pkg2.Module == nil {
			continue
		}
		// module pointers differ, must use path
		if pkg2.Module.Path == pkg.Module.Path {
			if err := fa.addPkg(pkg2, typ); err != nil {
				return err
			}
		}
	}
	return nil
}
func (fa *FilesToAnnotate) addFilename(filename string, typ AnnotationType) error {
	_, ok := fa.filesPkgs[filename]
	if !ok {
		return fmt.Errorf("file not found in loaded program: %v", filename)
	}
	fa.addToAnnotate(filename, typ)
	return nil
}

//----------

func (fa *FilesToAnnotate) addToAnnotate(filename string, typ AnnotationType) {
	typ0, ok := fa.toAnnotate[filename]
	add := !ok || typ > typ0
	if add {
		fa.toAnnotate[filename] = typ

		// set astfile node as well for the annotator to know from the start what type of annotation type is in the file
		if astFile, ok := fa.filesAsts[filename]; ok {
			fa.nodeAnnTypes[astFile] = typ
		}
	}
}

//----------

func (fa *FilesToAnnotate) loadPackages(ctx context.Context) ([]*packages.Package, error) {

	loadMode := 0 |
		//packages.NeedExportFile | // TODO
		//packages.NeedTypesSizes | // TODO
		packages.NeedCompiledGoFiles |
		packages.NeedDeps |
		packages.NeedFiles |
		packages.NeedImports |
		packages.NeedModule |
		packages.NeedName | // name and pkgpath
		packages.NeedSyntax |
		packages.NeedTypes |
		packages.NeedTypesInfo | // access to pkg.TypesInfo.*
		0

	cfg := &packages.Config{
		Context:    ctx,
		Fset:       fa.cmd.fset,
		Tests:      fa.cmd.flags.mode.test,
		Dir:        fa.cmd.Dir,
		Mode:       loadMode,
		Env:        fa.cmd.env,
		BuildFlags: fa.cmd.buildArgs(),
		//ParseFile:  parseFile,
		//Logf: func(f string, args ...interface{}) {
		//s := fmt.Sprintf(f, args...)
		//fmt.Print(s)
		//},
	}

	// There is a distinction between passing a file directly, or with the "file=" query. Passing without the file will pass a file argument to the underlying build tool, that could actually fail to properly load pkg.module var in the case of a simple [main.go go.mod] project. Because "go build" and "go build main.go" have slightly different behaviours. Check testdata/basic_gomod.txt test where it fails if the "file=" patterns are commented.
	p := []string{}

	for _, f := range fa.cmd.flags.unnamedArgs {
		p = append(p, "file="+f)
	}
	p = append(p, fa.cmd.flags.unnamedArgs...)

	pkgs, err := packages.Load(cfg, p...)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, pkg.Errors[0]
		}
	}

	//me := iout.MultiError{}
	//for _, pkg := range pkgs {
	//	for _, err := range pkg.Errors {
	//		me.Add(err)
	//	}
	//}
	//if err := me.Result(); err != nil {
	//	return nil, err
	//}

	return pkgs, nil
}

//----------

// commented: in favor of goutil.FindGoMod
//func (fa *FilesToAnnotate) GoModFilename() (string, bool) {
//	for _, pkg := range fa.filesPkgs {
//		fmt.Printf("** %v\n", pkg.Module)
//		if pkg.Module != nil {
//			fa.cmd.logf("gomod=%v", pkg.Module.GoMod)
//		}

//		//mod := pkg.Module
//		mod := pkgMod(pkg)
//		if mod != nil && mod.GoMod != "" {
//			//fa.cmd.logf("gomod(nomain?)=%v", mod.GoMod)
//			if mod.Main {
//				fa.cmd.logf("gomod=%v", mod.GoMod)
//				return mod.GoMod, true
//			}
//		}
//	}

//	// try env
//	env := goutil.GoEnv(fa.cmd.Dir)
//	filename := osutil.GetEnv(env, "GOMOD")
//	if filename != "" && filename != os.DevNull { // can be "/dev/null"!
//		return filename, true
//	}

//	return "", false
//}

//----------

func (fa *FilesToAnnotate) nodePkg(node ast.Node) (*packages.Package, error) {
	filename, err := nodeFilename(fa.cmd.fset, node)
	if err != nil {
		return nil, err
	}
	pkg, ok := fa.filesPkgs[filename]
	if !ok {
		return nil, fmt.Errorf("missing pkg for filename: %v", filename)
	}
	return pkg, nil
}

func (fa *FilesToAnnotate) pathPkg(path string) (*packages.Package, error) {
	pkg, ok := fa.pathsPkgs[path]
	if !ok {
		return nil, fmt.Errorf("missing pkg for path: %v", path)
	}
	return pkg, nil
}

//----------
//----------
//----------

func nodeImportPath(node ast.Node) (string, error) {
	// ex: direclty at *ast.ImportSpec
	// 	import (
	// 		//godebug:annotateimport
	// 		"pkg1"
	// 	)

	// ex: at *ast.GenDecl
	// 	//godebug:annotateimport
	// 	import "pkg1"
	// 	//godebug:annotateimport
	// 	import (
	// 		"pkg1"
	// 	)

	if gd, ok := node.(*ast.GenDecl); ok {
		if len(gd.Specs) > 0 {
			is, ok := gd.Specs[0].(*ast.ImportSpec)
			if ok {
				node = is
			}
		}
	}

	is, ok := node.(*ast.ImportSpec)
	if !ok {
		return "", fmt.Errorf("not at an import spec")
	}
	return strconv.Unquote(is.Path.Value)
}

func nodeFilename(fset *token.FileSet, node ast.Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("node is nil")
	}
	tokFile := fset.File(node.Pos())
	if tokFile == nil {
		return "", fmt.Errorf("missing token file: %v", node.Pos())
	}
	return tokFile.Name(), nil
}

//----------

func hashStringN(s string, n int) string {
	h := md5.New()
	_, _ = h.Write([]byte(s))
	b := h.Sum(nil)

	s2 := hex.EncodeToString(b)
	s2 = base64.RawStdEncoding.EncodeToString([]byte(s2))

	// cut short
	n = mathutil.Min(n, len(s2))
	s2 = s2[:n]

	return s2
}

//----------

func positionError(fset *token.FileSet, pos token.Pos, err error) error {
	p := fset.Position(pos)
	return fmt.Errorf("%v: %w", p, err)
}

//----------

// TODO: move to goutil?
func pkgMod(pkg *packages.Package) *packages.Module {
	mod := pkg.Module
	if mod != nil {
		for mod.Replace != nil {
			mod = mod.Replace
		}
	}
	return mod
}
