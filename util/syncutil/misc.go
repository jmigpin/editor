package syncutil

import (
	"sync"
)

func WaitDone(fn func(done func())) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	fn(wg.Done)
	wg.Wait()
}
