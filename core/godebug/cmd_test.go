package godebug

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmigpin/editor/util/testutil"

	//godebug:annotateimport
	"github.com/jmigpin/editor/core/godebug/debug"
)

func TestCmd1(t *testing.T) {
	scr := testutil.NewScript(os.Args)
	scr.ScriptsDir = "testdata"
	scr.Cmds = []*testutil.ScriptCmd{
		{"godebugtester", godebugTester},
	}
	scr.Run(t)
}
func godebugTester(t *testing.T, args []string) error {
	log.SetFlags(0)
	log.SetPrefix("godebugtester: ")

	args = args[1:]

	cmd := NewCmd()

	dir, _ := os.Getwd()
	cmd.Dir = dir

	ctx := context.Background()
	done, err := cmd.Start(ctx, args)
	if err != nil {
		return err
	}
	if done { // ex: "build", "-help"
		return nil
	}

	fn := func() error {
		pr := func(s string) { // util func
			fmt.Printf("recv: %v\n", s)
		}

		for {
			msg, ok, err := cmd.ProtoRead()
			if err != nil {
				return err
			}
			if !ok {
				break
			}

			switch t := msg.(type) {
			case *debug.LineMsg:
				pr(StringifyItem(t.Item))
			case *debug.LineMsgs:
				for _, m := range *t {
					pr(StringifyItem(m.Item))
				}
			default:
				return fmt.Errorf("unexpected type: %T, %v", msg, msg)
			}
		}
		return nil
	}

	ch := make(chan any)
	go func() {
		ch <- fn()
	}()
	if v := <-ch; v != nil {
		return v.(error)
	}

	return cmd.Wait()
}

//----------

func TestCmd2CtxCancel(t *testing.T) {
	// max time to run this test
	timer := time.AfterFunc(1000*time.Millisecond, func() {
		t.Fatal("test timeout")
	})
	defer timer.Stop()

	//----------

	cmd := NewCmd()
	args := []string{
		"connect", // just a connect (might have no timeouts set)
		"-editorisserver=true",
	}

	ctx := context.Background()
	ctx2, cancel := context.WithCancel(ctx)

	go func() {
		time.Sleep(250 * time.Millisecond)
		cancel()
	}()

	_, err := cmd.Start(ctx2, args)
	if err == nil {
		t.Fatal("got no error")
	}
	if strings.Index(err.Error(), "context canceled") < 0 {
		t.Fatal(err)
	}
	t.Log(err)
}
