package gosource

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type Info struct {
	FSet       *token.FileSet
	Info       types.Info
	Conf       types.Config
	Pkgs       map[string]*types.Package // if present, means imported
	Importable map[string]bool
	Parents    map[ast.Node]ast.Node

	astFiles   map[string]*ast.File
	astFilesMu sync.RWMutex

	extraPathFiles map[string]map[string]bool
}

func NewInfo() *Info {
	info := &Info{
		FSet: token.NewFileSet(),
		Info: types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
		},
		Pkgs:           make(map[string]*types.Package),
		Importable:     make(map[string]bool),
		Parents:        make(map[ast.Node]ast.Node),
		astFiles:       make(map[string]*ast.File),
		extraPathFiles: make(map[string]map[string]bool),
	}

	info.Conf = types.Config{
		DisableUnusedImportCheck: true, // faster? (works without it)
		Importer:                 ImportFn(info.PackageImporter),
		// it will exit on first error if not defined
		Error: func(err error) {},
	}

	return info
}

func (info *Info) AddPathFile(filename string) string {
	filename = info.FullFilename(filename)

	dir := filepath.Dir(filename)
	path := info.removeSrcDirPrefix(dir)
	m, ok := info.extraPathFiles[path]
	if !ok {
		m = make(map[string]bool)
		info.extraPathFiles[path] = m
	}
	m[filename] = true
	Logf("added %v to path %v", filename, path)
	return filename
}

func (info *Info) FullFilename(filename string) string {
	// find full filename if the package import path was given (not full path)
	bpkg, _ := build.Import(filepath.Dir(filename), "", 0)
	if bpkg.Dir != "" {
		filename = filepath.Join(bpkg.Dir, filepath.Base(filename))
		Logf("filename is now %v", filename)
		//v.Dump(bpkg)
	}
	return filename
}

func (info *Info) PosNode(pos token.Pos) ast.Node {
	if pos == token.NoPos {
		Logf("no pos")
		return nil
	}
	Logf("have pos %v", info.FSet.Position(pos).Offset)
	path := info.PosNodePath(pos)
	if len(path) > 0 {
		n := path[len(path)-1]

		// anon field with a SelectorExpr, need to continue with the selector
		if id, ok := n.(*ast.Ident); ok {
			if pn, ok := info.NodeParent(id); ok {
				if se, ok := pn.(*ast.SelectorExpr); ok {
					if pn2, ok := info.NodeParent(se); ok {
						if f, ok := pn2.(*ast.Field); ok && f.Names == nil {
							if se.Pos() == n.Pos() {
								n = se.Sel
							}
						}
					}
				}
			}
		}

		return n
	}
	return nil
}

// Path to innermost node.
func (info *Info) PosNodePath(pos token.Pos) []ast.Node {
	astFile := info.PosAstFile(pos)
	var path []ast.Node
	var path2 []ast.Node
	end := astFile.End()
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
		nend := node.End()
		if pos < nend && nend <= end {
			end = nend
			path2 = make([]ast.Node, len(path))
			copy(path2, path)
		}
		return true
	})

	// populate parents
	for i := 1; i < len(path2); i++ {
		info.Parents[path2[i]] = path2[i-1]
	}

	return path2
}

func (info *Info) NodeParent(node ast.Node) (ast.Node, bool) {
	Logf("")
	if pn, ok := info.Parents[node]; ok {
		return pn, true
	}
	path := info.NodePath(node)
	if len(path) >= 2 {
		return path[len(path)-2], true
	}
	return nil, false
}
func (info *Info) NodePath(node ast.Node) []ast.Node {
	// try cached path first (faster)
	path0 := info.cachedNodePath(node)
	if len(path0) > 0 {
		return path0
	}

	path := info.PosNodePath(node.Pos())
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == node {
			return path[:i+1]
		}
	}
	return nil
}
func (info *Info) cachedNodePath(node ast.Node) []ast.Node {
	var p []ast.Node
	n := node
	for {
		if pn, ok := info.Parents[n]; ok {
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

func (info *Info) PosAstFile(pos token.Pos) *ast.File {
	tokenFile := info.FSet.File(pos)
	return info.astFiles[tokenFile.Name()]
}

func (info *Info) PosFilePath(pos token.Pos) string {
	fp := info.FSet.File(pos).Name()
	dir := filepath.Dir(fp)
	return info.removeSrcDirPrefix(dir)
}
func (info *Info) removeSrcDirPrefix(path string) string {
	for _, d := range build.Default.SrcDirs() {
		d += "/"
		if strings.HasPrefix(path, d) {
			return path[len(d):]
		}
	}
	return path
}

func (info *Info) PackageImporter(path, dir string, mode types.ImportMode) (*types.Package, error) {
	if !info.Importable[path] {
		return nil, fmt.Errorf("not importable")
	}
	pkg, ok := info.Pkgs[path]
	if ok {
		return pkg, nil
	}
	//Logf("importing path %q", path)
	pkg, _ = info.ConfCheckPath(path, dir, build.ImportMode(mode))
	info.Pkgs[path] = pkg
	Logf("imported: %v", pkg)
	return pkg, nil
}

func (info *Info) ConfCheckPath(path, dir string, mode build.ImportMode) (*types.Package, error) {
	filenames := info.PathFilenames(path, dir, mode)
	//for _, f := range filenames {
	//	Logf("%v", f)
	//}
	files := info.AstFiles(filenames)
	return info.ConfCheckFiles(path, files)
}

func (info *Info) PathFilenames(path, dir string, mode build.ImportMode) []string {
	Logf("%v %v", path, dir)

	// build.package contains info to get the full filename
	bpkg, _ := build.Import(path, dir, mode)
	//Dump(bpkg, path)

	var names []string

	// add extra path files
	added := make(map[string]bool)
	if m, ok := info.extraPathFiles[path]; ok {
		for k, _ := range m {
			added[k] = true
			names = append(names, k)
		}
	}

	// package filenames
	a := append(bpkg.GoFiles, bpkg.CgoFiles...)
	a = append(a, bpkg.TestGoFiles...)
	for _, fname := range a {
		u := filepath.Join(bpkg.Dir, fname)
		if _, ok := added[u]; !ok {
			names = append(names, u)
		}
	}

	return names
}

func (info *Info) AstFiles(filenames []string) []*ast.File {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var files []*ast.File

	addFile := func(f *ast.File) {
		mu.Lock()
		files = append(files, f)
		mu.Unlock()
	}

	for _, filename := range filenames {
		info.astFilesMu.RLock()
		file, ok := info.astFiles[filename]
		info.astFilesMu.RUnlock()
		if ok {
			addFile(file)
			continue
		}

		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			file := info.ParseFile(filename, nil)
			addFile(file)
		}(filename)
	}
	wg.Wait()
	return files
}

func (info *Info) ParseFile(filename string, src interface{}) *ast.File {
	info.astFilesMu.RLock()
	file, ok := info.astFiles[filename]
	info.astFilesMu.RUnlock()
	if ok {
		return file
	}
	file, err := parser.ParseFile(info.FSet, filename, src, parser.AllErrors)
	Logf("%v (err=%v)", filepath.Base(filename), err)
	info.astFilesMu.Lock()
	info.astFiles[filename] = file
	info.astFilesMu.Unlock()
	return file
}

func (info *Info) ConfCheckFiles(path string, files []*ast.File) (*types.Package, error) {
	pkg, err := info.Conf.Check(path, info.FSet, files, &info.Info)
	Logf("%v (err=%v)", path, err)
	return pkg, err
}

func (info *Info) PrintPath(path []ast.Node) {
	var u []string
	for _, n := range path {
		s := reflect.TypeOf(n).String()
		if id, ok := n.(*ast.Ident); ok {
			s = id.String()
		}
		u = append(u, s)
	}
	fmt.Printf("path=[%s]\n", strings.Join(u, ","))
}

func (info *Info) PrintIdOffsets(astFile *ast.File) {
	ast.Inspect(astFile, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if id, ok := node.(*ast.Ident); ok {
			p := info.FSet.Position(id.Pos())
			fmt.Printf("ident %v %v\n", p.Offset, id)
		}
		return true
	})
}

func (info *Info) PrintNodeOnInfo(node ast.Node) {
	Logf("%v %v", reflect.TypeOf(node), node)
	// types
	if e, ok := node.(ast.Expr); ok {
		tv, ok := info.Info.Types[e]
		if ok {
			Logf("in types")
			Dump(tv)
		}
	}
	// defs
	if id, ok := node.(*ast.Ident); ok {
		obj, ok := info.Info.Defs[id]
		if ok {
			Logf("in defs")
			Dump(obj)
		}
	}
	// uses
	if id, ok := node.(*ast.Ident); ok {
		obj, ok := info.Info.Uses[id]
		if ok {
			Logf("in uses")
			Dump(obj)
		}
	}
	// implicits
	if true {
		obj, ok := info.Info.Implicits[node]
		if ok {
			Logf("in implicits")
			Dump(obj)
		}
	}
	// selections
	if se, ok := node.(*ast.SelectorExpr); ok {
		sel, ok := info.Info.Selections[se]
		if ok {
			Logf("in selections")
			Dump(sel)
		}
	}
	// scopes
	if true {
		scope, ok := info.Info.Scopes[node]
		if ok {
			Logf("in scopes")
			Dump(scope)
		}
	}
}

type ImportFn func(path, dir string, mode types.ImportMode) (*types.Package, error)

func (fn ImportFn) Import(path string) (*types.Package, error) {
	return fn.ImportFrom(path, "", 0)
}
func (fn ImportFn) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	return fn(path, dir, mode)
}
