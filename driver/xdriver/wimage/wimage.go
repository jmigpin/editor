package wimage

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Window image for drawing.
type WImage interface {
	Image() draw.Image
	PutImage(image.Rectangle) (completed bool, _ error)
	Resize(image.Rectangle) error
	Close() error
}

func NewWImage(opt *Options) (WImage, error) {
	// image using shared memory (better performance)
	wimg, err := NewShmWImage(opt)
	if err != nil {
		// output error, try next method
		fmt.Printf("warning: unable to use shmwimage: %v\n", err)
	} else {
		return wimg, nil
	}

	// default method via copy to pixmap
	return NewPixmapWImage(opt)
}

type Options struct {
	Conn       *xgb.Conn
	Window     xproto.Window
	ScreenInfo *xproto.ScreenInfo
	GCtx       xproto.Gcontext
}
