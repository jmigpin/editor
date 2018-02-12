package gosource

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

func DeclPosition(filename string, src interface{}, index int) (*token.Position, *token.Position, error) {
	conf := NewConfig()
	node, err := declPosition2(conf, filename, src, index)
	if err != nil {
		return nil, nil, err
	}
	// position
	posp := conf.FSet.Position(node.Pos())
	endp := conf.FSet.Position(node.End())
	return &posp, &endp, nil
}

func declPosition2(conf *Config, filename string, src interface{}, index int) (ast.Node, error) {
	astFile, err, ok := conf.ParseFile(filename, src, 0)
	if !ok {
		return nil, err
	}

	//// DEBUG: find positions for tests
	//ast.Inspect(astFile, func(node ast.Node) bool {
	//	if node == nil {
	//		return false
	//	}
	//	p := conf.FSet.Position(node.Pos())
	//	log.Printf("%v %v %v", reflect.TypeOf(node), node, p.Offset)
	//	return true
	//})

	// make package path importable and re-import (type check added astfile)
	conf.MakeFilePkgImportable(filename)
	_ = conf.ReImportImportables()

	// index token (need parsed file position)
	tf, err := conf.PosTokenFile(astFile.Package)
	if tf == nil {
		return nil, err
	}
	start := token.Pos(tf.Base() + index)

	// path to index in astFile
	path, exact := astutil.PathEnclosingInterval(astFile, start, start)

	// solve only for idents
	if len(path) == 0 || !exact {
		return nil, fmt.Errorf("index has no node")
	}
	id, ok := path[0].(*ast.Ident)
	if !ok {
		return nil, fmt.Errorf("node is not an ident")
	}

	res := NewDeclResolver(conf)

	n, err := res.ResolveDecl(id)
	if err != nil {
		return nil, err
	}

	// improve position
	switch t := n.(type) {
	case *Builtin:
		return conf.BuiltinLookup(t.Name)
	case *ast.FuncDecl:
		return t.Name, nil
	case *ast.TypeSpec:
		return t.Name, nil
	case *ast.AssignStmt:
		for _, e := range t.Lhs {
			if id2, ok := e.(*ast.Ident); ok && id2.Name == id.Name {
				return id2, nil
			}
		}
	case *ast.Field:
		for _, id2 := range t.Names {
			if id2.Name == id.Name {
				return id2, nil
			}
		}
	case *ast.ValueSpec:
		for _, id2 := range t.Names {
			if id2.Name == id.Name {
				return id2, nil
			}
		}
	}

	return n, nil
}
