package ctxutil

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Allows fn to return early on ctx cancel. If fn does not return in time, lateFn will run at the end of fn (async).
func Call(ctx context.Context, prefix string, fn func() error, lateFn func(error)) error {
	buildErr := func(e error) error {
		return fmt.Errorf("%v: %v", prefix, e)
	}

	type data struct {
		sync.Mutex
		exited   bool
		returned bool
		err      error
	}
	var d data

	// run fn in go routine
	ctx2, cancel := context.WithCancel(ctx)
	id := addCall(prefix)
	go func() {
		defer doneCall(id)

		err := fn() // goroutine leaks if fn never returns
		if err != nil {
			err = buildErr(err)
		}

		d.Lock()
		defer func() {
			d.returned = true
			cancel()
			d.Unlock()
		}()

		if d.exited {
			if lateFn != nil {
				lateFn(err)
			} else {
				// err is lost
			}
		} else {
			d.err = err
		}
	}()

	<-ctx2.Done()
	d.Lock()
	defer func() {
		d.exited = true
		d.Unlock()
	}()

	if d.returned {
		return d.err
	}

	return buildErr(ctx2.Err())
}

//----------

func Retry(ctx context.Context, sleep time.Duration, prefix string, fn func() error, lateFn func(error)) error {
	var err error
	for {
		err = Call(ctx, prefix, fn, lateFn)
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

//----------

type cdata struct {
	t time.Time
	s string
}

var cmu sync.Mutex
var callm = map[int]*cdata{}
var ci = 0

func addCall(s string) int {
	cmu.Lock()
	defer cmu.Unlock()
	ci++
	callm[ci] = &cdata{s: s, t: time.Now()}
	return ci
}

func doneCall(v int) {
	cmu.Lock()
	defer cmu.Unlock()
	delete(callm, v)
}

func CallsState() string {
	cmu.Lock()
	defer cmu.Unlock()
	u := []string{}
	now := time.Now()
	for _, d := range callm {
		s := fmt.Sprintf("%v: %v ago", d.s, now.Sub(d.t))
		u = append(u, s)
	}
	return fmt.Sprintf("%v entries\n%v\n", len(u), strings.Join(u, "\n"))
}
