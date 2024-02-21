package ctxutil

import (
	"context"
	"sync"
)

func WatchDone(ctx context.Context, cancel func()) func() {
	ch := make(chan struct{}, 1) // 1=receive clearwatching if ctx already done

	// ensure sync with the receiver, otherwise clearwatching could be called and exit and the ctx.done be called later on the same goroutine and still arrive before the clearwatching signal
	var cancelMu sync.Mutex

	go func() {
		select {
		case <-ch: // release goroutine on clearwatching()
		case <-ctx.Done():
			cancelMu.Lock()
			if cancel != nil { // could be cleared already by clearwatching()
				cancel()
			}
			cancelMu.Unlock()
		}
	}()

	clearWatching := func() {
		cancelMu.Lock()
		cancel = nil
		cancelMu.Unlock()
		ch <- struct{}{} // send to release goroutine
	}

	return clearWatching
}
