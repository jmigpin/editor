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
	pos, err := v.visitSource(filename, src, ci)
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
			"fmt"
			"time"
		)
		type type1 struct{
			t time.Time
		}
		func (t1 *type1) func1(){
			t1.t.String()
		}
	`
	testVisitSrc(t, src, 127) // String
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
			_,_=u.(*ttt.Package)
		}
	`
	testVisitSrc(t, src, 114) // ValueSpec
	testVisitSrc(t, src, 140) // Package
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
	testVisitSrc(t, src, 43) // int
}

func TestVisitFile1(t *testing.T) {
	filename := "image/image.go"
	testVisit(t, filename, nil, 1531) // Rectangle
}

// TEMPORARY TEST
func TestVisitFile2(t *testing.T) {
	filename := "github.com/jmigpin/editor/core/toolbarcmd.go"
	testVisit(t, filename, nil, 1713) // NewColumn
}

// TEMPORARY TEST
func TestVisitFile3(t *testing.T) {
	filename := "github.com/jmigpin/editor/core/erow.go"
	testVisit(t, filename, nil, 8354) // erow.row
}
