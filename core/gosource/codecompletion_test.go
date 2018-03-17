package gosource

import (
	"testing"
)

func testCCSrc(t *testing.T, src string, n, nres int) {
	t.Helper()

	src2, index, err := SourceCursor("●", src, n)
	if err != nil {
		t.Fatal(err)
	}

	filename := "t000/src.go"
	res, err := CodeCompletion(filename, src2, index)
	if err != nil {
		t.Fatal(err)
	}

	//for _, obj := range res.Objs {
	//	t.Logf("obj: %v\n", obj)
	//}

	t.Logf("result n objects: %v (exp=%v)", len(res.Objs), nres)
	t.Logf("str:\n%v", res.Str)

	if len(res.Objs) != nres {
		t.Fatalf("expecting %d results: got %v", nres, len(res.Objs))
	}

}

//------------

func TestCC1(t *testing.T) {
	src := `
		package pack1
		import "fmt"
		import "go/ast"
		func f1() {
			fmb:=2
			fmta:=1
			fmt●.●Print●(●)●
			var u ast.Ident
			u.●Name
		}
	`
	testCCSrc(t, src, 0, 2)  // fmt|. -> fmta, fmt
	testCCSrc(t, src, 1, 27) // fmt.| -> fmt.*
	testCCSrc(t, src, 2, 9)  // fmt.Print| -> fmt.*print*
	testCCSrc(t, src, 3, 45) // fmt.Print(|) -> *
	testCCSrc(t, src, 4, 0)  // fmt.Print()|
	testCCSrc(t, src, 5, 7)  // u.|Name -> ast.Ident.*
}

func TestCC2(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		func f1() {
			var s1, s2, s3 string
			ast.IsExported●(●s●,●s●)
		}
	`
	testCCSrc(t, src, 0, 1)
	testCCSrc(t, src, 1, 44)
	testCCSrc(t, src, 2, 7)
	testCCSrc(t, src, 3, 44)
	testCCSrc(t, src, 4, 7)
}

func TestCC3(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		import "go/token"
		func f1() {
			var v ast.Visitor
			var no1, no2, no3 ast.Node
			ast.Walk(●v●,●no●1)
		}
		func f2(node a●st.No●de) {
			var n1 ast.Node
			var fset *token.FileSet
			a := fset.Position(●n1.●Pos()).Offset
		}
	`
	testCCSrc(t, src, 0, 47)
	testCCSrc(t, src, 1, 2)
	testCCSrc(t, src, 2, 47)
	testCCSrc(t, src, 3, 3)
	testCCSrc(t, src, 4, 11) // ast.Node
	testCCSrc(t, src, 5, 2)  // ast.Node
	testCCSrc(t, src, 6, 47)
	testCCSrc(t, src, 7, 2)
}

func TestCC4(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		import "context"
		func f1() {
			var pos int
			var nt ast.Node
			switch t := ●nt.(type){
			case *●as●t.Impor●tSpec●:
				return ●nt●.SomeFunc(●t, pos)
			}			
			a,b := ●context.WithCancel(context.Backgr●ound())			
		}
	`
	testCCSrc(t, src, 0, 46)
	testCCSrc(t, src, 1, 46)
	testCCSrc(t, src, 2, 1)
	testCCSrc(t, src, 3, 4)
	testCCSrc(t, src, 4, 1)
	testCCSrc(t, src, 5, 46)
	testCCSrc(t, src, 6, 15)
	testCCSrc(t, src, 7, 46)
	testCCSrc(t, src, 8, 46)
	testCCSrc(t, src, 9, 2)
}

func TestCC5(t *testing.T) {
	src := `
		package pack1
		import "go/types"
		type A struct{
			a●bc int
		}
		func f1(){
			var o1,o2 types.Object
			for _, o := range []types.Object{o1,o2} {
				o.S●tring()
			}
			var u A
			u.●
		}
	`
	testCCSrc(t, src, 0, 11)
	testCCSrc(t, src, 1, 2)
	testCCSrc(t, src, 2, 1)
}
