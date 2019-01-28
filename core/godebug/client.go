package godebug

import (
	"context"
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

func NewClient(ctx context.Context) (*Client, error) {
	client := &Client{
		Messages: make(chan interface{}, 512),
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
	return client.Conn.Close()
	return nil
}

func (client *Client) connect(ctx context.Context) error {
	// connect to server with retries during a period
	end := time.Now().Add(5 * time.Second)
	for {
		// connect
		var dialer net.Dialer
		conn0, err := dialer.DialContext(ctx, debug.ServerNetwork, debug.ServerAddress)
		if err != nil {
			// retry while the end time is not reached
			if time.Now().Before(end) {
				timer := time.NewTimer(250 * time.Millisecond)
				select {
				case <-timer.C:
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return err
		}

		// connected
		client.Conn = conn0

		// close client if context gets canceled
		go func() {
			<-ctx.Done()
			_ = client.Close()
		}()

		return nil
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
