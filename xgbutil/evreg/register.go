package evreg

import "container/list"

type Register struct {
	m map[int]*list.List

	events chan<- interface{}

	UnhandledEventFunc func(ev *EventWrap)
}

func NewRegister() *Register {
	er := &Register{m: make(map[int]*list.List)}
	return er
}
func (er *Register) Add(evId int, cb *Callback) *Regist {
	l, ok := er.m[evId]
	if !ok {
		l = list.New()
		er.m[evId] = l
	}
	l.PushBack(cb)
	return &Regist{er, evId, cb}
}
func (er *Register) Remove(evId int, cb *Callback) {
	l, ok := er.m[evId]
	if !ok {
		return
	}
	for e := l.Front(); e != nil; e = e.Next() {
		cb2 := e.Value.(*Callback)
		if cb2 == cb {
			l.Remove(e)
			if l.Len() == 0 {
				delete(er.m, evId)
			}
			break
		}
	}
}

// TODO: rename runcallbacks
func (er *Register) Emit(evId int, ev interface{}) {
	l, ok := er.m[evId]
	if !ok {
		fn := er.UnhandledEventFunc
		if fn != nil {
			ev2 := &EventWrap{evId, ev}
			fn(ev2)
		}
		return
	}
	for e := l.Front(); e != nil; e = e.Next() {
		cb := e.Value.(*Callback)
		cb.F(ev)
	}
}

//func (er *Register) Enqueue(evId int, ev interface{}) {
//	// run inside goroutine to not allow deadlocks
//	go func() { er.events <- &EventWrap{evId, ev} }()
//}

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
