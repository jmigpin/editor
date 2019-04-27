package godebug

import (
	"context"
	"fmt"
	"log"
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
	filename := "test/src.go"
	args := []string{"run", filename}

	doCmd(t, "", src, args)
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

	filename := "test/src.go"
	args := []string{"run", filename}

	doCmd(t, "", src, args)
}

func TestCmd3(t *testing.T) {
	wd, _ := os.Getwd()
	proj := filepath.Join(wd, "./../../")
	filename := proj + "/editor.go"
	args := []string{
		"run",
		"-dirs=core",
		filename,
	}
	doCmd(t, proj, nil, args)
}

func TestCmd4(t *testing.T) {
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

	filename := "test/src.go"
	args := []string{"run", filename}

	doCmd(t, "", src, args)
}

//------------

func TestCmdFile1(t *testing.T) {
	proj := "./../../util/imageutil"
	args := []string{"test", "-run", "HSV1"}
	doCmd(t, proj, nil, args)
}

func TestCmdFile2(t *testing.T) {
	proj := "./../../util/imageutil"
	args := []string{"test", "-run", "HSV1"}
	doCmd(t, proj, nil, args)
}

func TestCmdFile3(t *testing.T) {
	proj := "./../../util/uiutil/widget/textutil"
	args := []string{"test"}
	doCmd(t, proj, nil, args)
}

func TestCmdFile4(t *testing.T) {
	proj := "./../.."
	args := []string{"run", "-dirs=driver/xgbutil/xwindow", "editor.go"}
	doCmd(t, proj, nil, args)
}

//------------

func doCmd(t *testing.T, dir string, src interface{}, args []string) {
	log.SetFlags(log.Lshortfile)
	t.Logf("DISABLED")
	//doCmd2(t, dir, src, args)
}

func doCmd2(t *testing.T, dir string, src interface{}, args []string) {
	cmd := NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = dir

	ctx := context.Background()
	if _, err := cmd.Start(ctx, args, src); err != nil {
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
			switch t := msg.(type) {
			case *debug.LineMsg:
				fmt.Printf("%v\n", StringifyItem(t.Item))
				//spew.Dump(msg)
			default:
				fmt.Printf("recv msg: %v\n", msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}
