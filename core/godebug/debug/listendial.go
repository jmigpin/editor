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
//----------
//----------

var listenReg = map[string]listenFunc{}
var dialReg = map[string]dialFunc{}

type listenFunc func(context.Context, Addr) (Listener, error)
type dialFunc func(context.Context, Addr) (Conn, error)

//----------

func listen(ctx context.Context, addr Addr) (Listener, error) {
	// listen timeout can be set in the ctx
	// accept timeout is done per implementation

	t := addr.Network()
	fn, ok := listenReg[t]
	if !ok {
		return nil, fmt.Errorf("missing listen network: %v", t)
	}
	return fn(ctx, addr)
}

//----------

func dial(ctx context.Context, addr Addr) (Conn, error) {
	t := addr.Network()
	fn, ok := dialReg[t]
	if !ok {
		return nil, fmt.Errorf("missing dial network: %v", t)
	}
	return fn(ctx, addr)
}
func dialRetry(ctx context.Context, addr Addr) (Conn, error) {
	// dial timeout
	ctx2 := ctx
	if val, ok := ctx2.Value("connectTimeout").(time.Duration); ok {
		ctx3, cancel := context.WithTimeout(ctx2, val)
		defer cancel()
		ctx2 = ctx3
	}

	for {
		conn, err := dial(ctx2, addr)
		if err != nil {
			if ctx2.Err() != nil {
				return nil, fmt.Errorf("dialretry: %v: %w", err)
			}
			// retry until ctx done
			time.Sleep(50 * time.Millisecond) // prevent hot loop
			continue
		}
		return conn, nil
	}
}

//----------
//----------
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
//----------
//----------

const websocketEntryPath = "/editor_debug_ws"

func websocketHostToUrl(host string) string {
	return fmt.Sprintf("ws://%s%s", host, websocketEntryPath)
}
