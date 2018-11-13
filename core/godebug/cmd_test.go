package godebug

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmigpin/editor/core/godebug/debug"
)

func TestCmdStart1(t *testing.T) {
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

	cmd := NewCmd()
	defer cmd.Cleanup()

	ctx := context.Background()
	args := []string{"run", filename}
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
			default:
				fmt.Printf("recv msg: %v\n", msg)
				//spew.Dump(msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestCmdStart2(t *testing.T) {
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

	cmd := NewCmd()
	defer cmd.Cleanup()

	ctx := context.Background()
	args := []string{"run", filename}
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

func TestCmdStart3(t *testing.T) {
	wd, _ := os.Getwd()
	proj := filepath.Join(wd, "./../../")
	//proj := "./../../"

	filename := proj + "/editor.go"
	args := []string{
		"run",

		//"-dirs=" +
		//	proj +
		//	"," + proj + "/core" +
		//	"," + proj + "/ui",

		"-dirs=" + strings.Join([]string{
			"core",
			"ui",
		}, ","),

		filename,
	}

	cmd := NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = proj

	ctx := context.Background()
	if _, err := cmd.Start(ctx, args, nil); err != nil {
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

	nMsgs := 0
	go func() {
		for msg := range cmd.Client.Messages {
			nMsgs++
			fmt.Printf("recv msg: %v\n", msg)
			//spew.Dump(msg)
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	if nMsgs == 0 {
		t.Fatalf("nmsgs=%v", nMsgs)
	}
}

//------------

func TestCmdTest1(t *testing.T) {
	proj := "./../../util/imageutil"

	args := []string{
		"test", "-run", "HSV1",
	}

	cmd := NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = proj
	ctx := context.Background()
	if _, err := cmd.Start(ctx, args, nil); err != nil {
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
			//fmt.Printf("recv msg: %v\n", msg)
			switch t := msg.(type) {
			case *debug.LineMsg:
				fmt.Printf("%v\n", StringifyItem(t.Item))
			default:
				fmt.Printf("recv msg: %v\n", msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestCmdTest2(t *testing.T) {
	proj := "./../../util/uiutil/widget/textutil"

	args := []string{
		"test",
		//"test", "-run", "HSV1",
	}

	cmd := NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = proj
	ctx := context.Background()
	if _, err := cmd.Start(ctx, args, nil); err != nil {
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
			//fmt.Printf("recv msg: %v\n", msg)
			switch t := msg.(type) {
			case *debug.LineMsg:
				fmt.Printf("%v\n", StringifyItem(t.Item))
			default:
				fmt.Printf("recv msg: %v\n", msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}
