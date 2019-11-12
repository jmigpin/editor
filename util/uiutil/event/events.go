package event

import (
	"image"
	"unicode"
)

//----------

type WindowClose struct{}
type WindowExpose struct{ Rect image.Rectangle }
type WindowResize struct{ Rect image.Rectangle }

type WindowPutImageDone struct{}

type WindowInput struct {
	Point image.Point
	Event interface{}
}

//----------

type Handle bool

const (
	NotHandled Handle = false
	Handled           = true
)

//----------

type MouseEnter struct{}
type MouseLeave struct{}

type MouseDown struct {
	Point  image.Point
	Button MouseButton
	Mods   KeyModifiers
}
type MouseUp struct {
	Point  image.Point
	Button MouseButton
	Mods   KeyModifiers
}
type MouseMove struct {
	Point   image.Point
	Buttons MouseButtons
	Mods    KeyModifiers
}

type MouseDragStart struct {
	Point  image.Point
	Button MouseButton
	Mods   KeyModifiers
}
type MouseDragEnd struct {
	Point  image.Point
	Button MouseButton
	Mods   KeyModifiers
}
type MouseDragMove struct {
	Point   image.Point
	Buttons MouseButtons
	Mods    KeyModifiers
}

type MouseClick struct {
	Point  image.Point
	Button MouseButton
	Mods   KeyModifiers
}
type MouseDoubleClick struct {
	Point  image.Point
	Button MouseButton
	Mods   KeyModifiers
}
type MouseTripleClick struct {
	Point  image.Point
	Button MouseButton
	Mods   KeyModifiers
}

//----------

type MouseButton int32

const (
	ButtonNone MouseButton = iota
	ButtonLeft MouseButton = 1 << (iota - 1)
	ButtonMiddle
	ButtonRight
	ButtonWheelUp
	ButtonWheelDown
	ButtonWheelLeft
	ButtonWheelRight
	ButtonBackward
	ButtonForward
)

type MouseButtons int32

func (mb MouseButtons) Has(b MouseButton) bool {
	return int32(mb)&int32(b) > 0
}
func (mb MouseButtons) HasAny(bs MouseButtons) bool {
	return int32(mb)&int32(bs) > 0
}
func (mb MouseButtons) Is(b MouseButton) bool {
	return int32(mb) == int32(b)
}

//----------

type KeyDown struct {
	Point  image.Point
	KeySym KeySym
	Mods   KeyModifiers
	Rune   rune
}

func (kd *KeyDown) LowerRune() rune {
	return unicode.ToLower(kd.Rune)
}

type KeyUp struct {
	Point  image.Point
	KeySym KeySym
	Mods   KeyModifiers
	Rune   rune
}

func (ku *KeyUp) LowerRune() rune {
	return unicode.ToLower(ku.Rune)
}

//----------

type KeyModifiers uint32

func (km KeyModifiers) HasAny(m KeyModifiers) bool {
	return km&m > 0
}
func (km KeyModifiers) Is(m KeyModifiers) bool {
	return km == m
}
func (km KeyModifiers) ClearLocks() KeyModifiers {
	w := []KeyModifiers{ModLock, ModNum}
	u := km
	for _, m := range w {
		u &^= m
	}
	return u
}

const (
	ModNone  KeyModifiers = 0
	ModShift KeyModifiers = 1 << (iota - 1)
	ModLock               // caps
	ModCtrl
	Mod1 // ~ alt
	Mod2 // ~ num lock
	Mod3
	Mod4 // ~ windows key
	Mod5 // ~ alt gr
)

const (
	ModAlt   = Mod1
	ModNum   = Mod2
	ModAltGr = Mod5
)

//----------

type KeySym int

const (
	KSymNone KeySym = 0

	// let ascii codes keep their values
	KSym_dummy_ KeySym = 256 + iota

	KSymSpace

	KSymBackspace
	KSymReturn
	KSymEscape
	KSymHome
	KSymLeft
	KSymUp
	KSymRight
	KSymDown
	KSymPageUp
	KSymPageDown
	KSymEnd
	KSymInsert
	KSymF1
	KSymF2
	KSymF3
	KSymF4
	KSymF5
	KSymF6
	KSymF7
	KSymF8
	KSymF9
	KSymF10
	KSymF11
	KSymF12

	KSymShiftL
	KSymShiftR
	KSymControlL
	KSymControlR
	KSymAltL
	KSymAltR
	KSymAltGr
	KSymSuperL // windows key
	KSymSuperR
	KSymDelete
	KSymTab
	KSymTabLeft

	KSymNumLock
	KSymCapsLock
	KSymShiftLock

	KSymGrave
	KSymAcute
	KSymCircumflex
	KSymTilde
	KSymMacron
	KSymBreve
	KSymDiaresis
	KSymRingAbove
	KSymCaron
	KSymCedilla

	KSymKeypadMultiply
	KSymKeypadAdd
	KSymKeypadSubtract
	KSymKeypadDecimal
	KSymKeypadDivide

	KSymKeypad0
	KSymKeypad1
	KSymKeypad2
	KSymKeypad3
	KSymKeypad4
	KSymKeypad5
	KSymKeypad6
	KSymKeypad7
	KSymKeypad8
	KSymKeypad9

	KSymVolumeUp
	KSymVolumeDown
	KSymMute

	KSymMultiKey
	KSymMenu
)

//----------

// drag and drop
type DndPosition struct {
	Point image.Point
	Types []DndType
	Reply func(DndAction)
}
type DndDrop struct {
	Point       image.Point
	ReplyAccept func(bool)
	RequestData func(DndType) ([]byte, error)
}

type DndAction int

const (
	DndADeny DndAction = iota
	DndACopy
	DndAMove
	DndALink
	DndAAsk
	DndAPrivate
)

type DndType int

const (
	TextURLListDndT DndType = iota
)

//----------

type CopyPasteIndex int

const (
	CPIPrimary CopyPasteIndex = iota
	CPIClipboard
)
