package shmimage

import (
	"image"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/pkg/errors"
)

type ShmImageWrap struct {
	conn     *xgb.Conn
	drawable xproto.Drawable
	depth    byte
	segId    shm.Seg
	simg     *ShmImage
}

func NewShmImageWrap(conn *xgb.Conn, drawable xproto.Drawable, depth byte) (*ShmImageWrap, error) {
	if err := shm.Init(conn); err != nil {
		return nil, err
	}
	siw := &ShmImageWrap{conn: conn, drawable: drawable, depth: depth}
	// server segment id
	segId, err := shm.NewSegId(siw.conn)
	if err != nil {
		return nil, err
	}
	siw.segId = segId
	// initial image
	r := image.Rect(0, 0, 1, 1)
	if err := siw.NewImage(&r); err != nil {
		return nil, err
	}
	return siw, nil
}
func (siw *ShmImageWrap) Close() error {
	return siw.simg.Close()
}
func (siw *ShmImageWrap) NewImage(r *image.Rectangle) error {
	simg, err := NewShmImage(r)
	if err != nil {
		return err
	}
	old := siw.simg
	siw.simg = simg
	// clean old img
	if old != nil {
		// need to detach to attach a new img id later
		_ = shm.Detach(siw.conn, siw.segId)

		err := old.Close()
		if err != nil {
			return err
		}
	}
	// attach to segId
	readOnly := false
	shmId := uint32(siw.simg.shmId)
	cookie := shm.AttachChecked(siw.conn, siw.segId, shmId, readOnly)
	if err := cookie.Check(); err != nil {
		return errors.Wrap(err, "shmimagewrap.newimage.attach")
	}

	return nil
}
func (siw *ShmImageWrap) Image() *imageutil.BGRA {
	return siw.simg.img
}
func (siw *ShmImageWrap) PutImage(gctx xproto.Gcontext, r *image.Rectangle) error {
	img := siw.simg.img
	b := img.Bounds()
	//c := shm.PutImageChecked(
	_ = shm.PutImage(
		siw.conn,
		siw.drawable,
		gctx,
		uint16(b.Dx()), uint16(b.Dy()), // total width/height
		uint16(r.Min.X), uint16(r.Min.Y), uint16(r.Dx()), uint16(r.Dy()), // src x,y,w,h
		int16(r.Min.X), int16(r.Min.Y), // dst x,y
		siw.depth,
		xproto.ImageFormatZPixmap,
		1, // send shm.CompletionEvent when done
		siw.segId,
		0) // offset
	//return c.Check()
	return nil
}
