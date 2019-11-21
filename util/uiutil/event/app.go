package event

import (
	"image"
)

// Events iniciated by the app to be sent to the driver

type App struct {
	Event interface{}
	Reply chan interface{}
}

func NewApp(ev interface{}) *App {
	return &App{Event: ev, Reply: make(chan interface{}, 1)}
}

type AppClose struct{}
type AppPutImage struct{ Rect image.Rectangle }
type AppPutImageReply struct{ Error error }
type AppResizeImage struct{ Rect image.Rectangle }
type AppSetCursor struct{ Cursor Cursor }
type AppQueryPointer struct{}
type AppQueryPointerReply struct {
	Point image.Point
	Error error
}
type AppWarpPointer struct{}
type AppGetPaste struct{}
type AppSetCopy struct{}

//----------

type EventPoster interface {
	PostEvent(ev interface{})
}

func PutImage(ep EventPoster, r image.Rectangle) error {
	app := NewApp(&AppPutImage{Rect: r})
	ep.PostEvent(app)
	reply := (<-app.Reply).(*AppPutImageReply)
	return reply.Error
}

func QueryPointer(ep EventPoster) (image.Point, error) {
	app := NewApp(&AppQueryPointer{})
	ep.PostEvent(app)
	reply := (<-app.Reply).(*AppQueryPointerReply)
	return reply.Point, reply.Error
}

func SetCursor(ep EventPoster, c Cursor) {
	app := NewApp(&AppSetCursor{c})
	ep.PostEvent(app)
	<-app.Reply
}
