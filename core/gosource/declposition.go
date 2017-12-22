package gosource

import (
	"fmt"
	"go/ast"
	"go/token"
)

// Find index declaration position.
func DeclPosition(filename string, src interface{}, index int) (*token.Position, *token.Position, error) {
	info := NewInfo()

	// parse main file
	filename = info.AddPathFile(filename)
	astFile := info.ParseFile(filename, src)
	if astFile == nil {
		return nil, nil, fmt.Errorf("unable to parse file")
	}

	if Debug {
		info.PrintIdOffsets(astFile)
	}

	// index node
	tokenFile := info.FSet.File(astFile.Package)
	// avoid panic from a bad index
	if index > tokenFile.Size() {
		return nil, nil, fmt.Errorf("index bigger than file size")
	}
	indexNode := info.PosNode(tokenFile.Pos(index))

	// must be an id
	id, ok := indexNode.(*ast.Ident)
	if !ok {
		return nil, nil, fmt.Errorf("index not at an id node")
	}

	// new resolver for the path
	path := info.PosFilePath(astFile.Package)
	res := NewResolver(info, path)

	// preemptively help the checker
	res.ResolveCertainPathNodeTypesToHelpChecker(id)

	// resolve id declaration
	node := res.ResolveDecl(id)
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
		lhsi, _ := res.IdAssignStmtRhs(id, t)
		if lhsi >= 0 {
			node = t.Lhs[lhsi]
		}
	case *ast.Field:
		for _, id2 := range t.Names {
			if id2.Name == id.Name {
				node = id2
				break
			}
		}
	case *ast.ValueSpec:
		for _, id2 := range t.Names {
			if id2.Name == id.Name {
				node = id2
				break
			}
		}
	default:
		//Logf("TODO")
		//Dump(node)
	}

	// node position
	posp := info.FSet.Position(node.Pos())
	endp := info.FSet.Position(node.End())
	Logf("***result: offset=%v %v", posp.Offset, posp)
	return &posp, &endp, nil
}
