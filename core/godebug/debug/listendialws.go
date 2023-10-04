//go:build !js && !editorDebugExecSide

// NOTE: provides websocket server for the editor
// NOTE: compiling this without !editorDebugExecSide, will require that the lib "golang.org/x/net/websocket" be available on the exec side for compilation
// NOTE: on the other hand, compiling with !editorDebugExecSide, will not have a ws dialer for the client in tests

package debug

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

func init() {
	listenReg["ws"] = listenWebsocket
	listenReg["wss"] = listenWebsocket
	dialReg["ws"] = dialWebsocket
	dialReg["wss"] = dialWebsocket
}

//----------
//----------
//----------

// websocket listener
type wsListener struct {
	Listener
	acceptCh chan interface{}
}

func listenWebsocket(ctx context.Context, addr Addr) (Listener, error) {
	addr2 := NewAddrI("tcp", addr.String())
	ln0, err := listen(ctx, addr2)
	if err != nil {
		return nil, err
	}

	ln := &wsListener{Listener: ln0}
	ln.acceptCh = make(chan interface{}, 2)

	// server
	srv := &http.Server{Addr: addr.String()}

	// each connection context
	//srv.BaseContext = func(ln net.Listener) context.Context {
	//	return ctx
	//}

	// server connection handler
	handler := websocket.Handler(func(wc *websocket.Conn) {
		wc.PayloadType = websocket.BinaryFrame

		//----------

		// keep connection alive, don't return from the function
		// TODO: in the case of the client closing, this is leaking since the conn.close is never called
		c2 := &ConnCloser{Conn: wc}
		alive := &sync.WaitGroup{}
		alive.Add(1)
		aliveDoneOnce := &sync.Once{}
		c2.closeFn = func() error {
			defer aliveDoneOnce.Do(alive.Done)
			return wc.Close()
		}
		defer alive.Wait() // blocks

		//----------

		ln.acceptCh <- c2
	})

	sm := &http.ServeMux{}
	sm.Handle(websocketEntryPath, handler)
	srv.Handler = sm

	// serve loop
	go func() {
		// flow: serve()->netln.accept->handler->wsln.accept
		err := srv.Serve(ln.Listener)
		if err != nil {
			ln.acceptCh <- err
		}
	}()

	return ln, nil
}
func (ln *wsListener) Accept() (Conn, error) {
	v, ok := <-ln.acceptCh
	if !ok {
		return nil, fmt.Errorf("chan closed")
	}
	switch t := v.(type) {
	case error:
		return nil, t
	case net.Conn:
		return t, nil
	default:
		panic(fmt.Errorf("unexpected: %T", t))
	}
}

//----------

func dialWebsocket(ctx context.Context, addr Addr, timeout time.Duration) (Conn, error) {

	u := websocketHostToUrl(addr.String())
	//origin := strings.Replace(u, "ws://", "http://", 1)
	origin := "http://" + addr.String()

	//----------

	//// NOTE: missing context support
	//wc, err := websocket.Dial(url, "", origin)
	//if err != nil {
	//	return nil, err
	//}
	//return wc, nil

	//----------

	conf, err := websocket.NewConfig(u, origin)
	if err != nil {
		return nil, err
	}
	//conf.Protocol = []string{"websocket"}

	addr2 := NewAddrI("tcp", addr.String())
	conn, err := dialNet(ctx, addr2, timeout)
	if err != nil {
		return nil, err
	}

	wc, err := websocket.NewClient(conf, conn)
	if err != nil {
		_ = conn.Close()
	}

	return wc, err
}

//----------
//----------
//----------

type ConnCloser struct {
	Conn
	closeFn func() error
}

func (cc *ConnCloser) Close() error {
	return cc.closeFn()
}
