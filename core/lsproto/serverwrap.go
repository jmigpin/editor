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

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
)

type ServerWrap struct {
	Cmd    *exec.Cmd
	cancel context.CancelFunc
	reg    *Registration

	rwc *rwc // can be nil
}

//----------

func NewServerWrapTCP(ctx context.Context, cmdTmpl string, reg *Registration) (*ServerWrap, string, error) {
	// random port to allow multiple editors to have multiple server wraps
	port := randPort()
	// template vars
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	cmd, err := cmdTemplate(cmdTmpl, addr)
	if err != nil {
		return nil, "", err
	}

	preStartFn := func(sw *ServerWrap) error {
		// allows reading lsp server output (ex: gopls)
		if logTestVerbose() {
			sw.Cmd.Stdout = os.Stdout
			sw.Cmd.Stderr = os.Stderr
		}
		return nil
	}

	sw, err := newServerWrapCommon(ctx, cmd, reg, preStartFn)
	if err != nil {
		return nil, "", err
	}
	return sw, addr, nil
}

func NewServerWrapIO(ctx context.Context, cmd string, stderr io.Writer, reg *Registration) (*ServerWrap, io.ReadWriteCloser, error) {

	preStartFn := func(sw *ServerWrap) error {
		// in/out/err pipes
		inp, err1 := sw.Cmd.StdinPipe()
		outp, err2 := sw.Cmd.StdoutPipe()
		err := iout.MultiErrors(err1, err2)
		if err != nil {
			inp.Close()  // ok if nil
			outp.Close() // ok if nil
			return err
		}

		// keep for later close
		sw.rwc = &rwc{}
		sw.rwc.WriteCloser = inp
		sw.rwc.ReadCloser = outp

		sw.Cmd.Stderr = stderr // can be nil

		return nil
	}

	sw, err := newServerWrapCommon(ctx, cmd, reg, preStartFn)
	if err != nil {
		return nil, nil, err
	}
	return sw, sw.rwc, nil
}

//----------

func newServerWrapCommon(ctx0 context.Context, cmd string, reg *Registration, preStartFn func(sw *ServerWrap) error) (*ServerWrap, error) {
	sw := &ServerWrap{reg: reg}
	sw.reg.cs.mc.Add(sw)

	// context with cancel
	ctx, cancel := context.WithCancel(ctx0)
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
	sw.Cmd = osutil.ExecCmdCtxWithAttr(ctx, args[0], args[1:]...)

	if preStartFn != nil {
		if err := preStartFn(sw); err != nil {
			return nil, err
		}
	}

	// cmd start
	if err := sw.Cmd.Start(); err != nil {
		return nil, err
	}
	startOk = true
	return sw, nil
}

//----------

func randPort() int {
	seed := time.Now().UnixNano() + int64(os.Getpid())
	ra := rand.New(rand.NewSource(seed))
	port := 27000 + ra.Intn(1000)
	return port
}

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

//----------

func (sw *ServerWrap) Close() error {
	sw.cancel() // cleanup context resource (cancels cmd)
	me := iout.MultiError{}
	if sw.rwc != nil {
		me.Add(sw.rwc.Close())
	}
	if sw.Cmd != nil {
		me.Add(sw.Cmd.Wait())
	}
	me.Add(sw.reg.cs.mc.CloseRest(sw)) // multiclose
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
