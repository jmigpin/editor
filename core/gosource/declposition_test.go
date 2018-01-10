package gosource

import (
	"testing"
)

func testVisit(t *testing.T, filename string, src interface{}, index int) {
	t.Helper()

	//LogDebug()

	pos, end, err := DeclPosition(filename, src, index)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = pos, end
	//t.Logf(pos.String())
	//t.Logf(end.String())
}

func testVisitSrc(t *testing.T, src interface{}, index int) {
	t.Helper()
	filename := "t000/src.go"
	testVisit(t, filename, src, index)
}

func TestVisit1(t *testing.T) {
	src := ` 
		package pack1
		import(
			"fmt"
			"time"
		)
		func func1() {
			fmt.Println(time.Now())
		}
	`
	testVisitSrc(t, src, 75) // Println
	testVisitSrc(t, src, 88) // Now
}

func TestVisit2(t *testing.T) {
	src := ` 
		package pack1
		import(
			"time"
		)
		type type1 struct{
			t time.Time
		}
		func (t1 *type1) func1(){
			t1.t.String()
		}
	`
	testVisitSrc(t, src, 118) // String
}

func TestVisit3(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"
			ttt "go/types"
		)
		func func1(){
			var u interface{}
			_,_=u.(*ast.ValueSpec)
			p,_:=u.(*ttt.Package)
			p.Complete()
		}
	`
	testVisitSrc(t, src, 114) // ValueSpec
	testVisitSrc(t, src, 141) // Package
}

func TestVisit4(t *testing.T) {
	src := ` 
		package pack1
		import(
			"time"
		)
		func func1(){
			var t *time.Time
			t.GobDecode(nil)
		}
	`
	testVisitSrc(t, src, 83) // GobDecode
}

func TestVisit5(t *testing.T) {
	src := ` 
		package pack1
		import(
			"time"
		)
		type type1 struct{
			t time.Time
		}
		type type2 struct{
			type1
		}
	`
	testVisitSrc(t, src, 106) // type1 inside type2
}

func TestVisit6(t *testing.T) {
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
			u.Year()
		}
	`
	testVisitSrc(t, src, 130) // Year
}

func TestVisit7(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"
			"image"
		)
		func func1(u interface{}){
			switch u.(type){
			case *ast.Field:
			case *image.Rectangle:
			}
		}
	`
	testVisitSrc(t, src, 117) // Field
	testVisitSrc(t, src, 139) // Rectangle
}

func TestVisit8(t *testing.T) {
	src := ` 
		package pack1
		func func1(){
			var u int
			_ = u
		}
	`
	testVisitSrc(t, src, 43) // int (basic type)
}

func TestVisit9(t *testing.T) {
	src := ` 
		package pack1
		import(
			ttt "go/types"
		)
		func func1(){
			var u interface{}
			p,ok:=u.(*ttt.Package)
			_=ok
			p.Complete()
		}
	`
	testVisitSrc(t, src, 118) // ok
	testVisitSrc(t, src, 126) // Complete
}

func TestVisit10(t *testing.T) {
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
			x := t1.img().Bounds().Max.X
			_=x
		}
	`
	testVisitSrc(t, src, 134) // img
	testVisitSrc(t, src, 140) // Bounds
	testVisitSrc(t, src, 149) // Max
	testVisitSrc(t, src, 153) // X
}

func TestVisit11(t *testing.T) {
	src := `
		package pack1
		func func1(){
			a,b:=false,0
			_,_=a,b
		}
	`
	testVisitSrc(t, src, 41) // false
}

func TestVisit12(t *testing.T) {
	src := `
		package pack1
		import "github.com/jmigpin/editor/util/uiutil/event"
		func func1(ev interface{}){
			switch evt:=ev.(type){
			case *event.KeyDown:
				_ = evt.Code
			}
		}
	`
	testVisitSrc(t, src, 164) // Code
}

func TestVisit13(t *testing.T) {
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
			img.Set(0,0,nil)
			_=img.Bounds()
		}
	`
	testVisitSrc(t, src, 199) // img.Set
	testVisitSrc(t, src, 221) // img.Bounds (inherited from image.Image in another pkg)
}

func TestVisit14(t *testing.T) {
	src := `
		package pack1
		type type1 struct{
			v int
		}
		func func1(){
			m:=make(map[type1]type1)
			for k,v:=range m{
				_=k.v
				_=v.v
			}
		}
	`
	testVisitSrc(t, src, 124) // k.v
	testVisitSrc(t, src, 134) // v.v
}

func TestVisit15(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		type type1 struct{
			v int
		}
		func (t1*type1)func1(node ast.Node){
			if id, ok := node.(*ast.Ident); ok {
				_=id.Pos()
			}
		}
	`
	testVisitSrc(t, src, 135) // ast.Ident
	testVisitSrc(t, src, 157) // id.Pos
}

func TestVisit16(t *testing.T) {
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
				_ = s1.Innermost(id.Pos())
			}
		}	
	`
	testVisitSrc(t, src, 162) // InnerMost
}

func TestVisit17(t *testing.T) {
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
			var t1 type1
			_ = t1.Ident.Pos()
		}	
	`
	testVisitSrc(t, src, 197) // Ident: ensure the position is ident instead of ast (:8:8)
	testVisitSrc(t, src, 203) // Pos: t1 overrides Pos, but want to access ident.pos
}

func TestVisit18(t *testing.T) {
	src := ` 
		package pack1
		import(
			"go/ast"
		)
		func func1(u interface{}){
			switch t:=u.(type){
			case *ast.Field:
				_=t.Pos()
			}
		}
	`
	testVisitSrc(t, src, 124) // Pos
}

func TestVisit19(t *testing.T) {
	src := ` 
		package pack1
		type type1 struct{
			v int
		}
		func func1(){
			a:=&type1{}
			_=a.v
		}
	`
	testVisitSrc(t, src, 90) // a.v
}

func TestVisit20(t *testing.T) {
	src := ` 
		package pack1
		func func1(){
			var ccc,aaa,bbb int
			_=aaa
		}
	`
	testVisitSrc(t, src, 62) // aaa: just "aaa" without getting "aaa int"
}

func TestVisit21(t *testing.T) {
	src := ` 
		package pack1
		import "go/ast"
		func func1(){
			var b[]*ast.Ident
			_=b[0].Pos()
		}
	`
	testVisitSrc(t, src, 83) // Pos
}

func TestVisit22(t *testing.T) {
	src := ` 
		package pack1
		import "go/ast"
		func func1(){
			var b func() *ast.Ident	
			_=b().Pos()
		}
	`
	testVisitSrc(t, src, 89) // Pos
}

func TestVisit23(t *testing.T) {
	src := ` 
		package pack1
		import (
			"go/ast"
			"go/token"
		)
		func func1(){
			var as *ast.AssignStmt
			as.Tok = token.NoPos			
		}
	`
	testVisitSrc(t, src, 107) // as.Tok
}

func TestVisit24(t *testing.T) {
	src := `
		package pack1
		import (
			"go/ast"
		)
		func IsExported()bool{
			return ast.IsExported("a")
		}
	`
	testVisitSrc(t, src, 84) // IsExported
}

func TestVisit25(t *testing.T) {
	src := `
		package pack1
		import (
			"testing"
		)
		func func1(t *testing.T){
			_ = t.Name()
		}
	`
	testVisitSrc(t, src, 82) // t.Name
}

func TestVisit26(t *testing.T) {
	src := `
		package pack1
		import "go/ast"
		func func1()(ast.Node, ast.Node, ast.Node){
			return nil,nil,nil
		}
		func func2(){
			a,b,c:=func1()
			_=a.Pos()
			_=b.Pos()
			_=c.Pos()
		}
	`
	testVisitSrc(t, src, 148) // a.Pos
	testVisitSrc(t, src, 161) // b.Pos
	testVisitSrc(t, src, 174) // c.Pos
}

//func TestVisit27(t *testing.T) {
//	src := `
//		package pack1
//		import "os"
//		func func1(){
//			_=os.Getenv("e1")
//			a:=[]string{}
//			_=append(a, os.Getenv("e2"))
//		}
//	`
//	testVisitSrc(t, src, 55)  // os.Getenv
//	testVisitSrc(t, src, 105) // os.Getenv
//}

func TestVisitFile1(t *testing.T) {
	filename := "image/image.go"
	testVisit(t, filename, nil, 1530) // Rectangle: same package but another file
}

//// TEMPORARY TEST
//func TestVisitFile2(t *testing.T) {
//	filename := "github.com/jmigpin/editor/core/cmdutil/codecompletion.go"
//	testVisit(t, filename, nil, 691)
//}
