package keybmap

import "github.com/BurntSushi/xgb/xproto"

type Modifiers uint16 // key and button mask

// Returns if it contains the modifiers flags.
func (m Modifiers) On(v uint) bool {
	return uint(m)&v > 0
}

// Returns if it is equal to the modifiers flags (no other flags are set).
func (m Modifiers) Is(v int) bool {
	return uint(m) == uint(v)
}

func (m Modifiers) IsNone() bool {
	return m.Is(0)
}
func (m Modifiers) IsShift() bool {
	return m.Is(xproto.KeyButMaskShift)
}
func (m Modifiers) IsControl() bool {
	return m.Is(xproto.KeyButMaskControl)
}
func (m Modifiers) IsControlShift() bool {
	return m.Is(xproto.KeyButMaskControl | xproto.KeyButMaskShift)
}
func (m Modifiers) IsControlMod1() bool {
	return m.Is(xproto.KeyButMaskControl | xproto.KeyButMaskMod1)
}
func (m Modifiers) IsControlShiftMod1() bool {
	return m.Is(xproto.KeyButMaskControl | xproto.KeyButMaskShift | xproto.KeyButMaskMod1)
}

//func (m Modifiers) Shift() bool {
//return m.On(xproto.KeyButMaskShift)
//}
//func (m Modifiers) CapsLock() bool {
//return m.On(xproto.KeyButMaskLock)
//}
//func (m Modifiers) Control() bool {
//return m.On(xproto.KeyButMaskControl)
//}
//func (m Modifiers) Mod1() bool { // Alt
//return m.On(xproto.KeyButMaskMod1)
//}
//func (m Modifiers) Mod2() bool {
//return m.On(xproto.KeyButMaskMod2)
//}
//func (m Modifiers) Mod3() bool { // Num lock
//return m.On(xproto.KeyButMaskMod3)
//}
//func (m Modifiers) Mod4() bool { // Windows key
//return m.On(xproto.KeyButMaskMod4)
//}
//func (m Modifiers) Mod5() bool { // AltGr
//return m.On(xproto.KeyButMaskMod5)
//}

func (m Modifiers) Button1() bool { // left
	return m.On(xproto.KeyButMaskButton1)
}
func (m Modifiers) Button2() bool { // middle
	return m.On(xproto.KeyButMaskButton2)
}
func (m Modifiers) Button3() bool { // right
	return m.On(xproto.KeyButMaskButton3)
}
func (m Modifiers) Button4() bool { // wheel up
	return m.On(xproto.KeyButMaskButton4)
}
func (m Modifiers) Button5() bool { // wheel down
	return m.On(xproto.KeyButMaskButton5)
}
