package xwindow

import (
	"fmt"
	"image"
	"image/draw"
	"sync"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/driver/xgbutil/shmimage"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

//----------

// Only image where everything is drawn.
type WindowImage interface {
	Image() draw.Image
	PutImage(*image.Rectangle) error
	Resize(*image.Rectangle) error
	Close() error
}

//----------

func NewWindowImage(win *Window) (WindowImage, error) {
	// image using shared memory (better performance)
	img, err := NewShmWImg(win)
	if err != nil {
		fmt.Printf("warning: unable to use shm: %v\n", err)
	} else {
		return img, nil
	}

	// default method via copy to pixmap
	return NewPixWImg(win)
}

//----------

type ShmWImg struct {
	si  *shmimage.ShmImageWrap
	win *Window
}

func NewShmWImg(win *Window) (*ShmWImg, error) {
	swi := &ShmWImg{win: win}
	si, err := shmimage.NewShmImageWrap(swi.win.Conn, xproto.Drawable(swi.win.Window), swi.win.Screen.RootDepth)
	if err != nil {
		return nil, err
	}
	swi.si = si
	return swi, nil
}

func (swi *ShmWImg) Image() draw.Image {
	return swi.si.Image()
}

func (swi *ShmWImg) PutImage(r *image.Rectangle) error {
	return swi.si.PutImage(swi.win.GCtx, r)
}

func (swi *ShmWImg) Resize(r *image.Rectangle) error {
	return swi.si.NewImage(r)
}

func (swi *ShmWImg) Close() error {
	return swi.si.Close()
}

//----------

type PixWImg struct {
	img    *imageutil.BGRA
	win    *Window
	pid    xproto.Pixmap
	events chan<- interface{}
}

func NewPixWImg(win *Window) (*PixWImg, error) {
	pwi := &PixWImg{win: win}

	r := image.Rect(0, 0, 1, 1) // initial image
	pwi.Resize(&r)

	return pwi, nil
}

func (pwi *PixWImg) Image() draw.Image {
	return pwi.img
}

func (pwi *PixWImg) PutImage(r *image.Rectangle) error {
	// X max data length = (2^16) * 4 = 262144, need to send it in chunks

	putImgReqSize := 28
	maxReqSize := (1 << 16) * 4
	maxSize := (maxReqSize - putImgReqSize) / 4
	if r.Dx() > maxSize {
		return fmt.Errorf("pixwimg: dy>max, %v>%v", r.Dx(), maxSize)
	}

	xsize := r.Dx()
	ysize := maxSize / xsize
	chunk := image.Point{xsize, ysize}

	getData := func(minY int) (int, int, int, int, []byte) {
		h := chunk.Y
		h2 := r.Max.Y - minY
		if h2 < h {
			h = h2
		}
		data := make([]uint8, chunk.X*h*4)
		for y := 0; y < h; y++ {
			i := y * chunk.X * 4
			j := pwi.img.PixOffset(r.Min.X, minY+y)
			copy(data[i:i+chunk.X*4], pwi.img.Pix[j:])
		}
		return r.Min.X, minY, chunk.X, h, data
	}

	send := func(x, y, w, h int, data []byte) error {
		//c := xproto.PutImageChecked(
		_ = xproto.PutImage(
			pwi.win.Conn,
			xproto.ImageFormatZPixmap,
			xproto.Drawable(pwi.win.Window),
			pwi.win.GCtx,
			uint16(w), uint16(h), // width/height
			int16(x), int16(y), // dst X/Y
			0, // left pad, must be 0 for ZPixmap format
			pwi.win.Screen.RootDepth,
			data)
		//return c.Check()
		return nil
	}

	wg := sync.WaitGroup{}
	for minY := r.Min.Y; minY < r.Max.Y; minY += chunk.Y {
		wg.Add(1)
		go func(minY int) {
			defer wg.Done()
			x, y, w, h, data := getData(minY)
			if err := send(x, y, w, h, data); err != nil {
				//return err
				fmt.Printf("error: %v", err)
			}
		}(minY)
	}
	wg.Wait()

	pwi.events <- &event.WindowPutImageDone{}

	return nil
}

func (pwi *PixWImg) Resize(r *image.Rectangle) error {
	pwi.img = imageutil.NewBGRA(r)

	init := pwi.pid == 0

	if init {
		pid, err := xproto.NewPixmapId(pwi.win.Conn)
		if err != nil {
			return err
		}
		pwi.pid = pid
	}

	// clear old pixmap
	if !init {
		c2 := xproto.FreePixmapChecked(pwi.win.Conn, pwi.pid)
		err := c2.Check()
		if err != nil {
			return err
		}
	}

	c := xproto.CreatePixmapChecked(
		pwi.win.Conn,
		pwi.win.Screen.RootDepth,
		pwi.pid,
		xproto.Drawable(pwi.win.Window),
		uint16(r.Dx()),
		uint16(r.Dy()))
	if err := c.Check(); err != nil {
		return err
	}

	return nil
}

func (pwi *PixWImg) Close() error {
	_ = xproto.FreePixmap(pwi.win.Conn, pwi.pid)
	pwi.img = &imageutil.BGRA{}
	return nil
}
