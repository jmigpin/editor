package debug

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestProto1(t *testing.T) {
	ctx := context.Background()

	addr := NewAddrI("tcp", ":12080")
	//addr := NewAddrI("unix", "/tmp/editor.sock1")
	//addr := NewAddrI("ws", ":12080") // needs go tag editorDebugExecSide to have ws client

	fmt.Printf("addr: %v, %q\n", addr.Network(), addr.String())

	eds := &ProtoEditorSide{}
	dp1 := NewProto(ctx, true, addr, eds)

	ch := make(chan int)

	msg := []byte("abc")
	msgOut := "[97 98 99]"
	item := IVi(msg)
	lineMsg := &LineMsg{Item: item}

	// accept
	go func() {
		ch <- 1
		if err := dp1.Connect(); err != nil {
			t.Log(err)
			return
		}

		for {
			u := (any)(nil)
			if err := dp1.Read(&u); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				t.Fatal(err)
			}

			lms := []*LineMsg{}
			switch t := u.(type) {
			case *LineMsg:
				lms = append(lms, t)
			case []*LineMsg:
				lms = t
			default:
				panic(fmt.Errorf("%T", u))
			}
			fmt.Printf("received: %T, n=%v\n", u, len(lms))

			for _, lm := range lms {
				iv, ok := lm.Item.(*ItemValue)
				if !ok {
					t.Fatal()
				}
				if iv.Str != string(msgOut) {
					t.Fatal(iv.Str)
				}
			}

		}
		fmt.Printf("read done\n")

		ch <- 2

		if err := dp1.Close(); err != nil {
			t.Fatal("dp1.close", err)
		}
		ch <- 3
	}()

	if <-ch != 1 {
		t.Fatal()
	}

	// DEBUG: let server start first (case of ws)?
	//time.Sleep(5 * time.Second)

	exs := &ProtoExecSide{
		fdata: &FilesDataMsg{},
		//NoWriteBuffering: true, // disable buffer
	}
	dp2 := NewProto(ctx, false, addr, exs)
	if err := dp2.Connect(); err != nil {
		t.Fatal(err)
	}

	//for i := 0; i < 100000; i++ {
	for i := 0; i < 100; i++ {
		if err := dp2.WriteLineMsg(lineMsg); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("write done\n")

	if err := dp2.Close(); err != nil {
		t.Fatal("dp2.close", err)
	}

	if <-ch != 2 {
		t.Fatal()
	}

	if <-ch != 3 {
		t.Fatal()
	}
}
