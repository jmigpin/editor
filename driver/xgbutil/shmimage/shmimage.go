package shmimage

import (
	"image"
	"reflect"
	"unsafe"

	"github.com/jmigpin/editor/util/imageutil"
)

type ShmImage struct {
	img   *imageutil.BGRA
	shmId uintptr
	addr  uintptr
}

func NewShmImage(r *image.Rectangle) (*ShmImage, error) {
	size := imageutil.BGRASize(r)
	shmId, addr, err := ShmOpen(size)
	if err != nil {
		return nil, err
	}

	// mask shared mem into a slice
	h := reflect.SliceHeader{Data: addr, Len: size, Cap: size}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	img := imageutil.NewBGRAFromBuffer(buf, r)
	simg := &ShmImage{img: img, shmId: shmId, addr: addr}
	return simg, nil
}
func (img *ShmImage) Close() error {
	return ShmClose(img.shmId, img.addr)
}
