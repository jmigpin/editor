package xgbutil

// Register events.
type EventRegister struct {
	m                map[int]*[]*ERCallback
	OnUnhandledEvent func(evId int, ev EREvent)
}

type EREvent interface{}
type ERCallback struct {
	F func(EREvent)
}

func NewEventRegister() *EventRegister {
	er := &EventRegister{m: make(map[int]*[]*ERCallback)}
	return er
}
func (er *EventRegister) Add(evId int, cb *ERCallback) {
	u, ok := er.m[evId]
	if !ok {
		u = &[]*ERCallback{}
		er.m[evId] = u
	}
	*u = append(*u, cb)
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
		if er.OnUnhandledEvent != nil {
			er.OnUnhandledEvent(evId, ev)
		}
		return
	}
	for _, cb := range *u {
		cb.F(ev)
	}
}
