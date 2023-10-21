package debug

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"
)

func TestProto1(t *testing.T) {
	ctx := context.Background()

	addr := NewAddrI("tcp", ":12080")
	//addr := NewAddrI("unix", "/tmp/editor.sock1")
	//addr := NewAddrI("ws", ":12080") // needs go tag editorDebugExecSide to have ws client

	tlogf(t, "addr: %v, %q\n", addr.Network(), addr.String())

	editorIsServer := true
	//editorIsServer := false
	ch := make(chan any)

	msg := []byte("abcdefg")
	msgOut := "[97 98 99 100 101 102 103]"
	lineMsg := &OffsetMsg{Item: IVi(msg)}

	// accept
	go func() {
		eds := &ProtoEditorSide{}
		eds.logOn = testing.Verbose()
		dp1 := NewProto(ctx, addr, eds, editorIsServer, false)

		ch <- 1

		tlogf(t, "1: connecting\n")
		if err := dp1.Connect(); err != nil {
			ch <- err
			return
		}
		tlogf(t, "1: connected\n")

		for {
			tlogf(t, "1: read\n")
			u := (any)(nil)
			if err := dp1.Read(&u); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				ch <- err
				return
			}
			tlogf(t, "1: read done\n")

			lms := OffsetMsgs{}
			switch t := u.(type) {
			case *FilesDataMsg:
				continue // for loop
			case *OffsetMsg:
				lms = append(lms, t)
			case *OffsetMsgs:
				lms = *t
			default:
				panic(fmt.Errorf("1: %T", u))
			}

			tlogf(t, "1: received: %T, n=%v\n", u, len(lms))

			for _, lm := range lms {
				iv, ok := lm.Item.(*ItemValue)
				if !ok {
					ch <- fmt.Errorf("1: bad type: %T", lm.Item)
					return
				}
				if iv.Str != string(msgOut) {
					ch <- fmt.Errorf("no match: %v", iv.Str)
					return
				}
			}
		}
		tlogf(t, "1: loop done\n")

		if err := dp1.WaitClose(); err != nil {
			ch <- fmt.Errorf("1: dp1.close: %w", err)
			return
		}

		tlogf(t, "1: out\n")

		ch <- 2
	}()

	//----------

	if v := <-ch; v != 1 {
		t.Fatal(v)
	}

	exs := &ProtoExecSide{fdata: &FilesDataMsg{}}
	exs.logOn = testing.Verbose()
	//exs.NoWriteBuffering = true // disable buffer
	dp2 := NewProto(ctx, addr, exs, !editorIsServer, false)

	tlogf(t, "2: connecting\n")
	if err := dp2.Connect(); err != nil {
		t.Fatal(err)
	}
	tlogf(t, "2: connected\n")

	now := time.Now()
	//for i := 0; i < 1000000; i++ {
	for i := 0; i < 10000; i++ {
		//for i := 0; i < 2; i++ {
		if err := dp2.WriteMsg(lineMsg); err != nil {
			t.Fatal(err)
		}
	}
	d := time.Now().Sub(now)
	tlogf(t, "2: write done: %v\n", d)

	if err := dp2.WaitClose(); err != nil {
		t.Fatal("2: dp2.close", err)
	}
	tlogf(t, "2: close done\n")

	if v := <-ch; v != 2 {
		t.Fatal(v)
	}

	tlogf(t, "2: out\n")
}

func TestProto2(t *testing.T) {
	ctx := context.Background()

	addr := NewAddrI("tcp", ":12080")
	tlogf(t, "addr: %v, %q\n", addr.Network(), addr.String())

	editorIsServer := true
	ch := make(chan any)

	// accept
	go func() {
		eds := &ProtoEditorSide{}
		eds.logOn = testing.Verbose()
		dp1 := NewProto(ctx, addr, eds, editorIsServer, false)

		ch <- 1

		if err := dp1.Connect(); err != nil {
			ch <- err
			return
		}

		ch <- 2

		for {
			u := (any)(nil)
			if err := dp1.Read(&u); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				ch <- err
				return
			}
		}

		if err := dp1.WaitClose(); err != nil {
			ch <- err
			return
		}

		ch <- 3
	}()

	//----------

	if v := <-ch; v != 1 {
		t.Fatal(v)
	}

	exs := &ProtoExecSide{fdata: &FilesDataMsg{}}
	exs.logOn = testing.Verbose()
	//exs.NoWriteBuffering = true // disable buffer
	dp2 := NewProto(ctx, addr, exs, !editorIsServer, false)
	if err := dp2.Connect(); err != nil {
		t.Fatal(err)
	}

	if v := <-ch; v != 2 {
		t.Fatal(v)
	}

	//for i := 0; i < 2; i++ {
	//	if err := dp2.WriteLineMsg(lineMsg); err != nil {
	//		t.Fatal(err)
	//	}
	//}

	if err := dp2.WaitClose(); err != nil {
		t.Fatal(err)
	}

	if v := <-ch; v != 3 {
		t.Fatal(v)
	}
}

//----------
//----------
//----------

func tlogf(t *testing.T, f string, args ...any) {
	if testing.Verbose() {
		fmt.Printf(f, args...)
	}
}
