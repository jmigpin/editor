package gosource

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strconv"
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
	info.Importable[res.mainPath] = true

	// first confcheck without extra imports
	res.info.ConfCheckPath(res.mainPath, "", 0)

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
		if n := res.GetIdDecl(t); n != nil {
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
			if n := res.GetIdDecl(t.Sel); n != nil {
				return n
			}
		}
	default:
		_ = t
		Logf("TODO")
		Dump(node)
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
						case *ast.StructType:
							return t3
						case *ast.InterfaceType:
							return t3
						case *ast.FuncType:
							if t3.Results != nil && lhsi < len(t3.Results.List) {
								return res.ResolveType(t3.Results.List[lhsi])
							}
						default:
							Logf("TODO id AssignStmt")
							Dump(lhsi)
							Dump(n)
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
		res.makeImportSpecImportableAndConfCheck(t)
		if res.importSpecImported(t) {
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
		return res.ResolveType(t.Type)
	case *ast.StructType:
		res.resolveAnonFieldsTypes(t.Fields)
		return t
	case *ast.InterfaceType:
		res.resolveAnonFieldsTypes(t.Methods)
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
		_ = t
		Logf("TODO")
		Dump(node)
	}

	Logf("not solved (%v)", reflect.TypeOf(node))
	return nil
}

func (res *Resolver) GetIdDecl(id *ast.Ident) ast.Node {
	Logf("%v", id)

	// solved by the parser
	if id.Obj != nil {
		if n, ok := id.Obj.Decl.(ast.Node); ok {
			Logf("in parser")
			return n
		}
		Logf("TODO 1")
		Dump(id.Obj)
	}

	// solved in info.uses
	obj := res.info.Info.Uses[id]
	if obj != nil {
		pos := obj.Pos()
		if pos != token.NoPos {
			Logf("in uses")
			return res.info.PosNode(pos)
		}
		// builtin package
		if pos == token.NoPos {
			b := "builtin"
			res.info.Importable[b] = true
			pkg, _ := res.info.PackageImporter(b, "", 0)
			obj2 := pkg.Scope().Lookup(id.Name)
			if obj2 != nil {
				return res.info.PosNode(obj2.Pos())
			}
		}
	}

	// solved in info.defs
	obj = res.info.Info.Defs[id]
	if obj != nil {
		Logf("in defs")
		return id
	}

	// can't use: not correct for some cases
	//// search in scopes
	//astFile := res.info.PosAstFile(id.Pos())
	//s1, ok := res.info.Info.Scopes[astFile]
	//if ok {
	//	s := s1.Innermost(id.Pos())
	//	if s != nil {
	//		_, obj := s.LookupParent(id.Name, id.Pos())
	//		if obj != nil {
	//			Logf("in scopes")
	//			return res.info.PosNode(obj.Pos())
	//		}
	//	}
	//}

	Logf("not found")
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
				Logf("TODO 1")
				Dump(n2)
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

func (res *Resolver) makeImportSpecImportableAndConfCheck(imp *ast.ImportSpec) {
	path := res.importSpecPath(imp)
	if _, ok := res.info.Importable[path]; !ok {
		Logf("%v", imp.Path)
		// make path importable
		res.info.Importable[path] = true
		// reset imported paths to clear cached pkgs
		res.info.Pkgs = make(map[string]*types.Package)
		// check main path that will now re-import available importables
		_, _ = res.info.ConfCheckPath(res.mainPath, "", 0)
	}
}
func (res *Resolver) importSpecImported(imp *ast.ImportSpec) bool {
	path := res.importSpecPath(imp)
	return res.info.Pkgs[path] != nil
}
func (res *Resolver) importSpecPath(imp *ast.ImportSpec) string {
	path, _ := strconv.Unquote(imp.Path.Value)
	return path
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
