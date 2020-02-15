package ctxutil

import "context"

func WatchDone(ctx context.Context, cancel context.CancelFunc, ctxs ...context.Context) func() {
	ch := make(chan struct{}, 1)
	clearWatching := func() {
		ch <- struct{}{}
	}
	// any ctx done will cancel
	for _, ctx := range ctxs {
		go func(ctx context.Context) {
			select {
			case <-ch: // release goroutine
			case <-ctx.Done():
				cancel()
			}
		}(ctx)
	}
	return clearWatching
}
