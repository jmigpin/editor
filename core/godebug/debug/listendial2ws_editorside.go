//go:build !js && !editorDebugExecSide

////go:build !js // DEBUG

package debug

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

func acceptWebsocket(conn net.Conn) (net.Conn, error) {
	// wrap connection to detect closing
	connWg := &sync.WaitGroup{}
	connWg.Add(1)
	closeBeforeWebsocketCh := make(chan bool, 1)
	closeOnce := sync.OnceFunc(func() {
		connWg.Done()
		closeBeforeWebsocketCh <- true
	})
	connWrap := &ConnFnCloser{
		Conn: conn,
		closeFn: func() error {
			closeOnce()
			return conn.Close()
		},
	}

	websocketCh := make(chan *websocket.Conn, 1)
	handler := websocket.Handler(func(wc *websocket.Conn) {
		wc.PayloadType = websocket.BinaryFrame
		websocketCh <- wc
		connWg.Wait() // blocks to keep connection alive
	})

	go func() {
		// there is no other way to access hijack logic easily other then srv.Serve(...), so using a single connection listener that will cause the srv.Serve(...) to exit after first connection
		ln := &singleConnListener{conn: connWrap}
		// ignore entry path, just serve
		//srv := &http.Server{Handler: handler}
		// must have the expected entry path
		smux := http.NewServeMux()
		smux.HandleFunc(websocketEntryPath, handler.ServeHTTP)
		srv := &http.Server{Handler: smux}

		_ = srv.Serve(ln)
	}()

	select {
	case wc := <-websocketCh:
		return wc, nil
	case <-closeBeforeWebsocketCh:
		return nil, fmt.Errorf("http accept websocket error")
	}
}

//----------

type singleConnListener struct {
	conn net.Conn
}

func (s *singleConnListener) Accept() (net.Conn, error) {
	if s.conn != nil {
		conn := s.conn
		s.conn = nil
		return conn, nil
	}
	return nil, fmt.Errorf("single connection already accepted")
}
func (s *singleConnListener) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
func (s *singleConnListener) Addr() Addr {
	return s.conn.LocalAddr()
}

//----------
//----------
//----------

func dialWebsocket(ctx context.Context, addr Addr, conn net.Conn) (Conn, error) {
	srv := websocketEntryPathUrl(addr.String())
	origin := "http://" + addr.String()
	conf, err := websocket.NewConfig(srv, origin)
	if err != nil {
		return nil, err
	}

	type result struct {
		conn net.Conn
		err  error
	}

	// start in goroutine to support ctx cancel
	connCh := make(chan *result, 1)
	go func() {
		wc, err := websocket.NewClient(conf, conn)
		connCh <- &result{wc, err}
	}()

	select {
	case <-ctx.Done():
		_ = conn.Close() // stops websocket.NewClient
		return nil, fmt.Errorf("dial websocket: %w", ctx.Err())
	case res := <-connCh:
		if res.err != nil {
			_ = conn.Close()
		}
		return res.conn, res.err
	}
}
