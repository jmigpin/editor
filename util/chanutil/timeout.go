package chanutil

import (
	"context"
	"fmt"
	"time"
)

// Run fn or return early with error after duration d. Note fn continues to run.
func CallTimeout(ctx context.Context, d time.Duration, errStrPrefix string, fn func() error) error {
	t := time.NewTicker(d)
	defer t.Stop()

	ch := make(chan error)
	go func() {
		ch <- fn()
	}()

	select {
	case err := <-ch:
		return err
	case <-t.C:
		return fmt.Errorf("call timeout: %v, %v", errStrPrefix, d)
	case <-ctx.Done():
		return ctx.Err()
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
