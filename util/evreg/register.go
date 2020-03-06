package evreg

import "container/list"

// The zero register is empty and ready for use.
type Register struct {
	m map[int]*list.List
}

//----------

// Remove is done via *Regist.Unregister().
func (reg *Register) Add(evId int, fn func(interface{})) *Regist {
	return reg.AddCallback(evId, &Callback{fn})
}

//----------

func (reg *Register) AddCallback(evId int, cb *Callback) *Regist {
	if reg.m == nil {
		reg.m = map[int]*list.List{}
	}
	l, ok := reg.m[evId]
	if !ok {
		l = list.New()
		reg.m[evId] = l
	}
	l.PushBack(cb)
	return &Regist{reg, evId, cb}
}

func (reg *Register) RemoveCallback(evId int, cb *Callback) {
	if reg.m == nil {
		return
	}
	l, ok := reg.m[evId]
	if !ok {
		return
	}
	// iterate to remove since the callback doesn't keep the element (allows callback to be added more then once, or at different evId's - this is probably a useless feature unless the *callback is being used to also be set in a map)
	for e := l.Front(); e != nil; e = e.Next() {
		cb2 := e.Value.(*Callback)
		if cb2 == cb {
			l.Remove(e)
			if l.Len() == 0 {
				delete(reg.m, evId)
				break
			}
			// Commented: to continue to remove if added more then once
			// break
		}
	}
}

//----------

// Returns number of callbacks done.
func (reg *Register) RunCallbacks(evId int, ev interface{}) int {
	if reg.m == nil {
		return 0
	}
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

// Number of registered callbacks for an event id.
func (reg *Register) NCallbacks(evId int) int {
	if reg.m == nil {
		return 0
	}
	l, ok := reg.m[evId]
	if !ok {
		return 0
	}
	return l.Len()
}

//----------

type Callback struct {
	F func(ev interface{})
}

//----------

type Regist struct {
	evReg *Register
	id    int
	cb    *Callback
}

func (reg *Regist) Unregister() {
	reg.evReg.RemoveCallback(reg.id, reg.cb)
}

//----------

// Utility to unregister big number of regists.
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
