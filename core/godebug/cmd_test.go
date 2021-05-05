package godebug

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
)

//godebug:annotatefile:cmd.go
//godebug:annotatefile:files.go
////godebug:annotatefile:modules.go
////godebug:annotatefile:annotatorset.go
////godebug:annotatefile:annotator.go

//----------

func init() {
	SimplifyStringifyItem = false
}

func TestCmd_simplifyStringify1(t *testing.T) {
	SimplifyStringifyItem = true
	defer func() { SimplifyStringifyItem = false }()

	src := `
		package main
		func f3(int) int { return 1 }
		func f2(int) int { return 1 }
		func f1(int) int { return 1 }
		func f0() []int{return []int{1,2,3}}
		func main(){
			a := f1(f2(f3(0)))
			_ = "_"
			b := "abc"
			c := f0()
			d := []string{"a","b","c"}
			_=a
			_=b
			_=c
			_=d
		}
	`
	msgs := doCmdSrc(t, src, false)
	// "1 := 1=f1(1=f2(1=f3(0)))" // remove duplicated result
	mustHaveString(t, msgs, "1 := f1(1=f2(1=f3(0)))")
}

//----------

func TestCmd_src1(t *testing.T) {
	src := `
		package main
		import "fmt"
		import "time"
		func main(){
			a:=1
			b:=a
			c:="testing"
			go func(){
				u:=a+b
				c+=fmt.Sprintf("%v", u)
			}()
			c+=fmt.Sprintf("%v", a+b)			
			time.Sleep(10*time.Millisecond)
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, `Sleep("10ms"=(10 * "1ms"))`)
}

func TestCmd_src2(t *testing.T) {
	src := `
		package main
		import "fmt"
		func f1() int{
			_=7
			return 1
		}
		func f2() string{
			_=5
			u := []int{9,1,2,3}
			_=5
			if 1 >= f1() && 1 <= f1() {
				b := 10
				u = u[:1-f1()]
				a := 10 + b
				return fmt.Sprintf("%v %v", a, u)
			}
			_=8
			return "aa"
		}
		func main(){
			_=f2()
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, `_ := "20 []"=f2()`)
}

func TestCmd_src3(t *testing.T) {
	src := `
		package main
		func main(){
			u:=float64(100)
			for i:=0; i<10; i++{
				u/=3
				_=u
			}
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, "false=(10 < 10)")
}

func TestCmd_src4(t *testing.T) {
	src := `
		package main__  // testing with other than "main"
		import "testing"
		func Test001(t*testing.T){
			_=1
		}
	`
	msgs := doCmdSrc(t, src, true)
	mustHaveString(t, msgs, "_ := 1")
}

func TestCmd_src5(t *testing.T) {
	src := `
		package main
		func main() {
			a:=1
			b:=2
			//godebug:annotateoff
			_=a+b
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, "2 := 2")
	mustNotHaveString(t, msgs, "_ := 3=(1 + 2)")
}

func TestCmd_src5b(t *testing.T) {
	src := `
		package main
		func main() {
			//godebug:annotateoff
			for i:=0; i<2;i++{
				//godebug:annotateblock
				_=i+3
			}
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, "_ := 3=(0 + 3)")
	mustNotHaveString(t, msgs, "true=(0 < 2)")
}

func TestCmd_src6(t *testing.T) {
	src := `
		package main
		func main() {
			a:=1
			b:=2
			// has extra ':' at the end in annotation type not expecting it
			//godebug:annotateblock:
			_=a+b
		}
	`
	_, err := doCmdSrc2(t, src, false)
	if err == nil {
		t.Fatal("expecting error")
	}
}

func TestCmd_src7(t *testing.T) {
	src := `
		package main
		//godebug:annotatepackage:not_used_here
		func main() {
			a:=1
			_=a
		}
	`
	_, _, es, err := doCmdSrc3(t, src, false)
	if err != nil {
		t.Fatal(err)
	}
	if !(strings.Index(es, "# warning") >= 0 &&
		strings.Index(es, "not_used_here") >= 0) {
		t.Fatal("missing warning")
	}
}

func TestCmd_src8(t *testing.T) {
	// check the panic function (not a real builtin here)
	src := `
		package main
		func main() {
			panic("!") // not a real builtin
			_=1
		}
		func panic(s string){
			_=2	
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, "_ := 1")
	mustHaveString(t, msgs, "_ := 2")
}

func TestCmd_src9(t *testing.T) {
	// check the panic function
	src := `
		package main
		func main() {
			panic("!")
			_=1 // not reachable
		}
	`
	msgs, err := doCmdSrc2(t, src, false)
	if err == nil {
		t.Fatal("expecting error from panic")
	}
	mustHaveString(t, msgs, `=> panic("!")`)
	mustNotHaveString(t, msgs, "_ := 1")
}

func TestCmd_src10(t *testing.T) {
	// should be able to compile big constants
	src := `
		package main
		import "math"
		func main() {
			_=uint64(1<<64 - 1)
			_=uint64(math.MaxUint64)
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, `_ := 18446744073709551615=uint64(18446744073709551615=(18446744073709551616=(1 << 64) - 1))`)
}

func TestCmd_src11(t *testing.T) {
	// support function returning multiple vars as args to another func
	src := `
		package main
		func main() {
			_=f1(f2())
		}
		func f1(a,b int)int{
			return a+b
		}
		func f2() (int,int){
			return 1,2
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, `=> f1((1, 2)=f2())`)
}

func TestCmd_src12(t *testing.T) {
	// comments in the middle of a stmt
	src := `
		package main
		func main() {
			a:=1
			/*aaa*/
			_=a
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, `_ := 1`)
}

func TestCmd_src13(t *testing.T) {
	// replacement of os.exit
	src := `
		package main
		import "os"
		func main() {
			os.Exit(0)
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, `=> Exit(0)`)
}

func TestCmd_src14(t *testing.T) {
	// replacement of os.exit
	src := `
		package main
		import "os"
		func main() {
			_=os.Getenv("a")
			os.Exit(0)
		}
	`
	msgs := doCmdSrc(t, src, false)
	mustHaveString(t, msgs, `=> Getenv("a")`)
	mustHaveString(t, msgs, `=> Exit(0)`)
}

//------------

func TestCmd_comments(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		func main() {
			a:=1
			_=a
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	ctx := context.Background()
	_, _, es, err := doCmd3(ctx, t, dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !(strings.Index(es, "# warning") >= 0 &&
		strings.Index(es, "not at an import spec") >= 0) {
		t.Fatal("missing warning")
	}
}

func TestCmd_comments2(t *testing.T) {
	// test single import line

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../w/example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		import "example.com/pkg1"
		func main() {
			_=pkg1.Fa()
		}
	`)
	d := "w/example.com/"
	tf.WriteFileInTmp2OrPanic(d+"pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic(d+"pkg1/fa.go", `
		package pkg1
		func Fa() string {
			return "Fa"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
}

func TestCmd_comments3(t *testing.T) {
	// test gendecl import line

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../w/example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import (
			//godebug:annotateimport
			"example.com/pkg1"
		)
		func main() {
			_=pkg1.Fa()
		}
	`)
	d := "w/example.com/"
	tf.WriteFileInTmp2OrPanic(d+"pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic(d+"pkg1/fa.go", `
		package pkg1
		func Fa() string {
			return "Fa"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
}

//------------

func TestCmd_goMod1(t *testing.T) {
	// not in gopath
	// has go.mod
	// pkg1 is in relative dir, not annotated
	// pkg2 is in relative dir, annotated, depends on pkg1

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		require example.com/pkg2 v0.0.0
		replace example.com/pkg1 => ../pkg1
		replace example.com/pkg2 => ../pkg2
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		import "example.com/pkg2"
		func main() {
			_=pkg1.F1()
			_=pkg2.F2()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/f1.go", `
		package pkg1
		func F1() string {
			return "F1"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/go.mod", `
		module example.com/pkg2
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/f2.go", `
		package pkg2
		import "example.com/pkg1"
		func F2() string {
			//godebug:annotateblock
			return "F2"+pkg1.F1()
		}
	`)

	dir1 := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	msgs := doCmd(t, dir1, cmd)
	mustNotHaveString(t, msgs, "F1")
	mustHaveString(t, msgs, `"F2F1"=("F2" + "F1"=F1())`)
}

func TestCmd_goMod2(t *testing.T) {
	// not in gopath
	// has go.mod
	// pkg1 is in relative dir, not annotated
	// pkg2 is in abs dir, annotated

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		require example.com/pkg2 v0.0.0
		replace example.com/pkg1 => ../pkg1
		replace example.com/pkg2 => `+filepath.Join(tf.Dir, "pkg2")+`
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		import "example.com/pkg2"
		func main() {
			_=pkg1.F1()
			_=pkg2.F2()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/f1.go", `
		package pkg1
		func F1() string {
			return "F1"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/go.mod", `
		module example.com/pkg2
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/f2.go", `
		package pkg2
		func F2() string {
			//godebug:annotateblock
			return "F2"
		}
	`)

	dir1 := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	msgs := doCmd(t, dir1, cmd)
	mustNotHaveString(t, msgs, `"F1"`)
	mustHaveString(t, msgs, `"F2"`)
}

func TestCmd_goMod2b(t *testing.T) {
	// not in gopath
	// has go.mod
	// pkg1 is in relative dir, annotated
	// pkg2 is in relative dir, not annotated, depends on pkg1

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		require example.com/pkg2 v0.0.0
		replace example.com/pkg1 => ../pkg1
		replace example.com/pkg2 => ../pkg2
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		import "example.com/pkg2"
		func main() {
			_=pkg1.F1()
			_=pkg2.F2()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/f1.go", `
		package pkg1
		func F1() string {
			//godebug:annotateblock
			return "F1"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/go.mod", `
		module example.com/pkg2
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/f2.go", `
		package pkg2
		import "example.com/pkg1"
		func F2() string {
			return "F2"+pkg1.F1()
		}
	`)

	dir1 := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	msgs := doCmd(t, dir1, cmd)
	mustHaveString(t, msgs, `"F1"`)
	mustNotHaveString(t, msgs, `"F2F1"=("F2" + "F1"=F1())`)

}

func TestCmd_goMod3(t *testing.T) {
	// func call should not be annotated: pkg with only one godebug annotate block inside another func

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		func main() {
			_=pkg1.F1a()
			_=pkg1.F1b("arg-from-main")
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/f1.go", `
		package pkg1
		func F1a() string {
			//godebug:annotateblock
			return "F1a"
		}
		func F1b(a string) string {
			return "F1b"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustNotHaveString(t, msgs, `"arg-from-main"`)
}

func TestCmd_goMod4(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		func main() {
			_=pkg1.F1a()
			_=pkg1.F1b()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/f1a.go", `
		package pkg1
		func F1a() string {
			//godebug:annotatepackage
			return "F1a"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/f1b.go", `
		package pkg1
		func F1b() string {
			return "F1b"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"F1a"`)
	mustHaveString(t, msgs, `"F1b"`)
}

func TestCmd_goMod5_test(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module example.com/main
		require example.com/pkg1 v0.0.0
		require example.com/pkg2 v0.0.0
		replace example.com/pkg1 => ../pkg1
		replace example.com/pkg2 => ../pkg2
	`)
	tf.WriteFileInTmp2OrPanic("main/main_test.go", `
		package main
		import "testing"
		import "example.com/pkg1"
		import "example.com/pkg2"
		func Test01(t*testing.T) {
			_=pkg1.F1()
			_=pkg2.F2()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/f1.go", `
		package pkg1
		func F1() string {
			return "F1"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/go.mod", `
		module example.com/pkg2
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/f2.go", `
		package pkg2
		func F2() string {
			//godebug:annotateblock
			return "F2"
		}
	`)

	dir1 := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"test",
		//"-work",
	}
	msgs := doCmd(t, dir1, cmd)
	mustNotHaveString(t, msgs, `"F1"`)
	mustHaveString(t, msgs, `"F2"`)
}

func TestCmd_goMod6(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		import "example.com/pkg1/sub1"
		//godebug:annotatemodule
		func main() {
			_=pkg1.Fa()
			_=pkg1.Fb()
			_=sub1.Fc()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fa.go", `
		package pkg1
		//godebug:annotatemodule
		func Fa() string {
			return "Fa"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fb.go", `
		package pkg1
		func Fb() string {
			return "Fb"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/sub1/fc.go", `
		package sub1
		func Fc() string {
			return "Fc"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
	mustHaveString(t, msgs, `"Fb"`)
	mustHaveString(t, msgs, `"Fc"`)
}

func TestCmd_goMod7(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		import "example.com/pkg1/sub1"
		//godebug:annotatepackage:example.com/pkg1/sub1
		func main() {
			_=pkg1.Fa()
			_=pkg1.Fb()
			_=sub1.Fc()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fa.go", `
		package pkg1
		func Fa() string {
			return "Fa"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fb.go", `
		package pkg1
		func Fb() string {
			return "Fb"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/sub1/fc.go", `
		package sub1
		func Fc() string {
			return "Fc"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustNotHaveString(t, msgs, `"Fa"`)
	mustNotHaveString(t, msgs, `"Fb"`)
	mustHaveString(t, msgs, `"Fc"`)
}

func TestCmd_goMod7b(t *testing.T) {
	// test //godebug:annotatemodule:<pkg-path>

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "example.com/pkg1"
		import "example.com/pkg1/sub1"
		//godebug:annotatemodule:example.com/pkg1/sub1
		func main() {
			_=pkg1.Fa()
			_=pkg1.Fb()
			_=sub1.Fc()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fa.go", `
		package pkg1
		func Fa() string {
			return "Fa"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fb.go", `
		package pkg1
		func Fb() string {
			return "Fb"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/sub1/fc.go", `
		package sub1
		func Fc() string {
			return "Fc"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
	mustHaveString(t, msgs, `"Fb"`)
	mustHaveString(t, msgs, `"Fc"`)
}

func TestCmd_goMod7c(t *testing.T) {
	// test passing build args to compiler

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		import "example.com/pkg1"
		func main() {
			_=pkg1.Fa()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fa_os1.go", `
		// +build OS1
		
		package pkg1
		func Fa() string {
			return "Fa_os1"
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fa_os2.go", `
		// +build OS2

		package pkg1
		func Fa() string {
			return "Fa_os2"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"-env=GODEBUG_BUILD_FLAGS=-tags=OS2",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustNotHaveString(t, msgs, `"Fa_os1"`)
	mustHaveString(t, msgs, `"Fa_os2"`)
}

func TestCmd_goMod8(t *testing.T) {
	// An empty go.mod with just the module name, will make "go build" try to fetch from the web the dependencies.
	// By using "go mod init", if there is no go.mod, it is created with the dependency (if already on the disk) and nothing is fetched from the web.
	// setting GOPROXY=off fails, but not sure why:
	// TODO: fails because go.mod is defined but doesn't declare the dependency location. Will fail with "cannot load...". It still fails without the go.mod but with GOPROXY=off.

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/BurntSushi/xgb"
		//godebug:annotatepackage:github.com/BurntSushi/xgb
		func main() {
			_=xgb.Pad(1)
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"-env=GOPROXY=off",
		"main.go",
	}
	_, err := doCmd2(t, dir, cmd)
	if err == nil {
		t.Fatal("expecting error")
	}
}

func TestCmd_goMod9(t *testing.T) {
	// Update: fails since go.mod is not present (not a module)
	// Old: if the os env doesn't have GOPROXY=off, having no go.mod should fetch the dependencies from the web at pre-build.

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/BurntSushi/xgb"
		import "golang.org/x/tools/godoc/util"
		//godebug:annotatepackage:github.com/BurntSushi/xgb
		//godebug:annotatepackage:golang.org/x/tools/godoc/util
		func main() {
			_=xgb.Pad(1)
			_=util.IsText([]byte("001"))
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"-env=GO111MODULE=on", // force modules mode (no go.mod)
		"main.go",
	}
	//msgs := doCmd(t, dir, cmd)
	//mustHaveString(t, msgs, `4=((4=(1 + 3)) & -4=^3)`)
	//mustHaveString(t, msgs, `[48 48 49]`)
	_, err := doCmd2(t, dir, cmd)
	if err == nil {
		t.Fatal("expecting error")
	}
}

func TestCmd_goMod10(t *testing.T) {
	// fails because GOPROXY=off won't fetch the module (no go.mod and outside of GOPATH)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/BurntSushi/xgb"
		import "golang.org/x/tools/godoc/util"
		//godebug:annotatepackage:github.com/BurntSushi/xgb
		//godebug:annotatepackage:golang.org/x/tools/godoc/util
		func main() {
			_=xgb.Pad(1)
			_=util.IsText([]byte("001"))
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		// force modules mode (no go.mod)
		"-env=GO111MODULE=on:GOPROXY=off",
		"main.go",
	}
	_, err := doCmd2(t, dir, cmd)
	if err == nil {
		t.Fatal("expecting error")
	}
}

func TestCmd_goMod11(t *testing.T) {
	// passes because is outside of GOPATH, but has go.mod, so it fetches from the web

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		`+requireStrBurntSushi+`
		`+requireStrGolangXTools+`
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/BurntSushi/xgb"
		import "github.com/BurntSushi/xgb/shm"
		import "golang.org/x/tools/godoc/util"
		//godebug:annotatepackage:github.com/BurntSushi/xgb
		//godebug:annotatepackage:golang.org/x/tools/godoc/util
		func main() {
			_=xgb.Pad(1)
			conn,err:=xgb.NewConnDisplay("")
			defer conn.Close()
			if err!=nil{
				_=shm.Init(conn)
			}
			_=util.IsText([]byte("001"))
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `4=((4=(1 + 3)) & -4=^3)`)
	mustHaveString(t, msgs, `[48 48 49]`)
}

func TestCmd_goMod12(t *testing.T) {
	// mod dependency is on xgb, but the annotated package is shm

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		`+requireStrBurntSushi+`
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/BurntSushi/xgb"
		//godebug:annotateimport
		import "github.com/BurntSushi/xgb/shm"
		func main() {
			_=xgb.Pad(1)
			conn,err:=xgb.NewConnDisplay("")
			defer conn.Close()
			if err!=nil{
				_=shm.Init(conn)
			}
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustNotHaveString(t, msgs, `4=((4=(1 + 3)) & -4=^3)`)
	mustHaveString(t, msgs, `map[]=NewExtErrorFuncs["MIT-SHM"] := map[]=make(type)`)
}

func TestCmd_goMod13(t *testing.T) {
	// annotate full external module (slow)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		`+requireStrBurntSushi+`
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/BurntSushi/xgb"
		import "github.com/BurntSushi/xgb/shm"
		//godebug:annotatemodule:github.com/BurntSushi/xgb/shm
		func main() {
			_=xgb.Pad(1)
			conn,err:=xgb.NewConnDisplay("")
			defer conn.Close()
			if err!=nil{
				_=shm.Init(conn)
			}
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `4=((4=(1 + 3)) & -4=^3)`)
	mustHaveString(t, msgs, `map[]=NewExtErrorFuncs["MIT-SHM"] := map[]=make(type)`)
}

func TestCmd_goMod14(t *testing.T) {
	// error when trying to annotate goroot package

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		import "fmt"
		func main() {
			fmt.Printf("aaa")
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	ctx := context.Background()
	_, _, es, err := doCmd3(ctx, t, dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !(strings.Index(es, "# warning") >= 0 &&
		strings.Index(es, "pkg path not found") >= 0) {
		t.Fatal("missing warning")
	}
}

func TestCmd_goMod15(t *testing.T) {
	// test ctx cancel

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "time"
		import "fmt"
		//godebug:annotatefile
		func main() {
			time.Sleep(10000*time.Second)
			fmt.Printf("aaa")
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
		t.Logf("cancel: %v", time.Now().Sub(start))
	}()
	_, _, _, err := doCmd3(ctx, t, dir, cmd)
	//if err == nil {
	//	t.Fatal("expecting error")
	//}
	t.Log(err)
	t.Logf("done: %v", time.Now().Sub(start))
}

func TestCmd_goMod16(t *testing.T) {
	// using the editor as a library but annotating another module
	// because the editor module is not annotated, no copy of the necessary modules were done, and so the "debug" package exists causing an ambiguous module

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		import "example.com/pkg1"
		func main() {
			_=pkg1.Fa()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
		`+requireStrJmigpinEditor+`
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fa.go", `
		package pkg1
		import "github.com/jmigpin/editor/util/mathutil"
		func Fa() string {
			_=mathutil.Max(0,1)
			return "Fa"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
	mustHaveString(t, msgs, `=> Max(0, 1)`)
}

func TestCmd_goMod17(t *testing.T) {
	// using the editor as a library that is being annotated as well (the chosen package has annotations)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require example.com/pkg1 v0.0.0
		replace example.com/pkg1 => ../pkg1
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		import "example.com/pkg1"
		func main() {
			_=pkg1.Fa()
		}
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module example.com/pkg1
		`+requireStrJmigpinEditor+`
	`)
	tf.WriteFileInTmp2OrPanic("pkg1/fa.go", `
		package pkg1
		import "github.com/jmigpin/editor/util/goutil"
		func Fa() string {
			_=goutil.GoPath()
			return "Fa"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
	mustHaveString(t, msgs, `=> GoPath()`)
}

//func TestCmd_goMod18(t *testing.T) {
//	// TODO: needs packages.load to be improved?

//	// debug editor pkg but from another directory (similar to prev test)

//	tf := newTmpFiles(t)
//	defer tf.RemoveAll()

//	tf.WriteFileInTmp2OrPanic("main/go.mod", `
//		module main
//		require example.com/pkg1 v0.0.0
//		replace example.com/pkg1 => ../pkg1
//	`)
//	//tf.WriteFileInTmp2OrPanic("main/go.sum", `
//	//	`+sumStrJmigpinEditor2+`
//	//`)
//	tf.WriteFileInTmp2OrPanic("main/main.go", `
//		package main
//		//godebug:annotateimport
//		import "example.com/pkg1"
//		func main() {
//			_=pkg1.Fa()
//		}
//	`)
//	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
//		module example.com/pkg1
//		`+requireStrJmigpinEditor+`
//	`)
//	//tf.WriteFileInTmp2OrPanic("pkg1/go.sum", `
//	//	`+sumStrJmigpinEditor2+`
//	//`)
//	tf.WriteFileInTmp2OrPanic("pkg1/fa.go", `
//		package pkg1
//		import "github.com/jmigpin/editor/util/mathutil"
//		func Fa() string {
//			_=mathutil.Max(0,1)
//			return "Fa"
//		}
//	`)

//	dir := filepath.Join(tf.Dir, "")
//	//dir := filepath.Join(tf.Dir, "main")
//	cmd := []string{
//		"run",
//		"-work",
//		"main/main.go",
//		//"main.go",
//	}
//	tryGoModTidy(t, dir)
//	msgs := doCmd(t, dir, cmd)
//	mustHaveString(t, msgs, `"Fa"`)
//	mustHaveString(t, msgs, `=> Max(0, 1)`)
//}

//----------

func TestCmd_goMod20(t *testing.T) {
	// using the editor as a library
	// nothing used other then the debug pkg
	// so the original require for the editor pkg could be dropped since it causes an ambiguous import

	// with const DebugPkgPath = "godebug0/debug", runs but:
	// 	panic: gob: registering duplicate types
	// with const DebugPkgPath = correct pkg path, fails to compile:
	// 	compile error: ... "ambiguous import"

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		`+requireStrJmigpinEditor+`
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/jmigpin/editor/core/godebug/debug"
		func main() {
			msg := &debug.ReqFilesDataMsg{}
			_, _ = debug.EncodeMessage(msg)
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `=> EncodeMessage(&ReqFilesDataMsg{})`)
}

func TestCmd_goMod21(t *testing.T) {
	// using the editor as a library (self debug)
	// uses a pkg that is *not-annotated*
	// must be handled as a special case due to being in the same module as the debug pkg

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		`+requireStrJmigpinEditor+`
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/jmigpin/editor/util/mathutil"
		func main() {
			_=mathutil.Max(0,1)
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `=> Max(0, 1)`)
}

func TestCmd_goMod22(t *testing.T) {
	// using the editor as a library (self debug)
	// uses a pkg that is *annotated*
	// must be handled as a special case due to being in the same module as the debug pkg

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		`+requireStrJmigpinEditor+`
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "github.com/jmigpin/editor/util/goutil"
		func main() {
			_=goutil.GoPath()
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `=> GoPath()`)
}

func TestCmd_goMod23(t *testing.T) {
	// test with specific package that triggered an infinite loop of msgs

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require golang.org/x/mod v0.4.2
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		import "golang.org/x/mod/modfile"
		func main() {
			src:=[]byte("asdfasdfasdfas asdfasdf")
			_, _ = modfile.Parse("aaa", src, nil)
			_=1
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `=> parseToFile("aaa", [97 115 100 102 97 1..., 0x0, true)`)
}

func TestCmd_goMod24(t *testing.T) {
	// sequence of "//godebug" directives

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "main/pkg1"
		func main() {
			_=pkg1.Fa()
		}
	`)
	tf.WriteFileInTmp2OrPanic("main/pkg1/fa.go", `
		package pkg1
		
		import(
			"fmt"
		)
		
		//godebug:annotatefile:fb.go
		//godebug:annotatefile:fc.go
		
		func Fa() string{
			_=Fb()
			_=Fc()
			_=Fd()
			return fmt.Sprintf("fa")
		}
	`)
	tf.WriteFileInTmp2OrPanic("main/pkg1/fb.go", `
		package pkg1
		func Fb() string{return "fb"}
	`)
	tf.WriteFileInTmp2OrPanic("main/pkg1/fc.go", `
		package pkg1
		func Fc() string{return "fc"}
	`)
	tf.WriteFileInTmp2OrPanic("main/pkg1/fd.go", `
		package pkg1
		func Fd() string{return "fd"}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		//"-work",
		"main.go",
	}
	//tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"fb"`)
	mustHaveString(t, msgs, `"fc"`)
	mustNotHaveString(t, msgs, `"fd"`)
}

func TestCmd_goMod25(t *testing.T) {
	// unused pkg2 (in testmode) is preventing compilation due dependency of a specific version of the editorPkg

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
		require pkg2 v0.0.0
		replace pkg2 => ../pkg2
	`)
	tf.WriteFileInTmp2OrPanic("main/go.sum", `
	`)
	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		import "pkg1"
		func main() {
			_=pkg1.Fa(1)
		}
	`)
	tf.WriteFileInTmp2OrPanic("main/pkg1/fa.go", `
		package pkg1
		import "fmt"
		func Fa(a int) string{ 
			return fmt.Sprintf("fa:%v",a)
		}
	`)
	tf.WriteFileInTmp2OrPanic("main/pkg1/fa_test.go", `
		package pkg1
		import "testing"
		func TestFa(t*testing.T) {
			_=Fa(2)
		}
	`)
	tf.WriteFileInTmp2OrPanic("main/pkg1/fb.go", `
		package pkg1
		import "pkg2"
		func Fb() string{
			return pkg2.Fb()
		}		
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/go.mod", `
		module pkg2
		`+requireStrJmigpinEditor+`
	`)
	tf.WriteFileInTmp2OrPanic("pkg2/fb.go", `
		package pkg2
		import "github.com/jmigpin/editor/core/godebug/debug"
		func Fb() string {
			msg := &debug.ReqFilesDataMsg{}
			_, _ = debug.EncodeMessage(msg)
			return "fb called debug"
		}
	`)

	dir := filepath.Join(tf.Dir, "main/pkg1")
	cmd := []string{
		"test",
		//"-work",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `_ := "fa:2"=Fa(2)`)
}

func TestCmd_goMod26(t *testing.T) {
	// running tests of the editorPkg (special handling)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("d1/go.mod", `
		module github.com/jmigpin/editor
	`)
	tf.WriteFileInTmp2OrPanic("d1/d2/some_test.go", `
		package d2
		import "testing"
		func TestSome1(t*testing.T) {
			_=1
		}
	`)

	dir := filepath.Join(tf.Dir, "d1/d2")
	cmd := []string{
		"test",
		//"-work",
	}
	tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `_ := 1`)
}

func TestCmd_goMod27(t *testing.T) {
	// editor pkg just being required in the go.mod, not used in the code, gives an ambiguos import if "go mod tidy" is not run

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("pkg1/go.mod", `
		module pkg1
		require github.com/jmigpin/editor v0.0.0-rc1
		replace github.com/jmigpin/editor => ../editor_rc1
	`)
	//tf.WriteFileInTmp2OrPanic("pkg1/main.go", `
	//	package pkg1
	//	func main() {
	//	}
	//`)
	tf.WriteFileInTmp2OrPanic("pkg1/main_test.go", `
		package pkg1
		import "testing"
		func TestMain1(t*testing.T) {
			_=1
		}
	`)
	tf.WriteFileInTmp2OrPanic("editor_rc1/go.mod", `
		module github.com/jmigpin/editor
	`)
	tf.WriteFileInTmp2OrPanic("editor_rc1/core/godebug/debug/fa.go", `
		package debug
		func Exit(a int) {} // ambiguous
	`)

	dir := filepath.Join(tf.Dir, "pkg1")
	cmd := []string{
		"test",
		//"-work",
	}
	//tryGoModTidy(t, dir)
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `_ := 1`)
}

//------------

func TestCmd_goPath1(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("src/main/main.go", `
		package main
		import "main/sub1"
		import "main/sub1/sub2"
		import "main/sub3"
		func main() {
			_=sub1.Sub1()
			_=sub2.Sub2()
			_=sub3.Sub3()
		}
	`)
	tf.WriteFileInTmp2OrPanic("src/main/sub1/sub1.go", `
		package sub1
		func Sub1() string {
			return "sub1"
		}
	`)
	tf.WriteFileInTmp2OrPanic("src/main/sub1/sub2/sub2.go", `
		package sub2
		func Sub2() string {
			//godebug:annotateblock
			return "sub2"
		}
	`)
	tf.WriteFileInTmp2OrPanic("src/main/sub3/sub3.go", `
		package sub3
		func Sub3() string {
			return "sub3"
		}
	`)

	dir := filepath.Join(tf.Dir, "src/main")
	cmd := []string{
		"run",
		//"-work",
		"-env=GO111MODULE=auto:GOPATH=" + tf.Dir,
		"main.go"}
	msgs := doCmd(t, dir, cmd)
	mustNotHaveString(t, msgs, `"sub1"`)
	mustHaveString(t, msgs, `"sub2"`)
}

func TestCmd_goPath2(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("aaa/src/main/main.go", `
		package main
		import "pkg1"
		func main() {
			_=1
			_=pkg1.Sub1()
		}
	`)
	tf.WriteFileInTmp2OrPanic("src/pkg1/sub1.go", `
		package pkg1
		func Sub1() string {
			//godebug:annotateblock
			return "sub1"
		}
	`)

	cmd := []string{
		"run",
		//"-work",
		"-env=GO111MODULE=auto:GOPATH=" + tf.Dir,
		"main.go",
	}

	dir := filepath.Join(tf.Dir, "aaa/src/main")
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `_ := 1`)
	mustHaveString(t, msgs, `"sub1"`)
}

func TestCmd_goPath3(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	// no go.mod, should run in GOPATH mode and succeed

	tf.WriteFileInTmp2OrPanic("main/main.go", `
		package main
		//godebug:annotateimport
		import "example.com/pkg1"
		func main() {
			_=pkg1.Fa()
		}
	`)
	tf.WriteFileInTmp2OrPanic("w/src/example.com/pkg1/fa.go", `
		package pkg1
		func Fa() string {
			return "Fa"
		}
	`)

	dir := filepath.Join(tf.Dir, "main")
	cmd := []string{
		"run",
		"-env=GO111MODULE=auto:GOPATH=" + filepath.Join(tf.Dir, "w"),
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
}

//----------

func TestCmd_simple1(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		import "io/ioutil"
		func main() {
			b,_:=ioutil.ReadFile("a.txt")
			_=string(b)
		}
	`)
	tf.WriteFileInTmp2OrPanic("a.txt", `aaa`)

	cmd := []string{
		"run",
		//"-work",
		"-env=GO111MODULE=auto",
		"dir1/main.go", // give location, but must run on dir
	}
	msgs := doCmd(t, tf.Dir, cmd)
	mustHaveString(t, msgs, `_ := "aaa"=string([97 97 97])`)
}

func TestCmd_simple2(t *testing.T) {
	// multiple file passed as package main

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		func main() {
			_=fn()
		}
	`)
	tf.WriteFileInTmp2OrPanic("dir1/fn.go", `
		package main
		func fn() string {
			return "1"
		}
	`)

	cmd := []string{
		"run",
		//"-work",
		"main.go",
		"fn.go",
	}
	dir := filepath.Join(tf.Dir, "dir1")
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `_ := "1"=fn()`)
	mustNotHaveString(t, msgs, `"1"`)
}

func TestCmd_simple2b(t *testing.T) {
	// multiple file passed as package main

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		func main() {
			_=fn()
		}
	`)
	tf.WriteFileInTmp2OrPanic("dir1/fn.go", `
		package main
		func fn() string {
			return "1"
		}
	`)

	cmd := []string{
		"run",
		//"-work",
		"dir1/main.go",
		"dir1/fn.go",
	}

	msgs := doCmd(t, tf.Dir, cmd)
	mustHaveString(t, msgs, `_ := "1"=fn()`)
	mustNotHaveString(t, msgs, `"1"`)
}

func TestCmd_simple3(t *testing.T) {
	// able to pass arguments to the built binary

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		import "flag"
		func main() {
			v:=flag.String("somearg","b","usage")
			flag.Parse()
			_=*v
		}
	`)

	cmd := []string{
		"run",
		"-env=GO111MODULE=auto",
		"dir1/main.go",
		"-somearg=a",
	}
	msgs, err := doCmd2(t, tf.Dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveString(t, msgs, `_ := "a"=*&"a"`)
}

func TestCmd_simple4(t *testing.T) {
	// don't annotate args

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		import "fmt"
		func main() {
			v:=T1(0)
			fmt.Sprintf("%v",v)
		}
		type T1 int
		func (t T1) String() string{
			return f1(t)
		}
		//godebug:annotateoff 	// testing
		func f1(t T1)string{
			return fmt.Sprintf("%d",t)
		}
	`)

	cmd := []string{
		"run",
		"dir1/main.go",
	}
	msgs, err := doCmd2(t, tf.Dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	// will be endless loop if it fails
	_ = msgs
	//mustHaveString(t, msgs, ``)
}

func TestCmd_simple5(t *testing.T) {
	// test annotation node

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		func main() {
			f1(1)
		}
		// some comment that is disabling annotateoff
		// some comment that is disabling annotateoff
		//godebug:annotateoff
		func f1(v int)string{
			return "f1"
		}
	`)

	cmd := []string{
		"run",
		"dir1/main.go",
	}
	msgs, err := doCmd2(t, tf.Dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	// will be endless loop if it fails
	mustNotHaveString(t, msgs, `"f1"`)
}

func TestCmd_simple6(t *testing.T) {
	// goto is changing the program

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		func main() {
			a:=[]int{1}
		redo:
			k := len(a)
			if k<=1{
				a=[]int{1,2,3}
				goto redo
			}
		}
	`)

	cmd := []string{
		"run",
		//"-work",
		"dir1/main.go",
	}
	msgs, err := doCmd2(t, tf.Dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	// will be endless loop if it fails
	mustHaveString(t, msgs, `false=(3 <= 1)`)
}

//------------

//func TestCmd_empty(t *testing.T) {}
//func TestCmd_testInOwnDir(t *testing.T) {
//	args := []string{
//		"test", "-run", "TestCmd_empty",
//	}
//	msgs := doCmd(t, "", args) // WARNING: runs in this directory
//	_ = msgs
//	//spew.Dump(msgs)
//}

//------------
//------------
//------------

//// Development test
//func TestCmd_srcDev(t *testing.T) {
//	src := `
//		package main
//		func main(){
//		}
//	`
//	msgs := doCmdSrc(t, src, false, false)
//	_ = msgs
//}

//// Launches the editor itself.
//func TestCmd_editor(t *testing.T) {
//	filename := "./../../editor.go"
//	args := []string{
//		"run",
//		//"-work",
//		//"-dirs=../../core",
//		//"-dirs=../../core,../../core/contentcmds",
//		filename,
//		"-sn=gui",
//	}
//	doCmd(t, "", args)
//}

//------------

func BenchmarkCmd1(b *testing.B) {
	// just searching for something odd, these tests envolve too much OS ops to be meaningful

	// N=def, parseFile==nil: 911875496: ns/op
	// N=10, parseFile==nil: 947289189 ns/op
	// N=15, parseFile==nil: 951260274 ns/op

	// N=def, parseFile!=nil: 917934864 ns/op
	// N=10, parseFile!=nil: 909638580 ns/op
	// N=15, parseFile!=nil: 924818313 ns/op

	// N=10, parseFile!=nil: 857348146 ns/op

	//BenchmarkCmd1-4   	      10	 863937070 ns/op

	b.N = 10
	for n := 0; n < b.N; n++ {
		bCmd1(b)
	}
}

func bCmd1(b *testing.B) {
	src := `
		package main
		import "image"
		type A struct{ p image.Point }
		func main(){
			a:=A{}
			b:=a.p.String()
			_ = b
		}
	`
	t := &testing.T{}
	_ = doCmdSrc(t, src, false)
}

//------------
//------------
//------------

func doCmd(t *testing.T, dir string, args []string) []string {
	t.Helper()
	msgs, err := doCmd2(t, dir, args)
	if err != nil {
		t.Fatal(err)
	}
	return msgs
}

func doCmd2(t *testing.T, dir string, args []string) ([]string, error) {
	t.Helper()
	ctx := context.Background()
	msgs, _, _, err := doCmd3(ctx, t, dir, args)
	return msgs, err
}

func doCmd3(ctx context.Context, t *testing.T, dir string, args []string) ([]string, string, string, error) {
	t.Helper()
	cmd := NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = dir
	cmd.NoPreBuild = true
	//cmd.fixedTmpDir.on = true
	//cmd.fixedTmpDir.pid = 1

	// log and get output (pid, build, work dir, warnings...)
	obuf := &bytes.Buffer{}
	ebuf := &bytes.Buffer{}
	ow := iout.FnWriter(func(p []byte) (int, error) {
		t.Helper()
		t.Logf(string(p))
		return obuf.Write(p)
	})
	ew := iout.FnWriter(func(p []byte) (int, error) {
		t.Helper()
		t.Logf(string(p))
		return ebuf.Write(p)
	})
	cmd.Stdout = ow
	cmd.Stderr = ew
	bs := func(buf *bytes.Buffer) string {
		return string(buf.Bytes())
	}

	if testing.Verbose() {
		// auto add "-verbose" flag to args
		u := []string{}
		for _, a := range args {
			u = append(u, a)
			// add after main arg
			switch a {
			case "run", "build", "test":
				u = append(u, "-verbose")
			}
		}
		args = u
	}

	done, err := cmd.Start(ctx, args)
	if err != nil {
		return nil, bs(obuf), bs(ebuf), err
	}
	if done { // ex: "build", "-help"
		return nil, bs(obuf), bs(ebuf), nil
	}

	go func() {
		if err := cmd.RequestFileSetPositions(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.RequestStart(); err != nil {
			t.Fatal(err)
		}
	}()

	msgs := []string{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// util func
		add := func(s string) {
			msgs = append(msgs, s)
			t.Logf("recv: %v", s)
		}

		for msg := range cmd.Client.Messages {
			switch mt := msg.(type) {
			case *debug.LineMsg:
				add(StringifyItem(mt.Item))
			case []*debug.LineMsg:
				for _, m := range mt {
					add(StringifyItem(m.Item))
				}
			default:
				add(fmt.Sprintf("(%T)%v", msg, msg))
			}
		}
	}()

	err = cmd.Wait()
	wg.Wait()
	return msgs, bs(obuf), bs(ebuf), err
}

//------------

func doCmdSrc(t *testing.T, src string, tests bool) []string {
	t.Helper()
	msgs, err := doCmdSrc2(t, src, tests)
	if err != nil {
		t.Fatal(err)
	}
	return msgs
}

func doCmdSrc2(t *testing.T, src string, tests bool) ([]string, error) {
	t.Helper()
	msgs, _, _, err := doCmdSrc3(t, src, tests)
	return msgs, err
}

func doCmdSrc3(t *testing.T, src string, tests bool) ([]string, string, string, error) {
	t.Helper()

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename := "main.go"
	if tests {
		filename = "main_test.go"
	}

	tf.WriteFileInTmp2OrPanic(filename, src)

	// environment
	env := []string{"GO111MODULE=off"}
	envArg := strings.Join(env, string(os.PathListSeparator))

	args := []string{}
	if tests {
		args = append(args, "test")
	} else {
		args = append(args, "run")
	}
	args = append(args, []string{
		// "-h",
		//"-verbose",
		//"-work",
	}...)
	if envArg != "" {
		args = append(args, "-env="+envArg)
	}
	if !tests {
		args = append(args, filename)
	}

	ctx := context.Background()
	return doCmd3(ctx, t, tf.Dir, args)
}

//------------

func newTmpFiles(t *testing.T) *osutil.TmpFiles {
	t.Helper()
	tf := osutil.NewTmpFiles("editor_godebug_tests_tmpfiles")
	t.Logf("tf.Dir: %v\n", tf.Dir)
	return tf
}

//------------

func mustHaveString(t *testing.T, u []string, s string) {
	t.Helper()
	if !hasStringIn(s, u) {
		t.Fatalf("missing string: %v", s)
	}
}
func mustNotHaveString(t *testing.T, u []string, s string) {
	t.Helper()
	if hasStringIn(s, u) {
		t.Fatalf("contains string: %v", s)
	}
}

func hasStringIn(s string, ss []string) bool {
	for _, u := range ss {
		if u == s {
			return true
		}
	}
	return false
}

//------------

func tryGoModTidy(t *testing.T, dir string) {
	if err := goutil.GoModTidy(context.Background(), dir, nil); err != nil {
		t.Log(err)
	}
}

func runGo(t *testing.T, dir string, args []string) error {
	t.Logf("runGo")
	defer t.Logf("runGo done")
	ctx := context.Background()
	args2 := append([]string{"go"}, args...)
	cmd := osutil.NewCmd(ctx, args2...)
	cmd.Dir = dir
	//cmd.Env = env
	//bout, err := osutil.RunCmdStdoutAndStderrInErr(cmd, nil)
	bout, err := osutil.RunCmdCombinedOutput(cmd, nil)
	if err != nil {
		t.Logf("runGo stdout:\n%v", string(bout))
		return fmt.Errorf("runGo: %w (args=%v, dir=%v)", err, args, dir)
	}
	return nil
}

//------------

// use these for reproducible tests
var requireStrJmigpinEditor = "require github.com/jmigpin/editor v1.3.1-0.20201028050500-3332844d68bb"
var requireStrBurntSushi = "require github.com/BurntSushi/xgb v0.0.0-20200324125942-20f126ea2843"
var requireStrGolangXTools = "require golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e"
