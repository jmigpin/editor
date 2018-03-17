package gosource

import (
	"fmt"
	"go/ast"
	"strconv"
)

type DeclResolver struct {
	conf *Config
}

func NewDeclResolver(conf *Config) *DeclResolver {
	res := &DeclResolver{conf: conf}
	return res
}

func (res *DeclResolver) ResolveDecl(node ast.Node) (ast.Node, error) {
	// node path
	path, _, ok := res.conf.PosAstPath(node.Pos())
	if !ok {
		return nil, fmt.Errorf("unable to find pos path")
	}

	// help checker by resolving path types
	res.tryToResolvePathTypes(path)

	return res.resolveDecl(node)
}

func (res *DeclResolver) resolveDecl(node ast.Node) (ast.Node, error) {
	switch t := node.(type) {

	//------------

	case *ast.FuncDecl:
		return t, nil

	//------------

	case *ast.TypeSpec:
		return t, nil
	case *ast.ValueSpec:
		return t, nil
	case *ast.ImportSpec:
		return t, nil

	//------------

	case *ast.AssignStmt:
		return t, nil

	//------------

	case *ast.SelectorExpr:
		//// help checker
		//if _, err := res.resolveType(t); err != nil {
		//	return nil, err
		//}

		return res.resolveDecl(t.Sel)

	//------------

	case *ast.Field:
		return t, nil

	//------------

	case *ast.BasicLit:
		path := res.conf.SurePosAstPath(t.Pos())
		switch path[1].(type) {
		case *ast.ImportSpec:
			return path[1], nil
		}
		return nil, fmt.Errorf("todo: basiclit")

	//------------

	case *ast.Ident:
		id := t
		// solved by the parser
		if id.Obj != nil {
			if n, ok := id.Obj.Decl.(ast.Node); ok {
				return res.resolveDecl(n)
			}
		}

		// solved in uses: obj.Pos() != id.Pos()
		if obj := res.conf.Info.Uses[id]; obj != nil {
			pos := obj.Pos()
			if !pos.IsValid() {
				// TODO: always builtin?
				return &Builtin{Name: id.Name}, nil
			}

			path := res.conf.SurePosAstPath(pos)

			// ex: type T1 struct{ fmt.●T2 }.
			// The resulting node of "pos" is "fmt". Need to solve SelectorExpr to solve T2.
			switch path[0].(type) {
			case *ast.Ident:
				switch t3 := path[1].(type) {
				case *ast.SelectorExpr:
					return t3.Sel, nil
					//return res.resolveDecl(t3)
				}
			}

			return res.resolveDecl(path[0])
			//return nil, fmt.Errorf("ident decl uses: %v", id.Name)
		}

		// solved in defs: obj.Pos() == id.Pos()
		if obj := res.conf.Info.Defs[id]; obj != nil {
			path := res.conf.SurePosAstPath(obj.Pos())
			// path[0] is id itself, solve parent node
			return res.resolveDecl(path[1])
		}

		return nil, fmt.Errorf("unable to resolve ident decl: %v", id.Name)

	default:
	}
	return nil, fmt.Errorf("todo: resolve decl: %T", node)
}

func (res *DeclResolver) ResolveType(node ast.Node) (ast.Node, error) {
	// node path
	path, _, ok := res.conf.PosAstPath(node.Pos())
	if !ok {
		return nil, fmt.Errorf("unable to find pos path")
	}

	// help checker by resolving path types
	res.tryToResolvePathTypes(path)

	return res.resolveType(node)
}

func (res *DeclResolver) resolveType(node ast.Node) (ast.Node, error) {
	switch t := node.(type) {

	//------------

	case *Builtin:
		return t, nil

	//------------

	case *ast.File:
		return t, nil

	//------------

	case *ast.GenDecl:
		// can have multiple specs, have to return itself
		return t, nil

	case *ast.FuncDecl:
		return res.resolveType(t.Type)

	//------------

	case *ast.TypeSpec:
		// help the checker
		if _, err := res.resolveType(t.Type); err != nil {
			return nil, err
		}

		return t, nil

	case *ast.ValueSpec:
		if t.Type == nil {
			// ex: "var a,b = 1,1.0"
			return t, nil
		}
		return res.resolveType(t.Type)

	case *ast.ImportSpec:
		// help the checker
		// make path importable
		path, _ := strconv.Unquote(t.Path.Value)
		if !res.conf.IsImportable(path) {
			res.conf.MakeImportable(path)
			_ = res.conf.ReImportImportables()
		}

		return t, nil

	//------------

	case *ast.FuncType:
		// help the checker
		if t.Results != nil {
			if _, err := res.resolveType(t.Results); err != nil {
				return nil, err
			}
		}

		return t, nil

	case *ast.StructType:
		// help the checker
		if err := res.resolveAnonFieldsTypes(t.Fields); err != nil {
			return nil, err
		}

		return t, nil

	case *ast.InterfaceType:
		// help the checker
		if err := res.resolveAnonFieldsTypes(t.Methods); err != nil {
			return nil, err
		}

		return t, nil

	case *ast.MapType:
		// help the checker
		//if _, err := res.resolveType(t.Key); err != nil {
		//return nil, err
		//}
		// help the checker
		if _, err := res.resolveType(t.Value); err != nil {
			return nil, err
		}

		return t, nil

	case *ast.ArrayType:
		return res.resolveType(t.Elt)

	//------------

	case *ast.DeclStmt:
		return res.resolveType(t.Decl)
	case *ast.ReturnStmt:
		return t, nil
	case *ast.BlockStmt:
		return t, nil
	case *ast.ExprStmt:
		return res.resolveType(t.X)
	case *ast.TypeSwitchStmt:
		return res.resolveType(t.Assign)
	case *ast.IfStmt:
		return t, nil
	case *ast.RangeStmt:
		// help the checker
		if _, err := res.resolveType(t.X); err != nil {
			return nil, err
		}
		//if _, err := res.resolveType(t.Key); err != nil {
		//	return nil, err
		//}
		//if _, err := res.resolveType(t.Value); err != nil {
		//	return nil, err
		//}

		return t, nil

	case *ast.AssignStmt:
		// help the checker
		for _, e := range t.Rhs {
			if _, err := res.resolveType(e); err != nil {
				return nil, err
			}
		}

		return t, nil

	//------------

	case *ast.CallExpr:
		return res.resolveType(t.Fun)
	case *ast.StarExpr:
		return res.resolveType(t.X)
	case *ast.UnaryExpr:
		return res.resolveType(t.X)
	case *ast.IndexExpr:
		return res.resolveType(t.X)
	case *ast.BinaryExpr:
		return t, nil

	case *ast.SelectorExpr:
		// help the checker
		_, err := res.resolveType(t.X)
		if err != nil {
			return nil, err
		}

		// solved at selections
		sel, ok := res.conf.Info.Selections[t]
		if ok {
			path := res.conf.SurePosAstPath(sel.Obj().Pos())

			// TODO:
			switch path[0].(type) {
			}

			return res.resolveType(path[0])
		}

		return res.resolveType(t.Sel)

	case *ast.TypeAssertExpr:
		// help the checker
		if _, err := res.resolveType(t.X); err != nil {
			return nil, err
		}

		if t.Type == nil {
			// ex: "switch x.(type)"
			return t, nil
		}
		return res.resolveType(t.Type)

	//------------

	case *ast.Ident:
		id := t

		// decl node
		dn, err := res.resolveDecl(id)
		if err != nil {
			return nil, err
		}
		if dn == id {
			return dn, nil
		}

		// decl node type
		n2, err := res.resolveType(dn)
		if err != nil {
			return nil, err
		}

		// TODO: need test in declposition_test for this, only tested on code completion
		// special cases that need to match the id
		switch t2 := n2.(type) {
		case *ast.AssignStmt:
			as := t2
			// lhs index
			lhsi := -1
			for i, e := range as.Lhs {
				if id2, ok := e.(*ast.Ident); ok && id2.Name == id.Name {
					lhsi = i
					break
				}
			}
			if lhsi < 0 {
				return nil, fmt.Errorf("unable to resolve ident type assign lhs index")
			}
			// rhs node
			var rhs ast.Expr
			if len(as.Rhs) == 1 {
				rhs = as.Rhs[0]
			} else {
				rhs = as.Rhs[lhsi]
			}
			// rhs type
			n3, err := res.resolveType(rhs)
			if err != nil {
				return nil, err
			}
			// rhs result
			switch t3 := n3.(type) {
			case *ast.FuncType:
				if t3.Results != nil && lhsi < len(t3.Results.List) {
					return res.resolveType(t3.Results.List[lhsi])
				}
			}
			return n3, nil
		}

		return n2, nil

	//------------

	case *ast.FieldList:
		// help the checker
		for _, f := range t.List {
			if _, err := res.resolveType(f); err != nil {
				return nil, err
			}
		}

		return t, nil

	case *ast.Field:
		return res.resolveType(t.Type)

	//------------

	case *ast.CaseClause:
		// help the checker
		for _, e := range t.List {
			if _, err := res.resolveType(e); err != nil {
				return nil, err
			}
		}

		return t, nil

	//------------

	case *ast.BasicLit:
		path := res.conf.SurePosAstPath(t.Pos())
		return res.resolveDecl(path[1])
	case *ast.CompositeLit:
		if t.Type == nil {
			return t, nil
		}
		return res.resolveType(t.Type)

	//------------

	default:
	}

	return nil, fmt.Errorf("todo: resolve type: %T", node)
}

//func (res *DeclResolver) resolveParentType(node ast.Node) (ast.Node, error) {
//	path, _, ok := res.conf.PosAstPath(node.Pos())
//	if !ok || len(path) < 2 {
//		return nil, fmt.Errorf("unable to get parent")
//	}
//	return res.resolveType(path[1])
//}

func (res *DeclResolver) resolveAnonFieldsTypes(fl *ast.FieldList) error {
	for _, f := range fl.List {
		if f.Names == nil {
			_, err := res.resolveType(f)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (res *DeclResolver) tryToResolvePathTypes(path []ast.Node) {
	// DEBUG
	//res.resolvePathTypes(path)

	// resolve top to bottom
	for i := len(path) - 1; i >= 0; i-- {
		_, _ = res.resolveType(path[i])
	}
}

func (res *DeclResolver) resolvePathTypes(path []ast.Node) error {
	// resolve top to bottom
	for i := len(path) - 1; i >= 0; i-- {
		_, err := res.resolveType(path[i])
		if err != nil {
			return err
		}
	}
	return nil
}
