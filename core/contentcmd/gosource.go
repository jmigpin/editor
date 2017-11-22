package contentcmd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/cmdutil"
)

func goSource(erow cmdutil.ERower, s string) bool {
	if !erow.IsRegular() {
		return false
	}
	if path.Ext(erow.Filename()) != ".go" {
		return false
	}
	//log.Printf("go source: %s", s)
	return parseERow(erow)
}

func parseERow(erow cmdutil.ERower) bool {
	ta := erow.Row().TextArea
	pos, err := visitGoSource(erow.Filename(), ta.Str(), ta.CursorIndex())
	if err != nil {
		//log.Print(err)
		return false
	}
	return filePos(erow, pos.String())
}

type importFn func(path, dir string, mode types.ImportMode) (*types.Package, error)

func (fn importFn) Import(path string) (*types.Package, error) {
	return fn.ImportFrom(path, "", 0)
}
func (fn importFn) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	return fn(path, dir, mode)
}

func visitGoSource(filename string, src interface{}, cursorIndex int) (*token.Position, error) {
	v := NewGSVisitor()
	return v.visitSource(filename, src, cursorIndex)
}

type GSVisitor struct {
	fset         *token.FileSet
	info         types.Info
	conf         types.Config
	mainFile     *ast.File
	mainFilename string
	imported     map[string]*types.Package
	astFilesMu   sync.RWMutex // just a load mutex, not used during ast traversal
	astFiles     astFiles
	Debug        bool
}

type astFiles map[string]*ast.File

func NewGSVisitor() *GSVisitor {
	v := &GSVisitor{
		fset: token.NewFileSet(),
		info: types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
		},
		conf: types.Config{
			// it will exit on first error if not defined
			Error: func(err error) {
				//log.Printf("conf error: %v", err)
			},
		},
		imported: make(map[string]*types.Package),
		astFiles: make(map[string]*ast.File),
	}

	// Calls to conf.check will only use cached imports imported first with packageImporter. This improves performance by not automatically parse/check all packages included in the source code.
	v.conf.Importer = importFn(v.cachedPackageImporter)

	return v
}

func (v *GSVisitor) visitSource(filename string, src interface{}, cursorIndex int) (*token.Position, error) {
	// find full filename if the package import path was given (not full path)
	bpkg, _ := build.Import(filepath.Dir(filename), "", 0)
	if bpkg.Dir != "" {
		filename = filepath.Join(bpkg.Dir, filepath.Base(filename))
		if v.Debug {
			log.Printf("filename is now %v", filename)
		}
		//spew.Dump(bpkg)
	}

	// parse source
	v.mainFilename = filename
	v.mainFile = v.parseFilename(filename, src)

	// first check pass without any imports (cached imports are empty)
	path1 := v.posFilePath(v.mainFile.Package)
	_, _ = v.confCheck(path1, []*ast.File{v.mainFile})

	id := v.resolveMainFileIdentNode(cursorIndex)
	if id != nil {
		pos := v.identObjPos(id)
		if pos != token.NoPos {
			u := v.fset.Position(pos)
			if v.Debug {
				log.Printf("result: %v", u.Offset)
			}
			return &u, nil
		}
	}

	return nil, fmt.Errorf("identifier object not found")
}

func (v *GSVisitor) identObjPos(id *ast.Ident) token.Pos {
	if id.Obj != nil {
		if n, ok := id.Obj.Decl.(ast.Node); ok {
			return n.Pos()
		}
	}
	obj, ok := v.info.Uses[id]
	if !ok {
		obj, ok = v.info.Defs[id]
	}
	if obj == nil {
		return token.NoPos
	}

	// builtin basic types
	switch t := obj.(type) {
	case *types.TypeName:
		switch t.Type().(type) {
		case *types.Basic:
			pkg, _ := v.packageImporter("builtin", "", 0)
			obj = pkg.Scope().Lookup(id.Name)
			if obj != nil {
				return obj.Pos()
			}
		}
	}

	return obj.Pos()
}

func (v *GSVisitor) posFilePath(pos token.Pos) string {
	name := v.fset.File(pos).Name() // file path
	name = filepath.Dir(name)
	name = v.normalizePath(name)
	return name
}

func (v *GSVisitor) posAstFile(pos token.Pos) *ast.File {
	tokenFile := v.fset.File(pos)
	return v.astFiles[tokenFile.Name()]
}

func (v *GSVisitor) resolveMainFileIdentNode(index int) *ast.Ident {
	var id *ast.Ident
	pvis := NewPathVisitor()
	pvis.OnVisit = func(node ast.Node) {
		switch t := node.(type) {
		case *ast.Ident:

			if v.Debug {
				log.Printf("ident %v %v", t, v.fset.Position(t.Pos()).Offset)
			}

			s := v.fset.Position(node.Pos()).Offset
			e := v.fset.Position(node.End()).Offset
			inside := index >= s && index < e
			if inside {
				id = t
				pvis.Stop = true
			}
		}
	}
	ast.Walk(pvis, v.mainFile)
	//pvis.PrintPath()
	for _, n := range pvis.path {
		v.resolveNode(n)
	}
	return id
}

func (v *GSVisitor) posNodeVisitor(pos token.Pos) *PathVisitor {
	pvis := NewPathVisitor()
	pvis.OnVisit = func(node ast.Node) {
		if node.Pos() == pos {
			pvis.Stop = true
		}
	}
	return pvis
}

func (v *GSVisitor) resolvePosNode(pos token.Pos) {
	pvis := v.posNodeVisitor(pos)
	astFile := v.posAstFile(pos)
	ast.Walk(pvis, astFile)
	v.resolveNode(pvis.path[len(pvis.path)-1])
}

func (v *GSVisitor) resolveNode(node ast.Node) {
	if v.Debug {
		extra := ""
		switch t := node.(type) {
		case *ast.Ident:
			extra = fmt.Sprintf("%q", node)
		case *ast.Field:
			extra = fmt.Sprintf("%q", t.Type)
		}
		log.Printf("resolveNode %v %v", reflect.TypeOf(node), extra)
	}

	switch t := node.(type) {
	case *ast.ValueSpec:
		v.resolveNode(t.Type)
	case *ast.StarExpr:
		v.resolveNode(t.X)
	case *ast.Field:
		v.resolveNode(t.Type)

	case *ast.Ident:
		if v.resolveObjOfIdent(t) {
			break
		}

		// could be a variable defined in another file (same package)
		path1 := v.posFilePath(t.Pos())
		_, _ = v.packageImporter(path1, "", 0)

		if v.resolveObjOfIdent(t) {
			break
		}

		// last resort
		v.importAllNodeFileImports(t)

	case *ast.SelectorExpr:
		sel, ok := v.info.Selections[t]
		if ok {
			v.resolveObj(t, sel.Obj())
			break
		}
		v.resolveNode(t.X)
		v.resolveNode(t.Sel)
	case *ast.ImportSpec:
		// import path referenced by the importspec
		ipath, _ := strconv.Unquote(t.Path.Value)
		_, _ = v.packageImporter(ipath, "", 0)
		// re-check the path that this node belongs to
		v.parseConfCheckNodePackage(t)
	}
}

func (v *GSVisitor) resolveObjOfIdent(id *ast.Ident) bool {
	// already solved
	obj, ok := v.info.Defs[id]
	if ok {
		return true
	}
	// if it is solved as a use, resolve the object to get to the definition
	obj, ok = v.info.Uses[id]
	if ok {
		v.resolveObj(id, obj)
		return true
	}
	return false
}

func (v *GSVisitor) resolveObj(node ast.Node, obj types.Object) {
	if v.Debug {
		log.Printf("resolveObj %v %q", reflect.TypeOf(obj), obj)
	}
	pos := obj.Pos()
	if pos != token.NoPos {
		v.resolvePosNode(pos)
	}
}

func (v *GSVisitor) parseConfCheckNodePackage(node ast.Node) {
	path1 := v.posFilePath(node.Pos())
	_, _ = v.parseConfCheck(path1, "", 0)
}

func (v *GSVisitor) importAllNodeFileImports(id *ast.Ident) {
	astFile := v.posAstFile(id.Pos())
	for _, imp := range astFile.Imports {
		ipath, _ := strconv.Unquote(imp.Path.Value)
		_, _ = v.packageImporter(ipath, "", 0)
	}
	v.parseConfCheckNodePackage(id)
}

//func (v *GSVisitor) importPackageMatchingIdent(id *ast.Ident) {
//	astFile := v.posAstFile(id.Pos())
//	for _, imp := range astFile.Imports {
//		// import path referenced by the importspec
//		ipath, _ := strconv.Unquote(imp.Path.Value)

//		// Slow operation, would import first and check name later
//		//pkg, _ := v.packageImporter(ipath, "", 0)
//		//if pkg.Name() == t.Name {

//		match := (imp.Name != nil && imp.Name.Name == id.Name) ||
//			filepath.Base(ipath) == id.Name

//		if match {
//			if v.Debug {
//				log.Printf("resolveNode ident: import match %v->%v", ipath, id.Name)
//			}
//			_, _ = v.packageImporter(ipath, "", 0)

//			// re-check the path that this node belongs to
//			v.parseConfCheckNodePackage(id)
//			break
//		}
//	}
//}

func (v *GSVisitor) packageImporter(path, dir string, mode types.ImportMode) (*types.Package, error) {
	key := path
	pkg, ok := v.imported[key]
	if ok {
		return pkg, nil
	}
	if v.Debug {
		log.Printf("importing: %q %q", path, dir)
	}
	pkg, _ = v.parseConfCheck(path, dir, build.ImportMode(mode))
	if v.Debug {
		log.Printf("imported: %v", pkg)
	}
	v.imported[key] = pkg
	return pkg, nil
}

func (v *GSVisitor) cachedPackageImporter(path, dir string, mode types.ImportMode) (*types.Package, error) {
	key := path
	pkg, ok := v.imported[key]
	if ok {
		return pkg, nil
	}
	return nil, fmt.Errorf("not cached")
}

func (v *GSVisitor) normalizePath(path string) string {
	for _, d := range build.Default.SrcDirs() {
		d += "/"
		if strings.HasPrefix(path, d) {
			return path[len(d):]
		}
	}
	return path
}

func (v *GSVisitor) pathFilenames(path, dir string, mode build.ImportMode) []string {
	// get build.package that contains info
	bpkg, _ := build.Import(path, dir, mode)
	//spew.Dump(bpkg)
	// package filenames
	a := append(bpkg.GoFiles, bpkg.CgoFiles...)
	var names []string
	for _, fname := range a {
		names = append(names, filepath.Join(bpkg.Dir, fname))
	}
	// include mainfile if a src string was parsed - bpkg doesn't have it
	if len(a) == 0 {
		path1 := v.posFilePath(v.mainFile.Package)
		if bpkg.ImportPath == path1 {
			names = append(names, v.mainFilename)
		}
	}
	return names
}

func (v *GSVisitor) parseFilenames(filenames []string) []*ast.File {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var files []*ast.File
	for _, filename := range filenames {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			file := v.parseFilename(filename, nil)
			mu.Lock()
			files = append(files, file)
			mu.Unlock()
		}(filename)
	}
	wg.Wait()
	return files
}

func (v *GSVisitor) parseFilename(filename string, src interface{}) *ast.File {
	v.astFilesMu.RLock()
	file, ok := v.astFiles[filename]
	v.astFilesMu.RUnlock()
	if ok {
		return file
	}
	file, err := parser.ParseFile(v.fset, filename, src, parser.AllErrors)
	if v.Debug {
		log.Printf("parseFilename: %v (err=%v)", filepath.Base(filename), err)
	}
	v.astFilesMu.Lock()
	v.astFiles[filename] = file
	v.astFilesMu.Unlock()
	return file
}

func (v *GSVisitor) parseConfCheck(path, dir string, mode build.ImportMode) (*types.Package, error) {
	filenames := v.pathFilenames(path, dir, mode)
	files := v.parseFilenames(filenames)
	return v.confCheck(path, files)
}

func (v *GSVisitor) confCheck(path string, files []*ast.File) (*types.Package, error) {
	pkg, err := v.conf.Check(path, v.fset, files, &v.info)
	if v.Debug {
		log.Printf("confCheck %v (err=%v)", path, err)
	}
	return pkg, err
}

type PathVisitor struct {
	path    []ast.Node
	parents map[ast.Node]ast.Node
	Stop    bool
	OnVisit func(ast.Node)
}

func NewPathVisitor() *PathVisitor {
	pv := &PathVisitor{
		parents: make(map[ast.Node]ast.Node),
	}
	return pv
}
func (pv *PathVisitor) Visit(node ast.Node) ast.Visitor {
	if pv.Stop {
		return nil
	}
	if node == nil {
		pv.path = pv.path[:len(pv.path)-1]
		return nil
	}
	if len(pv.path) > 0 {
		pv.parents[node] = pv.path[len(pv.path)-1]
	}
	pv.path = append(pv.path, node)
	pv.OnVisit(node)
	if pv.Stop {
		return nil
	}
	return pv
}
func (pv *PathVisitor) PrintPath() {
	for i, n := range pv.path {
		extra := ""
		switch n.(type) {
		case *ast.File,
			*ast.BlockStmt,
			*ast.GenDecl:
		default:
			extra = fmt.Sprintf(" %v", n)
		}
		log.Printf("%v: %v%v", i, reflect.TypeOf(n), extra)
	}
}
