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
	"runtime"
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
	ed := erow.Ed()
	m := make(map[cmdutil.ERower]bool)

	// any duplicate row that has the index already visible
	erows := ed.FindERowers(pos.Filename)
	for _, e := range erows {
		if e.Row().TextArea.IndexIsVisible(pos.Offset) {
			m[e] = true
		}
	}

	// choose a duplicate row that is not the current row
	if len(m) == 0 {
		erows := ed.FindERowers(pos.Filename)
		for _, e := range erows {
			if e != erow {
				m[e] = true
				break
			}
		}
	}

	// open new row
	if len(m) == 0 {
		col, nextRow := ed.GoodColumnRowPlace()
		e := ed.NewERowerBeforeRow(pos.Filename, col, nextRow)
		err := e.LoadContentClear()
		if err != nil {
			ed.Error(err)
			return
		}
		m[e] = true
	}

	// show position on selected rows
	for e, _ := range m {
		row2 := e.Row()
		row2.ResizeTextAreaIfVerySmall()
		ta2 := row2.TextArea
		ta2.MakeIndexVisible(pos.Offset)
		row2.TextArea.FlashIndexLen(pos.Offset, end.Offset-pos.Offset)
	}
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
	parents      map[ast.Node]ast.Node // just the id path node parents
	visited      [4]map[ast.Node]bool
	astFiles     struct {
		sync.RWMutex // used only during parse
		m            map[string]*ast.File
	}

	resolveDepth int
	Debug        bool
}

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
		imported:   make(map[string]*types.Package),
		importable: make(map[string]bool),
		parents:    make(map[ast.Node]ast.Node),
	}
	for i, _ := range v.visited {
		v.visited[i] = make(map[ast.Node]bool)
	}
	v.astFiles.m = make(map[string]*ast.File)

	v.conf = types.Config{
		// it will exit on first error if not defined
		Error: func(err error) {
			//v.debugf("conf error: %v", err)
		},
		DisableUnusedImportCheck: true, // faster? (works without it)
		Importer:                 importFn(v.packageImporter),
	}

	//v.Debug = true

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
		//v.Dump(bpkg)
	}

	// parse source (src string)
	v.mainFilename = filename
	v.mainFile = v.parseFilename(filename, src)

	// DEBUG
	if v.Debug {
		v.printAllIdentsOffsets()
	}

	// make main package importable
	path := v.posFilePath(v.mainFile.Package)
	v.importable[path] = true

	// first check pass without any imports cached
	v.confCheckMainFile()

	// cursor node
	mainTokenFile := v.fset.File(v.mainFile.Package)
	// avoid panic from a bad index
	if cursorIndex > mainTokenFile.Size() {
		return nil, nil, fmt.Errorf("bad cursor index")
	}
	cursorNode := v.posNode(mainTokenFile.Pos(cursorIndex))

	// must be an id
	cursorId, ok := cursorNode.(*ast.Ident)
	if !ok {
		return nil, nil, fmt.Errorf("cursor not at an id node")
	}

	// resolve id declaration
	node := v.resolveDecl(cursorId)
	if node == nil {
		return nil, nil, fmt.Errorf("id decl not found")
	}

	// improve final node to extract the position
	switch t := node.(type) {
	case *ast.FuncDecl:
		node = t.Name
	case *ast.TypeSpec:
		node = t.Name
	case *ast.AssignStmt:
		lhsi, _ := v.idAssignStmtRhs(cursorId, t)
		if lhsi >= 0 {
			node = t.Lhs[lhsi]
		}
	case *ast.Field:
		for _, id2 := range t.Names {
			if id2.Name == cursorId.Name {
				node = id2
				break
			}
		}
	default:
		//v.debugf("TODO")
		//v.Dump(node)
	}

	// node position
	posp := v.fset.Position(node.Pos())
	endp := v.fset.Position(node.End())
	if v.Debug {
		v.Printf("***result: offset=%v %v", posp.Offset, posp)
	}
	return &posp, &endp, nil
}

func (v *GSVisitor) posFilePath(pos token.Pos) string {
	name := v.fset.File(pos).Name() // file path
	name = filepath.Dir(name)
	name = v.normalizePath(name)
	return name
}

func (v *GSVisitor) posAstFile(pos token.Pos) *ast.File {
	tokenFile := v.fset.File(pos)
	return v.astFiles.m[tokenFile.Name()]
}

func (v *GSVisitor) posNode(pos token.Pos) ast.Node {
	if pos == token.NoPos {
		v.debugf("no pos")
		return nil
	}
	v.debugf("have pos %v", v.fset.Position(pos).Offset)
	path := v.posNodePath(pos)
	if len(path) > 0 {
		last := path[len(path)-1]
		return last
	}
	return nil
}

func (v *GSVisitor) vis(index int, node ast.Node) bool {
	if v.visited[index][node] {
		v.debugf("already visited")
		return true
	}
	v.visited[index][node] = true
	return false
}

func (v *GSVisitor) clearVis(index int, node ast.Node) {
	v.visited[index][node] = false
}

func (v *GSVisitor) resolveDecl(node ast.Node) ast.Node {
	v.debugf("%v %v", reflect.TypeOf(node), node)

	if v.Debug {
		v.resolveDepth++
		defer func() { v.resolveDepth-- }()
	}

	if v.vis(0, node) {
		return nil
	}
	defer v.clearVis(0, node)

	switch t := node.(type) {
	case *ast.Ident:

		// preemptively solve case clause types to help the checker
		v.resolvePathCaseClauseTypes(t)

		if n := v.getIdDecl(t); n != nil {
			return n
		}
		if pn, ok := v.nodeParent(t); ok {
			if n := v.resolveDecl(pn); n != nil {
				return n
			}
		}
	case *ast.SelectorExpr:
		if n := v.resolveType(t.X); n != nil {
			return v.getIdDecl(t.Sel)
		}
	default:
		_ = t
		v.debugf("TODO")
		v.Dump(node)
	}

	v.debugf("not solved")
	return nil
}

func (v *GSVisitor) resolveType(node ast.Node) ast.Node {
	v.debugf("%v %v", reflect.TypeOf(node), node)

	if v.Debug {
		v.resolveDepth++
		defer func() { v.resolveDepth-- }()
	}

	if v.vis(1, node) {
		return nil
	}
	defer v.clearVis(1, node)

	switch t := node.(type) {
	case *ast.Ident:
		var node2 ast.Node
		if n := v.resolveDecl(t); n != nil {
			if n == t {
				if pn, ok := v.nodeParent(t); ok {
					node2 = v.resolveType(pn)
				}
			} else {
				node2 = v.resolveType(n)
			}
		}
		if node2 != nil {
			switch t2 := node2.(type) {
			case *ast.AssignStmt:
				id := t
				as := t2
				lhsi, rhsn := v.idAssignStmtRhs(id, as)
				if rhsn != nil && lhsi >= 0 {
					if n := v.resolveType(rhsn); n != nil {
						if lhsi == 0 {
							switch t3 := n.(type) {
							case *ast.StructType:
								return t3
							case *ast.InterfaceType:
								return t3
							}
						}
						v.debugf("TODO id AssignStmt")
						v.Dump(lhsi)
						v.Dump(n)
					}
				}
			default:
				return node2
			}
		}
	case *ast.BasicLit:
		if pn, ok := v.nodeParent(t); ok {
			return v.resolveType(pn)
		}
	case *ast.ImportSpec:
		v.makeImportSpecImportableAndConfCheck(t)
		if v.importSpecImported(t) {
			return t
		}
	case *ast.SelectorExpr:
		var node2 ast.Node
		if n := v.getSelectorExprType(t); n != nil {
			node2 = n
		}
		if node2 == nil {
			if n := v.resolveType(t.X); n != nil {
				if n := v.getSelectorExprType(t); n != nil {
					node2 = n
				} else {
					node2 = v.resolveType(t.Sel)
				}
			}
		}
		if node2 != nil {
			switch t2 := node2.(type) {
			case *ast.FuncType:
				if t2.Results != nil && len(t2.Results.List) >= 1 {
					return v.resolveType(t2.Results.List[0])
				}
			case *ast.StructType:
				return t2
			case *ast.InterfaceType:
				return t2
			default:
				v.debugf("TODO selectorExpr node2")
				v.Dump(node2)
			}
		}
	case *ast.Field:
		return v.resolveType(t.Type)
	case *ast.TypeSpec:
		return v.resolveType(t.Type)
	case *ast.StructType:
		v.resolveAnonFieldsTypes(t.Fields)
		return t
	case *ast.InterfaceType:
		v.resolveAnonFieldsTypes(t.Methods)
		return t
	case *ast.ValueSpec:
		if t.Type == nil {
			// ex: "var a = 1"
			if pn, ok := v.nodeParent(t); ok {
				return v.resolveType(pn)
			}
			return nil
		} else {
			return v.resolveType(t.Type)
		}
	case *ast.StarExpr:
		return v.resolveType(t.X)
	case *ast.AssignStmt:
		return t
	case *ast.TypeAssertExpr:
		if t.Type == nil {
			// ex: "switch x.(type)"
			return t
		} else {
			return v.resolveType(t.Type)
		}
	case *ast.CallExpr:
		return v.resolveType(t.Fun)
	case *ast.FuncType:
		return t
	case *ast.FuncDecl:
		return v.resolveType(t.Type)
	case *ast.IndexExpr:
		return v.resolveType(t.X)
	case *ast.MapType:
		return v.resolveType(t.Value)
	default:
		_ = t
		v.debugf("TODO")
		v.Dump(node)
	}

	v.debugf("not solved")
	return nil
}

func (v *GSVisitor) getIdDecl(id *ast.Ident) ast.Node {
	v.debugf("%v", id)

	// solved by the parser
	if id.Obj != nil {
		if n, ok := id.Obj.Decl.(ast.Node); ok {
			v.debugf("in parser")
			return n
		}
		v.debugf("TODO 1")
		v.Dump(id.Obj)
	}

	// solved in info.uses
	obj := v.info.Uses[id]
	if obj != nil {
		pos := obj.Pos()
		if pos != token.NoPos {
			v.debugf("in uses")
			return v.posNode(pos)
		}
		// builtin package
		if pos == token.NoPos {
			b := "builtin"
			v.importable[b] = true
			pkg, _ := v.packageImporter(b, "", 0)
			obj2 := pkg.Scope().Lookup(id.Name)
			if obj2 != nil {
				return v.posNode(obj2.Pos())
			}
		}
	}

	// solved in info.defs
	obj = v.info.Defs[id]
	if obj != nil {
		v.debugf("in defs")
		return id
	}

	// search in scopes
	astFile := v.posAstFile(id.Pos())
	s1, ok := v.info.Scopes[astFile]
	if ok {
		s := s1.Innermost(id.Pos())
		if s != nil {
			_, obj := s.LookupParent(id.Name, id.Pos())
			if obj != nil {
				v.debugf("in scopes")
				return v.posNode(obj.Pos())
			}
		}
	}

	v.debugf("not found")
	return nil
}

func (v *GSVisitor) resolvePathCaseClauseTypes(node ast.Node) {
	// in some cases the CaseClause is not present in v.info.scopes

	path := v.nodePath(node)
	for _, n := range path {
		if cc, ok := n.(*ast.CaseClause); ok {
			for _, e := range cc.List {
				v.debugf("%v %v", e, node)
				_ = v.resolveType(e)
			}
		}
	}
}

func (v *GSVisitor) getSelectorExprType(se *ast.SelectorExpr) ast.Node {
	v.debugf("%v", se)
	// solved by the checker
	sel, ok := v.info.Selections[se]
	if ok {
		n := v.posNode(sel.Obj().Pos())
		return v.resolveType(n)
	}
	v.debugf("not found")
	return nil
}

func (v *GSVisitor) makeImportSpecImportableAndConfCheck(imp *ast.ImportSpec) {
	path := v.importSpecPath(imp)
	if _, ok := v.importable[path]; !ok {
		v.debugf("%v", imp.Path)
		// make path importable
		v.importable[path] = true
		// reset imported paths to clear cached pkgs
		v.imported = make(map[string]*types.Package)
		// check main file that will now re-import available importables
		v.confCheckMainFile()
	}
}
func (v *GSVisitor) importSpecImported(imp *ast.ImportSpec) bool {
	path := v.importSpecPath(imp)
	return v.imported[path] != nil
}
func (v *GSVisitor) importSpecPath(imp *ast.ImportSpec) string {
	path, _ := strconv.Unquote(imp.Path.Value)
	return path
}

func (v *GSVisitor) idAssignStmtRhs(id *ast.Ident, as *ast.AssignStmt) (int, ast.Node) {
	v.debugf("%v %v", id, as)
	// left-hand-side index
	lhsi := -1
	for i, e := range as.Lhs {
		if id2, ok := e.(*ast.Ident); ok && id2.Name == id.Name {
			lhsi = i
			break
		}
	}
	if lhsi < 0 {
		return 0, nil
	}
	// right-hand-side node
	if len(as.Rhs) == len(as.Lhs) {
		return lhsi, as.Rhs[lhsi]
	}
	if len(as.Rhs) == 1 {
		return lhsi, as.Rhs[0]
	}
	return lhsi, nil
}

func (v *GSVisitor) resolveAnonFieldsTypes(fl *ast.FieldList) {
	for _, f := range fl.List {
		if f.Names == nil {
			v.debugf("anon field")
			_ = v.resolveType(f)
		}
	}
}

func (v *GSVisitor) IsInvalidType(t types.Type) bool {
	if u, ok := t.(*types.Basic); ok {
		return u == types.Typ[types.Invalid]
	}
	return false
}

func (v *GSVisitor) debugInfo(node ast.Node) {
	v.debugf("%v %v", reflect.TypeOf(node), node)
	// types
	if e, ok := node.(ast.Expr); ok {
		tv, ok := v.info.Types[e]
		if ok {
			v.debugf("in types")
			v.Dump(tv)
		}
	}
	// defs
	if id, ok := node.(*ast.Ident); ok {
		obj, ok := v.info.Defs[id]
		if ok {
			v.debugf("in defs")
			v.Dump(obj)
		}
	}
	// uses
	if id, ok := node.(*ast.Ident); ok {
		obj, ok := v.info.Uses[id]
		if ok {
			v.debugf("in uses")
			v.Dump(obj)
		}
	}
	// implicits
	if true {
		obj, ok := v.info.Implicits[node]
		if ok {
			v.debugf("in implicits")
			v.Dump(obj)
		}
	}
	// selections
	if se, ok := node.(*ast.SelectorExpr); ok {
		sel, ok := v.info.Selections[se]
		if ok {
			v.debugf("in selections")
			v.Dump(sel)
		}
	}
	// scopes
	if true {
		scope, ok := v.info.Scopes[node]
		if ok {
			v.debugf("in scopes")
			v.Dump(scope)
		}
	}
}

func (v *GSVisitor) nodeParent(node ast.Node) (ast.Node, bool) {
	if pn, ok := v.parents[node]; ok {
		return pn, true
	}
	path := v.nodePath(node)
	if len(path) >= 2 {
		return path[len(path)-2], true
	}
	return nil, false
}

func (v *GSVisitor) nodePath(node ast.Node) []ast.Node {
	// try cached path first (faster)
	path0 := v.cachedNodePath(node)
	if len(path0) > 0 {
		return path0
	}

	path := v.posNodePath(node.Pos())
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == node {
			return path[:i+1]
		}
	}
	return nil
}

func (v *GSVisitor) cachedNodePath(node ast.Node) []ast.Node {
	var p []ast.Node
	n := node
	for {
		if pn, ok := v.parents[n]; ok {
			p = append(p, pn)
			n = pn
			continue
		}
		break
	}
	if len(p) > 0 {
		// reverse order
		l := len(p) - 1
		for i := 0; i <= l/2; i++ {
			p[i], p[l-i] = p[l-i], p[i]
		}
		return append(p, node)
	}
	return nil
}

// Path to innermost node.
func (v *GSVisitor) posNodePath(pos token.Pos) []ast.Node {
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
		// find innermost node that matches pos
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
	v.populateParents(path2)
	return path2
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
	a = append(a, bpkg.TestGoFiles...)
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
	v.astFiles.RLock()
	file, ok := v.astFiles.m[filename]
	v.astFiles.RUnlock()
	if ok {
		return file
	}
	file, err := parser.ParseFile(v.fset, filename, src, parser.AllErrors)
	if v.Debug {
		v.Printf("parseFilename: %v (err=%v)", filepath.Base(filename), err)
	}
	v.astFiles.Lock()
	v.astFiles.m[filename] = file
	v.astFiles.Unlock()
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

func (v *GSVisitor) printPath(path []ast.Node) {
	var u []string
	for _, n := range path {
		s := reflect.TypeOf(n).String()
		if id, ok := n.(*ast.Ident); ok {
			s = id.String()
		}
		u = append(u, s)
	}
	v.Printf("path=[%s]", strings.Join(u, ","))
}

func (v *GSVisitor) printParents(node ast.Node) {
	var u []string
	for n := v.parents[node]; n != nil; n = v.parents[n] {
		s := reflect.TypeOf(n).String()
		if id, ok := n.(*ast.Ident); ok {
			s = id.String()
		}
		u = append(u, s)
	}
	v.DepthPrintf("parents=[%v]", strings.Join(u, ","))
}

func (v *GSVisitor) printAllIdentsOffsets() {
	astFile := v.posAstFile(v.mainFile.Package)
	ast.Inspect(astFile, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if id, ok := node.(*ast.Ident); ok {
			p := v.fset.Position(id.Pos())
			v.Printf("ident %v %v", p.Offset, id)
		}
		return true
	})
}

func (v *GSVisitor) debugf(f string, a ...interface{}) {
	if !v.Debug {
		return
	}

	fname := ""
	fpcs := make([]uintptr, 1) // num of callers to get
	n := runtime.Callers(2, fpcs)
	if n != 0 {
		fun := runtime.FuncForPC(fpcs[0] - 1) // get info
		if fun != nil {
			s := fun.Name()
			i := strings.LastIndex(s, ".")
			if i >= 0 {
				s = s[i:]
			}
			fname = s + ": "
		}
	}

	u := append([]interface{}{v.resolveDepth * 4, ""}, a...)
	v.Printf("%*s"+fname+f, u...)
}

func (v *GSVisitor) DepthPrintf(f string, a ...interface{}) {
	u := append([]interface{}{v.resolveDepth * 4, ""}, a...)
	v.Printf("%*s"+f, u...)
}
func (v *GSVisitor) Printf(f string, a ...interface{}) {
	log.Printf(f, a...)
}

func (v *GSVisitor) Dump(a ...interface{}) {
	if !v.Debug {
		return
	}
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
