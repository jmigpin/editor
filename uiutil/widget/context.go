package widget

import (
	"image/draw"

	"golang.org/x/image/font"
)

type Context interface {
	Image() draw.Image
	FontFace1() font.Face
	SetCursor(Cursor)
}

type Cursor int

const (
	NoCursor Cursor = iota
	NSResizeCursor
	WEResizeCursor
	CloseCursor
	MoveCursor
	PointerCursor
)
