package godebug

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/testutil"
)

func TestScripts(t *testing.T) {
	scr := testutil.NewScript(os.Args)

	// uncomment to access work dir
	//scr.Work = true

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
		for msg := range cmd.Messages() {
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
