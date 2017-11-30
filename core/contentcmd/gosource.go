package contentcmd

import (
	"container/list"
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

	"github.com/davecgh/go-spew/spew"
	"github.com/jmigpin/editor/core/cmdutil"
)

func goSource(erow cmdutil.ERower) bool {
	if !erow.IsRegular() {
		return false
	}
	if path.Ext(erow.Filename()) != ".go" {
		return false
	}
	ta := erow.Row().TextArea
	pos, end, err := visitGoSource(erow.Filename(), ta.Str(), ta.CursorIndex())
	if err != nil {
		//log.Print(err)
		return false
	}

	goSourceOpenPosition(erow, pos, end)

	return true
}

func goSourceOpenPosition(erow cmdutil.ERower, pos, end *token.Position) {
	var erow2 cmdutil.ERower
	ed := erow.Ed()

	// highlight directly on the same row
	if erow.Filename() == pos.Filename {
		if erow.Row().TextArea.IndexIsVisible(pos.Offset) {
			erow2 = erow
		}
	}

	// choose a duplicate row that is not the current row
	if erow2 == nil {
		erows := ed.FindERowers(pos.Filename)
		for _, e := range erows {
			if e != erow {
				erow2 = e
				break
			}
		}
	}

	// open new row
	if erow2 == nil {
		col, nextRow := ed.GoodColumnRowPlace()
		erow2 = ed.NewERowerBeforeRow(pos.Filename, col, nextRow)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
		}
	}

	// goto index
	row2 := erow2.Row()
	row2.ResizeTextAreaIfVerySmall()
	ta2 := row2.TextArea
	ta2.MakeIndexVisible(pos.Offset)
	row2.TextArea.FlashIndexLen(pos.Offset, end.Offset-pos.Offset)
}

func visitGoSource(filename string, src interface{}, cursorIndex int) (*token.Position, *token.Position, error) {
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
	importable   map[string]bool
	visited      map[ast.Node]bool
	astFiles     map[string]*ast.File
	astFilesMu   sync.RWMutex // just a load mutex, not used during ast traversal
	idStack      list.List
	parents      map[ast.Node]ast.Node // just the id path node parents

	resolveDepth int
	Debug        bool
}

var universe = ast.NewScope(nil)

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

		astFiles:   make(map[string]*ast.File),
		imported:   make(map[string]*types.Package),
		importable: make(map[string]bool),
		visited:    make(map[ast.Node]bool),
		parents:    make(map[ast.Node]ast.Node),
	}

	v.conf.Importer = importFn(v.packageImporter)

	return v
}

func (v *GSVisitor) visitSource(filename string, src interface{}, cursorIndex int) (*token.Position, *token.Position, error) {
	// find full filename if the package import path was given (not full path)
	bpkg, _ := build.Import(filepath.Dir(filename), "", 0)
	if bpkg.Dir != "" {
		filename = filepath.Join(bpkg.Dir, filepath.Base(filename))
		if v.Debug {
			v.Printf("filename is now %v", filename)
		}
		//spew.Dump(bpkg)
	}

	// parse source (src string)
	v.mainFilename = filename
	v.mainFile = v.parseFilename(filename, src)

	// first check pass without any imports cached
	v.confCheckMainFile()

	id := v.mainFileIdentNode(cursorIndex)
	if id != nil {
		v.resolveNode(id)
		pos, end, ok := v.idPosEnd(id)
		if ok {
			posp := v.fset.Position(pos)
			endp := v.fset.Position(end)
			if v.Debug {
				v.Printf("result: offset=%v %v", posp.Offset, posp)
			}
			return &posp, &endp, nil
		}
	}

	return nil, nil, fmt.Errorf("identifier position not found")
}

func (v *GSVisitor) idPosEnd(id *ast.Ident) (token.Pos, token.Pos, bool) {
	pos := v.idObjPos(id)
	if pos == token.NoPos {
		return 0, 0, false
	}
	path := v.posDeepestPath(pos)
	node := path[len(path)-1]
	return node.Pos(), node.End(), true
}

func (v *GSVisitor) idObjPos(id *ast.Ident) token.Pos {
	// solved by the parser
	if id.Obj != nil {
		if n, ok := id.Obj.Decl.(ast.Node); ok {
			return n.Pos()
		}
	}
	// solved in info.uses
	obj, ok := v.info.Uses[id]
	if ok {
		return obj.Pos()
	}

	//// solved in info.defs
	//obj, ok := v.info.Defs[id]
	//if ok {
	//	return obj.Pos()
	//}

	return token.NoPos
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

func (v *GSVisitor) mainFileIdentNode(index int) *ast.Ident {
	pos := v.fset.File(v.mainFile.Package).Pos(index)
	path := v.posDeepestPath(pos)
	v.populateParents(path)
	if len(path) > 0 {
		last := path[len(path)-1]
		if id, ok := last.(*ast.Ident); ok {
			return id
		}
	}
	return nil
}

func (v *GSVisitor) resolveNode(node ast.Node) {
	if v.Debug {
		v.resolveDepth++
		defer func() { v.resolveDepth-- }()
	}
	if v.visited[node] {
		if v.Debug {
			v.DepthPrintf("resolveNode already visited %v", reflect.TypeOf(node))
		}
		return
	}
	v.visited[node] = true

	if v.Debug {
		v.DepthPrintf("resolveNode %v %v", reflect.TypeOf(node), v.fset.Position(node.Pos()))
	}

	switch t := node.(type) {
	case *ast.BasicLit:
		v.resolveParent(t)
	case *ast.Ident:
		v.resolveId(t)
	case *ast.ImportSpec:
		v.resolveImportSpec(t)
	case *ast.TypeAssertExpr:
		v.resolveNode(t.Type)
	case *ast.CallExpr:
		v.resolveNode(t.Fun)
	case *ast.ValueSpec:
		v.resolveNode(t.Type)
	case *ast.TypeSpec:
		v.resolveNode(t.Type)
	case *ast.Field:
		v.resolveNode(t.Type)
	case *ast.FuncDecl:
		v.resolveNode(t.Type)
	case *ast.StarExpr:
		v.resolveNode(t.X)
	case *ast.StructType:
		v.resolveFieldList(t.Fields)
	case *ast.InterfaceType:
		v.resolveFieldList(t.Methods)
	case *ast.SelectorExpr:
		sel, ok := v.info.Selections[t]
		if ok {
			v.resolvePos(sel.Obj().Pos())
			break
		}
		v.idStack.PushBack(t.Sel)
		v.resolveNode(t.X)
		v.idStack.Remove(v.idStack.Back())
		v.resolveNode(t.Sel)
	case *ast.AssignStmt:
		// TODO: should resolve only the necessary node
		for _, e := range t.Rhs {
			v.resolveNode(e)
		}
	case *ast.FuncType:
		// TODO: should resolve only the necessary node
		if t.Results != nil {
			for _, e := range t.Results.List {
				v.resolveNode(e)
			}
		}
	}
}

func (v *GSVisitor) resolveParent(node ast.Node) {
	pn, ok := v.parents[node]
	if ok {
		if v.Debug {
			v.DepthPrintf("resolveParent %v", pn)
		}
		v.resolveNode(pn)
		return
	}
	if v.Debug {
		v.DepthPrintf("resolveParent no parent")
	}
}

func (v *GSVisitor) resolveFieldList(fl *ast.FieldList) {
	for _, field := range fl.List {
		//obj, ok := v.info.Implicits[field]
		//if ok {
		//	v.Printf("***TODO***")
		//	v.Dump(obj)
		//}

		// resolve all inherited fields
		anonymous := field.Names == nil
		if anonymous {
			if v.Debug {
				v.DepthPrintf("resolveFieldList anonymous field %v", field)
			}
			v.resolveNode(field)
			continue
		}

		// check field if it matches an id being searched
		// TODO: should not have to check all ids in stack
		for e := v.idStack.Back(); e != nil; e = e.Prev() {
			id := e.Value.(*ast.Ident)
			for _, id2 := range field.Names {
				if id2.Name == id.Name {
					if v.Debug {
						v.DepthPrintf("resolveFieldList found field %v", id.Name)
					}
					v.resolveNode(field)
					break
				}
			}
		}
	}
}

func (v *GSVisitor) resolveId(id *ast.Ident) {
	//v.idStack.PushBack(id)
	//defer v.idStack.Remove(v.idStack.Back())

	if v.Debug {
		var u []string
		for e := v.idStack.Front(); e != nil; e = e.Next() {
			u = append(u, e.Value.(*ast.Ident).String())
		}
		v.DepthPrintf("resolveId %v idStack=[%v]", id, strings.Join(u, "->"))
		v.DepthPrintf("resolveId pos %v", v.fset.Position(id.Pos()))
	}

	// solved by the parser
	if id.Obj != nil {
		if n, ok := id.Obj.Decl.(ast.Node); ok {
			v.resolveNode(n)
			return
		}
	}
	// solved in info.uses
	obj := v.info.Uses[id]
	if obj != nil {
		v.resolveIdObj(id, obj)
		return
	}

	//// solved in info.defs
	//obj = v.info.Defs[id]
	//if obj != nil {
	//	v.resolveIdObj(id, obj)
	//	return
	//}

	//// implicit
	//obj, ok := v.info.Implicits[id]
	//if ok {
	//	if v.Debug {
	//		v.Printf("***TODO***")
	//		v.Dump(obj)
	//	}
	//}

	// could be an id defined in the same package but in another file
	path := v.posFilePath(v.mainFile.Package)
	if !v.importable[path] {
		if v.Debug {
			v.DepthPrintf("resolveId making mainfile path importable")
		}
		v.importable[path] = true
		v.confCheckMainFile()
		v.resolveId(id) // try again
		return
	}

	// nothing else todo other then trying to solve the parent node
	v.resolveParent(id)
}

func (v *GSVisitor) resolveIdObj(id *ast.Ident, obj types.Object) {
	pos := obj.Pos()

	// could be from the bulitin package
	if pos == token.NoPos {
		b := "builtin"
		v.importable[b] = true
		pkg, _ := v.packageImporter(b, "", 0)
		obj2 := pkg.Scope().Lookup(obj.Name())
		if obj2 != nil {
			v.info.Uses[id] = obj2 // TODO: don't use the info.uses?
			pos = obj2.Pos()
		}
	}

	v.resolvePos(pos)
}

func (v *GSVisitor) resolvePos(pos token.Pos) {
	if pos == token.NoPos {
		if v.Debug {
			v.DepthPrintf("resolvePos no pos")
		}
		return
	}
	if v.Debug {
		v.DepthPrintf("resolvePos: have pos %v %v", pos, v.fset.Position(pos))
	}
	path := v.posDeepestPath(pos)
	v.populateParents(path)
	last := path[len(path)-1]
	v.resolveNode(last)
}

func (v *GSVisitor) resolveImportSpec(imp *ast.ImportSpec) {
	// make this import importable
	path, _ := strconv.Unquote(imp.Path.Value)
	v.importable[path] = true

	// reset imported paths to allow re-confcheck all
	v.imported = make(map[string]*types.Package)

	// re check main file that will now re-import available importables
	v.confCheckMainFile()
}

//func (v *GSVisitor) posDeepestNode(pos token.Pos) ast.Node {
//	astFile := v.posAstFile(pos)
//	var n2 ast.Node
//	ast.Inspect(astFile, func(node ast.Node) bool {
//		if node == nil {
//			return false
//		}
//		if node.Pos() > pos {
//			return false
//		}
//		// find deepest node that matches pos
//		if node.Pos() == pos {
//			n2 = node
//		}
//		return true
//	})
//	return n2
//}

func (v *GSVisitor) posDeepestPath(pos token.Pos) []ast.Node {
	astFile := v.posAstFile(pos)
	var path []ast.Node
	var path2 []ast.Node
	size := 10000
	ast.Inspect(astFile, func(node ast.Node) bool {
		if node == nil {
			path = path[:len(path)-1]
			return false
		}
		path = append(path, node)
		if node.Pos() > pos {
			return false
		}
		// find deepest node that matches pos
		if pos < node.End() {
			s := int(node.End() - node.Pos())
			if s <= size {
				size = s
				path2 = make([]ast.Node, len(path))
				copy(path2, path)
			}
		}
		return true
	})
	if len(path2) > 0 {
		if v.Debug {
			v.printPath(path2)
		}
		return path2
	}
	return nil
}

func (v *GSVisitor) printPath(path []ast.Node) {
	var u []string
	for _, e := range path {
		u = append(u, fmt.Sprintf("%v", reflect.TypeOf(e)))
	}
	v.Printf("[%s]", strings.Join(u, ", "))
}

func (v *GSVisitor) populateParents(path []ast.Node) {
	for i := 1; i < len(path); i++ {
		v.parents[path[i]] = path[i-1]
	}
}

func (v *GSVisitor) confCheckMainFile() {
	path := v.posFilePath(v.mainFile.Package)

	// conf check the main file package
	if v.importable[path] {
		_, _ = v.confCheckPath(path, "", 0)
		return
	}

	// just conf check the main file
	_, _ = v.confCheckFiles(path, []*ast.File{v.mainFile})
}

func (v *GSVisitor) packageImporter(path, dir string, mode types.ImportMode) (*types.Package, error) {
	if !v.importable[path] {
		return nil, fmt.Errorf("not importable")
	}
	pkg, ok := v.imported[path]
	if ok {
		return pkg, nil
	}
	if v.Debug {
		v.Printf("importing: %q %q", path, dir)
	}
	pkg, _ = v.confCheckPath(path, dir, build.ImportMode(mode))
	v.imported[path] = pkg
	if v.Debug {
		v.Printf("imported: %v", pkg)
	}
	return pkg, nil
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
		v.Printf("parseFilename: %v (err=%v)", filepath.Base(filename), err)
	}
	v.astFilesMu.Lock()
	v.astFiles[filename] = file
	v.astFilesMu.Unlock()
	return file
}

func (v *GSVisitor) confCheckPath(path, dir string, mode build.ImportMode) (*types.Package, error) {
	filenames := v.pathFilenames(path, dir, mode)
	files := v.parseFilenames(filenames)
	return v.confCheckFiles(path, files)
}

func (v *GSVisitor) confCheckFiles(path string, files []*ast.File) (*types.Package, error) {
	pkg, err := v.conf.Check(path, v.fset, files, &v.info)
	if v.Debug {
		v.Printf("confCheckFiles %v (err=%v)", path, err)
	}
	return pkg, err
}

func (v *GSVisitor) DepthPrintf(f string, a ...interface{}) {
	u := append([]interface{}{(v.resolveDepth - 1) * 4, ""}, a...)
	v.Printf("%*s"+f, u...)
}
func (v *GSVisitor) Printf(f string, a ...interface{}) {
	log.Printf(f, a...)
}
func (v *GSVisitor) Dump(a ...interface{}) {
	v.Printf(v.Sdumpd(4, a...))
}
func (v *GSVisitor) Sdumpd(depth int, a ...interface{}) string {
	conf := spew.NewDefaultConfig()
	conf.MaxDepth = depth
	conf.Indent = "\t"
	return conf.Sdump(a...)
}

type importFn func(path, dir string, mode types.ImportMode) (*types.Package, error)

func (fn importFn) Import(path string) (*types.Package, error) {
	return fn.ImportFrom(path, "", 0)
}
func (fn importFn) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	return fn(path, dir, mode)
}
