//go:build !js

package debug

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

// besides direct tcp/unix, allows on demand websocket clients
type Listener2 struct {
	Listener
	addr Addr

	mu struct {
		sync.Mutex
		closed bool
		conn   net.Conn
	}
}

func listen2(ctx context.Context, addr Addr) (Listener, error) {
	lc := &net.ListenConfig{}

	network := addr.Network()
	if network == "ws" || network == "auto" {
		network = "tcp"
	}

	ln, err := lc.Listen(ctx, network, addr.String())
	if err != nil {
		return nil, err
	}
	ln2 := &Listener2{Listener: ln, addr: addr}
	return ln2, nil
}

//----------

func (ln *Listener2) Accept() (Conn, error) {
	conn, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// accept() can block while upgrading the connection (ex: websocket). It can be canceled by closing the listener. In this case, the conn already exists, and so it needs to be kept to be closed, such that it can unblock a read() with an error
	ln.mu.Lock()
	if ln.mu.closed {
		ln.mu.Unlock()
		return nil, fmt.Errorf("listener was closed")
	}
	ln.mu.conn = conn
	ln.mu.Unlock()

	switch ln.addr.Network() {
	case "ws":
		conn, err = acceptWebsocket(conn)
	case "auto":
		conn, err = ln.acceptAuto(conn)
	}

	ln.mu.Lock()
	ln.mu.conn = nil
	ln.mu.Unlock()

	return conn, err
}
func (ln *Listener2) acceptAuto(conn net.Conn) (Conn, error) {
	// peek for http header
	br := bufio.NewReader(conn)
	peek, err := br.Peek(10) // size to cover isHTTPRequest test
	// io.EOF error can have a valid peek
	if err != nil && err != io.EOF {
		return nil, err
	}

	// try to accept websocket
	pc := &PeekedConn{Conn: conn, peeker: br}
	if isHTTPRequest(peek) {
		return acceptWebsocket(pc)
	}

	// accept tcp conn
	return pc, nil
}

//----------

func (ln *Listener2) Close() error {
	// handle accept() canceling
	ln.mu.Lock()
	defer ln.mu.Unlock()
	if ln.mu.conn != nil {
		_ = ln.mu.conn.Close()
		ln.mu.closed = true
	}

	return ln.Listener.Close()
}

//----------

type PeekedConn struct {
	net.Conn
	peeker *bufio.Reader
}

func (c *PeekedConn) Read(b []byte) (int, error) {
	return c.peeker.Read(b)
}

//----------

func isHTTPRequest(data []byte) bool {
	methods := []string{
		"GET ", "POST ", "PUT ", "DELETE ",
		"HEAD ", "OPTIONS ", "PATCH ",
	}
	for _, method := range methods {
		if bytes.HasPrefix(data, []byte(method)) {
			return true
		}
	}
	return false
}

//----------
//----------
//----------

func dial2(ctx context.Context, addr Addr) (Conn, error) {
	network := addr.Network()
	if network == "ws" {
		network = "tcp"
	}

	d := &net.Dialer{}
	conn, err := d.DialContext(ctx, network, addr.String())
	if err != nil {
		return nil, err
	}

	if addr.Network() == "ws" {
		return dialWebsocket(ctx, addr, conn)
	}

	return conn, nil
}
