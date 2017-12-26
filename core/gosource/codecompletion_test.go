package gosource

import (
	"fmt"
	"testing"
)

func ccTest(t *testing.T, filename string, src interface{}, index int) {
	t.Helper()

	LogDebug()

	_, objs, err := CodeCompletion(filename, src, index)
	if err != nil {
		t.Fatal(err)
	}

	Logf("---")
	fmt.Printf("%v\n", FormatObjs(objs))
	Logf("---")
}

func ccTestSrc(t *testing.T, src interface{}, index int) {
	filename := "t000/src.go"
	ccTest(t, filename, src, index)
}

func TestCC1(t *testing.T) {
	src := ` 
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fmt.Prin
		}
	`
	ccTestSrc(t, src, 68) // fmt.|Prin: begginning of "Prin" (show all in list)
}

func TestCC2(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"		
		)
		func func1() {
			var u *ast.Ident
			_=u.Pos()
		}
	`
	//ccTestSrc(t, src, 90) // u.|Pos
	ccTestSrc(t, src, 91) // u.P|os
}
