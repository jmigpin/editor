package wimage

import (
	"fmt"
	"image"
	"image/draw"
	"sync"

	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/imageutil"
)

type PixmapWImage struct {
	opt        *Options
	pixId      xproto.Pixmap
	pixCreated bool
	img        *imageutil.BGRA
}

func NewPixmapWImage(opt *Options) (*PixmapWImage, error) {
	wi := &PixmapWImage{opt: opt}

	// pixmap id
	pixId, err := xproto.NewPixmapId(opt.Conn)
	if err != nil {
		return nil, err
	}
	wi.pixId = pixId

	// initial image
	r := image.Rect(0, 0, 1, 1)
	if err := wi.Resize(r); err != nil {
		return nil, err
	}

	return wi, nil
}

func (wi *PixmapWImage) Close() error {
	wi.img = &imageutil.BGRA{}
	if wi.pixCreated {
		return xproto.FreePixmapChecked(wi.opt.Conn, wi.pixId).Check()
	}
	return nil
}

//----------

func (wi *PixmapWImage) Resize(r image.Rectangle) error {
	// clear old pixmap
	if wi.pixCreated {
		err := xproto.FreePixmapChecked(wi.opt.Conn, wi.pixId).Check()
		if err != nil {
			return err
		}
		wi.pixCreated = false
	}

	// create new pixmap
	err := xproto.CreatePixmapChecked(
		wi.opt.Conn,
		wi.opt.ScreenInfo.RootDepth,
		wi.pixId,
		xproto.Drawable(wi.opt.Window),
		uint16(r.Dx()),
		uint16(r.Dy())).Check()
	if err != nil {
		return err
	}
	wi.pixCreated = true

	// new image
	wi.img = imageutil.NewBGRA(&r)
	return nil
}

//----------

func (wi *PixmapWImage) Image() draw.Image {
	return wi.img
}

func (wi *PixmapWImage) PutImage(r image.Rectangle) error {
	// X max data length = (2^16) * 4 = 262144, need to send it in chunks

	putImgReqSize := 28 // TODO: link to header size
	maxReqSize := (1 << 16) * 4
	maxSize := (maxReqSize - putImgReqSize) / 4
	if r.Dx() > maxSize {
		return fmt.Errorf("pixmapwimage: dx>max, %v>%v", r.Dx(), maxSize)
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
			j := wi.img.PixOffset(r.Min.X, minY+y)
			copy(data[i:i+chunk.X*4], wi.img.Pix[j:])
		}
		return r.Min.X, minY, chunk.X, h, data
	}

	send := func(x, y, w, h int, data []byte) error {
		_ = xproto.PutImage( // unchecked (performance; errors handled in ev loop)
			wi.opt.Conn,
			xproto.ImageFormatZPixmap,
			xproto.Drawable(wi.opt.Window),
			wi.opt.GCtx,
			uint16(w), uint16(h), // width/height
			int16(x), int16(y), // dst X/Y
			0, // left pad, must be 0 for ZPixmap format
			wi.opt.ScreenInfo.RootDepth,
			data)
		return nil
	}

	wg := sync.WaitGroup{}
	for minY := r.Min.Y; minY < r.Max.Y; minY += chunk.Y {
		wg.Add(1)
		go func(minY int) {
			defer wg.Done()
			x, y, w, h, data := getData(minY)
			if err := send(x, y, w, h, data); err != nil {
				fmt.Printf("pixmapwimage: putimage: %v", err)
			}
		}(minY)
	}
	wg.Wait()

	return nil
}

func (wi *PixmapWImage) PutImageCompleted() {
	panic("pixmapwimage: not expecting async put image completed")
}
