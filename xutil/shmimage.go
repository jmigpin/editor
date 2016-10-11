package xutil

import (
	"image"
	"unsafe"
)

type ShmImage struct {
	BGRA
	addr unsafe.Pointer
	id   uintptr
}

func NewShmImage(r *image.Rectangle) (*ShmImage, error) {
	w, h := r.Dx(), r.Dy()
	size := w * h * 4
	shmId, addr, err := shmOpen(size)
	if err != nil {
		return nil, err
	}
	// mask the shared memory into a slice
	buf := (*[1 << 31]uint8)(addr)[:size:size]

	p := BGRA{image.RGBA{buf, 4 * w, *r}}
	img := &ShmImage{BGRA: p, addr: addr, id: shmId}

	return img, nil
}
func (img *ShmImage) Close() error {
	return shmClose(img.addr)
}
