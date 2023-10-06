//go:build !js

package debug

import (
	"context"
	"fmt"
	"net"
	"time"
)

func init() {
	listenReg["tcp"] = listenNet
	listenReg["unix"] = listenNet
	dialReg["tcp"] = dialNet
	dialReg["unix"] = dialNet
}

//----------

type netListener struct {
	Listener
	ctx context.Context
}

func listenNet(ctx context.Context, addr Addr) (Listener, error) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(ctx, addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}
	ln2 := &netListener{ctx: ctx, Listener: ln}
	return ln2, nil
}

func (ln *netListener) Accept() (Conn, error) {
	// ln.ctx can cancel the accept

	// accept timeout
	ctx2 := ln.ctx
	if val, ok := ctx2.Value("connectTimeout").(time.Duration); ok {
		ctx3, cancel := context.WithTimeout(ctx2, val)
		defer cancel()
		ctx2 = ctx3
	}

	type result struct {
		c Conn
		e error
	}
	accept := make(chan *result, 1)
	go func() {
		conn, err := ln.Listener.Accept()
		accept <- &result{conn, err}
	}()

	select {
	case <-ctx2.Done():
		_ = ln.Close() // stops the listener.accept
		return nil, fmt.Errorf("accept: %w", ctx2.Err())
	case res := <-accept:
		return res.c, res.e
	}
}

//----------

func dialNet(ctx context.Context, addr Addr) (Conn, error) {
	// NOTE: use ctx to set a timeout

	d := &net.Dialer{}
	return d.DialContext(ctx, addr.Network(), addr.String())
}
