package godebug

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
)

// useful for testing with types
func getTypesInfo(fset *token.FileSet, astFile *ast.File) (*types.Info, error) {
	conf := types.Config{
		//Importer: importer.Default(), // failing
		Importer: importer.ForCompiler(fset, "source", nil),

		//Sizes:    nil,
		//DisableUnusedImportCheck: true,
		//IgnoreFuncBodies: true,
		//AllowTypeAssertions:      true,

		Error: func(err error) {
			// DEBUG
			fmt.Printf("typesinfo: error: %v\n", err)
		},
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Instances:  map[*ast.Ident]types.Instance{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Implicits:  map[ast.Node]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
		InitOrder:  nil,
	}
	pkg, err := conf.Check("main", fset, []*ast.File{astFile}, info)
	if err != nil {
		return nil, err
	}
	_ = pkg
	return info, nil
}

//----------
//----------
//----------
