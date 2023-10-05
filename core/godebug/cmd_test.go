package godebug

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmigpin/editor/util/testutil"

	////godebug:annotateimport
	"github.com/jmigpin/editor/core/godebug/debug"
)

func TestCmd(t *testing.T) {
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

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		// util func
		add := func(s string) {
			fmt.Printf("recv: %v\n", s)
		}
		for {
			msg, ok, err := cmd.ProtoRead()
			if err != nil {
				fmt.Printf("godebugtester msg loop error: %v", err)
				break
			}
			if !ok {
				break
			}

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
	return err
}

//----------

func TestCmdCtxCancel(t *testing.T) {
	// max time to run this test
	timer := time.AfterFunc(1000*time.Millisecond, func() {
		t.Fatal("test timeout")
	})
	defer timer.Stop()

	cmd := NewCmd()
	args := []string{
		"connect",
		//"-network=ws", // requires websocket client
		"-editorisserver=true",
		//"-addr=:8079",
	}

	ctx := context.Background()
	ctx2, cancel := context.WithCancel(ctx)

	go func() {
		time.Sleep(250 * time.Millisecond)
		cancel()
	}()

	_, err := cmd.Start(ctx2, args)
	if err == nil || err.Error() != "context canceled" {
		t.Fatal(err)
	}
	t.Log(err)
}
