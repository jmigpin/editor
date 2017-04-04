package xinput

import "github.com/BurntSushi/xgb/xproto"

type Modifiers uint16 // key and button mask

// Mod1: Alt
// Mod2:
// Mod3: Num lock
// Mod4: Windows key
// Mod5: AltGr
// button1: left
// button2: middle
// button3: right
// button4: wheel up
// button5: wheel down

// Returns true if it contains the modifiers flags.
func (m Modifiers) Has(v int) bool {
	return uint(m)&uint(v) > 0
}

// Returns true if it is equal to the modifiers flags (no other flags are set).
func (m Modifiers) Is(v int) bool {
	m2 := m.clearLock() // ignore lock state
	return uint(m2) == uint(v)
}

func (m Modifiers) clearLock() Modifiers {
	return m &^ xproto.KeyButMaskLock // caps lock
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
func (m Modifiers) IsMod1() bool {
	return m.Is(xproto.KeyButMaskMod1)
}
func (m Modifiers) IsShiftMod1() bool {
	return m.Is(xproto.KeyButMaskShift | xproto.KeyButMaskMod1)
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

func (m Modifiers) HasShift() bool {
	return m.Has(xproto.KeyButMaskShift)
}
func (m Modifiers) HasControl() bool {
	return m.Has(xproto.KeyButMaskControl)
}
func (m Modifiers) HasMod1() bool {
	return m.Has(xproto.KeyButMaskMod1)
}
func (m Modifiers) HasCapsLock() bool {
	return m.Has(xproto.KeyButMaskLock)
}

func (m Modifiers) IsButtonAnd(b int, flags int) bool {
	return m.Is(buttonMask(b) | flags)
}
func (m Modifiers) IsButton(b int) bool {
	return m.IsButtonAnd(b, 0)
}
func (m Modifiers) IsButtonAndShift(b int) bool {
	return m.IsButtonAnd(b, xproto.KeyButMaskShift)
}
func (m Modifiers) IsButtonAndControl(b int) bool {
	return m.IsButtonAnd(b, xproto.KeyButMaskControl)
}
func (m Modifiers) HasButton(b int) bool {
	return m.Has(buttonMask(b))
}
func buttonMask(b int) int {
	switch b {
	case 1:
		return xproto.KeyButMaskButton1
	case 2:
		return xproto.KeyButMaskButton2
	case 3:
		return xproto.KeyButMaskButton3
	case 4:
		return xproto.KeyButMaskButton4
	case 5:
		return xproto.KeyButMaskButton5
	default:
		panic("button index out of range")
	}
}
