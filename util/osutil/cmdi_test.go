package osutil

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCmdIRead1(t *testing.T) {
	// wait for stdin indefinitely

	cmd0 := exec.Command("sleep", "0.25")

	midT := 750 * time.Millisecond

	h := NewHanger(1 * time.Second)
	cmd0.Stdin = h

	cmd := NewCmdI(cmd0)

	now := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	if dur < midT {
		t.Fatalf("cmd did not wait for stdin")
	}
}
func TestCmdIRead2(t *testing.T) {
	// don't wait for stdin

	cmd0 := exec.Command("sleep", "0.25")

	midT := 750 * time.Millisecond

	h := NewHanger(1 * time.Second)
	cmd0.Stdin = h

	cmd := NewCmdI(cmd0)
	cmd = NewNoHangPipeCmd2(cmd, true, false, false)

	now := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	if dur > midT {
		t.Fatalf("cmd waited for stdin")
	}
}
func TestCmdIRead3(t *testing.T) {
	// ctx cancel stops the hang on stdin

	cmd0 := exec.Command("sleep", "0.5")

	midT := 750 * time.Millisecond

	h := NewHanger(1 * time.Second)
	cmd0.Stdin = h

	ctx, cancel := context.WithCancel(context.Background())

	cmd := NewCmdI(cmd0)
	cmd = NewCtxCmd(ctx, cmd)
	cmd = NewNoHangPipeCmd2(cmd, true, false, false)

	now := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := cmd.Wait(); err == nil {
		t.Fatal("expecting error")
	}
	dur := time.Since(now)
	if dur > midT {
		t.Fatalf("cmd waited for stdin")
	}
}
func TestCmdIRead4(t *testing.T) {
	// killing the process stops the hang on stdin

	cmd0 := exec.Command("sleep", "0.5")

	midT := 250 * time.Millisecond

	h := NewHanger(1 * time.Second)
	cmd0.Stdin = h

	cmd := NewCmdI(cmd0)
	cmd = NewNoHangPipeCmd2(cmd, true, false, false)

	now := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	cmd0.Process.Kill()
	if err := cmd.Wait(); err == nil {
		t.Fatal("expecting error")
	}
	dur := time.Since(now)
	if dur > midT {
		t.Fatalf("cmd waited for stdin")
	}
}

//----------

func TestCmdIWrite1(t *testing.T) {
	// wait for stdout indefinitely

	cmd0 := exec.Command("sh", "-c", "sleep 0.25; echo aaa")

	midT := 750 * time.Millisecond

	h := NewHanger(1 * time.Second)
	cmd0.Stdout = h

	cmd := NewCmdI(cmd0)

	now := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	dur := time.Since(now)
	if dur < midT {
		t.Fatalf("cmd did not wait for stdout")
	}

	s := string(h.buf.Bytes())
	if s != "aaa\n" {
		t.Fatalf("bad output: %v", s)
	}
}

//func TestCmdIWrite2(t *testing.T) {
//	// ctx cancel stops the hang on stdout

//	cmd0 := exec.Command("sh", "-c", "sleep 0.25; echo aaa")

//	midT := 750 * time.Millisecond

//	h := NewHanger(1 * time.Second)
//	cmd0.Stdout = h

//	ctx, cancel := context.WithCancel(context.Background())

//	cmd := NewCmdI(cmd0)
//	cmd = NewCtxCmd(ctx, cmd)
//	cmd = NewNoHangPipeCmd(cmd, false, true, true)

//	now := time.Now()
//	if err := cmd.Start(); err != nil {
//		t.Fatal(err)
//	}
//	if err := cmd.Wait(); err != nil {
//		t.Fatal(err)
//	}
//	dur := time.Since(now)
//	cancel()
//	if dur > midT {
//		t.Fatalf("cmd waited for stdout")
//	}

//	s := string(h.buf.Bytes())
//	if s != "" {
//		t.Fatalf("bad output: %v", s)
//	}
//}

//----------

func TestCmdIShell(t *testing.T) {
	args := []string{"a", "b c d", "e f"}
	script := `for arg in "$@"; do echo "$arg"; done; true`
	args = append([]string{script}, args...)

	c := NewCmdI2(args)
	c = NewShellCmd(c, false)
	buf := &bytes.Buffer{}
	c.Cmd().Stdout = buf
	c.Cmd().Stderr = os.Stderr
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	if err := c.Wait(); err != nil {
		t.Fatal(err)
	}
	//t.Fatal()
	out := buf.String()
	//t.Log(out)
	if !strings.Contains(out, "b c d\n") {
		t.Fatal(out)
	}
}

//----------
//----------
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
