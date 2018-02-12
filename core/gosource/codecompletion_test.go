package gosource

import (
	"strings"
	"testing"
)

func ccTest(t *testing.T, filename string, src interface{}, index int) *CCResult {
	t.Helper()
	res, err := CodeCompletion(filename, src, index)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

//func ccTestSrc(t *testing.T, src interface{}, index int) {
//	t.Helper()
//	filename := "t000/src.go"
//	_ = ccTest(t, filename, src, index)
//}

func ccTestSrcDetectIndex(t *testing.T, src string, n int) *CCResult {
	t.Helper()
	filename := "t000/src.go"

	// cursor positions
	pos := []int{}
	k := 0
	for {
		j := strings.Index(src[k:], "|")
		if j < 0 {
			break
		}
		k += j
		pos = append(pos, k)
		k++
	}

	// nth position
	if n >= len(pos) {
		t.Fatalf("nth index not found: len=%v", len(pos))
	}
	index := pos[n]

	// remove cursors
	index -= n
	src = strings.Replace(src, "|", "", -1)

	return ccTest(t, filename, src, index)
}

func ccTestSrcDetectIndexNResults(t *testing.T, src string, n, nres int) {
	t.Helper()
	res := ccTestSrcDetectIndex(t, src, n)
	_ = res

	//for _, obj := range res.Objs {
	//	t.Logf("obj: %v\n", obj)
	//}

	t.Logf("result n objects: %v", len(res.Objs))
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
			fmta:=1
			fmt|.|Print|(|)|
			var u ast.Ident
			u.|Name
		}
	`

	//ccTestSrcDetectIndexNResults(t, src, 0, 2)  // fmt|. -> fmta, fmt
	//ccTestSrcDetectIndexNResults(t, src, 1, 27) // fmt.| -> fmt.*
	//ccTestSrcDetectIndexNResults(t, src, 2, 9)  // fmt.Print| -> fmt.*print*
	////ccTestSrcDetectIndexNResults(t, src, 3, 42) // fmt.Print(|) -> *
	//ccTestSrcDetectIndexNResults(t, src, 4, 0) // fmt.Print()|
	ccTestSrcDetectIndexNResults(t, src, 5, 7) // u.|Name -> ast.Ident.*
}

func TestCC2(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		func f1() {
			var id *ast.Ident
			id.IsExported|()
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 0, 1)
}

func TestCC3(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		func f1() {
			a := ast.NewIdent|("a")
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 0, 1)
}
