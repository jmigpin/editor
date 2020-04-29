package osutil

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"
	"time"
)

//godebug:annotatepackage

//----------

func TestCmdRead1(t *testing.T) {
	// wait for stdin indefinitely (correct, cmd.cmd.wait waits)

	ctx := context.Background()
	cmd := NewCmd(ctx, "sh", "-c", "sleep 1")
	midT := 2 * time.Second
	h := NewHanger(3 * time.Second)
	cmd.Stdin = h // hangs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	go func() {
		<-ctx.Done()
		t.Logf("context: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur < midT {
		t.Fatalf("cmd did end, did not wait for stdin")
	}
}

func TestCmdRead2(t *testing.T) {
	// wait for stdin indefinitely (correct, cmd.cmd.wait waits)

	// (cmd differs from exec.cmd)
	// BUT: setupstdio doesn't wait because of the terminal feature, which could cause potential leaks

	ctx := context.Background()
	cmd := NewCmd(ctx, "sh", "-c", "sleep 1")
	midT := 2 * time.Second
	h := NewHanger(3 * time.Second)
	//cmd.Stdin = h // hangs
	cmd.SetupStdio(h, nil, nil) // doesn't hang (should it really hang?)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	go func() {
		<-ctx.Done()
		t.Logf("context: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur < midT {
		t.Fatalf("cmd did end, did not wait for stdin")
	}
}

func TestCmdRead2Ctx(t *testing.T) {
	// ctx cancel should be able to stop the hang on stdin
	// (cmd differs from exec.cmd)

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	cmd := NewCmd(ctx, "sh", "-c", "sleep 1")
	midT := 2 * time.Second
	h := NewHanger(3 * time.Second)
	//cmd.Stdin = h              // hangs (waits for it to be closed/fail)
	//ipwc, _ := cmd.StdinPipe() // doesn't hang
	//go func() {
	//	io.Copy(ipwc, h)
	//}()
	cmd.SetupStdio(h, nil, nil) // doesn't hang
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	go func() {
		<-ctx.Done()
		t.Logf("context: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err == nil {
		t.Fatal("expecting error")
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur > midT {
		t.Fatalf("cmd did not end at ctx cancel")
	}
}

//----------

func TestCmdWrite1(t *testing.T) {
	// wait for stdout indefinitely (correct, cmd.cmd.wait waits)

	ctx := context.Background()
	cmd := NewCmd(ctx, "sh", "-c", "sleep 1; echo aaa")
	midT := 2 * time.Second
	h := NewHanger(3 * time.Second)
	cmd.Stdout = h // hangs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	go func() {
		<-ctx.Done()
		t.Logf("context: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur < midT {
		t.Fatalf("cmd did end, did not wait for stdout")
	}
	s := string(h.buf.Bytes())
	if s != "aaa\n" {
		t.Fatalf("bad output: %v", s)
	}
}

func TestCmdWrite2(t *testing.T) {
	// wait for stdout indefinitely (correct, cmd.cmd.wait waits)

	ctx := context.Background()
	cmd := NewCmd(ctx, "sh", "-c", "sleep 1; echo aaa")
	midT := 2 * time.Second
	h := NewHanger(3 * time.Second)
	//cmd.Stdout = h // hangs
	cmd.SetupStdio(nil, h, nil) // hangs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	go func() {
		<-ctx.Done()
		t.Logf("context: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur < midT {
		t.Fatalf("cmd did end, did not wait for stdout")
	}
	s := string(h.buf.Bytes())
	if s != "aaa\n" {
		t.Fatalf("bad output: %v", s)
	}
}

func TestCmdWrite2Ctx(t *testing.T) {
	// ctx cancel should be able to stop the hang on stdout (correct, cmd.cmd cancels and sends kill sig)

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	midT := 2 * time.Second
	cmd := NewCmd(ctx, "sh", "-c", "sleep 3; echo aaa")
	h := NewHanger(4 * time.Second)
	//cmd.Stdout = h // doesn't hang
	cmd.SetupStdio(nil, h, nil) // doesn't hang
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	go func() {
		<-ctx.Done()
		t.Logf("context: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err == nil {
		t.Fatal("expecting error")
	} else {
		t.Logf("err: %v", err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur > midT {
		t.Fatalf("cmd did not end at ctx cancel")
	}
	s := string(h.buf.Bytes())
	if s != "" {
		t.Fatalf("bad output: %v", s)
	}
}

//----------

func TestCmdKill(t *testing.T) {
	// killing a process will still wait for the input to complete (correct, cmd.cmd.wait waits)
	// the hanger used in this test is sleeping, but in normal conditions, its write attempt would fail when the kill happens

	ctx := context.Background()
	killT := 1 * time.Second
	cmd := NewCmd(ctx, "sh", "-c", "sleep 2")
	midT := 2 * time.Second
	h := NewHanger(4 * time.Second)
	cmd.Stdin = h // hangs
	//cmd.SetupStdInOutErr(h, nil, nil) // hangs
	cmd.Start()
	now := time.Now()
	go func() {
		time.Sleep(killT)
		cmd.Process.Kill()
		t.Logf("kill: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err == nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	//if dur > 2*time.Second {
	//t.Fatalf("cmd did not end at kill")
	//}
	if dur < midT {
		t.Fatalf("cmd did end at kill")
	}
}

//----------

func TestCmdPipe(t *testing.T) {
	type strErr struct {
		s   string
		err error
	}
	for i := 0; i < 100; i++ {
		ctx := context.Background()
		cmd := NewCmd(ctx, "echo", "aaa")
		pr, pw := io.Pipe()
		c := make(chan strErr)
		cmd.SetupStdio(nil, pw, nil)
		go func() {
			b, err := ioutil.ReadAll(pr)
			c <- strErr{string(b), err}
		}()
		if err := cmd.Start(); err != nil {
			t.Errorf("%d. Start: %v", i, err)
			continue
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("%d. Wait: %v", i, err)
			continue
		}
		pw.Close() // must close after wait (handle pipe externally to cmd), or it will hang
		se := <-c
		if se.err != nil {
			t.Errorf("%d. echo: %v", i, se.err)
		}
		if se.s != "aaa\n" {
			t.Errorf("%d. echo: want %q, got %q", i, "aaa\n", se.s)
		}
	}
}

//----------

func TestStress1(t *testing.T) {
	// direct cmd.stoutpipe test for understanding

	type strErr struct {
		s   string
		err error
	}
	for i := 0; i < 1000; i++ {
		ctx := context.Background()
		cmd := NewCmd(ctx, "echo", "aaa")
		p, err := cmd.StdoutPipe()
		if err != nil {
			t.Errorf("%d. StdoutPipe: %v", i, err)
			continue
		}
		c := make(chan strErr)
		go func() {
			b, err := ioutil.ReadAll(p)
			c <- strErr{string(b), err}
		}()
		if err := cmd.Start(); err != nil {
			t.Errorf("%d. Start: %v", i, err)
			continue
		}
		se := <-c // must read before wait (using stdoutpipe directly)
		if err := cmd.Wait(); err != nil {
			t.Errorf("%d. Wait: %v", i, err)
			continue
		}
		//se := <-c // fails to get all output since wait will not have a copy loop
		if se.err != nil {
			t.Errorf("%d. echo: %v", i, se.err)
		}
		if se.s != "aaa\n" {
			t.Errorf("%d. echo: want %q, got %q", i, "aaa\n", se.s)
		}
	}
}

//----------

type Hanger struct {
	t   time.Duration
	buf bytes.Buffer
}

func NewHanger(t time.Duration) *Hanger {
	return &Hanger{t: t}
}
func (h *Hanger) Read(b []byte) (int, error) {
	time.Sleep(time.Duration(h.t))
	//return 0, io.EOF
	return h.buf.Read(b)
}
func (h *Hanger) Write(b []byte) (int, error) {
	time.Sleep(time.Duration(h.t))
	//return 0, io.EOF
	return h.buf.Write(b)
}
