package ctxutil

import (
	"context"
	"io"
	"time"
)

func RetryIncrease(ctx context.Context, retryPause time.Duration, fn func() error) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}
		if err2 := Sleep(ctx, retryPause); err2 != nil {
			return err // fn error
		}
		retryPause *= 2
	}
}

//----------

func Sleep(ctx context.Context, dur time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(dur):
		return nil
	}
}

//----------

func PipeWithContext(ctx context.Context) (*io.PipeReader, *io.PipeWriter) {
	r, w := io.Pipe()

	go func() {
		<-ctx.Done()
		// closing the reader propagates error to writer
		_ = r.CloseWithError(ctx.Err())
	}()

	return r, w
}

//----------

type cancelKey struct{ name string }

func WithNamedCancel(ctx context.Context, name string) context.Context {
	ctx2, cancel := context.WithCancel(ctx)
	return context.WithValue(ctx2, cancelKey{name}, cancel)
}

func GetNamedCancel(ctx context.Context, name string) (context.CancelFunc, bool) {
	v := ctx.Value(cancelKey{name})
	c, ok := v.(context.CancelFunc)
	return c, ok
}

func TryCancelNamed(ctx context.Context, name string) bool {
	if c, ok := GetNamedCancel(ctx, name); ok {
		c()
		return true
	}
	return false
}

func MustCancelNamed(ctx context.Context, name string) {
	if !TryCancelNamed(ctx, name) {
		panic("named cancel not found: " + name)
	}
}
