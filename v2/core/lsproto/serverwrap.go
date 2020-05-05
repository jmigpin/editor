package lsproto

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/jmigpin/editor/v2/util/iout"
	"github.com/jmigpin/editor/v2/util/osutil"
)

type ServerWrap struct {
	Cmd *osutil.Cmd
	rwc *rwc // just for IO mode (can be nil)
}

//----------

func StartServerWrapTCP(ctx context.Context, cmdTmpl string, w io.Writer) (*ServerWrap, string, error) {
	// multiple editors can have multiple server wraps
	//port, err := osutil.GetFreeTcpPort()
	//if err != nil {
	//	return nil, "", err
	//}
	port := osutil.RandomPort(3, 10000, 65000)

	// template vars
	addr := fmt.Sprintf("127.0.0.1:%v", port)

	cmd, err := cmdTemplate(cmdTmpl, addr)
	if err != nil {
		return nil, "", err
	}

	sw := newServerWrapCommon(ctx, cmd)

	// get lsp server output in tcp mode
	if w != nil {
		if err := sw.Cmd.SetupStdio(nil, w, w); err != nil {
			sw.Cmd.Cancel() // start will not run, clear ctx
			return nil, "", err
		}
	}

	if err := sw.Cmd.Start(); err != nil {
		return nil, "", err
	}
	return sw, addr, nil
}

func StartServerWrapIO(ctx context.Context, cmd string, stderr io.Writer, li *LangInstance) (*ServerWrap, io.ReadWriteCloser, error) {
	sw := newServerWrapCommon(ctx, cmd)

	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()
	if err := sw.Cmd.SetupStdio(pr1, pw2, stderr); err != nil {
		sw.Cmd.Cancel() // start will not run, clear ctx
		return nil, nil, err
	}
	sw.rwc = &rwc{} // also keep for later close
	sw.rwc.WriteCloser = pw1
	sw.rwc.ReadCloser = pr2

	if err := sw.Cmd.Start(); err != nil {
		sw.rwc.Close() // wait will not be called, clear resources
		return nil, nil, err
	}

	return sw, sw.rwc, nil
}

func newServerWrapCommon(ctx context.Context, cmd string) *ServerWrap {
	sw := &ServerWrap{}
	args := strings.Split(cmd, " ") // TODO: escapes
	sw.Cmd = osutil.NewCmd(ctx, args...)
	return sw
}

//----------

func (sw *ServerWrap) Wait() error {
	if sw.rwc != nil { // can be nil if in tcp mode
		// was set outside cmd, close after cmd.wait
		defer sw.rwc.Close()
	}

	return sw.Cmd.Wait()
}

//----------

type rwc struct {
	io.ReadCloser
	io.WriteCloser
}

func (rwc *rwc) Close() error {
	me := iout.MultiError{}
	me.Add(rwc.ReadCloser.Close())
	me.Add(rwc.WriteCloser.Close())
	return me.Result()
}

//----------

func cmdTemplate(cmdTmpl, addr string) (string, error) {
	// build template
	tmpl, err := template.New("").Parse(cmdTmpl)
	if err != nil {
		return "", err
	}
	// fill template
	var data = struct{ Addr string }{addr}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}
