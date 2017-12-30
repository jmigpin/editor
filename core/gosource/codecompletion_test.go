package gosource

import (
	"strings"
	"testing"
)

func ccTest(t *testing.T, filename string, src interface{}, index int) *CC {
	t.Helper()

	//LogDebug()

	cc := &CC{}
	err := cc.Run(filename, src, index)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("**result**\n")
	t.Logf("%v\n", cc.str)
	return cc
}

func ccTestSrc(t *testing.T, src interface{}, index int) {
	t.Helper()
	filename := "t000/src.go"
	ccTest(t, filename, src, index)
}
func ccTestSrcDetectIndex(t *testing.T, src string) *CC {
	t.Helper()
	filename := "t000/src.go"
	i := strings.Index(src, "|")
	if i < 0 {
		t.Fatal("test index '|' not detected")
	}
	src2 := src[:i] + src[i+1:]
	return ccTest(t, filename, src2, i)
}
func ccTestSrcDetectIndexNResults(t *testing.T, src string, n int) {
	t.Helper()
	cc := ccTestSrcDetectIndex(t, src)
	if len(cc.objs) != n {
		t.Fatalf("expecting %d results: got %v", n, len(cc.objs))
	}
}

func TestCC1(t *testing.T) {
	src := `
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fm|t.Print
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 1)
}
func TestCC2(t *testing.T) {
	src := `
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fmt|.Print
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 1)
}
func TestCC3(t *testing.T) {
	src := `
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fmt.Print|
			fmt.Print(|)
			fmt.Print|()
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 9)
}
func TestCC4(t *testing.T) {
	src := `
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fmt.Print|()
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 9)
}
func TestCC5(t *testing.T) {
	src := `
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fmt.Print(|)
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 41) // TODO: test for more then 10
}
func TestCC6(t *testing.T) {
	src := `
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fmt.|
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 25) // TODO: test for more then 10
}
func TestCC7(t *testing.T) {
	src := `
		package pack1
		import(
			"fmt"			
		)
		var yyy string
		func func1() {
			var zzz string
			fmt.Print("%v %v", zzz|, yyy)
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 1)
}
func TestCC8(t *testing.T) {
	src := `
		package pack1
		import(
			"go/ast"			
		)
		func func1() {
			var n ast.Node
			switch t := n.(type){
			case *ast.Ident:
				_=t.Pos().|
			}
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 1)
}
func TestCC9(t *testing.T) {
	src := `
		package pack1
		import(
			"go/ast"			
		)
		func func1() {
			n:=&ast.Ident{}
			_=n.Pos().|			
		}
	`
	ccTestSrcDetectIndexNResults(t, src, 1)
}

//func TestCC2(t *testing.T) {
//	src := `
//		package pack1
//		import(
//			"go/ast"
//		)
//		func func1() {
//			var u *ast.Ident
//			_=u.Pos()
//		}
//	`
//	//ccTestSrc(t, src, 90) // u.|Pos
//	ccTestSrc(t, src, 91) // u.P|os
//	//ccTestSrc(t, src, 93) // u.P|os
//}

//func TestCC3(t *testing.T) {
//	src := `
//		package pack1
//		import(
//			"go/ast"
//		)
//		func func1() {
//			var u *ast.Ident
//			_=u.Pos().IsValid()
//		}
//	`
//	ccTestSrc(t, src, 98) // Is|Valid
//}

//func TestCC4(t *testing.T) {
//	src := `
//		package pack1
//		import(
//			"go/ast"
//			"go/types"
//		)
//		func func1() {
//			var aa *ast.Ident
//			var aab *types.Func
//			_ = aa
//			_ = aab
//		}
//	`
//	ccTestSrc(t, src, 141) // a|ab
//}

//func TestCC5(t *testing.T) {
//	src := `
//		package pack1
//		import(
//			"runtime/pprof"
//		)
//		func func1() {
//			pprof.StopCPUProfile()
//		}
//	`
//	//ccTestSrc(t, src, 79) // .|StopCPUProfile
//	ccTestSrc(t, src, 88) // .StopCPUPr|ofile
//}
