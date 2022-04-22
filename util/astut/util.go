package astut

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
)

func NodeFilename(fset *token.FileSet, node ast.Node) (string, error) {
	f := fset.File(node.Pos())
	if f == nil {
		return "", fmt.Errorf("not found")
	}
	return f.Name(), nil
}

//----------

// print ast notes
// TODO: without tabwidth set, it won't output the source correctly
// Fail: has struct fields without spaces "field int"->"fieldint"
//cfg := &printer.Config{Mode: printer.SourcePos | printer.TabIndent}
// Fail: has stmts split with comments in the middle
//cfg := &printer.Config{Mode: printer.SourcePos | printer.TabIndent | printer.UseSpaces}
// debug
//cfg := &printer.Config{Tabwidth: 4}
//cfg := &printer.Config{Mode: printer.SourcePos, Tabwidth: 4}

func PrintNode(fset *token.FileSet, n ast.Node) {
	fmt.Println(SprintNode(fset, n))
}
func SprintNode(fset *token.FileSet, n ast.Node) string {
	s, err := SprintNode2(fset, n)
	if err != nil {
		return fmt.Sprintf("<sprintnodeerr:%v>", err)
	}
	return s
}
func SprintNode2(fset *token.FileSet, n ast.Node) (string, error) {
	buf := &bytes.Buffer{}
	cfg := &printer.Config{Mode: printer.RawFormat}
	if err := cfg.Fprint(buf, fset, n); err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}
