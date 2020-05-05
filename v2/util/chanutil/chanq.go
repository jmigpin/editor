package chanutil

import (
	"container/list"
)

// Flexible channel queue (no fixed length). Note: consider using syncutil.* instead.
type ChanQ struct {
	q       list.List
	in, out chan interface{}
}

func NewChanQ(inSize, outSize int) *ChanQ {
	ch := &ChanQ{}
	ch.in = make(chan interface{}, inSize)
	ch.out = make(chan interface{}, outSize)
	go ch.loop()
	return ch
}

func (ch *ChanQ) In() chan<- interface{} {
	return ch.in
}

func (ch *ChanQ) Out() <-chan interface{} {
	return ch.out
}

func (ch *ChanQ) loop() {
	var next interface{}
	var out chan<- interface{}
	for {
		select {
		case v := <-ch.in:
			if next == nil {
				next = v
				out = ch.out
			} else {
				ch.q.PushBack(v)
			}
		case out <- next:
			elem := ch.q.Front()
			if elem == nil {
				next = nil
				out = nil
			} else {
				next = elem.Value
				out = ch.out
				ch.q.Remove(elem)
			}
		}
	}
}
