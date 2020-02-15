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
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
)

//godebug:annotatepackage

func init() {
	SimplifyStringifyItem = false
}

//------------

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
	msgs := doCmdSrc(t, src, false, false)
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
	msgs := doCmdSrc(t, src, false, false)
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
	msgs := doCmdSrc(t, src, false, false)
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
	msgs := doCmdSrc(t, src, false, false)
	mustHaveString(t, msgs, "false=(10 < 10)")
}

func TestCmd_src4(t *testing.T) {
	src := `
		package main
		import "testing"
		func Test001(t*testing.T){
			_=1
		}
	`
	msgs := doCmdSrc(t, src, true, false)
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
	msgs := doCmdSrc(t, src, false, false)
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
	msgs := doCmdSrc(t, src, false, false)
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
	_, err := doCmdSrc2(t, src, false, false)
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
	_, _, es, err := doCmdSrc3(t, src, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !(strings.Index(es, "# warning") >= 0 &&
		strings.Index(es, "not_used_here") >= 0) {
		t.Fatal("missing warning")
	}
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
		//"-work",
		"main.go",
	}
	msgs := doCmd(t, dir1, cmd)
	mustNotHaveString(t, msgs, `"F1"`)
	mustHaveString(t, msgs, `"F2"`)
}

func TestCmd_goMod3(t *testing.T) {
	// func call should not be annotated: pkg with only one godebug annotate block inside another func

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
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
	// TODO: This is failing at pre-build?

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
		//"-verbose",
		"-env=GOPROXY=off",
		"main.go",
	}
	_, err := doCmd2(t, dir, cmd)
	if err == nil {
		t.Fatal("expecting error")
	}
}

func TestCmd_goMod9(t *testing.T) {
	// if the os env doesn't have GOPROXY=off, having no go.mod should fetch the dependencies from the web at pre-build.

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
		//"-verbose",
		"-env=GO111MODULE=on", // force modules mode (no go.mod)
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `4=((4=(1 + 3)) & -4=^3)`)
	mustHaveString(t, msgs, `[48 48 49]`)
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
		//"-verbose",
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
		//"-verbose",
		"main.go",
	}
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
		//"-verbose",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustNotHaveString(t, msgs, `4=((4=(1 + 3)) & -4=^3)`)
	mustHaveString(t, msgs, `map[]=["MIT-SHM"] := map[]=make(type)`)
}

func TestCmd_goMod13(t *testing.T) {
	// annotate full external module (slow)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("main/go.mod", `
		module main
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
		//"-verbose",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `4=((4=(1 + 3)) & -4=^3)`)
	mustHaveString(t, msgs, `map[]=["MIT-SHM"] := map[]=make(type)`)
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
		//"-verbose",
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
		//"-verbose",
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
		//"-verbose",
		"main.go",
	}
	msgs := doCmd(t, dir, cmd)
	mustHaveString(t, msgs, `"Fa"`)
	mustHaveString(t, msgs, `=> GoPath()`)
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
		//"-verbose",
		//"-work",
		"-env=GOPATH=" + tf.Dir,
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
		//"-verbose",
		//"-work",
		"-env=GOPATH=" + tf.Dir,
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
		//"-verbose",
		"-env=GOPATH=" + filepath.Join(tf.Dir, "w"),
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
		func main() {
			_=1
		}
	`)

	cmd := []string{
		"run",
		//"-verbose",
		//"-work",
		"dir1/main.go", // give location to run
	}
	msgs := doCmd(t, tf.Dir, cmd)
	mustHaveString(t, msgs, `_ := 1`)
}

func TestCmd_simple2(t *testing.T) {
	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	tf.WriteFileInTmp2OrPanic("dir1/main.go", `
		package main
		func main() {
			_=1
		}
	`)

	cmd := []string{
		"build",
		"dir1/main.go",
		"-tags=aaa",
	} // some arg after the filename
	_, err := doCmd2(t, tf.Dir, cmd)
	if err != nil {
		t.Fatal(err) // ok - just be able to build
	}
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
//		//"-verbose",
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
	_ = doCmdSrc(t, src, false, false)
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
	//cmd.FixedTmpDir = true
	//cmd.FixedTmpDirPid = 1

	// log and get output (pid, build, work dir, warnings...)
	obuf := &bytes.Buffer{}
	ebuf := &bytes.Buffer{}
	ow := iout.FnWriter(func(p []byte) (int, error) {
		t.Log(string(p))
		return obuf.Write(p)
	})
	ew := iout.FnWriter(func(p []byte) (int, error) {
		t.Log(string(p))
		return ebuf.Write(p)
	})
	cmd.Stdout = ow
	cmd.Stderr = ew
	bs := func(buf *bytes.Buffer) string {
		return string(buf.Bytes())
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

func doCmdSrc(t *testing.T, src string, tests bool, noModules bool) []string {
	t.Helper()
	msgs, err := doCmdSrc2(t, src, tests, noModules)
	if err != nil {
		t.Fatal(err)
	}
	return msgs
}

func doCmdSrc2(t *testing.T, src string, tests bool, noModules bool) ([]string, error) {
	t.Helper()
	msgs, _, _, err := doCmdSrc3(t, src, tests, noModules)
	return msgs, err
}

func doCmdSrc3(t *testing.T, src string, tests bool, noModules bool) ([]string, string, string, error) {
	t.Helper()

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename := "main.go"
	if tests {
		filename = "main_test.go"
	}

	tf.WriteFileInTmp2OrPanic(filename, src)

	// environment
	env := []string{}
	if noModules {
		env = append(env, "GO111MODULE=off")
	}
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
