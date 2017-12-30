package gosource

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strings"
)

func CodeCompletion(filename string, src interface{}, index int) (int, string, [][2]int, bool) {
	cc := &CC{}
	err := cc.Run(filename, src, index)
	if err != nil {
		//log.Print(err)
		return 0, "", nil, false
	}
	return cc.index, cc.str, cc.segments, true
}

type CC struct {
	info     *Info
	objs     []types.Object
	segments [][2]int
	index    int
	str      string
}

func (cc *CC) Run(filename string, src interface{}, index int) error {
	//LogDebug()

	cc.info = NewInfo()
	cc.info.ParserMode = parser.ParseComments

	// parse main file
	filename = cc.info.AddPathFile(filename)
	astFile := cc.info.ParseFile(filename, src)
	if astFile == nil {
		return fmt.Errorf("unable to parse file")
	}

	if Debug {
		cc.info.PrintIdOffsets(astFile)
	}

	// translate index to a position
	tokenFile := cc.info.FSet.File(astFile.Package)
	if tokenFile == nil {
		return fmt.Errorf("unable to get token file")
	}
	ipos := cc.info.SafeOffsetPos(tokenFile, index)
	if !ipos.IsValid() {
		return fmt.Errorf("invalid index")
	}

	inode := cc.info.PosNode(ipos)

	// try to correct ipos/inode for certain positions
	previous := 0
	switch t := inode.(type) {
	case *ast.BlockStmt:
		previous = 1
	case *ast.SelectorExpr:
		// at "a|.b"
		if ipos == t.X.End() {
			previous = 1
		}
		// at "a.|_"
		if ipos == t.X.End()+1 {
			inode = t.Sel
		}
	case *ast.CallExpr:
		// at "(" or ")"
		if ipos == t.Lparen || ipos == t.Rparen {
			previous = 1
		}
		// in an arg separator ","
		for i, a := range t.Args {
			if i < len(t.Args) && ipos == a.End() {
				previous = 1
			}
		}
	}
	if previous > 0 {
		inode = cc.info.PosNode(ipos - token.Pos(previous))
		if inode == nil {
			return fmt.Errorf("index node not found")
		}
	}

	Logf("inode after previous (if any)")
	Dump(inode)

	// extract candidates from node according to type
	switch t := inode.(type) {
	case *ast.Ident:
		id := t
		str := ""
		diff := ipos - id.Pos()
		if diff > 0 {
			str = id.Name[:diff]
		}
		cc.candidatesInId(id)
		cc.filterCandidates(str)
	case *ast.CallExpr:
		if ipos > t.Lparen && ipos <= t.Rparen {
			cc.candidatesInScope(t)
		}
	default:
		LogTODO()
		Dump(t)
	}

	if len(cc.objs) == 0 {
		return fmt.Errorf("no objects found")
	}

	// format output
	//cc.str = FormatObjsDefaults(cc.objs)
	cc.str = FormatObjsSegs(cc.objs, cc.segments)

	// TODO: show comment if there is a match to the first func in sort
	// show comments if there is only one result
	if len(cc.objs) == 1 {
		obj := cc.objs[0]
		s, ok := cc.getComment(obj)
		if ok {
			cc.str += "\n\n" + s
		}
	}

	// index on where to "attach" the output
	inpos := cc.info.FSet.Position(inode.Pos())
	cc.index = inpos.Offset

	return nil
}

func (cc *CC) candidatesInId(id *ast.Ident) {
	if pn, ok := cc.info.NodeParent(id); ok {
		switch t2 := pn.(type) {
		case *ast.SelectorExpr:
			se := t2
			if id == se.Sel {
				cc.candidatesInNode(se.X)
				return
			}
		case *ast.FuncDecl:

			// TODO: complete inherited anon methods to overwrite

			return
		case *ast.CallExpr:
			// inside arguments
		default:
			LogTODO(t2)
		}
	}
	cc.candidatesInScope(id)
}

func (cc *CC) candidatesInScope(node ast.Node) {
	Logf("")

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
	for s := scope.Innermost(node.Pos()); s != nil; s = s.Parent() {
		for _, n := range s.Names() {
			if _, ok := m[n]; !ok {
				m[n] = true
				obj := s.Lookup(n)
				cc.objs = append(cc.objs, obj)
			}
		}
	}
}

func (cc *CC) candidatesInNode(node ast.Node) {
	Logf("")

	path := cc.info.PosFilePath(node.Pos())
	res := NewResolver(cc.info, path, node)
	n := res.ResolveType(node)

	Logf("%v", reflect.TypeOf(n))

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

func (cc *CC) filterCandidates(str string) {
	Logf("searching for %q", str)

	type entry struct {
		obj           types.Object
		indexStrIndex int
		segment       [2]int
	}

	var entries []entry
	indexStrLow := strings.ToLower(str)
	for _, obj := range cc.objs {
		name := obj.Name()
		nameLow := strings.ToLower(name)
		i1 := strings.Index(nameLow, indexStrLow)
		if i1 >= 0 {
			seg := [2]int{i1, i1 + len(indexStrLow)}
			entries = append(entries, entry{obj, i1, seg})
		}
	}

	sort.Slice(entries, func(a, b int) bool {
		ea, eb := entries[a], entries[b]
		if ea.indexStrIndex == eb.indexStrIndex {
			na, nb := ea.obj.Name(), eb.obj.Name()
			return na < nb
		}
		return ea.indexStrIndex < eb.indexStrIndex
	})

	var objs []types.Object
	var segs [][2]int
	for _, e := range entries {
		objs = append(objs, e.obj)
		segs = append(segs, e.segment)
	}
	cc.objs = objs
	cc.segments = segs
}

func (cc *CC) wantNotExported(n1, n2 ast.Node) bool {
	path1 := cc.info.PosFilePath(n1.Pos())
	path2 := cc.info.PosFilePath(n2.Pos())
	return path1 == path2
}

func (cc *CC) getComment(obj types.Object) (string, bool) {
	n := cc.info.PosNode(obj.Pos())
	if n == nil {
		return "", false
	}
	if n2, ok := cc.info.NodeParent(n); ok {
		n = n2
	} else {
		return "", false
	}
	s := ""
	switch t := n.(type) {
	case *ast.FuncDecl:
		s = t.Doc.Text()
	case *ast.TypeSpec:
		s = t.Doc.Text()
	case *ast.ImportSpec:
		s = t.Doc.Text()
	case *ast.ValueSpec:
		s = t.Doc.Text()
	default:
		LogTODO()
		Logf("%v", reflect.TypeOf(t))
		return "", false
	}
	s = strings.TrimRight(s, "\n")
	if len(s) == 0 {
		return "", false
	}
	return s, true
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

func FormatObjsSegs(objs []types.Object, segs [][2]int) string {
	var buf bytes.Buffer
	ws := buf.WriteString

	for i, o := range objs {
		if i > 0 {
			ws("\n")
		}

		wsseg := func(s string) {
			l := buf.Len()
			if segs != nil {
				seg := &segs[i]
				seg[0] += l
				seg[1] += l
			}
			ws(s)
		}

		switch t := o.(type) {
		case *types.Const:
			//ws("const ")
			wsseg(t.Name())
			//ws(t.Val().String())
		case *types.Var:
			wsseg(t.Name())
			//ws(": var")
			//ws(" ")
			//ws(t.Type().String())
		case *types.Builtin:
			//ws("builtin ")
			wsseg(t.Name())
			ws("(...)")
			//ws(t.Type().String())
		case *types.PkgName:
			//ws("package ")
			wsseg(t.Name())
			//ws("package ")
		case *types.Nil:
			wsseg(t.Name())
		case *types.TypeName:
			//ws("type ")
			wsseg(t.Name())
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
			wsseg(t.Name())
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
	}
	return buf.String()
}
