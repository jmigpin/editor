package xutil

import (
	"fmt"
	"sync"

	"image"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"
)

type Shm struct {
	conn     *xgb.Conn
	drawable xproto.Drawable
	depth    byte

	segId shm.Seg
	img   struct {
		sync.RWMutex
		img *ShmImage
	}
}

func NewShm(conn *xgb.Conn, drawable xproto.Drawable, depth byte) (*Shm, error) {
	if err := shm.Init(conn); err != nil {
		return nil, err
	}
	sm := &Shm{conn: conn, drawable: drawable, depth: depth}

	// server segment id
	segId, err := shm.NewSegId(sm.conn)
	if err != nil {
		return nil, err
	}
	sm.segId = segId

	// initial image to ensure it works
	r := image.Rect(0, 0, 1, 1)
	if err := sm.newImage(&r); err != nil {
		return nil, err
	}

	return sm, nil
}

//func (sm *Shm) Close() error {
//sm.img.Lock()
//defer sm.img.Unlock()
//if sm.img.img != nil {
//return sm.img.img.Close()
//}
//return nil
//}
func (sm *Shm) Resize(r *image.Rectangle) error {
	return sm.newImage(r)
}
func (sm *Shm) newImage(r *image.Rectangle) error {
	// shared memory image
	img, err := NewShmImage(r)
	if err != nil {
		return err
	}

	sm.img.Lock()
	defer sm.img.Unlock()

	// assign
	old := sm.img.img
	sm.img.img = img
	// close old img
	if old != nil {
		// need to detach to attach a new img id later
		_ = shm.Detach(sm.conn, sm.segId)

		err := old.Close()
		if err != nil {
			return err
		}
	}

	// attach to segId
	readOnly := false
	_ = shm.Attach(sm.conn, sm.segId, uint32(sm.img.img.id), readOnly)

	return nil
}
func (sm *Shm) Image() *ShmImage {
	sm.img.RLock()
	defer sm.img.RUnlock()
	return sm.img.img
}
func (sm *Shm) PutImage(gctx xproto.Gcontext, r *image.Rectangle) {
	sm.img.RLock()
	defer sm.img.RUnlock()

	img := sm.img.img
	if img == nil {
		return
	}

	b := img.Bounds()
	//_ = shm.PutImage(
	cookie := shm.PutImageChecked(
		sm.conn,
		sm.drawable,
		gctx,
		uint16(b.Dx()), uint16(b.Dy()), // total width/height
		uint16(r.Min.X), uint16(r.Min.Y), uint16(r.Dx()), uint16(r.Dy()), // src x,y,w,h
		int16(r.Min.X), int16(r.Min.Y), // dst x,y
		sm.depth,
		xproto.ImageFormatZPixmap,
		0, // send shm.CompletionEvent when done
		sm.segId,
		0) // offset

	// Checked waits for the function to complete
	// Prevents flickering because it doesn't map the image to the screen while a function might be changing it due to having returned without waiting.
	// The flickring is visible when dragging a column. The toolbar text starts flickering because the background is being drawn already for the next frame before starting to draw the text.
	// TODO: double buffering without waits?
	if err := cookie.Check(); err != nil {
		fmt.Println(err)
	}
}
