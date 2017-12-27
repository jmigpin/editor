package event

import (
	"image"
)

type MouseEnter struct{}
type MouseLeave struct{}

type MouseDown struct {
	Point     image.Point
	Button    MouseButton
	Modifiers KeyModifiers
}
type MouseUp struct {
	Point     image.Point
	Button    MouseButton
	Modifiers KeyModifiers
}
type MouseMove struct {
	Point     image.Point
	Buttons   MouseButtons
	Modifiers KeyModifiers
}

type MouseDragStart struct {
	Point     image.Point
	Button    MouseButton
	Modifiers KeyModifiers
}
type MouseDragEnd struct {
	Point     image.Point
	Button    MouseButton
	Modifiers KeyModifiers
}
type MouseDragMove struct {
	Point     image.Point
	Buttons   MouseButtons
	Modifiers KeyModifiers
}

type MouseClick struct {
	Point     image.Point
	Button    MouseButton
	Modifiers KeyModifiers
}
type MouseDoubleClick struct {
	Point     image.Point
	Button    MouseButton
	Modifiers KeyModifiers
}
type MouseTripleClick struct {
	Point     image.Point
	Button    MouseButton
	Modifiers KeyModifiers
}

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

type KeyModifiers int32

func (km KeyModifiers) HasAny(m KeyModifiers) bool {
	return int32(km)&int32(m) > 0
}
func (km KeyModifiers) Is(m KeyModifiers) bool {
	return int32(km) == int32(m)
}

const (
	ModNone  KeyModifiers = iota
	ModShift KeyModifiers = 1 << (iota - 1)
	ModControl
	ModAlt
	ModMeta
)

type KeyDown struct {
	Point     image.Point
	Code      KeyCode
	Modifiers KeyModifiers
	Rune      rune
}

type KeyUp struct {
	Point     image.Point
	Code      KeyCode
	Modifiers KeyModifiers
	Rune      rune
}

type KeyCode int

// Other codes not defined here will be their first column key symbol (like 'a','b',..., but not 'A'), going from 0 upwards.
const (
	KCodeNone KeyCode = iota - 300
	KCodeBackspace
	KCodeReturn
	KCodeEscape
	KCodeHome
	KCodeLeft
	KCodeUp
	KCodeRight
	KCodeDown
	KCodePageUp
	KCodePageDown
	KCodeEnd
	KCodeInsert
	KCodeF1
	KCodeF2
	KCodeF3
	KCodeF4
	KCodeF5
	KCodeF6
	KCodeF7
	KCodeF8
	KCodeF9
	KCodeF10
	KCodeF11
	KCodeF12
	KCodeShiftL
	KCodeShiftR
	KCodeControlL
	KCodeControlR
	KCodeAltL
	KCodeAltR
	KCodeAltGr
	KCodeSuperL // windows key
	KCodeSuperR
	KCodeDelete
	KCodeTab

	KCodeNumLock
	KCodeCapsLock

	KCodeVolumeUp
	KCodeVolumeDown
	KCodeMute
)

type WindowClose struct{}
type WindowExpose struct{}
type WindowPutImageDone struct{}
type WindowInput struct {
	Point image.Point
	Event interface{}
}

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
	DenyDndA DndAction = iota
	CopyDndA
	MoveDndA
	LinkDndA
	AskDndA
	PrivateDndA
)

type DndType int

const (
	TextURLListDndT DndType = iota
)

// copy/paste
type CopyPasteIndex int

const (
	PrimaryCPI CopyPasteIndex = iota
	ClipboardCPI
)
