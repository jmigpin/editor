package chanutil

import (
	"context"
	"fmt"
	"time"
)

// Run fn or return early with error after duration d. Note fn continues to run.
func CallTimeout(ctx context.Context, d time.Duration, msgPrefix string, asyncErrors chan<- error, fn func() error) error {
	t := time.NewTicker(d)
	defer t.Stop()

	// keep track if fn ever returns
	fnDone := make(chan interface{})
	if asyncErrors != nil {
		go waitForDone(fnDone, asyncErrors, msgPrefix)
	}

	// run fn in go routine
	ch := make(chan error, 1)
	go func() {
		defer close(fnDone)
		err := fn() // goroutine leaks if fn never returns
		select {
		case <-ctx.Done(): // too late, send error async (else err is lost)
			if asyncErrors != nil {
				asyncErrors <- err
			}
		default: // don't block select
			ch <- err // won't block due to size 1 in chan
		}
	}()

	select {
	case err := <-ch:
		return err
	case <-t.C:
		return fmt.Errorf("call timeout: %v, %v", msgPrefix, d)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func waitForDone(done <-chan interface{}, asyncErrors chan<- error, msgPrefix string) {
	start := time.Now()
	t := time.NewTicker(10 * time.Second) // send err every x secs
	defer t.Stop()
	sentErr := false
	for {
		select {
		case <-t.C:
			sentErr = true
			d := time.Now().Sub(start)
			asyncErrors <- fmt.Errorf("waiting for fn (%s): %v", msgPrefix, d)
		case <-done:
			// send final error if it had sent errors before
			if sentErr {
				d := time.Now().Sub(start)
				asyncErrors <- fmt.Errorf("fn returned (%s): %v", msgPrefix, d)
			}

			return
		}
	}
}

//----------

func RetryTimeout(ctx context.Context, retry, sleep time.Duration, errStrPrefix string, fn func() error) error {
	t := time.NewTicker(retry)
	defer t.Stop()

	buildErr := func(e error) error {
		return fmt.Errorf("%v: %v", errStrPrefix, e)
	}

	var err error
	for {
		ch := make(chan error)
		go func() {
			ch <- fn()
		}()

		select {
		case err = <-ch:
			if err == nil {
				return nil
			}
			// continue and retry while there is time
			// sleep to avoid high cpu usage
			time.Sleep(sleep)
		case <-t.C:
			// return last known error
			if err != nil {
				return err
			}
			// timeout error
			return buildErr(fmt.Errorf("retry timeout: %v", retry))
		case <-ctx.Done():
			return buildErr(ctx.Err())
		}
	}
}
