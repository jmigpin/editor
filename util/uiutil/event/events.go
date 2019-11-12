package event

import (
	"image"
	"unicode"
)

//----------

type WindowClose struct{}
type WindowResize struct{ Rect image.Rectangle }
type WindowExpose struct{ Rect image.Rectangle } // empty = full area
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
