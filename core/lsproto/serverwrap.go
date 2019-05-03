package lsproto

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"text/template"
	"time"

	"os"
	"os/exec"
)

type ServerWrap struct {
	Cmd    *exec.Cmd
	cancel context.CancelFunc
}

//----------

func NewServerWrapTCP(cmdTmpl string) (*ServerWrap, string, error) {
	// random port to allow multiple editors to have multiple server wraps
	seed := time.Now().UnixNano() + int64(os.Getpid())
	ra := rand.New(rand.NewSource(seed))
	port := 27000 + ra.Intn(1000)

	// template vars
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	sw := &ServerWrap{}
	if err := sw.initTCP(cmdTmpl, addr); err != nil {
		return nil, "", err
	}
	return sw, addr, nil
}

func (sw *ServerWrap) initTCP(cmdTmpl, addr string) error {
	// build template
	tmpl, err := template.New("").Parse(cmdTmpl)
	if err != nil {
		return err
	}
	// fill template
	var data = struct{ Addr string }{addr}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return err
	}
	cmd := out.String() // cmd string

	// context
	bg := context.Background()
	ctx, cancel := context.WithCancel(bg)
	sw.cancel = cancel

	// early ctx cleanup
	startOk := false
	defer func() {
		if !startOk {
			cancel() // cleanup context resource
		}
	}()

	// cmd
	args := strings.Split(cmd, " ") // TODO: escapes
	sw.Cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	sw.Cmd.Stdout = os.Stdout
	sw.Cmd.Stderr = os.Stderr

	// cmd start
	if err := sw.Cmd.Start(); err != nil {
		return err
	}
	startOk = true
	return nil
}

//----------

func NewServerWrapIO(cmd string, stderr io.Writer) (*ServerWrap, io.ReadWriteCloser, error) {
	sw := &ServerWrap{}
	rwc, err := sw.initIO(cmd, stderr)
	if err != nil {
		return nil, nil, err
	}
	return sw, rwc, nil
}

func (sw *ServerWrap) initIO(cmd string, stderr io.Writer) (io.ReadWriteCloser, error) {
	// context
	bg := context.Background()
	ctx, cancel := context.WithCancel(bg)
	sw.cancel = cancel

	// early ctx cleanup
	startOk := false
	defer func() {
		if !startOk {
			cancel() // cleanup context resource
		}
	}()

	// cmd
	args := strings.Split(cmd, " ") // TODO: escapes
	sw.Cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	inp, err := sw.Cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	outp, err := sw.Cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	sw.Cmd.Stderr = stderr

	// io.ReadWriteCloser
	type rws_ struct {
		io.Reader
		io.Writer
		io.Closer
	}
	var rws rws_
	rws.Writer = inp
	rws.Closer = inp
	rws.Reader = outp

	// cmd start
	if err := sw.Cmd.Start(); err != nil {
		return nil, err
	}
	startOk = true
	return rws, nil
}

//----------

func (sw *ServerWrap) CloseWait() error {
	sw.cancel() // cleanup context resource
	if sw.Cmd != nil {
		return sw.Cmd.Wait()
	}
	return nil
}
