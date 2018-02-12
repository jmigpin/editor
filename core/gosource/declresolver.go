package gosource

import (
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
)

//var noDeclNode = fmt.Errorf("node has no declaration")

type DeclResolver struct {
	conf *Config
}

func NewDeclResolver(conf *Config) *DeclResolver {
	res := &DeclResolver{conf: conf}
	return res
}

func (res *DeclResolver) tryToResolvePathTypes(path []ast.Node) {
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
	case *Builtin:
		return t, nil

	case *ast.FuncDecl:
		return res.resolveType(t.Type)
	case *ast.ExprStmt:
		return res.resolveType(t.X)
	case *ast.CallExpr:
		return res.resolveType(t.Fun)
	case *ast.Field:
		return res.resolveType(t.Type)
	case *ast.StarExpr:
		return res.resolveType(t.X)
	case *ast.UnaryExpr:
		return res.resolveType(t.X)
	case *ast.IndexExpr:
		return res.resolveType(t.X)
	case *ast.MapType:
		return res.resolveType(t.Value)
	case *ast.ArrayType:
		return res.resolveType(t.Elt)

	case *ast.CompositeLit:
		if t.Type != nil {
			return res.resolveType(t.Type)
		}
		return nil, fmt.Errorf("todo: composite literal type nil")

	case *ast.FuncType:
		// help the checker
		if t.Results != nil {
			for _, f := range t.Results.List {
				if _, err := res.resolveType(f); err != nil {
					return nil, err
				}
			}
		}

		return t, nil

	case *ast.AssignStmt:
		// help the checker, discard result
		for _, e := range t.Rhs {
			_, err := res.resolveType(e)
			if err != nil {
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

	case *ast.SelectorExpr:
		// help the checker
		n, err := res.resolveType(t.X)
		if err != nil {
			return nil, err
		}

		// help the checker
		switch t2 := n.(type) {
		//case *ast.BasicLit:
		//	// ex: "go/ast" in `import "go/ast"`
		//	if _, err := res.resolveParentType(t2); err != nil {
		//		return nil, err
		//	}
		case *ast.FuncType:
			// function call to a func that has 1 return value
			if t2.Results != nil && len(t2.Results.List) == 1 {
				_, err := res.resolveType(t2.Results.List[0])
				if err != nil {
					return nil, err
				}
			}
		}

		// solved at selections
		sel, ok := res.conf.Info.Selections[t]
		if ok {
			path := res.conf.SurePosAstPath(sel.Obj().Pos())
			return res.resolveType(path[0])
		}

		return res.resolveType(t.Sel)

	case *ast.Ident:
		id := t

		// decl node
		dn, err := res.resolveDecl(id)
		if err != nil {
			return nil, err
		}

		// avoid loop lock
		if dn == id {
			// ex: ttt in `import ttt "go/types"`

			// help checker
			_, err := res.resolveParentType(dn)
			if err != nil {
				return nil, err
			}

			return dn, nil
		}

		// decl node type
		n2, err := res.resolveType(dn)
		if err != nil {
			return nil, err
		}

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

			// special case for boolean value of typeassertexpr
			// TODO: map and range
			switch rhs.(type) {
			case *ast.TypeAssertExpr:
				if lhsi == 1 {
					return &Builtin{Name: "bool"}, nil
				}
			}

			n3, err := res.resolveType(rhs)
			if err != nil {
				return nil, err
			}
			switch t3 := n3.(type) {
			case *ast.FuncType:
				if t3.Results != nil && lhsi < len(t3.Results.List) {
					return res.resolveType(t3.Results.List[lhsi])
				}
			}
			return n3, nil
		}
		return n2, nil

	case *ast.ImportSpec:
		imp := t
		// make path importable
		path, _ := strconv.Unquote(imp.Path.Value)
		if !res.conf.IsImportable(path) {
			res.conf.MakeImportable(path)
			_ = res.conf.ReImportImportables()
		}
		return t, nil

	case *ast.BasicLit:
		// just resolving the parent leads to loop lock

		// parent
		path, _, ok := res.conf.PosAstPath(t.Pos())
		if ok && len(path) > 1 {
			// ImportSpec parent
			// ex: "go/ast" in `import "go/ast"`
			if is, ok := path[1].(*ast.ImportSpec); ok {
				return res.resolveType(is)
			}
		}

		return t, nil

	case *ast.TypeAssertExpr:
		if t.Type == nil {
			// ex: "switch x.(type)"
			//return t, nil
			return res.resolveType(t.X) // TODO: review
		} else {
			// help the checker
			_, err := res.resolveType(t.X)
			if err != nil {
				return nil, err
			}

			return res.resolveType(t.Type)
		}

	case *ast.TypeSwitchStmt:
		// help the checker
		if _, err := res.resolveType(t.Assign); err != nil {
			return nil, err
		}

		return t, nil

	case *ast.CaseClause:
		// help the checker
		for _, e := range t.List {
			if _, err := res.resolveType(e); err != nil {
				return nil, err
			}
		}

		return t, nil

	case *ast.TypeSpec:
		//switch t2 := t.Type.(type) {
		//case *ast.Ident:
		//	// ex: "type A B"
		//	return res.ResolveType(t2)
		//}
		//return t, nil
		return res.resolveType(t.Type)

	case *ast.ValueSpec:
		if t.Type == nil {
			// ex: "var a = 1"
			return t, nil
		}
		return res.resolveType(t.Type)

	case *ast.Ellipsis:
		return res.resolveType(t.Elt)

	default:
		//log.Printf("todo %v %v", reflect.TypeOf(node), node)
		return t, nil
	}
	//return nil, fmt.Errorf("todo")
}

func (res *DeclResolver) resolveParentType(node ast.Node) (ast.Node, error) {
	path, _, ok := res.conf.PosAstPath(node.Pos())
	if !ok || len(path) < 2 {
		return nil, fmt.Errorf("unable to get parent")
	}
	return res.resolveType(path[1])
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
	case *ast.Ident:
		id := t
		// solved by the parser
		if id.Obj != nil {
			if n, ok := id.Obj.Decl.(ast.Node); ok {
				return n, nil
			}
		}
		// solved in uses
		obj := res.conf.Info.Uses[id]
		if obj != nil {
			pos := obj.Pos()
			if !pos.IsValid() {
				// possible builtin
				return &Builtin{Name: id.Name}, nil
			}
			// node
			path := res.conf.SurePosAstPath(pos)
			return path[0], nil
		}
		// solved in defs
		obj = res.conf.Info.Defs[id]
		if obj != nil {
			return id, nil
		}
		return nil, fmt.Errorf("unable to resolve ident decl: %v", id.Name)
	default:
	}
	return nil, fmt.Errorf("todo: resole decl for %v", reflect.TypeOf(node))
}

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
