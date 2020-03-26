package fontutil

import (
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

//----------

var DPI float64 // default: github.com/golang/freetype/truetype/face.go:31:3
var FontsMan = NewFontsManager()

func DefaultFontFace() *FontFace {
	f, err := FontsMan.Font(goregular.TTF)
	if err != nil {
		panic(err)
	}
	opt := truetype.Options{} // defaults: size=12, dpi=72, ~14px
	return f.FontFace(opt)
}

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
	Font       *truetype.Font
	facesCache map[truetype.Options]*FontFace
}

func NewFont(ttf []byte) (*Font, error) {
	font, err := truetype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	f := &Font{Font: font}
	f.ClearFacesCache()
	return f, nil
}

func (f *Font) ClearFacesCache() {
	f.facesCache = map[truetype.Options]*FontFace{}
}

func (f *Font) FontFace(opt truetype.Options) *FontFace {
	if opt.DPI == 0 {
		opt.DPI = DPI
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
	opt := truetype.Options{Size: size}
	return f.FontFace(opt)
}

//----------

type FontFace struct {
	Font       *Font
	Face       font.Face
	Size       float64 // in points, readonly
	Metrics    *font.Metrics
	lineHeight fixed.Int26_6
}

func NewFontFace(font *Font, opt truetype.Options) *FontFace {
	face := truetype.NewFace(font.Font, &opt)
	face = NewFaceRunes(face)
	// TODO: allow cache choice
	//face = NewFaceCache(face) // can safely be used only in ui loop (read)
	face = NewFaceCacheL(face) // safe for concurrent calls
	//face = NewFaceCacheL2(face)

	ff := &FontFace{Font: font, Face: face}
	m := face.Metrics()
	ff.Metrics = &m
	ff.lineHeight = ff.calcLineHeight()

	ff.Size = opt.Size
	if ff.Size == 0 {
		// github.com/golang/freetype/truetype/face.go:26:2
		ff.Size = 12
	}

	return ff
}

func (ff *FontFace) calcLineHeight() fixed.Int26_6 {
	// TODO: failing: m.Height
	m := ff.Metrics
	lh := m.Ascent + m.Descent
	return fixed.I(lh.Ceil()) // ceil for stable lines
}

func (ff *FontFace) LineHeight() fixed.Int26_6 {
	return ff.lineHeight
}
func (ff *FontFace) LineHeightInt() int {
	return ff.LineHeight().Floor()
}
func (ff *FontFace) LineHeightFloat() float64 {
	return Fixed266ToFloat64(ff.LineHeight())
}

func (ff *FontFace) BaseLine() fixed.Point26_6 {
	return fixed.Point26_6{0, ff.Metrics.Ascent}
}
