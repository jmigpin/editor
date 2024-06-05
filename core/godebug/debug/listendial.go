package debug

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type Listener = net.Listener
type Conn = net.Conn
type Addr = net.Addr

//----------

func listen(ctx context.Context, addr Addr) (Listener, error) {
	return listen2(ctx, addr)
}

//----------

func dial(ctx context.Context, addr Addr) (Conn, error) {
	return dial2(ctx, addr)
}

func dialRetry(ctx context.Context, addr Addr) (Conn, error) {
	sleep := 50 * time.Millisecond
	for {
		conn, err := dial(ctx, addr)
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("dialretry: %w: %w", ctx.Err(), err)
			}

			// prevent hot loop
			time.Sleep(sleep)
			sleep *= 2 // next time have a longer wait

			continue
		}
		return conn, nil
	}
}

//----------

// a simple addr implementation
type AddrImpl struct {
	net, str string
}

func NewAddrI(network, str string) *AddrImpl {
	return &AddrImpl{net: network, str: str}
}
func (addr *AddrImpl) Network() string {
	return addr.net
}
func (addr *AddrImpl) String() string {
	h := addr.str
	if k := strings.Index(h, ":"); k == 0 { // ex: ":8080"
		//h = "localhost" + h
		h = "127.0.0.1" + h
	}
	return h
}

//----------

const websocketEntryPath = "/editor_debug_ws"

func websocketEntryPathUrl(host string) string {
	return fmt.Sprintf("ws://%s%s", host, websocketEntryPath)
}
