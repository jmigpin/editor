package fontutil

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

func DefaultFont() *Font {
	f, err := FontsMan.Font(goregular.TTF)
	if err != nil {
		panic(err)
	}
	return f
}

func DefaultFontFace() *FontFace {
	f := DefaultFont()
	opt := opentype.FaceOptions{} // defaults: size=12, dpi=72, ~14px
	return f.FontFace(opt)
}

//----------

var FontsMan = NewFontsManager()

//----------

type FontsManager struct {
	fontsCache map[string]*Font
}

func NewFontsManager() *FontsManager {
	fm := &FontsManager{}
	fm.ClearFontsCache()
	return fm
}

func (fm *FontsManager) ClearFontsCache() {
	fm.fontsCache = map[string]*Font{}
}

func (fm *FontsManager) Font(ttf []byte) (*Font, error) {
	f, ok := fm.fontsCache[string(ttf)]
	if ok {
		return f, nil
	}
	f, err := NewFont(ttf)
	if err != nil {
		return nil, err
	}
	fm.fontsCache[string(ttf)] = f
	return f, nil
}

//----------

type Font struct {
	Font       *sfnt.Font
	facesCache map[opentype.FaceOptions]*FontFace
}

func NewFont(ttf []byte) (*Font, error) {
	font, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	f := &Font{Font: font}
	f.ClearFacesCache()

	return f, nil
}

func (f *Font) ClearFacesCache() {
	f.facesCache = map[opentype.FaceOptions]*FontFace{}
}

func (f *Font) FontFace(opt opentype.FaceOptions) *FontFace {
	// avoid divide by zero; also ensure face.metrics() works
	if opt.Size == 0 {
		opt.Size = 12 // internal opentype default
	}
	if opt.DPI == 0 {
		opt.DPI = 72
	}

	ff, ok := f.facesCache[opt]
	if ok {
		return ff
	}
	ff = NewFontFace(f, opt)
	f.facesCache[opt] = ff
	return ff
}

func (f *Font) FontFace2(size float64) *FontFace {
	opt := opentype.FaceOptions{Size: size}
	return f.FontFace(opt)
}

//----------

type FontFace struct {
	Font    *Font
	Face    font.Face
	Size    float64 // in points, readonly
	Metrics *font.Metrics

	lineHeight fixed.Int26_6
	baselineY  fixed.Int26_6
}

func NewFontFace(font *Font, opt opentype.FaceOptions) *FontFace {
	// should be set from font.fontface
	if opt.Size == 0 || opt.DPI == 0 {
		panic("!")
	}

	face, err := opentype.NewFace(font.Font, &opt)
	if err != nil { // currently, no error is being returned
		panic(err)
	}

	face = NewFaceRunes(face)
	// TODO: allow cache choice
	//face = NewFaceCache(face) // safe for ui loop thread only (read)
	face = NewFaceCacheL(face) // safe for concurrent calls
	//face = NewFaceCacheL2(face)

	ff := &FontFace{Font: font, Face: face, Size: opt.Size}
	m := face.Metrics()
	ff.Metrics = &m

	//ff.lineHeight = ff.Metrics.Height
	//ff.baselineY = ff.Metrics.Ascent
	ff.lineHeight = max(
		ff.Metrics.Ascent+ff.Metrics.Descent,
		ff.Metrics.Height)
	ff.baselineY = min(
		ff.Metrics.Ascent,
		ff.lineHeight-ff.Metrics.Descent)

	return ff
}

func (ff *FontFace) LineHeight() fixed.Int26_6 {
	return ff.lineHeight
}
func (ff *FontFace) LineHeightInt() int {
	return ff.LineHeight().Ceil()
}
func (ff *FontFace) LineHeightFloat() float64 {
	return Fixed266ToFloat64(ff.LineHeight())
}

func (ff *FontFace) BaseLine() fixed.Point26_6 {
	return fixed.Point26_6{0, ff.baselineY}
}
