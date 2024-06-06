package debug

import (
	"context"
	"fmt"
	"net"
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
