package wimage

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/syncutil"
)

type ShmWImage struct {
	opt          *Options
	segId        shm.Seg
	imgWrap      *ShmImgWrap
	putCompleted *syncutil.WaitForSet
}

func NewShmWImage(opt *Options) (*ShmWImage, error) {
	//// init shared memory extension
	//if err := shm.Init(opt.Conn); err != nil {
	//	return nil, err
	//}
	// get error from early init
	if initErr != nil {
		return nil, initErr
	}

	wi := &ShmWImage{opt: opt}
	wi.putCompleted = syncutil.NewWaitForSet()

	// server segment id
	segId, err := shm.NewSegId(wi.opt.Conn)
	if err != nil {
		return nil, err
	}
	wi.segId = segId

	// initial image
	r := image.Rect(0, 0, 1, 1)
	if err := wi.Resize(r); err != nil {
		return nil, err
	}

	return wi, nil
}
func (wi *ShmWImage) Close() error {
	return wi.imgWrap.Close()
}

func (wi *ShmWImage) Resize(r image.Rectangle) error {
	imgWrap, err := NewShmImgWrap(r)
	if err != nil {
		return err
	}
	old := wi.imgWrap
	wi.imgWrap = imgWrap
	// clean old img
	if old != nil {
		// need to detach to attach a new img id later
		_ = shm.Detach(wi.opt.Conn, wi.segId)

		err := old.Close()
		if err != nil {
			return err
		}
	}
	// attach to segId
	readOnly := false
	shmId := uint32(wi.imgWrap.shmId)
	cookie := shm.AttachChecked(wi.opt.Conn, wi.segId, shmId, readOnly)
	if err := cookie.Check(); err != nil {
		return fmt.Errorf("shmwimage.resize.attach: %w", err)
	}

	return nil
}

func (wi *ShmWImage) Image() draw.Image {
	return wi.imgWrap.Img
}

func (wi *ShmWImage) PutImage(r image.Rectangle) error {
	wi.putCompleted.Start(500 * time.Millisecond)
	if err := wi.putImage2(r); err != nil {
		wi.putCompleted.Cancel()
		return err
	}
	// wait for shm.CompletionEvent that should call PutImageCompleted()
	// Returns early if the server fails to send the msg (failsafe)
	_, err := wi.putCompleted.WaitForSet()
	if err != nil {
		err = fmt.Errorf("shm putCompleted: get, %w", err)
	}
	return err
}

func (wi *ShmWImage) putImage2(r image.Rectangle) error {
	gctx := wi.opt.GCtx
	img := wi.imgWrap.Img
	drawable := xproto.Drawable(wi.opt.Window)
	depth := wi.opt.ScreenInfo.RootDepth
	b := img.Bounds()
	c1 := shm.PutImageChecked(
		wi.opt.Conn,
		drawable,
		gctx,
		uint16(b.Dx()), uint16(b.Dy()), // total width/height
		uint16(r.Min.X), uint16(r.Min.Y), uint16(r.Dx()), uint16(r.Dy()), // src x,y,w,h
		int16(r.Min.X), int16(r.Min.Y), // dst x,y
		depth,
		xproto.ImageFormatZPixmap,
		1, // send shm.CompletionEvent when done
		wi.segId,
		0) // offset
	return c1.Check()
}

func (wi *ShmWImage) PutImageCompleted() {
	err := wi.putCompleted.Set(nil)
	if err != nil {
		err = fmt.Errorf("shm putCompleted: set, %w", err)
		log.Println(err)
	}
}

//----------

var initErr error

// initialize early to avoid concurrent map read/write (XGB library issue)
func Init(conn *xgb.Conn) {
	initErr = shm.Init(conn)
}
