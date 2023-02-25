package testutil

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/jmigpin/editor/util/iout"
)

func CollectLog(t *testing.T, fn func() error) ([]byte, []byte, error) {
	return CollectLog2(t, t.Logf, fn)
}
func CollectLog2(t *testing.T, logf func(string, ...any), fn func() error) ([]byte, []byte, error) {
	t.Helper()

	// keep for later restoration
	orig1, orig2 := os.Stdout, os.Stderr
	defer func() { // restore
		os.Stdout, os.Stderr = orig1, orig2
	}()

	// build pipes to catch stdout/stderr for ouput comparison
	stdoutBuf, stderrBuf := &bytes.Buffer{}, &bytes.Buffer{}
	pr1, pw1, err1 := os.Pipe()
	if err1 != nil {
		return nil, nil, err1
	}
	pr2, pw2, err2 := os.Pipe()
	if err2 != nil {
		return nil, nil, err2
	}
	os.Stdout, os.Stderr = pw1, pw2

	// setup logger
	logWriter := func(wr io.Writer, buf *bytes.Buffer) io.Writer {
		return iout.FnWriter(func(b []byte) (int, error) {
			t.Helper()

			// commented: prints many lines without log prefix
			//logf("%s", b)

			// ensure a call to logf() sends a complete line
			k := 0
			for i, c := range b {
				if c == '\n' {
					b2 := append(buf.Bytes(), b[k:i]...)
					logf("%s", b2)
					k = i + 1
					buf.Reset()
				}
			}
			if k < len(b) {
				buf.Write(b[k:])
			}
			return len(b), nil
		})
	}
	flushLogWriter := func(buf *bytes.Buffer) {
		if len(buf.Bytes()) > 0 {
			logf("%s", buf.Bytes())
		}
	}

	// copy loop
	copyLoops := &sync.WaitGroup{}
	copyLoops.Add(1)
	go func() {
		defer copyLoops.Done()

		buf := &bytes.Buffer{}
		lw := logWriter(orig1, buf)

		rd := io.Reader(pr1)
		rd = io.TeeReader(rd, lw)
		_, _ = io.Copy(stdoutBuf, rd)
		pr1.Close()

		flushLogWriter(buf) // final bytes
	}()

	// copy loop
	copyLoops.Add(1)
	go func() {
		defer copyLoops.Done()

		buf := &bytes.Buffer{}
		lw := logWriter(orig2, buf)

		rd := io.Reader(pr2)
		rd = io.TeeReader(rd, lw)
		_, _ = io.Copy(stderrBuf, rd)
		pr2.Close()

		flushLogWriter(buf) // final bytes
	}()

	// run
	err := fn()

	// cmd done, close writers
	pw1.Close()
	pw2.Close()
	// wait for readers
	copyLoops.Wait()

	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}
