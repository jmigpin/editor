package godebug

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmigpin/editor/core/godebug/debug"
)

func TestCmd1(t *testing.T) {
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
	doCmdSrc(t, src, false)
}

func TestCmd2(t *testing.T) {
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
	doCmdSrc(t, src, false)
}

func TestCmd3(t *testing.T) {
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
	doCmdSrc(t, src, false)
}

func TestCmd4(t *testing.T) {
	src := `
		package main
		import "testing"
		import "github.com/jmigpin/editor/core/godebug/debug"
		func Test001(t*testing.T){
			debug.NoAnnotations()
			println("on testcmd4")
			for i:=0; i<2;i++{
				debug.AnnotateBlock()
				println("i=",i)
			}
		}
	`
	doCmdSrc(t, src, true)
}

//------------

// Launches the editor itself.
//func TestCmd5(t *testing.T) {
//	filename := "./../../editor.go"
//	args := []string{
//		"run",
//		"-dirs=../../core",
//		filename,
//	}
//	doCmd(t, "", args)
//}

//------------

func doCmdSrc(t *testing.T, src string, tests bool) {
	filename := "main.go"
	if tests {
		filename = "main_test.go"
	}
	tmpFile, tmpDir := createTmpFileFromSrc(t, filename, src)
	defer os.RemoveAll(tmpDir)
	args := []string{"run", tmpFile}
	if tests {
		args = []string{"test"} // no file
		//args = []string{"test", "-work"} // no file
	}
	doCmd(t, tmpDir, args)
}

func doCmd(t *testing.T, dir string, args []string) {
	cmd := NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = dir

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

	go func() {
		for msg := range cmd.Client.Messages {
			switch mt := msg.(type) {
			case *debug.LineMsg:
				t.Logf("line msg: %v\n", StringifyItem(mt.Item))
			case []*debug.LineMsg:
				for _, m := range mt {
					t.Logf("line msg: %v\n", StringifyItem(m.Item))
				}
			default:
				t.Logf("recv msg: %T %v\n", msg, msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

//------------

func createTmpFileFromSrc(t *testing.T, filename, src string) (string, string) {
	tmpDir := createTmpDir(t)
	tmpFile := createTmpFile(t, tmpDir, filename, src)
	return tmpFile, tmpDir
}

func createTmpDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "godebug_tests")
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
