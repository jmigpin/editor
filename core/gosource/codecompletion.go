package gosource

import (
	"fmt"
	"go/ast"
)

func CodeCompletion(filename string, src interface{}, index int) error {
	info := NewInfo()

	// parse main file
	filename = info.AddPathFile(filename)
	astFile := info.ParseFile(filename, src)
	if astFile == nil {
		return fmt.Errorf("unable to parse file")
	}

	if Debug {
		info.PrintIdOffsets(astFile)
	}

	// index node
	tokenFile := info.FSet.File(astFile.Package)
	// avoid panic from a bad index
	if index > tokenFile.Size() {
		return fmt.Errorf("index bigger than file size")
	}
	indexNode := info.PosNode(tokenFile.Pos(index))

	// must be an id
	id, ok := indexNode.(*ast.Ident)
	if !ok {
		return fmt.Errorf("index not at an id node")
	}

	Logf("")
	Dump(id)

	//// new resolver for the path
	//path := info.PosFilePath(astFile.Package)
	//res := NewResolver(info, path)

	//// preemptively help the checker
	//res.ResolveCertainPathNodeTypesToHelpChecker(id)

	//// resolve id declaration
	//node := res.ResolveDecl(id)
	//if node == nil {
	//	return fmt.Errorf("id decl not found")
	//}

	//Logf("")
	//Dump(node)

	return nil
}
