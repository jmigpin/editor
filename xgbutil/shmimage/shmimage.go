package shmimage

import (
	"image"
	"unsafe"

	"github.com/jmigpin/editor/imageutil"
)

type ShmImage struct {
	img   *imageutil.BGRA
	shmId uintptr
	addr  unsafe.Pointer
}

func NewShmImage(r *image.Rectangle) (*ShmImage, error) {
	size := imageutil.BGRASize(r)
	shmId, addr, err := ShmOpen(size)
	if err != nil {
		return nil, err
	}

	// mask shared mem into a slice - gives go vet warning
	buf := (*[1 << 30]byte)(addr)[:size:size]

	img := imageutil.NewBGRAFromBuffer(buf, r)
	simg := &ShmImage{img: img, shmId: shmId, addr: addr}
	return simg, nil
}
func (img *ShmImage) Close() error {
	return ShmClose(img.shmId, img.addr)
}
