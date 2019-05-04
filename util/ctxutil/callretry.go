package ctxutil

import (
	"context"
	"fmt"
	"time"
)

// Returns fn result or return early on ctx cancel. If fn does not return in time, lateFn will run at the end of fn (async). Timed errors will be sent if asyncErrors!=nil.
func Call(ctx context.Context, prefix string, asyncErrors chan<- error, fn func() error, lateFn func(error)) error {
	buildErr := func(e error) error {
		return fmt.Errorf("%v: %v", prefix, e)
	}

	// keep track if fn ever returns
	fnDone := make(chan interface{})
	if asyncErrors != nil {
		go waitForDone(prefix, fnDone, asyncErrors)
	}

	// run fn in go routine
	ch := make(chan error, 1)
	go func() {
		defer close(fnDone)
		err := fn() // goroutine leaks if fn never returns
		select {
		case <-ctx.Done():
			// too late, send error async
			//asyncErrors <- buildErr(err)
			if lateFn != nil {
				lateFn(buildErr(err))
			}
		default: // don't block select
			ch <- err // won't block due to size 1 in chan
		}
	}()

	select {
	case err := <-ch:
		return buildErr(err)
	case <-ctx.Done():
		return buildErr(ctx.Err())
	}
}

func waitForDone(prefix string, done <-chan interface{}, asyncErrors chan<- error) {
	start := time.Now()
	t := time.NewTicker(10 * time.Second) // send err every x secs
	defer t.Stop()
	sentErr := false
	for {
		select {
		case <-t.C:
			sentErr = true
			d := time.Now().Sub(start)
			asyncErrors <- fmt.Errorf("%v: waiting for fn: %v", prefix, d)
		case <-done:
			// send error if it had sent errors before
			if sentErr {
				d := time.Now().Sub(start)
				asyncErrors <- fmt.Errorf("%v: fn returned: %v", prefix, d)
			}

			return
		}
	}
}

//----------

func Retry(ctx context.Context, sleep time.Duration, prefix string, asyncErrors chan<- error, fn func() error, lateFn func(error)) error {
	var err error
	for {
		err = Call(ctx, prefix, asyncErrors, fn, lateFn)
		if err == nil {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		default: // non-blocking select
			time.Sleep(sleep) // sleep before next retry
		}
	}
}

//func Retry(ctx context.Context, sleep time.Duration, prefix string, fn func() error) error {
//	buildErr := func(e error) error {
//		return fmt.Errorf("%v: %v", prefix, e)
//	}

//	var err error
//	for {
//		ch := make(chan error, 1) // size 1 to avoid goroutine leak
//		go func() {
//			ch <- fn() // able to exit at the end with chan size 1
//		}()

//		select {
//		case err = <-ch:
//			if err == nil {
//				return nil
//			}
//			// sleep before next retry
//			time.Sleep(sleep)
//		case <-ctx.Done():
//			// include previous set error if any
//			err2 := me.MultiErrors(err, ctx.Err())
//			return buildErr(err2)
//		}
//	}
//}
