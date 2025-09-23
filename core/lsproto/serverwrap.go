package lsproto

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
)

type ServerWrap struct {
	Cmd osutil.CmdI
}

func newServerWrap(ctx context.Context, cmd string) *ServerWrap {
	sw := &ServerWrap{}
	args := strings.Split(cmd, " ") // TODO: escapes
	sw.Cmd = osutil.NewCmdINoHangPipeShell(ctx, args...)
	return sw
}
func (sw *ServerWrap) Wait() error {
	return sw.Cmd.Wait()
}

//----------
//----------
//----------

func startServerWrapTCP(ctx context.Context, cmdTmpl string, outw io.Writer) (*ServerWrap, string, error) {
	host := "127.0.0.1"

	// multiple editors can have multiple server wraps, need unique port
	port, err := osutil.GetFreeTcpPort()
	if err != nil {
		return nil, "", err
	}

	// run cmd template
	cmd, addr, err := cmdTemplate(cmdTmpl, host, port)
	if err != nil {
		return nil, "", err
	}

	sw := newServerWrap(ctx, cmd)

	// get lsp server output in tcp mode
	if outw != nil {
		sw.Cmd.Cmd().Stdout = outw
		sw.Cmd.Cmd().Stderr = outw
	}

	// ensure ctx cancel in case of error after start
	sw.Cmd = osutil.NewOnWaitDoneCmd(sw.Cmd, func(err error) {
		mustCancelLangInstance(ctx)
	})

	if err := sw.Cmd.Start(); err != nil {
		return nil, "", err
	}

	return sw, addr, nil
}

func startServerWrapIO(ctx context.Context, cmd string, stderr io.Writer) (*ServerWrap, io.ReadWriteCloser, error) {
	sw := newServerWrap(ctx, cmd)

	pr1, pw1 := ctxutil.PipeWithContext(ctx)
	pr2, pw2 := ctxutil.PipeWithContext(ctx)

	sw.Cmd.Cmd().Stdin = pr1
	sw.Cmd.Cmd().Stdout = pw2
	sw.Cmd.Cmd().Stderr = stderr

	rwc := &iout.RWC{}
	rwc.Writer = pw1
	rwc.Reader = pr2
	rwc.Closer = iout.FnCloser(func() error {
		err1 := pw1.Close()
		err2 := pw2.Close()
		return iout.MultiErrors(err1, err2)
	})

	// ensure pipe close in case of error after start()
	sw.Cmd = osutil.NewOnWaitDoneCmd(sw.Cmd, func(err error) {
		rwc.Close()
		mustCancelLangInstance(ctx)
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
