package debug

import (
	"net"
	"testing"
)

func init() {
	//logger = log.New(os.Stdout, "debug: ", 0)
	ServerNetwork = "tcp"
	ServerAddress = "127.0.0.1:10002"
	AcceptOnlyFirstClient = true
}

func TestServer1(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}

	c, err := newTestClient()
	if err != nil {
		t.Fatal(err)
	}
	_ = c

	srv.Close()
}

//----------

func newTestClient() (*testClient, error) {
	var dialer net.Dialer
	conn, err := dialer.Dial(ServerNetwork, ServerAddress)
	if err != nil {
		return nil, err
	}
	c := &testClient{conn: conn}
	go c.receiveLoop()
	return c, nil
}

type testClient struct {
	conn net.Conn
}

func (c *testClient) receiveLoop() {
	for {
		msg, err := DecodeMessage(c.conn)
		_, _ = msg, err
		//println(msg, err)
	}
}
