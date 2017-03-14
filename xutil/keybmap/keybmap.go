package keybmap

import (
	"fmt"
	"image"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

// $ man keymaps
// https://tronche.com/gui/x/xlib/input/XGetKeyboardMapping.html

// TODO: update map in case of key map change (mapping notify)

type KeybMap struct {
	conn      *xgb.Conn
	setupInfo *xproto.SetupInfo
	evReg     *xgbutil.EventRegister
	keybMap   *xproto.GetKeyboardMappingReply
	modMap    *xproto.GetModifierMappingReply
}

func NewKeybMap(conn *xgb.Conn) (*KeybMap, error) {
	si := xproto.Setup(conn)
	km := &KeybMap{conn: conn, setupInfo: si}

	if err := km.getMappings(); err != nil {
		return nil, err
	}

	return km, nil
}
func (km *KeybMap) getMappings() error {
	// keyboard mapping
	count := byte(km.setupInfo.MaxKeycode - km.setupInfo.MinKeycode + 1)
	cookie := xproto.GetKeyboardMapping(km.conn, km.setupInfo.MinKeycode, count)
	reply, err := cookie.Reply()
	if err != nil {
		return err
	}
	km.keybMap = reply
	//fmt.Printf("%+#v\n",reply)

	//// debug
	//min:=km.setupInfo.MinKeycode
	//max:=km.setupInfo.MaxKeycode
	//for i:=min;i<max;i++{
	//km.debug(i,0)
	//}

	//// debug
	//fmt.Printf("min: %d\n",km.setupInfo.MinKeycode)
	//for i,ks:=range km.keybMap.Keysyms{
	//fmt.Printf("%d: ('%c',%x)\n",i, ks,ks)
	//}

	//fmt.Printf("min %v, width %v\n",km.setupInfo.MinKeycode, km.keybMap.KeysymsPerKeycode)

	// modifier mapping
	cookie2 := xproto.GetModifierMapping(km.conn)
	reply2, err := cookie2.Reply()
	if err != nil {
		return err
	}
	km.modMap = reply2 // TODO: not being used
	//fmt.Printf("%+#v\n",reply2)

	return nil
}
func (km *KeybMap) debug(keycode xproto.Keycode, mods Modifiers) {
	fmt.Printf("*kb: m=%x, kc=%x, syms:", mods, keycode)
	w := int(km.keybMap.KeysymsPerKeycode) // ~7
	for j := 0; j < w; j++ {
		u := km.KeysymColumn(keycode, j)
		fmt.Printf("('%c',%x)", u, u)
	}
	fmt.Printf("\n")
}

func (km *KeybMap) KeysymColumn(keycode xproto.Keycode, column int) xproto.Keysym {
	kc := int(keycode - km.setupInfo.MinKeycode)
	w := int(km.keybMap.KeysymsPerKeycode)
	i := kc*w + column
	return km.keybMap.Keysyms[i]
}
func (km *KeybMap) ModKeysym(keycode xproto.Keycode, mods Modifiers) xproto.Keysym {
	col := km.modifiersColumn(mods)
	return km.KeysymColumn(keycode, col)
}
func (km *KeybMap) modifiersColumn(mods Modifiers) int {
	//alt := mods.Mod1()
	altGr := mods.Mod5()

	caps := mods.CapsLock()
	shift := mods.Shift()
	shift = (shift && !caps) || (!shift && caps)

	ctrl := mods.Control()

	// TODO: rules
	// https://tronche.com/gui/x/xlib/input/keyboard-encoding.html

	// missing: 3,6
	i := 0
	if altGr {
		i = 4
		if shift {
			i = 5
		}
	} else if shift {
		i = 1
	} else if ctrl {
		i = 2
	}

	if i >= int(km.keybMap.KeysymsPerKeycode) {
		panic("i>=keysymsperkeycode")
	}

	return i
}
func (km *KeybMap) NewKey(keycode xproto.Keycode, state uint16) *Key {
	return newKey(km, keycode, state)
}
func (km *KeybMap) NewButton(button xproto.Button, state uint16) *Button {
	return newButton(km, button, state)
}
func (km *KeybMap) NewModifiers(state uint16) Modifiers {
	// TODO: use modmap just like the keymap is being used
	return Modifiers(state)
}

// event register support

func (km *KeybMap) SetupEventRegister(evReg *xgbutil.EventRegister) {
	km.evReg = evReg
	fn := &xgbutil.ERCallback{km.onEvRegKeyPress}
	km.evReg.Add(xproto.KeyPress, fn)
	fn = &xgbutil.ERCallback{km.onEvRegKeyRelease}
	km.evReg.Add(xproto.KeyRelease, fn)
	fn = &xgbutil.ERCallback{km.onEvRegButtonPress}
	km.evReg.Add(xproto.ButtonPress, fn)
	fn = &xgbutil.ERCallback{km.onEvRegButtonRelease}
	km.evReg.Add(xproto.ButtonRelease, fn)
	fn = &xgbutil.ERCallback{km.onEvRegMotionNotify}
	km.evReg.Add(xproto.MotionNotify, fn)
}
func (km *KeybMap) onEvRegKeyPress(ev xgbutil.EREvent) {
	ev0 := ev.(xproto.KeyPressEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	k := newKey(km, ev0.Detail, ev0.State)
	ev2 := &KeyPressEvent{p, k}
	km.evReg.Emit(KeyPressEventId, ev2)
}
func (km *KeybMap) onEvRegKeyRelease(ev xgbutil.EREvent) {
	ev0 := ev.(xproto.KeyReleaseEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	k := newKey(km, ev0.Detail, ev0.State)
	ev2 := &KeyReleaseEvent{p, k}
	km.evReg.Emit(KeyReleaseEventId, ev2)
}
func (km *KeybMap) onEvRegButtonPress(ev xgbutil.EREvent) {
	ev0 := ev.(xproto.ButtonPressEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	b := newButton(km, ev0.Detail, ev0.State)
	ev2 := &ButtonPressEvent{p, b}
	km.evReg.Emit(ButtonPressEventId, ev2)
}
func (km *KeybMap) onEvRegButtonRelease(ev xgbutil.EREvent) {
	ev0 := ev.(xproto.ButtonReleaseEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	b := newButton(km, ev0.Detail, ev0.State)
	ev2 := &ButtonReleaseEvent{p, b}
	km.evReg.Emit(ButtonReleaseEventId, ev2)
}
func (km *KeybMap) onEvRegMotionNotify(ev xgbutil.EREvent) {
	ev0 := ev.(xproto.MotionNotifyEvent)
	p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
	m := Modifiers(ev0.State)
	ev2 := &MotionNotifyEvent{p, m}
	km.evReg.Emit(MotionNotifyEventId, ev2)
}

const (
	KeyPressEventId = iota + 1100
	KeyReleaseEventId
	ButtonPressEventId
	ButtonReleaseEventId
	MotionNotifyEventId
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
	Point     *image.Point
	Modifiers Modifiers
}
