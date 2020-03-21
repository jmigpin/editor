package fontutil

//func NewFace(font *truetype.Font, opt *truetype.Options) font.Face {
//	face := truetype.NewFace(font, opt)
//	f2 := NewFaceRunes(face)
//	//f3 := NewFaceCache(f2)
//	f3 := NewFaceCacheL(f2)
//	//f3 := NewFaceCacheL2(f2)
//	return f3
//}

////----------

//func GetTestFace() font.Face {
//	ttf := goregular.TTF
//	f, err := truetype.Parse(ttf)
//	if err != nil {
//		panic(err)
//	}
//	ttOpt := &truetype.Options{} // defaults: size=12, dpi=72, ~14px
//	return truetype.NewFace(f, ttOpt)
//}
