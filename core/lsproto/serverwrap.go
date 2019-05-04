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

	"github.com/jmigpin/editor/util/chanutil"
	"github.com/jmigpin/editor/util/iout"
)

type ServerWrap struct {
	Cmd    *exec.Cmd
	cancel context.CancelFunc
	reg    *Registration

	rwc *rwc
}

//----------

func NewServerWrapTCP(cmdTmpl string, reg *Registration) (*ServerWrap, string, error) {
	// random port to allow multiple editors to have multiple server wraps
	port := randPort()
	// template vars
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	cmd, err := cmdTemplate(cmdTmpl, addr)
	if err != nil {
		return nil, "", err
	}
	sw, err := newServerWrapCommon(cmd, reg, nil)
	if err != nil {
		return nil, "", err
	}
	return sw, addr, nil
}

func NewServerWrapIO(cmd string, stderr io.Writer, reg *Registration) (*ServerWrap, io.ReadWriteCloser, error) {

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

	sw, err := newServerWrapCommon(cmd, reg, preStartFn)
	if err != nil {
		return nil, nil, err
	}
	return sw, sw.rwc, nil
}

//----------

func newServerWrapCommon(cmd string, reg *Registration, preStartFn func(sw *ServerWrap) error) (*ServerWrap, error) {
	sw := &ServerWrap{reg: reg}

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
		// cmd.wait can be slow to return, use timeout
		timeout := 200 * time.Millisecond
		err := chanutil.CallTimeout(context.Background(), timeout, "sw close", sw.reg.asyncErrors, sw.Cmd.Wait)
		me.Add(err)
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
