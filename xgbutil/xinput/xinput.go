package xinput

import (
	"image"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type XInput struct {
	km    *KMap
	evReg *evreg.Register

	// detect buttons double/triple clicks, only for buttons 1,2,3, not wheel buttons
	buttonPressedTime [3]struct {
		p      image.Point
		t      time.Time
		action int
	}
}

func NewXInput(conn *xgb.Conn, evReg *evreg.Register) (*XInput, error) {
	km, err := NewKMap(conn)
	if err != nil {
		return nil, err
	}

	xi := &XInput{km: km, evReg: evReg}

	xi.evReg = evReg
	xi.evReg.Add(xproto.KeyPress,
		&evreg.Callback{xi.onEvRegKeyPress})
	xi.evReg.Add(xproto.KeyRelease,
		&evreg.Callback{xi.onEvRegKeyRelease})
	xi.evReg.Add(xproto.ButtonPress,
		&evreg.Callback{xi.onEvRegButtonPress})
	xi.evReg.Add(xproto.ButtonRelease,
		&evreg.Callback{xi.onEvRegButtonRelease})
	xi.evReg.Add(xproto.MotionNotify,
		&evreg.Callback{xi.onEvRegMotionNotify})

	return xi, nil
}
func (xi *XInput) onEvRegKeyPress(ev0 interface{}) {
	ev := ev0.(xproto.KeyPressEvent)
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	k := NewKey(xi.km, ev.Detail, ev.State)
	ev2 := &KeyPressEvent{p, k}
	xi.evReg.Emit(KeyPressEventId, ev2)
}
func (xi *XInput) onEvRegKeyRelease(ev0 interface{}) {
	ev := ev0.(xproto.KeyReleaseEvent)
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	k := NewKey(xi.km, ev.Detail, ev.State)
	ev2 := &KeyReleaseEvent{p, k}
	xi.evReg.Emit(KeyReleaseEventId, ev2)
}
func (xi *XInput) onEvRegButtonPress(ev0 interface{}) {
	ev := ev0.(xproto.ButtonPressEvent)
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	b := NewButton(xi.km, ev.Detail, ev.State)

	// double and triple clicks
	// buttons 4 and 5 are wheel up/down, double/tripple click should not affect them
	// TODO: mods mapping could affect this
	index := int(b.button)
	if index >= 1 && index <= 3 {
		bpt := &xi.buttonPressedTime[index-1]

		t0 := bpt.t
		p0 := bpt.p

		// update time and point
		bpt.t = time.Now()
		bpt.p = *p

		d := bpt.t.Sub(t0)
		if d > 400*time.Millisecond {
			bpt.action = 0
		} else {
			pad := image.Point{1, 1}
			var r image.Rectangle
			r.Min = p0.Sub(pad)
			r.Max = p0.Add(pad)

			if p.In(r) {
				bpt.action = (bpt.action + 1) % 3
			} else {
				bpt.action = 0
			}

			switch bpt.action {
			case 1:
				ev2 := &DoubleClickEvent{p, b}
				xi.evReg.Emit(DoubleClickEventId, ev2)
				return
			case 2:
				ev2 := &TripleClickEvent{p, b}
				xi.evReg.Emit(TripleClickEventId, ev2)
				return
			}
		}
	}

	ev2 := &ButtonPressEvent{p, b}
	xi.evReg.Emit(ButtonPressEventId, ev2)
}
func (xi *XInput) onEvRegButtonRelease(ev interface{}) {
	ev0 := ev.(xproto.ButtonReleaseEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	b := NewButton(xi.km, ev0.Detail, ev0.State)
	ev2 := &ButtonReleaseEvent{p, b}
	xi.evReg.Emit(ButtonReleaseEventId, ev2)
}
func (xi *XInput) onEvRegMotionNotify(ev interface{}) {
	ev0 := ev.(xproto.MotionNotifyEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	m := Modifiers(ev0.State)
	ev2 := &MotionNotifyEvent{p, m}
	xi.evReg.Emit(MotionNotifyEventId, ev2)
}

const (
	KeyPressEventId = evreg.XInputEventIdStart + iota
	KeyReleaseEventId
	ButtonPressEventId
	ButtonReleaseEventId
	MotionNotifyEventId
	DoubleClickEventId
	TripleClickEventId
)

type KeyPressEvent struct {
	Point *image.Point
	Key   *Key
}
type KeyReleaseEvent struct {
	Point *image.Point
	Key   *Key
}
type ButtonPressEvent struct {
	Point  *image.Point
	Button *Button
}
type ButtonReleaseEvent struct {
	Point  *image.Point
	Button *Button
}
type MotionNotifyEvent struct {
	Point *image.Point
	Mods  Modifiers
}

type DoubleClickEvent struct {
	Point  *image.Point
	Button *Button
}
type TripleClickEvent struct {
	Point  *image.Point
	Button *Button
}
