package widget

import (
	"image/draw"
)

type ImageContext interface {
	Image() draw.Image
}
type CursorContext interface {
	SetCursor(Cursor)
}
