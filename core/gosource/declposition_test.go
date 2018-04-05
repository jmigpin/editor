package gosource

import (
	"strings"
	"testing"
)

func testVisit(t *testing.T, filename string, src interface{}, index int) {
	t.Helper()
	posp, endp, err := DeclPosition(filename, src, index)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = posp, endp
	t.Logf("result: %v:%v:%v", posp.Filename, posp.Line, posp.Column)
}

func testVisitSrc(t *testing.T, src interface{}, index int) {
	t.Helper()
	filename := "t000/src.go"
	testVisit(t, filename, src, index)
}

//------------

func testDeclSrc(t *testing.T, src string, n int, expOffset int) {
	t.Helper()
	src2, index, err := SourceCursor("●", src, n)
	if err != nil {
		t.Fatal(err)
	}
	filename := "t000/src.go"
	posp, endp, err := DeclPosition(filename, src2, index)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = posp, endp

	t.Logf("result: offset %v", posp.Offset)
	t.Logf("result: %v", posp)

	if posp.Offset != expOffset {
		t.Fatal()
	}
}

func testDeclSrcFail(t *testing.T, src string, n int, expOffset int, filename string) {
	t.Helper()
	src2, index, err := SourceCursor("●", src, n)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = DeclPosition(filename, src2, index)
	if err == nil {
		t.Fatal("expecting error")
	}
	t.Log(err)
}

//------------

func TestDecl1(t *testing.T) {
	src := ` 
		package pack1
		import(
			"fmt"
			"time"
		)
		func func1() {
			fmt.●Println(time.●Now())
		}
	`
	testDeclSrc(t, src, 0, 7600)
	testDeclSrc(t, src, 1, 31788)
}

func TestDecl2(t *testing.T) {
	src := ` 
		package pack1
		import(
			"time"
		)
		type type1 struct{
			t time.Time
		}
		func (t1 *type1) func1(){
			t1.t.S●tring()
		}
	`
	testDeclSrc(t, src, 0, 13960)
}

func TestDecl3(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"
			ttt "go/types"
		)
		func func1(){
			var u interface{}
			_,_=u.(*ast.●ValueSpec)
			p,_:=u.(*ttt.●Package)
			p.Complete()
		}
	`
	//testDeclSrc(t, src, 0, 26395)
	testDeclSrc(t, src, 1, 248)
}

func TestDecl4(t *testing.T) {
	src := ` 
		package pack1
		import(
			"time"
		)
		func func1(){
			var t *time.Time
			t.●GobDecode(nil)
		}
	`
	testDeclSrc(t, src, 0, 36040)
}

func TestDecl5(t *testing.T) {
	src := `
		package pack1
		import(
			"time"
		)
		type type1 struct{
			t time.Time
		}
		type type2 struct{
			ty●pe1
		}
	`
	testDeclSrc(t, src, 0, 48)
}

func TestDecl6(t *testing.T) {
	src := ` 
		package pack1
		import(
			"time"
		)
		type type1 struct{
			t time.Time
		}
		func func1(){
			var t1 type1
			u:=t1.t
			u.●Year()
		}
	`
	testDeclSrc(t, src, 0, 17137)
}

func TestDecl7(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"
			"image"
		)
		func func1(u interface{}){
			switch u.(type){
			case *ast.●Field:
			case *image.●Rectangle:
			}
		}
	`
	testDeclSrc(t, src, 0, 4449)
	testDeclSrc(t, src, 1, 1999)
}

func TestDecl8(t *testing.T) {
	src := ` 
		package pack1
		func func1(){
			var u ●int
			_ = u
		}
	`
	testDeclSrc(t, src, 0, 2254)
}

func TestDecl9(t *testing.T) {
	src := `
		package pack1
		import(
			ttt "go/types"
		)
		func func1(){
			var u interface{}
			p,●ok:=u.(*ttt.Package)
			_=ok
			p.●Complete()
		}
	`
	testDeclSrc(t, src, 0, 91)
	testDeclSrc(t, src, 1, 1395)
}

func TestDecl10(t *testing.T) {
	src := `
		package pack1
		import(
			"image"
		)
		type type1 interface{
			img() image.Image
		}
		func func1(){
			var t1 type1
			y := t1.●img().●Bounds().●Max.●Y
			_=y
		}
	`
	testDeclSrc(t, src, 0, 69)
	testDeclSrc(t, src, 1, 1521)
	testDeclSrc(t, src, 2, 2024)
	testDeclSrc(t, src, 3, 310)
}

func TestDecl11(t *testing.T) {
	src := `
		package pack1
		func func1(){
			a,b:=●false,0
			_,_=a,b
		}
	`
	testDeclSrc(t, src, 0, 593)
}

func TestDecl12(t *testing.T) {
	src := `
		package pack1
		import "github.com/jmigpin/editor/util/uiutil/event"
		func func1(ev interface{}){
			switch evt:=ev.(type){
			case *event.KeyDown:
				_ = evt.●Code
			}
		}
	`
	testDeclSrc(t, src, 0, 1840)
}

func TestDecl13(t *testing.T) {
	src := `
		package pack1
		import "image/draw"
		type type1 struct{
			img draw.Image
		}
		func (t1*type1) Image() draw.Image{
			return t1.img
		}
		func func1(){
			var t1 type1
			img:=t1.Image()
			img.●Set(0,0,nil)
			_=img.●Bounds()
		}
	`
	testDeclSrc(t, src, 0, 609)
	testDeclSrc(t, src, 1, 1521)
}

func TestDecl14(t *testing.T) {
	src := `
		package pack1
		type type1 struct{
			v int
		}
		func func1(){
			m:=make(map[type1]type1)
			for k,v:=range m{
				_=k.●v
				_=v.●v
			}
		}
	`
	testDeclSrc(t, src, 0, 41)
	testDeclSrc(t, src, 1, 41)
}

func TestDecl15(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		type type1 struct{
			v int
		}
		func (t1*type1)func1(node ast.Node){
			if id, ok := node.(*ast.●Ident); ok {
				_=id.●Pos()
			}
		}
	`
	testDeclSrc(t, src, 0, 6430)
	testDeclSrc(t, src, 1, 12060)
}

func TestDecl16(t *testing.T) {
	src := `
		package pack1
		import (
			"go/ast"		
			"go/types"
		)
		func func1(id *ast.Ident){
			var info types.Info
			s1, ok := info.Scopes[id]
			if ok{
				_ = s1.●Innermost(id.Pos())
			}
		}	
	`
	testDeclSrc(t, src, 0, 4502)
}

func TestDecl17(t *testing.T) {
	src := `
		package pack1
		import (
			"go/ast"
			"go/token"
		)
		type type1 struct{
			ast.Ident
		}
		func (t1*type1)Pos()token.Pos{
			return token.NoPos
		}
		func func1(){
			var ●t1 type1
			_ = t1.●Ident.●Pos()
			_ = t1.●Pos()
		}	
	`
	testDeclSrc(t, src, 0, 178)
	testDeclSrc(t, src, 1, 86)    // the position is ident instead of "go/ast" or ident at ast
	testDeclSrc(t, src, 2, 12060) // Pos: t1 overrides Pos, but want to access ident.pos
	testDeclSrc(t, src, 3, 113)
}

func TestDecl18(t *testing.T) {
	src := `
		package pack1
		import(
			"go/ast"
		)
		func func1(u interface{}){
			switch t:=u.(type){
			case *ast.Field:
				_=t.●Pos()
			}
		}
	`
	testDeclSrc(t, src, 0, 4770)
}

func TestDecl19(t *testing.T) {
	src := `
		package pack1
		type type1 struct{
			v int
		}
		func func1(){
			a:=&type1{}
			_=a.●v
		}
	`
	testDeclSrc(t, src, 0, 41)
}

func TestDecl20(t *testing.T) {
	src := `
		package pack1
		func func1(){
			var ccc,●aaa,bbb int
			_=●aaa
			_=bbb
			_=ccc
		}
	`
	testDeclSrc(t, src, 0, 44)
	testDeclSrc(t, src, 1, 44)
}

func TestDecl21(t *testing.T) {
	src := ` 
		package pack1
		import "go/ast"
		func func1(){
			var b[]*ast.Ident
			_=b[0].●Pos()
			var c func() *ast.Ident	
			_=c().●Pos()
		}
	`
	testDeclSrc(t, src, 0, 12060)
	testDeclSrc(t, src, 1, 12060)
}

func TestDecl23(t *testing.T) {
	src := ` 
		package pack1
		import (
			"go/ast"
			"go/token"
		)
		func func1(){
			var as *ast.AssignStmt
			as.●TokPos = token.NoPos			
		}
	`
	testDeclSrc(t, src, 0, 18398)
}

func TestDecl24(t *testing.T) {
	src := `
		package pack1
		import (
			"go/ast"
		)
		func IsExported()bool{
			return ast.●IsExported("a")
		}
	`
	testDeclSrc(t, src, 0, 16391)
}

func TestDecl25(t *testing.T) {
	src := `
		package pack1
		import (
			"testing"
		)
		func func1(t *testing.T){
			_ = t.●Name()
		}
	`
	testDeclSrc(t, src, 0, 17834)
}

func TestDecl26(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		func func1()(ast.Node, ast.Node){
			return nil,nil
		}
		func func2(){
			a,b:=func1()
			_=●a.●Pos()
			_=●b.●Pos()
		}
	`
	testDeclSrc(t, src, 0, 112)
	testDeclSrc(t, src, 1, 1231)
	testDeclSrc(t, src, 2, 114)
	testDeclSrc(t, src, 3, 1231)
}

func TestDecl27(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		func func1(node ast.Node){			
			switch t:=node.(type){
			case *ast.SelectorExpr:
				var n ast.Node = t.X
				switch t2:=n.(type){
				case *ast.FuncType:
					if t2.●Results!=nil{
					}
				}
			}
		}
	`
	testDeclSrc(t, src, 0, 11177)
}

func TestDecl28(t *testing.T) {
	src := `
		package pack1
		func func1(){			
			var a=1
			●a=40			
		}
	`
	testDeclSrc(t, src, 0, 43)
}

func TestDecl29(t *testing.T) {
	src := `
		package pack1
		import "fmt"
		func f1(){
			●fmt.P
		}
	`
	testDeclSrc(t, src, 0, 26) // needs to not fail on path error since P doesn't exist
}

func TestDecl30(t *testing.T) {
	src := `
		package pack1
		import "fmt"
		func f1(){
			a:=[]int{}
			for i:=0;●len(●a
		}
	`
	testDeclSrc(t, src, 0, 5738)
	testDeclSrc(t, src, 1, 48)
}

func TestDecl31(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		func f1(){
			var n ast.Node
			switch t:=n.(type){
			case *ast.BadExpr:
			case *ast.Call●Expr:
				// TODO: some comment
				for _,a:=range t.
			case *ast.FieldList:
			}	
			return		
		}
	`
	testDeclSrc(t, src, 0, 8954)
}

func TestDecl32(t *testing.T) {
	src := `
		package pack1
		import "go/types"
		func f1(){
			var o1,o2 types.Object
			for _, o := range []types.Object{o1,o2} {
				o.S●tring()
			}
		}
	`
	testDeclSrc(t, src, 0, 1013)
}

func TestDecl33_cycleImporter_endlessLoop(t *testing.T) {
	src := `
		package gosource
		import "github.com/jmigpin/editor/core/gosource" // cycle importer
		func f1(){
			gosource.●DeclPosition // cycle 
		}
	`
	filename := "github.com/jmigpin/editor/core/gosource/src.go"
	testDeclSrcFail(t, src, 0, 1013, filename)
}

//------------

func TestDeclFile1(t *testing.T) {
	filename := "image/image.go"
	testVisit(t, filename, nil, 1530) // Rectangle: same package but another file
}

//------------

func TestFullFilenameDirectory1(t *testing.T) {
	if n := FullFilename("image/image.go"); !strings.HasSuffix(n, "src/image/image.go") {
		t.Fatalf(n)
	}
	if n := FullFilename("/a/b.txt"); n != "/a/b.txt" {
		t.Fatalf(n)
	}
	if n := FullFilename("a/b.txt"); n != "a/b.txt" {
		t.Fatalf(n)
	}
	if n := FullDirectory("fmt"); !strings.HasSuffix(n, "src/fmt") {
		t.Fatalf(n)
	}
	if n := FullDirectory("os/exec"); !strings.HasSuffix(n, "src/os/exec") {
		t.Fatalf(n)
	}

	//{
	//	a, b, c, d := PkgFilenames("fmt", false)
	//	log.Printf("%v %v %v %v", a, b, c, d)
	//}

	//{
	//	a, b, c, d := PkgFilenames("..", false)
	//	log.Printf("%v %v %v %v", a, b, c, d)
	//}
}
