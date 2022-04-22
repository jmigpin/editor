package debug

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func init() {
	//logger = log.New(os.Stdout, "debug: ", 0)
	hasGenConfig = true
	ServerNetwork = "tcp"
	ServerAddress = "127.0.0.1:10002"
	acceptOnlyFirstClient = true

	// manually register to run tests
	RegisterStructsForEncodeDecode(encoderId)
}

func TestServer1(t *testing.T) {
	clientWait := &sync.WaitGroup{}
	clientWait.Add(1)
	go func() {
		defer clientWait.Done()

		time.Sleep(500 * time.Millisecond)
		c, err := newTestClient()
		if err != nil {
			t.Fatal(err)
		}

		msg := &ReqStartMsg{}
		b, err := EncodeMessage(msg)
		if err != nil {
			t.Fatal(err)
		}
		c.conn.Write(b)

		c.receiveLoop()
		//go c.receiveLoop()
		//c.conn.Close()
	}()

	srv, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}

	srv.Close()

	clientWait.Wait()
}

//----------

type testClient struct {
	conn net.Conn
}

func newTestClient() (*testClient, error) {
	dialer := &net.Dialer{}
	conn, err := dialer.Dial(ServerNetwork, ServerAddress)
	if err != nil {
		return nil, err
	}
	c := &testClient{conn: conn}
	return c, nil
}
func (c *testClient) receiveLoop() {
	for {
		msg, err := DecodeMessage(c.conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Printf("decode error: %v\n", err)
			}
			return
		}
		fmt.Printf("msg: %#v\n", msg)
	}
}
