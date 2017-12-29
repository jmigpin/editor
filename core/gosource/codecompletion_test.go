package gosource

import (
	"fmt"
	"testing"
)

func ccTest(t *testing.T, filename string, src interface{}, index int) {
	t.Helper()

	//LogDebug()

	_, str, err := CodeCompletion(filename, src, index)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("**result**\n")
	fmt.Printf("%v\n", str)
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
	//ccTestSrc(t, src, 68) // fmt.|Prin: begginning of "Prin" (show all in list)
	//ccTestSrc(t, src, 69) // fmt.P|rin:
	ccTestSrc(t, src, 70) // fmt.Pr|in:
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
	ccTestSrc(t, src, 90) // u.|Pos
	//ccTestSrc(t, src, 91) // u.P|os
}

func TestCC3(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"		
		)
		func func1() {
			var u *ast.Ident
			_=u.Pos().IsValid()
		}
	`
	ccTestSrc(t, src, 98) // Is|Valid
}

func TestCC4(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"		
			"go/types"		
		)
		func func1() {
			var aa *ast.Ident
			var aab *types.Func
			_ = aa
			_ = aab
		}
	`
	ccTestSrc(t, src, 141) // a|ab
}
