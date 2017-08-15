package xgbutil

type EventRegister struct {
	m map[int]*[]*ERCallback
}

func NewEventRegister() *EventRegister {
	er := &EventRegister{m: make(map[int]*[]*ERCallback)}
	return er
}
func (er *EventRegister) Add(evId int, cb *ERCallback) *ERRegist {
	u, ok := er.m[evId]
	if !ok {
		u = &[]*ERCallback{}
		er.m[evId] = u
	}
	*u = append(*u, cb)
	return &ERRegist{er, evId, cb}
}
func (er *EventRegister) Remove(evId int, cb *ERCallback) {
	u, ok := er.m[evId]
	if !ok {
		return
	}
	for i, cb0 := range *u {
		if cb0 == cb {
			// remove
			*u = append((*u)[:i], (*u)[i+1:]...)
			// copy to ensure a short slice
			u2 := make([]*ERCallback, len(*u))
			copy(u2, *u)
			*u = u2

			if len(*u) == 0 {
				delete(er.m, evId)
			}
			break
		}
	}
}
func (er *EventRegister) Emit(evId int, ev EREvent) {
	u, ok := er.m[evId]
	if !ok {
		//log.Printf("unhandled event id: %v, %#v", evId, ev)
		return
	}
	for _, cb := range *u {
		cb.F(ev)
	}
}

type EREvent interface{} // TODO: remove, and use interface{}
type ERCallback struct {
	F func(EREvent)
}

type ERRegist struct {
	evReg *EventRegister
	id    int
	cb    *ERCallback
}

func (reg *ERRegist) Unregister() {
	reg.evReg.Remove(reg.id, reg.cb)
}

type EventDeregister struct {
	v []*ERRegist
}

func (d *EventDeregister) Add(u ...*ERRegist) {
	d.v = append(d.v, u...)
}
func (d *EventDeregister) UnregisterAll() {
	for _, e := range d.v {
		e.Unregister()
	}
	d.v = []*ERRegist{}
}
