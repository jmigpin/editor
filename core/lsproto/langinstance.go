package lsproto

import (
	"context"
	"fmt"
	"io"
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

	// start new client/server
	if err := li.start(ctx2); err != nil {
		li.cancelCtx() // clear resources
		return nil, err
	}

	// initialize client
	if err := li.cli.Initialize(ctx2); err != nil {
		li.cancelCtx()
		_ = li.Wait() // wait for server/client
		return nil, err
	}

	return li, nil
}

//----------

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
	ctx2, sw, addr, err := startServerWrapTCP(ctx, li.lang.Reg.Cmd, li.srvOutW())
	if err != nil {
		return err
	}
	li.sw = sw
	// client
	if err := li.startClientTCP(ctx2, addr); err != nil {
		li.cancelCtx()
		_ = sw.Wait()
		return err
	}
	return nil
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

//----------

func (li *LangInstance) startClientServerStdio(ctx context.Context) error {
	// server wrap; the server can rwc.close, which will stop the client
	sw, rwc, err := startServerWrapIO(ctx, li.lang.Reg.Cmd, li.srvOutW())
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

func (li *LangInstance) srvOutW() io.Writer {
	if w := li.lang.man.serverWrapW; w != nil {
		return w
	}

	if li.lang.Reg.HasOptional("stderr") {
		//// useful for testing to see the server output msgs for debug
		//return os.Stderr

		// get server output in manager messages (editor msgs)
		return iout.FnWriter(func(p []byte) (int, error) {
			li.lang.man.Message(string(p))
			return len(p), nil
		})
	}
	return nil
}

//----------

func (li *LangInstance) Wait() error {
	defer li.cancelCtx()
	me := iout.MultiError{}
	if li.sw != nil { // might be nil: "tcpclient" option (or not started)
		me.Add(li.sw.Wait())
	}
	if li.cli != nil { // might be nil: not started in case of sw start error
		me.Add(li.cli.Wait())
	}
	return me.Result()
}
