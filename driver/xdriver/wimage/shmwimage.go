package wimage

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"log"
	"reflect"
	"time"
	"unsafe"

	"github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/syncutil"
)

type ShmWImage struct {
	opt          *Options
	segId        shm.Seg
	imgWrap      *ShmImgWrap
	closed       bool
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

	// initial image
	r := image.Rect(0, 0, 1, 1)
	if err := wi.Resize(r); err != nil {
		return nil, err
	}

	return wi, nil
}
func (wi *ShmWImage) Close() error {
	wi.closed = true
	imgWrap := wi.imgWrap
	wi.imgWrap = nil
	return wi.closeImage(wi.segId, imgWrap)
}

func (wi *ShmWImage) Resize(r image.Rectangle) error {
	if wi.closed {
		return fmt.Errorf("shmwimage: resize after close")
	}
	imgWrap, err := NewShmImgWrap(r)
	if err != nil {
		return err
	}
	segId, err := wi.resize2(imgWrap)
	if err != nil {
		return errors.Join(err, imgWrap.Close())
	}
	// Keep the old image valid until the server accepts the new segment.
	oldSegId, oldImgWrap := wi.segId, wi.imgWrap
	wi.segId, wi.imgWrap = segId, imgWrap
	return wi.closeImage(oldSegId, oldImgWrap)
}

func (wi *ShmWImage) resize2(imgWrap *ShmImgWrap) (shm.Seg, error) {
	segId, err := shm.NewSegId(wi.opt.Conn)
	if err != nil {
		return 0, err
	}
	if err := shm.AttachChecked(wi.opt.Conn, segId, uint32(imgWrap.shmId), false).Check(); err != nil {
		return 0, fmt.Errorf("shmwimage.resize.attach: %w", err)
	}
	return segId, nil
}

func (wi *ShmWImage) closeImage(segId shm.Seg, imgWrap *ShmImgWrap) error {
	if imgWrap == nil {
		return nil
	}
	err := shm.DetachChecked(wi.opt.Conn, segId).Check()
	if err != nil {
		err = fmt.Errorf("shm.detach: %w", err)
	}
	return errors.Join(err, imgWrap.Close())
}

func (wi *ShmWImage) Image() draw.Image {
	if wi.imgWrap == nil {
		return nil
	}
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
	if wi.closed || wi.imgWrap == nil {
		return fmt.Errorf("shmwimage: draw without an active buffer")
	}
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

type ShmImgWrap struct {
	Img   *imageutil.BGRA
	shmId uintptr
	addr  uintptr
}

func NewShmImgWrap(r image.Rectangle) (*ShmImgWrap, error) {
	size := imageutil.BGRASize(&r)
	shmId, addr, err := ShmOpen(size)
	if err != nil {
		return nil, err
	}

	// mask shared mem into a slice
	h := reflect.SliceHeader{Data: addr, Len: size, Cap: size}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	img := imageutil.NewBGRAFromBuffer(buf, &r)
	imgWrap := &ShmImgWrap{Img: img, shmId: shmId, addr: addr}
	return imgWrap, nil
}

func (imgWrap *ShmImgWrap) Close() error {
	if imgWrap.Img == nil {
		return nil
	}
	imgWrap.Img = nil
	id, addr := imgWrap.shmId, imgWrap.addr
	imgWrap.shmId, imgWrap.addr = 0, 0
	return ShmClose(id, addr)
}
