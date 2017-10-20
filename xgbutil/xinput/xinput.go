package xinput

import (
	"image"
	"log"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xgbutil/evreg"

	"github.com/jmigpin/editor/uiutil/event"
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
	xi.evReg.Add(xproto.KeyPress, xi.onEvRegKeyPress)
	xi.evReg.Add(xproto.KeyRelease, xi.onEvRegKeyRelease)
	xi.evReg.Add(xproto.ButtonPress, xi.onEvRegButtonPress)
	xi.evReg.Add(xproto.ButtonRelease, xi.onEvRegButtonRelease)
	xi.evReg.Add(xproto.MotionNotify, xi.onEvRegMotionNotify)
	xi.evReg.Add(xproto.MappingNotify, xi.onEvRegMappingNotify)

	return xi, nil
}
func (xi *XInput) onEvRegMappingNotify(ev0 interface{}) {
	err := xi.km.ReadTable()
	if err != nil {
		log.Print(err)
	}
}
func (xi *XInput) onEvRegKeyPress(ev0 interface{}) {
	ev := ev0.(xproto.KeyPressEvent)
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	//k := NewKey(xi.km, ev.Detail, ev.State)
	//ev2 := &KeyPressEvent{p, k}
	//xi.evReg.RunCallbacks(KeyPressEventId, ev2)

	keycode := xproto.Keycode(ev.Detail)
	mods := Modifiers(ev.State)
	ru, code := xi.km.Lookup(keycode, mods)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev3 := &event.KeyDown{*p, code, m2, ru}
	ev4 := &InputEvent{*p, ev3}
	xi.evReg.RunCallbacks(InputEventId, ev4)
}
func (xi *XInput) onEvRegKeyRelease(ev0 interface{}) {
	ev := ev0.(xproto.KeyReleaseEvent)
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	//k := NewKey(xi.km, ev.Detail, ev.State)
	//ev2 := &KeyReleaseEvent{p, k}
	//xi.evReg.RunCallbacks(KeyReleaseEventId, ev2)

	keycode := xproto.Keycode(ev.Detail)
	mods := Modifiers(ev.State)
	ru, code := xi.km.Lookup(keycode, mods)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev3 := &event.KeyUp{*p, code, m2, ru}
	ev4 := &InputEvent{*p, ev3}
	xi.evReg.RunCallbacks(InputEventId, ev4)
}
func (xi *XInput) onEvRegButtonPress(ev0 interface{}) {
	ev := ev0.(xproto.ButtonPressEvent)
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	b := NewButton(xi.km, ev.Detail, ev.State)

	//ev2 := &ButtonPressEvent{p, b}
	//xi.evReg.RunCallbacks(ButtonPressEventId, ev2)

	b2 := translateButtonToEventButton(b.XButton)
	m2 := translateModifiersToEventKeyModifiers(b.Mods)
	ev3 := &event.MouseDown{*p, b2, m2}
	ev4 := &InputEvent{*p, ev3}
	xi.evReg.RunCallbacks(InputEventId, ev4)
}
func (xi *XInput) onEvRegButtonRelease(ev interface{}) {
	ev0 := ev.(xproto.ButtonReleaseEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	b := NewButton(xi.km, ev0.Detail, ev0.State)
	//ev2 := &ButtonReleaseEvent{p, b}
	//xi.evReg.RunCallbacks(ButtonReleaseEventId, ev2)

	b2 := translateButtonToEventButton(b.XButton)
	m2 := translateModifiersToEventKeyModifiers(b.Mods)
	ev3 := &event.MouseUp{*p, b2, m2}
	ev4 := &InputEvent{*p, ev3}
	xi.evReg.RunCallbacks(InputEventId, ev4)
}
func (xi *XInput) onEvRegMotionNotify(ev interface{}) {
	ev0 := ev.(xproto.MotionNotifyEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	m := Modifiers(ev0.State)
	//ev2 := &MotionNotifyEvent{p, m}
	//xi.evReg.RunCallbacks(MotionNotifyEventId, ev2)

	var b2 event.MouseButtons
	if m.HasButton(1) {
		b2 |= event.MouseButtons(event.ButtonLeft)
	}
	if m.HasButton(2) {
		b2 |= event.MouseButtons(event.ButtonMiddle)
	}
	if m.HasButton(3) {
		b2 |= event.MouseButtons(event.ButtonRight)
	}
	if m.HasButton(4) {
		b2 |= event.MouseButtons(event.ButtonWheelUp)
	}
	if m.HasButton(5) {
		b2 |= event.MouseButtons(event.ButtonWheelDown)
	}
	if m.HasButton(6) {
		b2 |= event.MouseButtons(event.ButtonWheelLeft)
	}
	if m.HasButton(7) {
		b2 |= event.MouseButtons(event.ButtonWheelRight)
	}
	if m.HasButton(8) {
		b2 |= event.MouseButtons(event.ButtonBackward)
	}
	if m.HasButton(9) {
		b2 |= event.MouseButtons(event.ButtonForward)
	}

	m2 := translateModifiersToEventKeyModifiers(m)
	ev3 := &event.MouseMove{*p, b2, m2}
	ev4 := &InputEvent{*p, ev3}
	xi.evReg.RunCallbacks(InputEventId, ev4)
}

func translateButtonToEventButton(b xproto.Button) event.MouseButton {
	var b2 event.MouseButton
	switch b {
	case 1:
		b2 = event.ButtonLeft
	case 2:
		b2 = event.ButtonMiddle
	case 3:
		b2 = event.ButtonRight
	case 4:
		b2 = event.ButtonWheelUp
	case 5:
		b2 = event.ButtonWheelDown
	case 6:
		b2 = event.ButtonWheelLeft
	case 7:
		b2 = event.ButtonWheelRight
	case 8:
		b2 = event.ButtonBackward
	case 9:
		b2 = event.ButtonForward
	}
	return b2
}

func translateModifiersToEventKeyModifiers(u Modifiers) event.KeyModifiers {
	var m event.KeyModifiers
	if u.HasShift() {
		m |= event.ModShift
	}
	if u.HasControl() {
		m |= event.ModControl
	}
	if u.HasMod1() {
		m |= event.ModAlt
	}
	return m
}

const (
	KeyPressEventId = evreg.XInputEventIdStart + iota
	KeyReleaseEventId
	ButtonPressEventId
	ButtonReleaseEventId
	MotionNotifyEventId
	DoubleClickEventId
	TripleClickEventId
	InputEventId
)

type InputEvent struct {
	Point image.Point
	Event interface{}
}
