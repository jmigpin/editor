package drawutil

import "github.com/golang/freetype/truetype"

type Face struct {
	*FaceRunes
}

func NewFace(font *truetype.Font, opt *truetype.Options) *Face {
	face := truetype.NewFace(font, opt)
	faceCache := NewFaceCache(face)
	faceRunes := NewFaceRunes(faceCache)
	return &Face{faceRunes}
}
