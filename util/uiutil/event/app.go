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
type AppResizeImage struct{ Rect image.Rectangle }
type AppSetCursor struct{ Cursor Cursor }
type AppQueryPointer struct{}
type AppWarpPointer struct{}
type AppGetPaste struct{}
type AppSetCopy struct{}

//----------

type PostEventer interface {
	PostEvent(ev interface{}) error
}

func PutImage(ep PostEventer, r image.Rectangle) error {
	app := NewApp(&AppPutImage{Rect: r})
	if err := ep.PostEvent(app); err != nil {
		return err
	}
	res := <-app.Reply
	switch t := res.(type) {
	case nil:
		return nil
	case error:
		return t
	default:
		panic(res)
	}
}

func QueryPointer(ep PostEventer) (image.Point, error) {
	app := NewApp(&AppQueryPointer{})
	if err := ep.PostEvent(app); err != nil {
		return image.ZP, err
	}
	res := <-app.Reply
	switch t := res.(type) {
	case error:
		return image.ZP, t
	case image.Point:
		return t, nil
	default:
		panic(res)
	}
}

func SetCursor(ep PostEventer, c Cursor) error {
	app := NewApp(&AppSetCursor{c})
	if err := ep.PostEvent(app); err != nil {
		return err
	}
	res := <-app.Reply
	switch t := res.(type) {
	case nil:
		return nil
	case error:
		return t
	default:
		panic(res)
	}
}

func Close(ep PostEventer) error {
	app := NewApp(&AppClose{})
	if err := ep.PostEvent(app); err != nil {
		return err
	}
	res := <-app.Reply
	switch res.(type) {
	case nil:
		return nil
	default:
		panic(res)
	}
}
