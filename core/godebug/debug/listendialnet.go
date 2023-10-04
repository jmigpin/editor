//go:build !js

package debug

import (
	"context"
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
	case <-ln.ctx.Done():
		_ = ln.Close() // stop the accept
		return nil, ln.ctx.Err()
	case res := <-accept:
		return res.c, res.e
	}
}

//----------

func dialNet(ctx context.Context, addr Addr, timeout time.Duration) (Conn, error) {
	d := &net.Dialer{Timeout: timeout}
	return d.DialContext(ctx, addr.Network(), addr.String())
}
