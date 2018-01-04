package xinput

import (
	"image"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type XInput struct {
	km *KMap
}

func NewXInput(conn *xgb.Conn) (*XInput, error) {
	km, err := NewKMap(conn)
	if err != nil {
		return nil, err
	}
	xi := &XInput{km: km}
	return xi, nil
}

func (xi *XInput) ReadMapTable() {
	err := xi.km.ReadTable()
	if err != nil {
		log.Print(err)
	}
}

func (xi *XInput) KeyPress(ev *xproto.KeyPressEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	keycode := xproto.Keycode(ev.Detail)
	mods := Modifiers(ev.State)
	ru, code := xi.km.Lookup(keycode, mods)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev2 := &event.KeyDown{*p, code, m2, ru}
	return &event.WindowInput{*p, ev2}
}
func (xi *XInput) KeyRelease(ev *xproto.KeyReleaseEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	keycode := xproto.Keycode(ev.Detail)
	mods := Modifiers(ev.State)
	ru, code := xi.km.Lookup(keycode, mods)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev2 := &event.KeyUp{*p, code, m2, ru}
	return &event.WindowInput{*p, ev2}
}

func (xi *XInput) ButtonPress(ev *xproto.ButtonPressEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	b := NewButton(xi.km, ev.Detail, ev.State)
	b2 := translateButtonToEventButton(b.XButton)
	m2 := translateModifiersToEventKeyModifiers(b.Mods)
	ev2 := &event.MouseDown{*p, b2, m2}
	return &event.WindowInput{*p, ev2}
}
func (xi *XInput) ButtonRelease(ev *xproto.ButtonReleaseEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	b := NewButton(xi.km, ev.Detail, ev.State)
	b2 := translateButtonToEventButton(b.XButton)
	m2 := translateModifiersToEventKeyModifiers(b.Mods)
	ev2 := &event.MouseUp{*p, b2, m2}
	return &event.WindowInput{*p, ev2}
}
func (xi *XInput) MotionNotify(ev *xproto.MotionNotifyEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	m := Modifiers(ev.State)
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
	ev2 := &event.MouseMove{*p, b2, m2}
	return &event.WindowInput{*p, ev2}
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
