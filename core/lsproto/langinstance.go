package lsproto

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/pkg/errors"
)

type LangInstance struct {
	lang *LangManager
	mu   struct {
		sync.Mutex
		cli        *Client
		sw         *ServerWrap
		connCancel context.CancelFunc
	}
}

func NewLangInstance(lang *LangManager) *LangInstance {
	return &LangInstance{lang: lang}
}

//----------

func (li *LangInstance) client(ctx context.Context, filename string) (*Client, error) {
	li.mu.Lock()
	defer li.mu.Unlock()
	// client/server already setup
	if li.mu.cli != nil {
		return li.mu.cli, nil
	}
	// start new client/server
	if err := li.startClientServer(ctx); err != nil {
		err = li.lang.WrapError(err)
		return nil, err
	}
	// initialize client
	if err := li.mu.cli.Initialize(ctx, filename); err != nil {
		_ = li.lang.Close()
		return nil, err
	}
	return li.mu.cli, nil
}

func (li *LangInstance) startClientServer(ctx context.Context) error {
	// independent context for client/server conn
	connCtx, cancel := context.WithCancel(context.Background())
	li.mu.connCancel = cancel
	// new client/server
	return li.connectClientServer(ctx, connCtx)
}

func (li *LangInstance) connectClientServer(reqCtx, connCtx context.Context) error {
	switch li.lang.Reg.Network {
	case "tcp":
		return li.connClientServerTCP(reqCtx, connCtx)
	case "tcpclient":
		return li.connClientTCP(reqCtx, connCtx, li.lang.Reg.Cmd)
	case "stdio":
		return li.connClientServerStdio(connCtx)
	default:
		return fmt.Errorf("unexpected network: %v", li.lang.Reg.Network)
	}
}

//----------

func (li *LangInstance) connClientServerTCP(reqCtx, connCtx context.Context) error {
	// server wrap
	sw, addr, err := NewServerWrapTCP(connCtx, li.lang.Reg.Cmd, li)
	if err != nil {
		return err
	}
	li.mu.sw = sw
	// client
	return li.connClientTCP(reqCtx, connCtx, addr)
}

//----------

func (li *LangInstance) connClientTCP(reqCtx, connCtx context.Context, addr string) error {
	// client connect with retries
	var cli *Client
	fn := func() error {
		cli0, err := NewClientTCP(connCtx, addr, li)
		if err != nil {
			return err
		}
		cli = cli0
		return nil
	}
	lateFn := func(err error) {
		if err != nil {
			err = errors.Wrap(err, "call late")
			li.lang.ErrorAsync(err)
			_ = li.lang.Close()
		}
	}
	sleep := 250 * time.Millisecond
	err := ctxutil.Retry(reqCtx, sleep, "clienttcp", fn, lateFn)
	if err != nil {
		return err
	}

	li.mu.cli = cli
	return nil
}

//----------

func (li *LangInstance) connClientServerStdio(ctx context.Context) error {
	var stderr io.Writer
	if li.lang.Reg.HasOptional("stderr") {
		stderr = os.Stderr
	}

	// server wrap
	sw, rwc, err := NewServerWrapIO(ctx, li.lang.Reg.Cmd, stderr, li)
	if err != nil {
		return err
	}
	li.mu.sw = sw

	// client
	cli := NewClientIO(rwc, li)
	li.mu.cli = cli

	return nil
}

//----------

func (li *LangInstance) closeFromLangManager() (err error) {
	li.mu.Lock()
	defer li.mu.Unlock()

	var me iout.MultiError
	me.Add(li.mu.cli.closeFromLangInstance())
	me.Add(li.mu.sw.closeFromLangInstance())

	// clear resources and force close
	if li.mu.connCancel != nil {
		li.mu.connCancel()
	}

	return me.Result()
}
