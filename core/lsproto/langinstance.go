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
	lang *LangManager
	ctx  context.Context
	cli  *Client
	sw   *ServerWrap // might be nil: "tcpclient" option
}

func NewLangInstance(ctx context.Context, lang *LangManager) (*LangInstance, error) {
	li := &LangInstance{lang: lang}

	li.ctx = withLangInstanceNamedCancel(ctx)
	earlyErrClear := func() {
		mustCancelLangInstance(li.ctx)
	}

	// start new client/server
	if err := li.start(li.ctx); err != nil {
		earlyErrClear()
		_ = li.Wait() // wait for server/client
		return nil, err
	}

	// initialize client
	if err := li.cli.Initialize(li.ctx); err != nil {
		earlyErrClear()
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
	sw, addr, err := startServerWrapTCP(ctx, li.lang.Reg.Cmd, li.srvOutW())
	if err != nil {
		return err
	}
	li.sw = sw

	// client
	if err := li.startClientTCP(ctx, addr); err != nil {
		// NOTE: not waiting for server in case of error, handled upstream
		return err
	}
	return nil
}

func (li *LangInstance) startClientTCP(ctx context.Context, addr string) error {
	// client connect with retries
	fn := func() error {
		cli0, err := NewClientTCP(ctx, addr, li)
		if err != nil {
			return err
		}
		li.cli = cli0
		return nil
	}
	return ctxutil.RetryIncrease(ctx, 100*time.Millisecond, fn)
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
	cli, err := NewClientIO(ctx, rwc, li)
	if err != nil {
		// NOTE: not waiting for server
		return err
	}
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
	// clear resources
	defer mustCancelLangInstance(li.ctx)

	err := (error)(nil)
	if li.sw != nil { // might be nil: "tcpclient" option (or not started)
		err = iout.MultiErrors(err, li.sw.Wait())
	}
	if li.cli != nil { // might be nil: not started in case of sw start error
		err = iout.MultiErrors(err, li.cli.Wait())
	}
	return err
}

//----------
//----------
//----------

var liNamedCancelStr = "langInstance"

func withLangInstanceNamedCancel(ctx context.Context) context.Context {
	return ctxutil.WithNamedCancel(ctx, liNamedCancelStr)
}
func mustCancelLangInstance(ctx context.Context) {
	ctxutil.MustCancelNamed(ctx, liNamedCancelStr)
}
