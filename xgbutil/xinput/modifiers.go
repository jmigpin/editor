package xinput

import (
	"strings"

	"github.com/BurntSushi/xgb/xproto"
)

// Mod1: Alt
// Mod2: Num lock
// Mod3:
// Mod4: Windows key
// Mod5: AltGr
// button1: left
// button2: middle
// button3: right
// button4: wheel up
// button5: wheel down

type Modifiers uint16 // key and button mask

func (m Modifiers) String() string {
	//return strconv.FormatInt(int64(m), 2)

	var u []string
	if m.Has(xproto.KeyButMaskShift) {
		u = append(u, "shift")
	}
	if m.Has(xproto.KeyButMaskLock) {
		u = append(u, "lock")
	}
	if m.Has(xproto.KeyButMaskControl) {
		u = append(u, "ctrl")
	}
	if m.Has(xproto.KeyButMaskMod1) {
		u = append(u, "mod1")
	}
	if m.Has(xproto.KeyButMaskMod2) {
		u = append(u, "mod2")
	}
	if m.Has(xproto.KeyButMaskMod3) {
		u = append(u, "mod3")
	}
	if m.Has(xproto.KeyButMaskMod4) {
		u = append(u, "mod4")
	}
	if m.Has(xproto.KeyButMaskMod5) {
		u = append(u, "mod5")
	}
	if m.Has(xproto.KeyButMaskButton1) {
		u = append(u, "button1")
	}
	if m.Has(xproto.KeyButMaskButton2) {
		u = append(u, "button2")
	}
	if m.Has(xproto.KeyButMaskButton3) {
		u = append(u, "button3")
	}
	if m.Has(xproto.KeyButMaskButton4) {
		u = append(u, "button4")
	}
	if m.Has(xproto.KeyButMaskButton5) {
		u = append(u, "button5")
	}

	return strings.Join(u, "|")
}

// Returns true if it contains the modifiers flags.
func (m Modifiers) Has(v int) bool {
	return uint(m)&uint(v) > 0
}

// Returns true if it is equal to the modifiers flags (no other flags are set).
func (m Modifiers) Is(v int) bool {
	// ignore locks state
	m = m.clearLock()
	m = m.clearMod2()

	return uint(m) == uint(v)
}

func (m Modifiers) clearLock() Modifiers {
	return m &^ xproto.KeyButMaskLock // caps lock
}
func (m Modifiers) clearMod2() Modifiers {
	return m &^ xproto.KeyButMaskMod2 // num lock
}

func (m Modifiers) ClearButtons() Modifiers {
	m = m &^ xproto.KeyButMaskButton1
	m = m &^ xproto.KeyButMaskButton2
	m = m &^ xproto.KeyButMaskButton3
	m = m &^ xproto.KeyButMaskButton4
	m = m &^ xproto.KeyButMaskButton5
	return m
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
	case 6:
		return xproto.KeyButMaskButton5 << 1
	case 7:
		return xproto.KeyButMaskButton5 << 2
	case 8:
		return xproto.KeyButMaskButton5 << 3
	case 9:
		return xproto.KeyButMaskButton5 << 4
	default:
		panic("button index out of range")
	}
}
