package xgbutil

import (
	"fmt"
	"unsafe"

	"image"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/imageutil"
)

type ShmWrap struct {
	conn     *xgb.Conn
	drawable xproto.Drawable
	depth    byte
	segId    shm.Seg
	simg     *ShmWrapImage
}

type ShmWrapImage struct {
	img   *imageutil.BGRA
	shmId uintptr
	addr  unsafe.Pointer
}

func NewShmWrapImage(r *image.Rectangle) (*ShmWrapImage, error) {
	size := imageutil.BGRASize(r)
	shmId, addr, err := ShmOpen(size)
	if err != nil {
		return nil, err
	}
	img := imageutil.NewBGRAFromAddr(addr, r)
	simg := &ShmWrapImage{img: img, shmId: shmId, addr: addr}
	return simg, nil
}
func (img *ShmWrapImage) Close() error {
	return ShmClose(img.addr)
}

func NewShmWrap(conn *xgb.Conn, drawable xproto.Drawable, depth byte) (*ShmWrap, error) {
	if err := shm.Init(conn); err != nil {
		return nil, err
	}
	smw := &ShmWrap{conn: conn, drawable: drawable, depth: depth}
	// server segment id
	segId, err := shm.NewSegId(smw.conn)
	if err != nil {
		return nil, err
	}
	smw.segId = segId
	// initial image
	r := image.Rect(0, 0, 1, 1)
	if err := smw.NewImage(&r); err != nil {
		return nil, err
	}
	return smw, nil
}

func (smw *ShmWrap) Close() error {
	if smw.simg.img != nil {
		return smw.simg.Close()
	}
	return nil
}

func (smw *ShmWrap) NewImage(r *image.Rectangle) error {
	simg, err := NewShmWrapImage(r)
	if err != nil {
		return err
	}
	old := smw.simg
	smw.simg = simg
	// clean old img
	if old != nil {
		// need to detach to attach a new img id later
		_ = shm.Detach(smw.conn, smw.segId)

		err := old.Close()
		if err != nil {
			return err
		}
	}
	// attach to segId
	readOnly := false
	shmId := uint32(smw.simg.shmId)
	_ = shm.Attach(smw.conn, smw.segId, shmId, readOnly)

	return nil
}
func (smw *ShmWrap) Image() *imageutil.BGRA {
	return smw.simg.img
}
func (smw *ShmWrap) PutImage(gctx xproto.Gcontext, r *image.Rectangle) {
	img := smw.simg.img
	b := img.Bounds()
	//_ = shm.PutImage(
	cookie := shm.PutImageChecked(
		smw.conn,
		smw.drawable,
		gctx,
		uint16(b.Dx()), uint16(b.Dy()), // total width/height
		uint16(r.Min.X), uint16(r.Min.Y), uint16(r.Dx()), uint16(r.Dy()), // src x,y,w,h
		int16(r.Min.X), int16(r.Min.Y), // dst x,y
		smw.depth,
		xproto.ImageFormatZPixmap,
		0, // send shm.CompletionEvent when done
		smw.segId,
		0) // offset

	// Checked waits for the function to complete
	// Prevents flickering because it doesn't map the image to the screen while a function might be changing it due to having returned without waiting.
	// The flickering is visible when resizing a column. The toolbar text starts flickering because the background is being drawn already for the next frame before starting to draw the text.
	if err := cookie.Check(); err != nil {
		fmt.Println(err)
	}
}
