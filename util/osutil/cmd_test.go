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

func TestCmdCtx1(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	cmd := NewCmd(ctx, "sh", "-c", "sleep 3")
	cmd.Stdin = Hanger(4 * time.Second)
	cmd.Start()
	now := time.Now()
	go func() {
		<-ctx.Done()
		t.Logf("context: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err != nil {
		//t.Fatal(err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur > 2*time.Second {
		t.Fatalf("cmd did not end at ctx cancel")
	}
}

func TestCmdKill(t *testing.T) {
	ctx := context.Background()
	cmd := NewCmd(ctx, "sh", "-c", "sleep 3")
	cmd.Stdin = Hanger(4 * time.Second)
	cmd.Start()
	now := time.Now()
	go func() {
		time.Sleep(1 * time.Second)
		cmd.Process.Kill()
		t.Logf("kill: %v\n", time.Since(now))
	}()
	if err := cmd.Wait(); err != nil {
		//t.Fatal(err)
	}
	dur := time.Since(now)
	t.Logf("wait done: %v\n", dur)
	if dur > 2*time.Second {
		t.Fatalf("cmd did not end at kill")
	}
}

func TestStress1(t *testing.T) {
	for i := 0; i < 1000; i++ {
		ctx := context.Background()
		cmd := NewCmd(ctx, "echo", "foo")
		p, err := cmd.StdoutPipe()
		if err != nil {
			t.Errorf("%d. StdoutPipe: %v", i, err)
			continue
		}
		type strErr struct {
			s   string
			err error
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
		se := <-c // passes test
		if err := cmd.Wait(); err != nil {
			t.Errorf("%d. Wait: %v", i, err)
			continue
		}
		//se := <-c // fails to get all output, should not read after wait
		if se.err != nil {
			t.Errorf("%d. echo: %v", i, se.err)
		}
		if se.s != "foo\n" {
			t.Errorf("%d. echo: want %q, got %q", i, "foo\n", se.s)
		}
	}
}

func TestStress2(t *testing.T) {
	for i := 0; i < 1000; i++ {
		ctx := context.Background()
		cmd := NewCmd(ctx, "echo", "foo")
		pr, pw := io.Pipe()
		cmd.Stdout = pw
		type strErr struct {
			s   string
			err error
		}
		c := make(chan strErr)
		go func() {
			b, err := ioutil.ReadAll(pr)
			c <- strErr{string(b), err}
		}()
		if err := cmd.Start(); err != nil {
			t.Errorf("%d. Start: %v", i, err)
			continue
		}
		//se := <-c // fails: reallall loop will hang since it doesn't know when pw is done
		if err := cmd.Wait(); err != nil {
			t.Errorf("%d. Wait: %v", i, err)
			continue
		}
		pw.Close() // needed to let readall loop know it has ended
		se := <-c  // passes test
		if se.err != nil {
			t.Errorf("%d. echo: %v", i, se.err)
		}
		if se.s != "foo\n" {
			t.Errorf("%d. echo: want %q, got %q", i, "foo\n", se.s)
		}
	}
}

func TestStress3(t *testing.T) {
	// test: must close descriptors on ctx cancel or it will hang

	for i := 0; i < 10; i++ {
		ctx := context.Background()
		cmd := NewCmd(ctx, "echo", "foo")
		buf := &bytes.Buffer{}
		cmd.Stdout = buf
		type strErr struct {
			s   string
			err error
		}
		c := make(chan strErr)
		waitDone := make(chan bool)
		go func() {
			<-waitDone
			b, err := ioutil.ReadAll(buf)
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
		close(waitDone)
		se := <-c // passes test
		if se.err != nil {
			t.Errorf("%d. echo: %v", i, se.err)
		}
		if se.s != "foo\n" {
			t.Errorf("%d. echo: want %q, got %q", i, "foo\n", se.s)
		}
	}
}

//----------

type Hanger time.Duration

func (h Hanger) Read(b []byte) (int, error) {
	time.Sleep(time.Duration(h))
	return 0, io.EOF
}
func (h Hanger) Write(b []byte) (int, error) {
	time.Sleep(time.Duration(h))
	return 0, io.EOF
}
