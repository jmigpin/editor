package evreg

import "container/list"

type Register struct {
	m map[int]*list.List

	Events chan<- interface{}
}

func NewRegister() *Register {
	reg := &Register{m: make(map[int]*list.List)}
	return reg
}
func (reg *Register) AddCallback(evId int, cb *Callback) *Regist {
	l, ok := reg.m[evId]
	if !ok {
		l = list.New()
		reg.m[evId] = l
	}
	l.PushBack(cb)
	return &Regist{reg, evId, cb}
}
func (reg *Register) Add(evId int, fn func(interface{})) *Regist {
	return reg.AddCallback(evId, &Callback{fn})
}
func (reg *Register) Remove(evId int, cb *Callback) {
	l, ok := reg.m[evId]
	if !ok {
		return
	}
	for e := l.Front(); e != nil; e = e.Next() {
		cb2 := e.Value.(*Callback)
		if cb2 == cb {
			l.Remove(e)
			if l.Len() == 0 {
				delete(reg.m, evId)
			}
			break
		}
	}
}

func (reg *Register) RunCallbacks(evId int, ev interface{}) int {
	l, ok := reg.m[evId]
	if !ok {
		return 0
	}
	c := 0
	for e := l.Front(); e != nil; e = e.Next() {
		cb := e.Value.(*Callback)
		cb.F(ev)
		c++
	}
	return c
}

func (reg *Register) Enqueue(evId int, ev interface{}) {
	// run inside goroutine to not allow deadlocks
	//go func() { reg.Events <- &EventWrap{evId, ev} }()

	// ensures call event ordreg if not inside a goroutine
	reg.Events <- &EventWrap{evId, ev}
}
func (reg *Register) EnqueueError(err error) {
	reg.Enqueue(ErrorEventId, err)
}

type Callback struct {
	F func(interface{})
}

type EventWrap struct {
	EventId int
	Event   interface{}
}

type Regist struct {
	evReg *Register
	id    int
	cb    *Callback
}

func (reg *Regist) Unregister() {
	reg.evReg.Remove(reg.id, reg.cb)
}

type Unregister struct {
	v []*Regist
}

func (unr *Unregister) Add(u ...*Regist) {
	unr.v = append(unr.v, u...)
}
func (unr *Unregister) UnregisterAll() {
	for _, e := range unr.v {
		e.Unregister()
	}
	unr.v = []*Regist{}
}
