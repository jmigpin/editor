package gosource

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"
)

func CodeCompletion(filename string, src interface{}, index int) (int, string, error) {
	cc := &CC{}
	index2, objs, err := cc.Run(filename, src, index)
	if err != nil {
		return 0, "", err
	}
	if len(objs) == 0 {
		return 0, "", fmt.Errorf("no objects")
	}
	//return index2, FormatObjsDefault(objs), nil
	return index2, FormatObjs(objs), nil
}

type CC struct {
	info *Info
	objs []types.Object

	// TODO: errors/todo_msgs
}

func (cc *CC) Run(filename string, src interface{}, index int) (int, []types.Object, error) {
	//LogDebug()

	cc.info = NewInfo()

	// parse main file
	filename = cc.info.AddPathFile(filename)
	astFile := cc.info.ParseFile(filename, src)
	if astFile == nil {
		return 0, nil, fmt.Errorf("unable to parse file")
	}

	if Debug {
		cc.info.PrintIdOffsets(astFile)
	}

	// index node
	tokenFile := cc.info.FSet.File(astFile.Package)
	if tokenFile == nil {
		return 0, nil, fmt.Errorf("unable to get token file")
	}
	indexNode := cc.info.PosNode(cc.info.SafeOffsetPos(tokenFile, index))
	if indexNode == nil {
		return 0, nil, fmt.Errorf("index node not found")
	}
	inpos := cc.info.FSet.Position(indexNode.Pos())

	cc.getCandidates(indexNode)
	cc.filterCandidates(indexNode, index)
	return inpos.Offset, cc.objs, nil
}

func (cc *CC) getCandidates(node ast.Node) {
	switch t := node.(type) {
	case *ast.Ident:
		id := t
		if pn, ok := cc.info.NodeParent(id); ok {
			switch t2 := pn.(type) {
			case *ast.SelectorExpr:
				se := t2
				if id == se.Sel {
					cc.getCandidatesIn(se.X)
					return
				}
			case *ast.FuncDecl:
				return
			default:
				LogTODO(t2)
			}
		}
		cc.getCandidatesInScope(node)
	case *ast.SelectorExpr:
		// at the dot of the SE: "a|.b"
		//cc.getCandidatesIn(t.X)
		cc.getCandidatesInScope(node)
	default:
		LogTODO(t)
	}
}

func (cc *CC) getCandidatesInScope(node ast.Node) {
	path := cc.info.PosFilePath(node.Pos())
	res := NewResolver(cc.info, path, node)
	_ = res.ResolveType(node)

	// get scope
	astFile := cc.info.PosAstFile(node.Pos())
	scope, ok := cc.info.Info.Scopes[astFile]
	if !ok {
		return
	}

	// search scope and parent scopes
	m := make(map[string]bool)
	for s := scope; s != nil; s = s.Parent() {
		for _, n := range s.Names() {
			if _, ok := m[n]; !ok {
				m[n] = true
				obj := s.Lookup(n)
				cc.objs = append(cc.objs, obj)
			}
		}
	}
}

func (cc *CC) getCandidatesIn(node ast.Node) {
	path := cc.info.PosFilePath(node.Pos())
	res := NewResolver(cc.info, path, node)
	n := res.ResolveType(node)

	switch t := n.(type) {
	case *ast.ImportSpec:
		cc.getCandidatesInImportSpec(node, t)
	case *ast.TypeSpec:
		cc.getCandidatesInTypeSpec(node, t)
	case *ast.FuncType:
		if t.Results != nil && len(t.Results.List) > 0 {
			n2 := t.Results.List[0]
			n3 := res.ResolveType(n2)
			switch t2 := n3.(type) {
			case *ast.TypeSpec:
				cc.getCandidatesInTypeSpec(node, t2)
			default:
				LogTODO(t2)
			}
		}
	default:
		LogTODO(n)
	}
}

func (cc *CC) getCandidatesInImportSpec(node ast.Node, imp *ast.ImportSpec) {
	impPath := cc.info.ImportSpecPath(imp)
	nodePath := cc.info.PosFilePath(node.Pos())
	wantNotExp := nodePath == impPath
	if pkg, ok := cc.info.Pkgs[impPath]; ok {
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj != nil && (wantNotExp || obj.Exported()) {
				cc.objs = append(cc.objs, obj)
			}
		}
	}
}

func (cc *CC) getCandidatesInTypeSpec(node ast.Node, ts *ast.TypeSpec) {
	// get scope
	astFile := cc.info.PosAstFile(ts.Pos())
	scope, ok := cc.info.Info.Scopes[astFile]
	if !ok {
		return
	}

	// search obj in scope chain
	s2 := scope.Innermost(ts.Pos())
	if s2 == nil {
		return
	}
	_, obj := s2.LookupParent(ts.Name.Name, ts.Pos())
	if obj == nil {
		return
	}

	typ := obj.Type()

	wantNotExp := cc.wantNotExported(node, ts)

	switch t := typ.Underlying().(type) {
	case *types.Struct:
		s := t
		for i := 0; i < s.NumFields(); i++ {
			vobj := s.Field(i)
			if wantNotExp || vobj.Exported() {
				Logf("adding 2 %v", obj.Name())
				cc.objs = append(cc.objs, vobj)
			}
		}
	}

	// methods
	// using a pointer seems to include pointer and non-pointer methods, otherwise it includes only non-pointer or if iterating through both it can include duplicates.
	t := types.NewPointer(typ)
	mset := types.NewMethodSet(t)
	for i := 0; i < mset.Len(); i++ {
		obj := mset.At(i).Obj()
		if wantNotExp || obj.Exported() {
			Logf("adding 1 %v", obj.Name())
			cc.objs = append(cc.objs, obj)
		}
	}
}

func (cc *CC) filterCandidates(node ast.Node, index int) {
	switch t := node.(type) {
	case *ast.Ident:
		cc.filterIdCandidates(t, index)
	}
}
func (cc *CC) filterIdCandidates(id *ast.Ident, index int) {
	// get id string up to index
	indexStr := ""
	diff := index - cc.info.FSet.Position(id.Pos()).Offset
	if diff > 0 {
		indexStr = id.Name[:diff]
	}

	Logf("searching for %q (%q)", indexStr, id.Name)

	type entry struct {
		obj           types.Object
		indexStrIndex int
		//idNameIndex   int
	}

	var entries []entry
	indexStrLow := strings.ToLower(indexStr)
	//idNameLow := strings.ToLower(id.Name)
	for _, obj := range cc.objs {
		name := obj.Name()
		nameLow := strings.ToLower(name)
		i1 := strings.Index(nameLow, indexStrLow)
		if i1 >= 0 {
			//i2 := strings.Index(nameLow, idNameLow)
			//entries = append(entries, entry{obj, i1, i2})
			entries = append(entries, entry{obj, i1})
		}
	}

	sort.Slice(entries, func(a, b int) bool {
		ea, eb := entries[a], entries[b]

		//if ea.idNameIndex >= 0 {
		//	if eb.idNameIndex >= 0 {
		//		return ea.idNameIndex < eb.idNameIndex
		//	}
		//	return true
		//} else if eb.idNameIndex >= 0 {
		//	return false
		//}

		if ea.indexStrIndex == eb.indexStrIndex {
			na, nb := ea.obj.Name(), eb.obj.Name()
			return na < nb
		}

		return ea.indexStrIndex < eb.indexStrIndex
	})

	var objs []types.Object
	for _, e := range entries {
		objs = append(objs, e.obj)
	}
	cc.objs = objs
}

func (cc *CC) wantNotExported(n1, n2 ast.Node) bool {
	path1 := cc.info.PosFilePath(n1.Pos())
	path2 := cc.info.PosFilePath(n2.Pos())
	return path1 == path2
}

func FormatObjsDefault(objs []types.Object) string {
	var u []string
	for _, o := range objs {
		var buf bytes.Buffer
		buf.WriteString(o.String())
		u = append(u, buf.String())
	}
	return strings.Join(u, "\n")
}

func FormatObjs(objs []types.Object) string {
	var u []string
	for _, o := range objs {
		var buf bytes.Buffer
		ws := buf.WriteString

		switch t := o.(type) {
		case *types.Const:
			//ws("const ")
			ws(t.Name())
			//ws(t.Val().String())
		case *types.Var:
			ws(t.Name())
			//ws(": var")
			//ws(" ")
			//ws(t.Type().String())
		case *types.Builtin:
			//ws("builtin ")
			ws(t.Name())
			ws("(...)")
			//ws(t.Type().String())
		case *types.PkgName:
			//ws("package ")
			ws(t.Name())
			//ws("package ")
		case *types.Nil:
			//ws(t.Name())
			//ws(" <nil>")

		case *types.TypeName:
			//ws("type ")
			ws(t.Name())
			//switch t2 := t.Type().Underlying().(type) {
			//case *types.Interface:
			//	ws(" interface")
			//case *types.Struct:
			//	ws(" struct")
			//case *types.Slice:
			//	ws(" []")
			//	//ws(t2.Underlying().String())
			//case *types.Basic:
			//	//ws(t2.Name())
			//	//ws(" basic")
			//default:
			//	Logf("TODO 3")
			//	Dump(t2)
			//}

		case *types.Func:
			ws(t.Name())
			switch t2 := t.Type().(type) {
			case *types.Signature:
				ws("(")
				if tuple := t2.Params(); tuple != nil {
					if tuple.Len() > 0 {
						ws("...")
					}
					//w := []string{}
					//for i := 0; i < tuple.Len(); i++ {
					//	v := tuple.At(i)
					//	w = append(w, v.Name())
					//	w = append(w, " ")
					//	w = append(w, v.Type().String())
					//}
					//ws(strings.Join(w, ","))
				}
				ws(")")
			default:
				LogTODO(t2)
			}

		default:
			Logf("TODO 1")
			Dump(o)
			continue
		}

		u = append(u, buf.String())
	}
	return strings.Join(u, "\n")
}
