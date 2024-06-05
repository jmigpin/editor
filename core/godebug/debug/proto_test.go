package debug

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
)

//----------

func TestProto1(t *testing.T) {
	host := "localhost:8923"
	addr1 := NewAddrI("tcp", host)
	addr2 := NewAddrI("tcp", host)
	testProto1a(t, addr1, addr2, true, true)

	addr1 = NewAddrI("auto", host)
	addr2 = NewAddrI("tcp", host)
	testProto1a(t, addr1, addr2, true, true)

	addr1 = NewAddrI("auto", host)
	addr2 = NewAddrI("ws", host)
	testProto1a(t, addr1, addr2, true, true)

	addr1 = NewAddrI("ws", host)
	addr2 = NewAddrI("ws", host)
	testProto1a(t, addr1, addr2, true, true)

}
func testProto1a(t *testing.T, addr1, addr2 Addr, addr1IsServer, pass bool) {
	t.Helper()
	err := testProto1b(t, addr1, addr2, addr1IsServer, false)
	if pass && err != nil {
		t.Fatal(err)
	}
	if !pass && err == nil {
		t.Fatal("test should fail but had no error")
	}
}
func testProto1b(t *testing.T, addr1, addr2 Addr, addr1IsServer, continueServing bool) (err0 error) {
	t.Helper()

	ctx := context.Background()
	stdout := verboseStdout()

	//----------

	tlogf(t, "addr: %v, %q\n", addr1.Network(), addr1.String())

	go func() {
		eds := &ProtoEditorSide{}
		p1, err := NewProto(ctx, addr1, eds, addr1IsServer, addr1IsServer && continueServing, stdout)
		if err != nil {
			err0 = err
			return
		}

		for {
			u := (any)(nil)
			if err := p1.Read(&u); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				err0 = err
				return
			}
		}

		if err := p1.CloseOrWait(); err != nil {
			err0 = err
			return
		}
	}()

	//----------

	exs := &ProtoExecSide{fdata: &FilesDataMsg{}}
	//exs.NoWriteBuffering = true // disable buffer
	addr2IsServer := !addr1IsServer
	p2, err := NewProto(ctx, addr2, exs, addr2IsServer, addr2IsServer && continueServing, stdout)
	if err != nil {
		return err
	}

	msg := []byte("abcdefg")
	//msgOut := "[97 98 99 100 101 102 103]"
	lineMsg := &OffsetMsg{Item: IVi(msg)}

	for i := 0; i < 2; i++ {
		if err := p2.WriteMsg(lineMsg); err != nil {
			t.Fatal(err)
		}
	}

	if err := p2.CloseOrWait(); err != nil {
		return err
	}

	return err0
}

//----------

//func TestProto2(t *testing.T) {
//	ctx := context.Background()

//	addr := NewAddrI("tcp", ":12080")
//	//addr := NewAddrI("unix", "/tmp/editor.sock1")
//	//addr := NewAddrI("ws", ":12080") // needs go tag editorDebugExecSide to have ws client

//	tlogf(t, "addr: %v, %q\n", addr.Network(), addr.String())

//	editorIsServer := true
//	//editorIsServer := false
//	//ch := make(chan any)

//	msg := []byte("abcdefg")
//	msgOut := "[97 98 99 100 101 102 103]"
//	lineMsg := &OffsetMsg{Item: IVi(msg)}

//	// accept
//	go func() {
//		eds := &ProtoEditorSide{}
//		eds.logStdout = verboseStdout()
//		p1, err := NewProto(ctx, addr, eds, editorIsServer, false, os.Stderr)

//		ch <- 1

//		tlogf(t, "1: connecting\n")
//		if err := p1.Connect(ctx); err != nil {
//			ch <- err
//			return
//		}
//		tlogf(t, "1: connected\n")

//		for {
//			tlogf(t, "1: read\n")
//			u := (any)(nil)
//			if err := p1.Read(&u); err != nil {
//				if errors.Is(err, io.EOF) {
//					break
//				}
//				ch <- err
//				return
//			}
//			tlogf(t, "1: read done\n")

//			lms := OffsetMsgs{}
//			switch t := u.(type) {
//			case *FilesDataMsg:
//				continue // for loop
//			case *OffsetMsg:
//				lms = append(lms, t)
//			case *OffsetMsgs:
//				lms = *t
//			default:
//				panic(fmt.Errorf("1: %T", u))
//			}

//			tlogf(t, "1: received: %T, n=%v\n", u, len(lms))

//			for _, lm := range lms {
//				iv, ok := lm.Item.(*ItemValue)
//				if !ok {
//					ch <- fmt.Errorf("1: bad type: %T", lm.Item)
//					return
//				}
//				if iv.Str != string(msgOut) {
//					ch <- fmt.Errorf("no match: %v", iv.Str)
//					return
//				}
//			}
//		}
//		tlogf(t, "1: loop done\n")

//		if err := p1.Wait(); err != nil {
//			ch <- fmt.Errorf("1: dp1.close: %w", err)
//			return
//		}

//		tlogf(t, "1: out\n")

//		ch <- 2
//	}()

//	//----------

//	if v := <-ch; v != 1 {
//		t.Fatal(v)
//	}

//	exs := &ProtoExecSide{fdata: &FilesDataMsg{}}
//	exs.logStdout = verboseStdout()
//	//exs.NoWriteBuffering = true // disable buffer
//	p2 := NewProto(addr, exs, !editorIsServer, false, os.Stderr)

//	tlogf(t, "2: connecting\n")
//	if err := p2.Connect(ctx); err != nil {
//		t.Fatal(err)
//	}
//	tlogf(t, "2: connected\n")

//	now := time.Now()
//	//for i := 0; i < 1000000; i++ {
//	//for i := 0; i < 1000; i++ {
//	for i := 0; i < 10; i++ {
//		if err := p2.WriteMsg(lineMsg); err != nil {
//			t.Fatal(err)
//		}
//	}
//	d := time.Now().Sub(now)
//	tlogf(t, "2: write done: %v\n", d)

//	if err := p2.Wait(); err != nil {
//		t.Fatal("2: dp2.close", err)
//	}
//	tlogf(t, "2: close done\n")

//	if v := <-ch; v != 2 {
//		t.Fatal(v)
//	}

//	tlogf(t, "2: out\n")
//}

//----------
//----------
//----------

func tlogf(t *testing.T, f string, args ...any) {
	if testing.Verbose() {
		fmt.Printf(f, args...)
	}
}
