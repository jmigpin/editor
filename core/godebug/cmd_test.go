package godebug

import (
	"context"
	"errors"
	"fmt"
	"io"
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

//godebug:annotatefile:debug/proto.go
//godebug:annotatepackage:github.com/jmigpin/editor/util/testutil

func TestCmd1(t *testing.T) {
	scr := testutil.NewScript(os.Args)
	scr.ScriptsDir = "testdata"
	scr.Cmds = []*testutil.ScriptCmd{
		{"godebugtester", godebugTester},
	}
	scr.Run(t)
}
func godebugTester(t *testing.T, st *testutil.ScriptTest, args []string) error {
	log.SetFlags(0)
	log.SetPrefix("godebugtester: ")

	args = args[1:] // clear "godebugtester"

	cmd := NewCmd()
	cmd.Testing = true

	cmd.Dir = st.CurDir
	cmd.env = st.Env.Environ()
	cmd.Stdout = os.Stdout

	//----------
	// TODO: tests failing without this; this prevents running tests in parallel
	tmp, _ := os.Getwd()
	defer func() { _ = os.Chdir(tmp) }()
	_ = os.Chdir(st.CurDir)
	//----------

	ctx := context.Background()
	done, err := cmd.Start(ctx, args)
	if err != nil {
		return err
	}
	if done { // ex: "build", "-help"
		return nil
	}

	//----------

	pr := func(s string) { // util func
		//st.Logf(t, "recv: %v\n", s)
		fmt.Printf("recv: %v\n", s)
	}
	for {
		v, err := cmd.ProtoRead()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
			break
		}

		switch t := v.(type) {
		case *debug.FilesDataMsg:
			pr(fmt.Sprintf("%#v", t))

			// DEBUG: can make some tests fail if active: ex: will print filenames that can match output that should not be seen
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
	//}()
	//return nil
	//}
	//go func() {
	//	if err := fn(); err != nil {
	//		fmt.Printf("error: %v\n", err)
	//	}
	//}()

	return cmd.Wait()
}

//----------
//----------
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
		"-nodebugmsg",
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

//----------

func TestCmd3Reconnect(t *testing.T) {
	ctx := context.Background()
	addr := debug.NewAddrI("tcp", ":9158")
	isServer := true

	running := sync.WaitGroup{}

	//----------

	ctx2, cancel2 := context.WithCancel(ctx)
	//ctx2 := ctx

	nClients := 3
	nMsgs := 2

	running.Add(1)
	go func() {
		defer running.Done()
		runServer(t, ctx2, addr, isServer, nClients, nClients*(1+nMsgs))
	}()

	//----------

	running.Add(1)
	go func() {
		defer running.Done()

		for nc := 0; nc < nClients; nc++ {
			runClient(t, ctx, addr, !isServer, nMsgs)
			time.Sleep(500 * time.Millisecond) // wait to avoid the server getting the close before some message

			//go runClient(t, ctx, addr, !isServer, nMsgs) // DEBUG
		}

		time.Sleep(1000 * time.Millisecond)
		cancel2() // stop server
	}()

	//----------

	running.Wait()
}

func runServer(t *testing.T, ctx context.Context, addr debug.Addr, isServer bool, nClients, nMsgs int) {
	tw := &TestWriter{t: t}
	cmd := NewCmd()
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	cmd.Stdout = tw
	cmd.Stderr = tw
	args := []string{
		"connect", // just a connect (might have no timeouts set)
		"-addr=" + addr.String(),
		"-editorisserver=" + strconv.FormatBool(isServer),
		"-continueserving",
		//"-nodebugmsg",
	}
	_, err := cmd.Start(ctx, args)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	//nErrs := 0
	for {
		v, err := cmd.ProtoRead()
		if err != nil {
			if errors.Is(err, io.EOF) ||
				errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded) {
				t.Log(err)
				break
				//nErrs++
				//if nErrs == nClients {
				//	break
				//}
				//continue
			}

			t.Fatal(err)
		}
		count++
		t.Logf("<- %T\n", v)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	//if nErrs != nClients {
	//	t.Fatalf("expecting %v clients, got %v", nClients, nErrs)
	//}
	if count != nMsgs {
		t.Fatalf("expecting %v msgs, got %v", nMsgs, count)
	}
}
func runClient(t *testing.T, ctx context.Context, addr debug.Addr, isServer bool, nMsgs int) {
	tw := &TestWriter{t: t}
	//logw := debug.NewPrefixWriter(os.Stderr, "# godebug.exec: ")
	logw := debug.NewPrefixWriter(tw, "# godebug.exec: ")

	fd := &debug.FilesDataMsg{Data: nil}

	//for i := 0; i < 10; i++ {
	//	afd := &debug.AnnotatorFileData{
	//		FileIndex:   0,
	//		NMsgIndexes: 0,
	//		Filename:    "a.go",
	//	}
	//	fd.Data = append(fd.Adata, afd)
	//}

	pexs := &debug.ProtoExecSide{FData: fd}
	p, err := debug.NewProto(ctx, addr, pexs, isServer, false, logw)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("abc")
	//msgOut := "[97 98 99]"
	lineMsg := &debug.OffsetMsg{Item: debug.IVi(msg)}
	//for i := 0; i < 10000; i++ {
	for i := 0; i < nMsgs; i++ {
		//if err := p.WriteMsg(lineMsg); err != nil {
		//t.Logf("-> %T\n", lineMsg)
		if err := p.Write(lineMsg); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.CloseOrWait(); err != nil {
		t.Fatal(err)
	}
}

//----------
//----------
//----------

type TestWriter struct {
	t *testing.T
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.t.Logf("%s", string(p))
	return len(p), nil
}
