package godebug

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmigpin/editor/util/testutil"

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
	cmd.Testing = true

	dir, _ := os.Getwd()
	cmd.Dir = dir
	cmd.Stdout = os.Stdout

	ctx := context.Background()
	done, err := cmd.Start(ctx, args)
	if err != nil {
		return err
	}
	if done { // ex: "build", "-help"
		return nil
	}

	//----------

	fn := func() error {
		pr := func(s string) { // util func
			fmt.Printf("recv: %v\n", s)
		}
		for {
			v, err, ok := cmd.ProtoRead()
			if !ok {
				break
			}
			if err != nil {
				t.Log(err)
				continue
			}

			switch t := v.(type) {
			case *debug.FilesDataMsg:
				pr(fmt.Sprintf("%#v", t))
				//for i, afd := range t.Data {
				//	pr(fmt.Sprintf("%v: %#v", i, afd))
				//}
			case *debug.OffsetMsg:
				pr(StringifyItem(t.Item))
			case *debug.OffsetMsgs:
				for _, m := range *t {
					pr(StringifyItem(m.Item))
				}
			default:
				return fmt.Errorf("unexpected type: %T, %v", v, v)
			}
		}
		return nil
	}
	go func() {
		if err := fn(); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}()

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
		"-addr=:9158",
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

func TestCmd3Reconnect(t *testing.T) {
	ctx := context.Background()
	addr := debug.NewAddrI("tcp", ":9158")
	isServer := true

	running := sync.WaitGroup{}

	//----------

	ctx2, cancel2 := context.WithCancel(ctx)

	nClients := 3
	nMsgs := 2

	running.Add(1)
	go func() {
		defer running.Done()

		cmd := NewCmd()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		args := []string{
			"connect", // just a connect (might have no timeouts set)
			"-addr=" + addr.String(),
			"-editorisserver=" + strconv.FormatBool(isServer),
			"-continueserving",
		}
		_, err := cmd.Start(ctx2, args)
		if err != nil {
			t.Fatal(err)
		}

		count := 0
		for {
			v, err, ok := cmd.ProtoRead()
			if !ok {
				break
			}
			if err != nil {
				t.Log(err)
				continue
			}
			count++
			t.Logf("<- %T\n", v)
		}

		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}

		if n := nClients * nMsgs; count != n {
			t.Fatalf("expecting %v msgs, got %v", n, count)
		}
	}()

	//----------

	running.Add(1)
	go func() {
		defer running.Done()

		for nc := 0; nc < nClients; nc++ {
			//fd := &debug.FilesDataMsg{Data: nil}
			pexs := &debug.ProtoExecSide{}
			p, err := debug.NewProto(ctx, addr, pexs, !isServer, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			msg := []byte("abc")
			//msgOut := "[97 98 99]"
			lineMsg := &debug.OffsetMsg{Item: debug.IVi(msg)}
			//for i := 0; i < 10000; i++ {
			for i := 0; i < nMsgs; i++ {
				//if err := p.WriteMsg(lineMsg); err != nil {
				t.Logf("-> %T\n", lineMsg)
				if err := p.Write(lineMsg); err != nil {
					t.Fatal(err)
				}
			}
			if err := p.CloseOrWait(); err != nil {
				t.Fatal(err)
			}
		}

		time.Sleep(1 * time.Second)
		cancel2() // stop server
	}()

	//----------

	running.Wait()
}
