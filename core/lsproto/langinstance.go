package lsproto

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/iout"
)

type LangInstance struct {
	lang      *LangManager
	cli       *Client
	sw        *ServerWrap // might be nil: "tcpclient" option
	cancelCtx context.CancelFunc
}

func NewLangInstance(ctx context.Context, lang *LangManager) (*LangInstance, error) {
	li := &LangInstance{lang: lang}

	ctx2, cancel := context.WithCancel(ctx)
	li.cancelCtx = cancel

	if err := li.startAndInit(ctx2); err != nil {
		cancel()
		_ = li.Wait()
		return nil, err
	}
	return li, nil
}

//----------

func (li *LangInstance) startAndInit(ctx context.Context) error {
	// start new client/server
	if err := li.start(ctx); err != nil {
		return err
	}
	// initialize client
	if err := li.cli.Initialize(ctx); err != nil {
		return err
	}
	return nil
}

func (li *LangInstance) start(ctx context.Context) error {
	switch li.lang.Reg.Network {
	case "tcp":
		return li.startClientServerTCP(ctx)
	case "tcpclient":
		return li.startClientTCP(ctx, li.lang.Reg.Cmd)
	case "stdio":
		return li.startClientServerStdio(ctx)
	default:
		return fmt.Errorf("unexpected network: %v", li.lang.Reg.Network)
	}
}

//----------

func (li *LangInstance) startClientServerTCP(ctx context.Context) error {
	// server wrap
	sw, addr, err := StartServerWrapTCP(ctx, li.lang.Reg.Cmd, li.lang.man.serverWrapW)
	if err != nil {
		return err
	}
	li.sw = sw
	// client
	return li.startClientTCP(ctx, addr)
}

func (li *LangInstance) startClientTCP(ctx context.Context, addr string) error {
	// client connect with retries
	var cli *Client
	fn := func() error {
		cli0, err := NewClientTCP(ctx, addr, li)
		if err != nil {
			return err
		}
		cli = cli0
		return nil
	}
	lateFn := func(err error) {
		if err != nil {
			// no connection close, it was handled already on late error
			err = fmt.Errorf("call late: %w", err)
			li.lang.PrintWrapError(err)
		}
	}
	retryPause := 300 * time.Millisecond
	err := ctxutil.Retry(ctx, retryPause, "clienttcp", fn, lateFn)
	if err != nil {
		return err
	}
	li.cli = cli
	return nil
}

func (li *LangInstance) startClientServerStdio(ctx context.Context) error {
	var stderr io.Writer
	if li.lang.Reg.HasOptional("stderr") {
		stderr = os.Stderr
	}
	// server wrap
	sw, rwc, err := StartServerWrapIO(ctx, li.lang.Reg.Cmd, stderr, li)
	if err != nil {
		return err
	}
	li.sw = sw
	// client
	cli := NewClientIO(ctx, rwc, li)
	li.cli = cli
	return nil
}

//----------

func (li *LangInstance) Wait() error {
	defer li.cancelCtx()
	var me iout.MultiError
	if li.sw != nil { // might be nil: "tcpclient" option (or not started)
		me.Add(li.sw.Wait())
	}
	if li.cli != nil { // might be nil: not started in case of sw start error
		me.Add(li.cli.Wait())
	}
	return me.Result()
}
