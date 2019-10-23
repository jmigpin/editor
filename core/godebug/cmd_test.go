package godebug

import (
	"context"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
)

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
	msgs := doCmdGoPathSrc(t, src, false)
	if !hasStringIn("Sleep(10ms=(10 * 1ms))", msgs) {
		t.Fatal()
	}
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
	msgs := doCmdGoPathSrc(t, src, false)
	if !hasStringIn("_ := 20 []=f2()", msgs) {
		t.Fatal()
	}
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
	msgs := doCmdGoPathSrc(t, src, false)
	if !hasStringIn("false=(10 < 10)", msgs) {
		t.Fatal()
	}
}

func TestCmd_src4(t *testing.T) {
	src := `
		package main
		import "testing"
		import "github.com/jmigpin/editor/core/godebug/debug"
		func Test001(t*testing.T){
			debug.NoAnnotations()
			for i:=0; i<2;i++{
				debug.AnnotateBlock()
				_=i
			}
		}
	`
	msgs := doCmdGoPathSrc(t, src, true)
	if !hasStringIn("_ := 1", msgs) {
		t.Fatal()
	}
}

//------------

func TestCmd_goMod5(t *testing.T) {
	tmpDir := createTmpDir(t)
	defer os.RemoveAll(tmpDir)
	t.Logf("tmpDir: %v\n", tmpDir)

	createFilesForTestCmd_goMod5(t, tmpDir)

	dir1 := filepath.Join(tmpDir, "main")
	cmd := []string{"run", "main.go"}
	//cmd := []string{"run", "-verbose", "-work", "main.go"}
	msgs := doCmd(t, dir1, cmd)

	if hasStringIn("F1", msgs) { // must not be annotated
		t.Fatal(msgs)
	}
	if !hasStringIn("F2F1=(F2 + F1=F1())", msgs) { // must be annotated
		t.Fatal(msgs)
	}
}

func createFilesForTestCmd_goMod5(t *testing.T, tmpDir string) {
	// not in gopath
	// has go.mod
	// pkg1 is in relative dir, not annotated
	// pkg2 is in relative dir, annotated, depends on pkg1

	mainMainGo := `
		package main
		import "example.com/pkg1"
		import "example.com/pkg2"
		func main() {
			_=pkg1.F1()
			_=pkg2.F2()
		}
	`
	mainGoMod := `
		module main
		replace example.com/pkg1 => ../pkg1
		replace example.com/pkg2 => ../pkg2
	`
	pkg1F1Go := `
		package pkg1
		func F1() string {
			return "F1"
		}
	`
	pkg1GoMod := `
		module example.com/pkg1
	`
	pkg2F2Go := `
		package pkg2
		import "example.com/pkg1"
		import "github.com/jmigpin/editor/core/godebug/debug"
		func F2() string {
			debug.AnnotateBlock()
			return "F2"+pkg1.F1()
		}
	`
	pkg2GoMod := `
		module example.com/pkg2
		replace example.com/pkg1 => ../pkg1
	`

	createTmpFileFromSrc2(t, tmpDir, "main/main.go", mainMainGo)
	createTmpFileFromSrc2(t, tmpDir, "main/go.mod", mainGoMod)
	createTmpFileFromSrc2(t, tmpDir, "pkg1/f1.go", pkg1F1Go)
	createTmpFileFromSrc2(t, tmpDir, "pkg1/go.mod", pkg1GoMod)
	createTmpFileFromSrc2(t, tmpDir, "pkg2/f2.go", pkg2F2Go)
	createTmpFileFromSrc2(t, tmpDir, "pkg2/go.mod", pkg2GoMod)
}

//------------

func TestCmd_goMod6(t *testing.T) {
	tmpDir := createTmpDir(t)
	defer os.RemoveAll(tmpDir)
	t.Logf("tmpDir: %v\n", tmpDir)

	createFilesForTest_gomod6(t, tmpDir)

	dir1 := filepath.Join(tmpDir, "main")
	cmd := []string{"run", "main.go"}
	//cmd := []string{"run", "-verbose", "-work", "main.go"}
	msgs := doCmd(t, dir1, cmd)

	if hasStringIn("F1", msgs) { // must not be annotated
		t.Fatal(msgs)
	}
	if !hasStringIn("F2", msgs) { // must be annotated
		t.Fatal(msgs)
	}
}

func createFilesForTest_gomod6(t *testing.T, tmpDir string) {
	// not in gopath
	// has go.mod
	// pkg1 is in relative dir, not annotated
	// pkg2 is in abs dir, annotated

	mainMainGo := `
		package main
		import "example.com/pkg1"
		import "example.com/pkg2"
		func main() {
			_=pkg1.F1()
			_=pkg2.F2()
		}
	`
	mainGoMod := `
		module main
		replace example.com/pkg1 => ../pkg1
		replace example.com/pkg2 => ` + filepath.Join(tmpDir, "pkg2") + `
	`
	pkg1F1Go := `
		package pkg1
		func F1() string {
			return "F1"
		}
	`
	pkg1GoMod := `
		module example.com/pkg1
	`
	pkg2F2Go := `
		package pkg2
		import "github.com/jmigpin/editor/core/godebug/debug"
		func F2() string {
			debug.AnnotateBlock()
			return "F2"
		}
	`
	pkg2GoMod := `
		module example.com/pkg2
	`

	createTmpFileFromSrc2(t, tmpDir, "main/main.go", mainMainGo)
	createTmpFileFromSrc2(t, tmpDir, "main/go.mod", mainGoMod)
	createTmpFileFromSrc2(t, tmpDir, "pkg1/f1.go", pkg1F1Go)
	createTmpFileFromSrc2(t, tmpDir, "pkg1/go.mod", pkg1GoMod)
	createTmpFileFromSrc2(t, tmpDir, "pkg2/f2.go", pkg2F2Go)
	createTmpFileFromSrc2(t, tmpDir, "pkg2/go.mod", pkg2GoMod)
}

//------------

func TestCmd_goMod7_test(t *testing.T) {
	tmpDir := createTmpDir(t)
	defer os.RemoveAll(tmpDir)
	t.Logf("tmpDir: %v\n", tmpDir)

	createFilesForTestCmd_gomod7_test(t, tmpDir)

	dir1 := filepath.Join(tmpDir, "main")
	cmd := []string{"test"}
	//cmd := []string{"test", "-verbose", "-work"}
	msgs := doCmd(t, dir1, cmd)

	if hasStringIn("F1", msgs) { // must not be annotated
		t.Fatal(msgs)
	}
	if !hasStringIn("F2", msgs) { // must be annotated
		t.Fatal(msgs)
	}
}

func createFilesForTestCmd_gomod7_test(t *testing.T, tmpDir string) {
	mainMainTestsGo := `
		package main
		import "testing"
		import "example.com/pkg1"
		import "example.com/pkg2"
		func Test01(t*testing.T) {
			_=pkg1.F1()
			_=pkg2.F2()
		}
	`
	mainGoMod := `
		module example.com/main
		replace example.com/pkg1 => ../pkg1
		replace example.com/pkg2 => ../pkg2
	`
	pkg1F1Go := `
		package pkg1
		func F1() string {
			return "F1"
		}
	`
	pkg1GoMod := `
		module example.com/pkg1
	`
	pkg2F2Go := `
		package pkg2
		import "github.com/jmigpin/editor/core/godebug/debug"
		func F2() string {
			debug.AnnotateBlock()
			return "F2"
		}
	`
	pkg2GoMod := `
		module example.com/pkg2
	`

	//createTmpFileFromSrc2(t, tmpDir, "main/main.go", mainMainGo)
	createTmpFileFromSrc2(t, tmpDir, "main/main_test.go", mainMainTestsGo)
	createTmpFileFromSrc2(t, tmpDir, "main/go.mod", mainGoMod)
	createTmpFileFromSrc2(t, tmpDir, "pkg1/f1.go", pkg1F1Go)
	createTmpFileFromSrc2(t, tmpDir, "pkg1/go.mod", pkg1GoMod)
	createTmpFileFromSrc2(t, tmpDir, "pkg2/f2.go", pkg2F2Go)
	createTmpFileFromSrc2(t, tmpDir, "pkg2/go.mod", pkg2GoMod)
}

//------------

func TestCmd_goPath1(t *testing.T) {
	tmpDir := createTmpDir(t)
	defer os.RemoveAll(tmpDir)
	t.Logf("tmpDir: %v\n", tmpDir)

	createFilesForTestCmd_goPath1(t, tmpDir)

	dir := filepath.Join(tmpDir, "src/main")
	cmd := []string{"run", "main.go"}
	//cmd := []string{"run", "-verbose", "-work", "main.go"}
	msgs := doCmd2(t, dir, cmd, true, tmpDir)

	if hasStringIn("sub1", msgs) { // not annotated
		t.Fatal(msgs)
	}
	if !hasStringIn("sub2", msgs) { // annotated
		t.Fatal(msgs)
	}
}

func createFilesForTestCmd_goPath1(t *testing.T, tmpDir string) {
	mainMainGo := `
		package main
		import "main/sub1"
		import "main/sub1/sub2"
		import "main/sub3"
		func main() {
			_=sub1.Sub1()
			_=sub2.Sub2()
			_=sub3.Sub3()
		}
	`
	mainSub1Sub1Go := `
		package sub1
		func Sub1() string {
			return "sub1"
		}
	`
	mainSub1Sub2Sub2Go := `
		package sub2
		import "github.com/jmigpin/editor/core/godebug/debug"
		func Sub2() string {
			debug.AnnotateBlock()
			return "sub2"
		}
	`
	mainSub3Sub3Go := `
		package sub3
		func Sub3() string {
			return "sub3"
		}
	`
	createTmpFileFromSrc2(t, tmpDir, "src/main/main.go", mainMainGo)
	createTmpFileFromSrc2(t, tmpDir, "src/main/sub1/sub1.go", mainSub1Sub1Go)
	createTmpFileFromSrc2(t, tmpDir, "src/main/sub1/sub2/sub2.go", mainSub1Sub2Sub2Go)
	createTmpFileFromSrc2(t, tmpDir, "src/main/sub3/sub3.go", mainSub3Sub3Go)
}

//----------

func TestCmd_simple1(t *testing.T) {
	tmpDir := createTmpDir(t)
	defer os.RemoveAll(tmpDir)
	t.Logf("tmpDir: %v\n", tmpDir)

	createFilesForTestCmd_simple1(t, tmpDir)

	dir := filepath.Join(tmpDir, "dir1")
	cmd := []string{"run", "main.go"}
	//cmd := []string{"run", "-verbose", "-work", "main.go"}
	msgs := doCmd2(t, dir, cmd, false, tmpDir)

	if !hasStringIn("2 := 2", msgs) { // annotated
		t.Fatal(msgs)
	}
}

func createFilesForTestCmd_simple1(t *testing.T, tmpDir string) {
	mainMainGo := `
		package main
		import "github.com/jmigpin/editor/core/godebug/debug"
		func main() {
			a:=1
			b:=2
			debug.NoAnnotations()
			_=a+b
		}
	`
	createTmpFileFromSrc2(t, tmpDir, "dir1/main.go", mainMainGo)
}

//------------

//func TestCmd_simple2(t *testing.T) {
//	tmpDir := createTmpDir(t)
//	defer os.RemoveAll(tmpDir)
//	t.Logf("tmpDir: %v\n", tmpDir)

//	createFilesForTestCmd_simple2(t, tmpDir)

//	dir := filepath.Join(tmpDir, "dir1")
//	cmd := []string{"run", "main.go"}
//	//cmd := []string{"run", "-verbose", "-work", "main.go"}
//	msgs := doCmd2(t, dir, cmd, false, tmpDir)

//	if !hasStringIn("2 := 2", msgs) { // annotated
//		t.Fatal(msgs)
//	}
//}

//func createFilesForTestCmd_simple2(t *testing.T, tmpDir string) {
//	mainMainGo := `
//		package main
//		import "time"
//		func main() {
//			time.Sleep(time.Second)
//		}
//	`
//	createTmpFileFromSrc2(t, tmpDir, "dir1/main.go", mainMainGo)
//}

//------------

// Launches the editor itself.
func TestCmd2_editor(t *testing.T) {
	filename := "./../../editor.go"
	args := []string{
		"run",
		"-verbose", "-work",
		//"-dirs=../../core,../../core/contentcmds",
		//"-dirs=../../core",
		//"-dirs=",
		filename,
	}
	doCmd(t, "", args)
}

//------------

func doCmd(t *testing.T, dir string, args []string) []string {
	return doCmd2(t, dir, args, false, "")
}

func doCmd2(t *testing.T, dir string, args []string, noModules bool, goPathDir string) []string {
	t.Helper()

	cmd := NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = dir
	cmd.NoModules = noModules

	if noModules && goPathDir != "" {
		// ensure the directory (possibly just created on tmp) is in gopath for tests to be able to find non-annotated files not copied to the tmp dir
		prependToGoPathAndUpdateGoBuildPkg(goPathDir)
	}

	ctx := context.Background()
	if _, err := cmd.Start(ctx, args); err != nil {
		t.Fatal(err)
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
				add(fmt.Sprintf("%T %v", msg, msg))
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	return msgs
}

//------------

func doCmdGoPathSrc(t *testing.T, src string, tests bool) []string {
	filename := "main.go"
	if tests {
		filename = "main_test.go"
	}
	tmpFile, tmpDir := createTmpFileFromSrc(t, filename, src)
	defer os.RemoveAll(tmpDir)

	args := []string{"run", tmpFile}
	if tests {
		args = []string{"test"} // no file
	}
	//args = append(args, "-work")
	noModules := true
	return doCmd2(t, tmpDir, args, noModules, "")
}

//------------

func createTmpFileFromSrc2(t *testing.T, tmpDir, filename, src string) string {
	fname := filepath.Join(tmpDir, filename)
	baseDir := filepath.Dir(fname)
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		t.Fatal(t)
	}
	if err := ioutil.WriteFile(fname, []byte(src), 0660); err != nil {
		t.Fatal(err)
	}
	return fname
}

//------------

func createTmpFileFromSrc(t *testing.T, filename, src string) (string, string) {
	tmpDir := createTmpDir(t)
	tmpFile := createTmpFile(t, tmpDir, filename, src)
	return tmpFile, tmpDir
}

func createTmpDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "editor_godebug_tests")
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func createTmpFile(t *testing.T, dir, filename, src string) string {
	f := filepath.Join(dir, filename)
	if err := ioutil.WriteFile(f, []byte(src), 0660); err != nil {
		t.Fatal(err)
	}
	return f
}

//------------

func prependToGoPathAndUpdateGoBuildPkg(dir string) {
	goPaths := append([]string{dir}, goutil.GoPath()...)
	goPath := strings.Join(goPaths, string(os.PathListSeparator))
	os.Setenv("GOPATH", goPath)
	// Update "build" path since it is set at init time.
	// This is then needed by: github.com/jmigpin/editor/util/goutil/misc.go:34
	build.Default.GOPATH = goPath
}

//------------

func hasStringIn(s string, ss []string) bool {
	for _, u := range ss {
		if u == s {
			return true
		}
	}
	return false
}
