package debug

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"
)

func TestListener2(t *testing.T) {
	host := "localhost:8921"
	saddr := NewAddrI("tcp", host)
	caddr := NewAddrI("tcp", host)
	testListener2a(t, saddr, caddr, true)

	saddr = NewAddrI("auto", host)
	caddr = NewAddrI("tcp", host)
	testListener2a(t, saddr, caddr, true)

	saddr = NewAddrI("auto", host)
	caddr = NewAddrI("ws", host)
	testListener2a(t, saddr, caddr, true)

	saddr = NewAddrI("ws", host)
	caddr = NewAddrI("ws", host)
	testListener2a(t, saddr, caddr, true)

	saddr = NewAddrI("tcp", host)
	caddr = NewAddrI("ws", host)
	testListener2a(t, saddr, caddr, false)

	//saddr = NewAddrI("unix", "/tmp/editor_godebug_t1.txt")
	//caddr = NewAddrI("unix", "/tmp/editor_godebug_t1.txt")
	//testListener2a(t, saddr, caddr, true)
}
func testListener2a(t *testing.T, saddr, caddr Addr, pass bool) {
	t.Helper()
	err := testListener2b(t, saddr, caddr)
	if pass && err != nil {
		t.Fatal(err)
	}
	if !pass && err == nil {
		t.Fatal("test should fail but had no error")
	}
}
func testListener2b(t *testing.T, saddr, caddr Addr) (err0 error) {
	t.Helper()

	ctx := context.Background()

	// server: ctx for listen/accept, not connection once accepted
	// client: ctx for dial
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	ln, err := listen2(ctx, saddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	// client
	msg := "+abc\n"
	go func() {
		conn2, err := dial2(ctx, caddr)
		if err != nil {
			err0 = err
			return
		}
		// if commented, io.copy below will not exit
		defer conn2.Close()

		if _, err := conn2.Write([]byte(msg)); err != nil {
			err0 = err
			return
		}
	}()

	conn, err := ln.Accept()
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, conn); err != nil {
		return err
	}
	if s := buf.String(); s != msg {
		return fmt.Errorf("msg mismatch:\n%v", s)
	}

	return err0
}
