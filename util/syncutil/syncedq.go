package syncutil

import (
	"container/list"
	"sync"
)

type SyncedQ struct {
	sync.Mutex
	cond *sync.Cond
	q    list.List
}

func NewSyncedQ() *SyncedQ {
	sq := &SyncedQ{}
	sq.cond = sync.NewCond(sq)
	return sq
}

func (sq *SyncedQ) PushBack(v any) {
	sq.Lock()
	sq.q.PushBack(v)
	sq.Unlock()
	sq.cond.Signal()
}

// Waits until a value is available
func (sq *SyncedQ) PopFront() any {
	sq.Lock()
	for sq.q.Len() == 0 {
		sq.cond.Wait()
	}
	defer sq.Unlock()

	e := sq.q.Front()
	v := e.Value
	sq.q.Remove(e)
	return v
}

func (sq *SyncedQ) PeekFront() (any, bool) {
	sq.Lock()
	defer sq.Unlock()
	if sq.q.Len() == 0 {
		return nil, false
	}
	e := sq.q.Front()
	return e.Value, true
}
