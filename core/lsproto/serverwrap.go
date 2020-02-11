package lsproto

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"

	"os"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
)

type ServerWrap struct {
	Cmd *osutil.Cmd
	rwc *rwc // just for IO mode (can be nil)
}

//----------

func NewServerWrapTCP(ctx context.Context, cmdTmpl string, li *LangInstance) (*ServerWrap, string, error) {
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

	preStartFn := func(sw *ServerWrap) error {
		// allows reading lsp server output
		if logTestVerbose() {
			if err := sw.Cmd.SetupStdInOutErr(nil, os.Stdout, os.Stderr); err != nil {
				return err
			}
		}
		return nil
	}

	sw, err := newServerWrapCommon(ctx, cmd, li, preStartFn)
	if err != nil {
		return nil, "", err
	}
	return sw, addr, nil
}

//----------

func NewServerWrapIO(ctx context.Context, cmd string, stderr io.Writer, li *LangInstance) (*ServerWrap, io.ReadWriteCloser, error) {

	preStartFn := func(sw *ServerWrap) error {
		pr1, pw1 := io.Pipe()
		pr2, pw2 := io.Pipe()
		if err := sw.Cmd.SetupStdInOutErr(pr1, pw2, stderr); err != nil {
			return err
		}
		sw.rwc = &rwc{} // also keep for later close
		sw.rwc.WriteCloser = pw1
		sw.rwc.ReadCloser = pr2
		return nil
	}

	sw, err := newServerWrapCommon(ctx, cmd, li, preStartFn)
	if err != nil {
		return nil, nil, err
	}
	return sw, sw.rwc, nil
}

//----------

func newServerWrapCommon(ctx context.Context, cmd string, li *LangInstance, preStartFn func(sw *ServerWrap) error) (*ServerWrap, error) {
	sw := &ServerWrap{}

	args := strings.Split(cmd, " ") // TODO: escapes
	sw.Cmd = osutil.NewCmd(ctx, args...)

	if preStartFn != nil {
		if err := preStartFn(sw); err != nil {
			sw.Cmd.Cancel() // start will not run, clear ctx
			return nil, err
		}
	}

	if err := sw.Cmd.Start(); err != nil {
		return nil, err
	}
	return sw, nil
}

//----------

func (sw *ServerWrap) closeFromLangInstance() error {
	if sw == nil {
		return nil
	}
	me := iout.MultiError{}
	if sw.rwc != nil {
		me.Add(sw.rwc.Close())
	}
	if sw.Cmd != nil {
		me.Add(sw.Cmd.Wait())
	}
	return me.Result()
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
