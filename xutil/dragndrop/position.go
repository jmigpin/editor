package dragndrop

import (
	"image"

	"github.com/BurntSushi/xgb/xproto"
)

type PositionEvent struct {
	Window      xproto.Window // sender
	ScreenPoint image.Point
	Timestamp   xproto.Timestamp
	Action      xproto.Atom

	dnd        *Dnd
	enterEvent *EnterEvent
	Replied    bool // external use to ensure a reply is made
}

func ParsePositionEvent(buf []uint32, enterEvent *EnterEvent, dnd *Dnd) *PositionEvent {
	return &PositionEvent{
		Window: xproto.Window(buf[0]),
		//buf[1] // flags
		ScreenPoint: image.Point{
			int(buf[2] >> 16),
			int(buf[2] & 0xffff),
		},
		Timestamp:  xproto.Timestamp(buf[3]),
		Action:     xproto.Atom(buf[4]),
		dnd:        dnd,
		enterEvent: enterEvent,
	}
}
func (pos *PositionEvent) WindowPoint() (*image.Point, error) {
	geom, err := pos.dnd.getWindowGeometry()
	if err != nil {
		return nil, err
	}
	x := int(geom.X) + int(geom.BorderWidth)
	y := int(geom.Y) + int(geom.BorderWidth)
	p := pos.ScreenPoint.Sub(image.Point{x, y})
	return &p, nil
}
func (pos *PositionEvent) SupportsType(typ xproto.Atom) bool {
	return pos.enterEvent.SupportsType(typ)
}
func (pos *PositionEvent) ReplyDeny() {
	pos.Replied = true
	pos.dnd.sendStatus(pos.Window, pos.Action, false)
}
func (pos *PositionEvent) ReplyAccept(action xproto.Atom) {
	pos.Replied = true
	pos.dnd.sendStatus(pos.Window, action, true)
}
