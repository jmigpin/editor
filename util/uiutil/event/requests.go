package event

import (
	"image"
	"image/draw"
)

type Request interface{}

//----------

type ReqClose struct{}
type ReqWindowSetName struct{ Name string }
type ReqImage struct{ ReplyImg draw.Image }
type ReqImagePut struct{ Rect image.Rectangle }
type ReqImageResize struct{ Rect image.Rectangle }
type ReqCursorSet struct{ Cursor Cursor }
type ReqPointerQuery struct{ ReplyP image.Point }
type ReqPointerWarp struct{ P image.Point }

type ReqClipboardDataGet struct {
	Index  ClipboardIndex
	ReplyS string
}
type ReqClipboardDataSet struct {
	Index ClipboardIndex
	Str   string
}

// TODO: possibly lower level requests like drawtriangle
