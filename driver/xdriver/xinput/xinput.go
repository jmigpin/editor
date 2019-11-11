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

//----------

func (xi *XInput) ReadMapTable() {
	err := xi.km.ReadTable()
	if err != nil {
		log.Print(err)
	}
}

//----------

func (xi *XInput) KeyPress(ev *xproto.KeyPressEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	keycode := xproto.Keycode(ev.Detail)
	mods := Modifiers(ev.State)
	ru, ks := xi.km.Lookup(keycode, mods)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev2 := &event.KeyDown{*p, ks, m2, ru}
	return &event.WindowInput{*p, ev2}
}
func (xi *XInput) KeyRelease(ev *xproto.KeyReleaseEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	keycode := xproto.Keycode(ev.Detail)
	mods := Modifiers(ev.State)
	ru, ks := xi.km.Lookup(keycode, mods)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev2 := &event.KeyUp{*p, ks, m2, ru}
	return &event.WindowInput{*p, ev2}
}

func (xi *XInput) ButtonPress(ev *xproto.ButtonPressEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	b := xproto.Button(ev.Detail)
	mods := Modifiers(ev.State)
	b2 := translateButtonToEventButton(b)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev2 := &event.MouseDown{*p, b2, m2}
	return &event.WindowInput{*p, ev2}
}
func (xi *XInput) ButtonRelease(ev *xproto.ButtonReleaseEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	b := xproto.Button(ev.Detail)
	mods := Modifiers(ev.State)
	b2 := translateButtonToEventButton(b)
	m2 := translateModifiersToEventKeyModifiers(mods)
	ev2 := &event.MouseUp{*p, b2, m2}
	return &event.WindowInput{*p, ev2}
}

func (xi *XInput) MotionNotify(ev *xproto.MotionNotifyEvent) *event.WindowInput {
	p := &image.Point{int(ev.EventX), int(ev.EventY)}
	m := Modifiers(ev.State)
	b2 := translateModifiersToEventMouseButtons(m)
	m2 := translateModifiersToEventKeyModifiers(m)
	ev2 := &event.MouseMove{*p, b2, m2}
	return &event.WindowInput{*p, ev2}
}

//----------

type Modifiers uint32 // key and button mask

func (m Modifiers) HasAny(v Modifiers) bool {
	return m&v > 0
}

func (m Modifiers) Remove(u Modifiers) Modifiers {
	return m &^ u
}

//----------

func translateButtonToEventButton(xb xproto.Button) event.MouseButton {
	var b event.MouseButton
	switch xb {
	case 1:
		b = event.ButtonLeft
	case 2:
		b = event.ButtonMiddle
	case 3:
		b = event.ButtonRight
	case 4:
		b = event.ButtonWheelUp
	case 5:
		b = event.ButtonWheelDown
	case 6:
		b = event.ButtonWheelLeft
	case 7:
		b = event.ButtonWheelRight
	case 8:
		b = event.ButtonBackward
	case 9:
		b = event.ButtonForward
	}
	return b
}

func translateModifiersToEventMouseButtons(m Modifiers) event.MouseButtons {
	var b event.MouseButtons
	if m.HasAny(xproto.KeyButMaskButton1) {
		b |= event.MouseButtons(event.ButtonLeft)
	}
	if m.HasAny(xproto.KeyButMaskButton2) {
		b |= event.MouseButtons(event.ButtonMiddle)
	}
	if m.HasAny(xproto.KeyButMaskButton3) {
		b |= event.MouseButtons(event.ButtonRight)
	}
	if m.HasAny(xproto.KeyButMaskButton4) {
		b |= event.MouseButtons(event.ButtonWheelUp)
	}
	if m.HasAny(xproto.KeyButMaskButton5) {
		b |= event.MouseButtons(event.ButtonWheelDown)
	}
	if m.HasAny(xproto.KeyButMaskButton5 << 1) {
		b |= event.MouseButtons(event.ButtonWheelLeft)
	}
	if m.HasAny(xproto.KeyButMaskButton5 << 2) {
		b |= event.MouseButtons(event.ButtonWheelRight)
	}
	if m.HasAny(xproto.KeyButMaskButton5 << 3) {
		b |= event.MouseButtons(event.ButtonBackward)
	}
	if m.HasAny(xproto.KeyButMaskButton5 << 4) {
		b |= event.MouseButtons(event.ButtonForward)
	}
	return b
}

func translateModifiersToEventKeyModifiers(u Modifiers) event.KeyModifiers {
	var m event.KeyModifiers
	if u.HasAny(xproto.KeyButMaskShift) {
		m |= event.ModShift
	}
	if u.HasAny(xproto.KeyButMaskControl) {
		m |= event.ModCtrl
	}
	if u.HasAny(xproto.KeyButMaskLock) {
		m |= event.ModLock
	}
	if u.HasAny(xproto.KeyButMaskMod1) {
		m |= event.Mod1
	}
	if u.HasAny(xproto.KeyButMaskMod2) {
		m |= event.Mod2
	}
	if u.HasAny(xproto.KeyButMaskMod3) {
		m |= event.Mod3
	}
	if u.HasAny(xproto.KeyButMaskMod4) {
		m |= event.Mod4
	}
	if u.HasAny(xproto.KeyButMaskMod5) {
		m |= event.Mod5
	}
	return m
}
