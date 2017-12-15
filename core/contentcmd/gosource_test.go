package contentcmd

import (
	"log"
	"testing"
)

func testVisit(t *testing.T, filename string, src interface{}, ci int) {
	t.Helper()
	v := NewGSVisitor()
	log.SetFlags(0)
	//v.Debug = true
	pos, _, err := v.visitSource(filename, src, ci)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf(pos.String())
}

func testVisitSrc(t *testing.T, src interface{}, ci int) {
	filename := "t000/src.go"
	testVisit(t, filename, src, ci)
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
		import "github.com/jmigpin/editor/uiutil/event"
		func func1(ev interface{}){
			switch evt:=ev.(type){
			case *event.KeyDown:
				_ = evt.Code
			}
		}
	`
	testVisitSrc(t, src, 159) // Code
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
			m:=make(map[type1]int)
			for k,v:=range m{
				_=k.v
				_=v
			}
		}
	`
	testVisitSrc(t, src, 122) // k.v
	testVisitSrc(t, src, 130) // v
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

func TestVisitFile1(t *testing.T) {
	filename := "image/image.go"
	testVisit(t, filename, nil, 1530) // Rectangle
}

// TEMPORARY TEST
func TestVisitFile2(t *testing.T) {
	filename := "github.com/jmigpin/editor/core/toolbarcmd.go"
	testVisit(t, filename, nil, 1708) // NewColumn
}

// TEMPORARY TEST
func TestVisitFile3(t *testing.T) {
	filename := "github.com/jmigpin/editor/ui/toolbar.go"
	testVisit(t, filename, nil, 743) // MarkNeedsPaint
}

// TEMPORARY TEST
func TestVisitFile4(t *testing.T) {
	filename := "github.com/jmigpin/editor/core/editor.go"
	testVisit(t, filename, nil, 832) // loopers.wraplinerune
}

// TEMPORARY TEST
func TestVisitFile5(t *testing.T) {
	filename := "github.com/jmigpin/editor/drawutil2/loopers/wraplinelooper.go"
	testVisit(t, filename, nil, 247) // loopers.wraplinerune
}

// TEMPORARY TEST
func TestVisitFile6(t *testing.T) {
	filename := "github.com/jmigpin/editor/core/contentcmd/gosource_test.go"
	testVisit(t, filename, nil, 76) // testing.T
}

// TEMPORARY TEST
func TestVisitFile7(t *testing.T) {
	filename := "github.com/jmigpin/editor/core/contentcmd/gosource.go"
	testVisit(t, filename, nil, 7826) // makeImportSpecImportable
}
