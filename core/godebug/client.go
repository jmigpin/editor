package godebug

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/chanutil"
)

type Client struct {
	Conn     net.Conn
	Messages chan interface{}
	waitg    sync.WaitGroup
	done     chan interface{}
}

func NewClient(ctx context.Context) (*Client, error) {
	client := &Client{
		Messages: make(chan interface{}, 512), // TODO: group server msgs
	}
	if err := client.connect(ctx); err != nil {
		return nil, err
	}

	client.Messages <- "connected"

	// receive msgs from server and send to channel
	client.waitg.Add(1)
	go func() {
		defer client.waitg.Done()
		client.receiveLoop()
	}()

	return client, nil
}

func (client *Client) Wait() {
	client.waitg.Wait()
}

func (client *Client) Close() error {
	close(client.done) // on close, ensure no goroutine leaks
	return client.Conn.Close()
}

func (client *Client) connect(ctx context.Context) error {
	client.done = make(chan interface{})

	retry := 5 * time.Second
	sleep := 200 * time.Millisecond
	err := chanutil.RetryTimeout(ctx, retry, sleep, "connect", func() error {
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, debug.ServerNetwork, debug.ServerAddress)
		if err != nil {
			return err
		}
		client.Conn = conn

		// on context cancel, ensure client close
		go func() {
			select {
			case <-client.done: // ensure no goroutine leaks
			case <-ctx.Done():
				_ = client.Close()
			}
		}()
		return nil
	})
	return err
}

func (client *Client) receiveLoop() {
	defer close(client.Messages)
	for {
		msg, err := debug.DecodeMessage(client.Conn)
		if err != nil {
			// unable to read (server was probably closed)
			if operr, ok := err.(*net.OpError); ok {
				if operr.Op == "read" {
					break
				}
			}
			// connection ended gracefully by the client
			if err == io.EOF {
				break
			}

			client.Messages <- err
			continue
		}

		client.Messages <- msg
	}
}
