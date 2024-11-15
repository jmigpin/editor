package lsproto

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
)

type ServerWrap struct {
	Cmd osutil.CmdI
}

func newServerWrap(ctx context.Context, cmd string) *ServerWrap {
	sw := &ServerWrap{}
	args := strings.Split(cmd, " ") // TODO: escapes
	sw.Cmd = osutil.NewCmdIShell(ctx, args...)
	return sw
}
func (sw *ServerWrap) Wait() error {
	return sw.Cmd.Wait()
}

//----------
//----------
//----------

func startServerWrapTCP(ctx context.Context, cmdTmpl string, outw io.Writer) (context.Context, *ServerWrap, string, error) {
	host := "127.0.0.1"

	// multiple editors can have multiple server wraps, need unique port
	port, err := osutil.GetFreeTcpPort()
	if err != nil {
		return nil, nil, "", err
	}

	// run cmd template
	cmd, addr, err := cmdTemplate(cmdTmpl, host, port)
	if err != nil {
		return nil, nil, "", err
	}

	ctx2, cancel := context.WithCancelCause(ctx)

	sw := newServerWrap(ctx2, cmd)

	// get lsp server output in tcp mode
	if outw != nil {
		sw.Cmd.Cmd().Stdout = outw
		sw.Cmd.Cmd().Stderr = outw
	}

	// ensure ctx cancel in case of error after start
	sw.Cmd = osutil.NewOnWaitDoneCmd(sw.Cmd, func(err error) {
		cancel(err)
	})

	if err := sw.Cmd.Start(); err != nil {
		cancel(err) // clear resources
		return nil, nil, "", err
	}

	return ctx2, sw, addr, nil
}

func startServerWrapIO(ctx context.Context, cmd string, stderr io.Writer) (*ServerWrap, io.ReadWriteCloser, error) {
	sw := newServerWrap(ctx, cmd)

	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()

	sw.Cmd.Cmd().Stdin = pr1
	sw.Cmd.Cmd().Stdout = pw2
	sw.Cmd.Cmd().Stderr = stderr

	rwc := &rwc{} // also keep for later close
	rwc.WriteCloser = pw1
	rwc.ReadCloser = pr2
	// ensure pipe close in case of error after start()
	sw.Cmd = osutil.NewOnWaitDoneCmd(sw.Cmd, func(err error) {
		rwc.Close()
	})

	if err := sw.Cmd.Start(); err != nil {
		rwc.Close()
		return nil, nil, err
	}

	return sw, rwc, nil
}

//----------
//----------
//----------

type rwc struct {
	io.ReadCloser
	io.WriteCloser
}

func (rwc *rwc) Close() error {
	err1 := rwc.ReadCloser.Close()
	err2 := rwc.WriteCloser.Close()
	return iout.MultiErrors(err1, err2)
}

//----------

func cmdTemplate(cmdTmpl string, host string, port int) (string, string, error) {
	// build template
	tmpl, err := template.New("").Parse(cmdTmpl)
	if err != nil {
		return "", "", err
	}

	// template data
	type tdata struct {
		Addr string
		Host string
		Port int
	}
	data := &tdata{}
	data.Host = host
	data.Port = port
	data.Addr = fmt.Sprintf("%s:%d", host, port)

	// fill template
	out := &bytes.Buffer{}
	if err := tmpl.Execute(out, data); err != nil {
		return "", "", err
	}
	return out.String(), data.Addr, nil
}
