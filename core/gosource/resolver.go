package gosource

import (
	"go/ast"
	"reflect"
)

type Resolver struct {
	info     *Info
	visited1 map[ast.Node]bool
	visited2 map[ast.Node]bool
	mainPath string
}

func NewResolver(info *Info, mainPath string, checkerNode ast.Node) *Resolver {
	res := &Resolver{
		info:     info,
		visited1: make(map[ast.Node]bool),
		visited2: make(map[ast.Node]bool),
		mainPath: mainPath,
	}

	// make path importable (imports other files on the same path)
	info.Importable[res.mainPath] = struct{}{}

	// first confcheck without extra imports
	res.info.ConfCheckPath(res.mainPath)

	// preemptively help the checker
	if checkerNode != nil {
		res.resolveCertainPathNodeTypesToHelpChecker(checkerNode)
	}

	return res
}

func (res *Resolver) ResolveDecl(node ast.Node) ast.Node {
	Logf("%v %v", reflect.TypeOf(node), node)
	LogDepth++
	defer func() { LogDepth-- }()

	if res.visited1[node] {
		return nil
	}
	res.visited1[node] = true
	defer func() { res.visited1[node] = false }()

	switch t := node.(type) {
	case *ast.Ident:
		if n := res.info.GetIdDecl(t); n != nil {
			return n
		}
		if pn, ok := res.info.NodeParent(t); ok {
			if n := res.ResolveDecl(pn); n != nil {
				return n
			}
		}
	case *ast.SelectorExpr:
		if n := res.ResolveType(t.X); n != nil {
			switch t2 := n.(type) {
			case *ast.FuncType:
				if t2.Results != nil && len(t2.Results.List) >= 1 {
					_ = res.ResolveType(t2.Results.List[0])
				}
			}
			if n := res.info.GetIdDecl(t.Sel); n != nil {
				return n
			}
		}
	default:
		LogTODO(node)
	}

	Logf("not solved (%v)", reflect.TypeOf(node))
	return nil
}

func (res *Resolver) ResolveType(node ast.Node) ast.Node {
	Logf("%v %v", reflect.TypeOf(node), node)
	LogDepth++
	defer func() { LogDepth-- }()

	if res.visited2[node] {
		return nil
	}
	res.visited2[node] = true
	defer func() { res.visited2[node] = false }()

	switch t := node.(type) {
	case *ast.Ident:
		var node2 ast.Node
		if n := res.ResolveDecl(t); n != nil {
			if n == t {
				if pn, ok := res.info.NodeParent(t); ok {
					node2 = res.ResolveType(pn)
				}
			} else {
				node2 = res.ResolveType(n)
			}
		}
		if node2 != nil {
			switch t2 := node2.(type) {
			case *ast.AssignStmt:
				id := t
				as := t2
				lhsi, rhsn := res.IdAssignStmtRhs(id, as)
				if rhsn != nil && lhsi >= 0 {
					if n := res.ResolveType(rhsn); n != nil {
						switch t3 := n.(type) {
						case *ast.FuncType:
							if t3.Results != nil && lhsi < len(t3.Results.List) {
								return res.ResolveType(t3.Results.List[lhsi])
							}
						default:
							//LogTODO(lhsi, t3)
							return n
						}
					}
				}
			default:
				return node2
			}
		}
	case *ast.BasicLit:
		if pn, ok := res.info.NodeParent(t); ok {
			return res.ResolveType(pn)
		}
	case *ast.ImportSpec:
		res.info.MakeImportSpecImportableAndConfCheck(t)
		if res.info.ImportSpecImported(t) {
			return t
		}
	case *ast.SelectorExpr:
		if n := res.getSelectorExprType(t); n != nil {
			return n
		}
		if n := res.ResolveType(t.X); n != nil {
			switch t2 := n.(type) {
			case *ast.FuncType:
				if t2.Results != nil && len(t2.Results.List) >= 1 {
					_ = res.ResolveType(t2.Results.List[0])
				}
			}
			if n := res.getSelectorExprType(t); n != nil {
				return n
			}
			if n := res.ResolveType(t.Sel); n != nil {
				return n
			}
		}
	case *ast.Field:
		return res.ResolveType(t.Type)
	case *ast.TypeSpec:
		switch t2 := t.Type.(type) {
		case *ast.StructType:
			res.resolveAnonFieldsTypes(t2.Fields)
		case *ast.InterfaceType:
			res.resolveAnonFieldsTypes(t2.Methods)
		case *ast.Ident:
			// Ex: "type A B", resolving B here, but returning it
			_ = res.ResolveType(t2)
		default:
			LogTODO(t2)
		}
		return t
	case *ast.ValueSpec:
		if t.Type == nil {
			// ex: "var a = 1"
			if pn, ok := res.info.NodeParent(t); ok {
				return res.ResolveType(pn)
			}
		} else {
			return res.ResolveType(t.Type)
		}
	case *ast.StarExpr:
		return res.ResolveType(t.X)
	case *ast.AssignStmt:
		return t
	case *ast.TypeAssertExpr:
		// preemptively solve to help the checker
		_ = res.ResolveType(t.X)

		if t.Type == nil {
			// ex: "switch x.(type)"
			return t
		} else {
			return res.ResolveType(t.Type)
		}
	case *ast.CallExpr:
		return res.ResolveType(t.Fun)
	case *ast.FuncType:
		return t
	case *ast.FuncDecl:
		return res.ResolveType(t.Type)
	case *ast.IndexExpr:
		return res.ResolveType(t.X)
	case *ast.MapType:
		return res.ResolveType(t.Value)
	case *ast.UnaryExpr:
		return res.ResolveType(t.X)
	case *ast.CompositeLit:
		if t.Type != nil {
			return res.ResolveType(t.Type)
		}
	case *ast.ArrayType:
		return res.ResolveType(t.Elt)
	default:
		LogTODO(node)
	}

	Logf("not solved (%v)", reflect.TypeOf(node))
	return nil
}

func (res *Resolver) resolveCertainPathNodeTypesToHelpChecker(node ast.Node) {
	// in some cases the CaseClause is not present in res.info.scopes

	path := res.info.NodePath(node)

	Logf("")
	if Debug {
		res.info.PrintPath(path)
	}

	for _, n := range path {
		switch t := n.(type) {
		case *ast.TypeSwitchStmt:
			var n2 ast.Node = t.Assign
		L1:
			switch t2 := n2.(type) {
			case *ast.AssignStmt:
				if len(t2.Rhs) >= 1 {
					n2 = t2.Rhs[0]
					goto L1
				}
			case *ast.TypeAssertExpr:
				// TODO: only need to solve t2.X?
				Logf("typeswitchstmtexpr %v %v", node, t2)
				_ = res.ResolveType(t2)
			default:
				LogTODO(n2)
			}

		case *ast.TypeAssertExpr:
			Logf("typeassertexpr %v %v", node, t)
			_ = res.ResolveType(t)
		case *ast.CaseClause:
			for _, e := range t.List {
				Logf("caseclause %v %v", node, e)
				_ = res.ResolveType(e)
			}
		case *ast.AssignStmt:
			// left side of AssignStmt, need to solve right side
			if node.Pos() < t.TokPos {
				for _, e := range t.Rhs {
					Logf("leftsideof assignstmt %v %v", node, e)
					_ = res.ResolveType(e)
				}
			}
		}
	}
}

func (res *Resolver) getSelectorExprType(se *ast.SelectorExpr) ast.Node {
	Logf("%v", se)
	// solved by the checker
	sel, ok := res.info.Info.Selections[se]
	if ok {
		n := res.info.PosNode(sel.Obj().Pos())
		return res.ResolveType(n)
	}
	Logf("not found")
	return nil
}

func (res *Resolver) IdAssignStmtRhs(id *ast.Ident, as *ast.AssignStmt) (int, ast.Node) {
	Logf("%v %v", id, as)
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

func (res *Resolver) resolveAnonFieldsTypes(fl *ast.FieldList) {
	for _, f := range fl.List {
		if f.Names == nil {
			Logf("anon field")
			_ = res.ResolveType(f)
		}
	}
}
