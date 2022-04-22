package godebug

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/godebug/debug"
)

type Client struct {
	Conn     net.Conn
	Messages chan interface{}
	waitg    sync.WaitGroup
}

func NewClient(ctx context.Context, network, addr string) (*Client, error) {
	client := &Client{
		Messages: make(chan interface{}, 64),
	}
	if err := client.connect(ctx, network, addr); err != nil {
		return nil, fmt.Errorf("client connect: %w", err)
	}

	// ensure connection close on ctx cancel
	go func() {
		select {
		case <-ctx.Done():
			client.Conn.Close()
		}
	}()

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
	if client.Conn != nil {
		return client.Conn.Close()
	}
	return nil
}

func (client *Client) connect(ctx context.Context, network, addr string) error {
	// timeout to get a sucessful connection
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	retry := 250 * time.Millisecond
	for {
		dialer := &net.Dialer{}
		dialer.Timeout = retry
		conn, err := dialer.DialContext(ctx, network, addr)
		if err == nil {
			client.Conn = conn
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default: // non-blocking
			if retry < 5*time.Second {
				retry += retry // increase for slower conns
			}
		}
	}
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
