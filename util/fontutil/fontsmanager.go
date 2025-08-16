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
	return DefaultFont().FontFace(DefaultFaceOptions())
}
func DefaultFaceOptions() FaceOptions {
	return NewFaceOptions(12, 72)
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
func (f *Font) FontFace(fopts FaceOptions) *FontFace {
	ff, ok := f.facesCache[fopts.opts]
	if ok {
		return ff
	}
	ff = NewFontFace(f, fopts)
	f.facesCache[fopts.opts] = ff
	return ff
}

//----------

type FontFace struct {
	Font    *Font
	Face    font.Face
	Opts    FaceOptions // readonly, make copy and change
	Metrics *font.Metrics

	lineHeight fixed.Int26_6
	baselineY  fixed.Int26_6
}

func NewFontFace(font *Font, fopts FaceOptions) *FontFace {
	face := mustNewFace(font.Font, &fopts.opts)

	face = NewFaceRunes(face)
	// TODO: allow cache choice
	//face = NewFaceCache(face) // safe for ui loop thread only (read)
	face = NewFaceCacheL(face) // safe for concurrent calls
	//face = NewFaceCacheL2(face)

	ff := &FontFace{Font: font, Face: face, Opts: fopts}
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

//----------
//----------
//----------

func mustNewFace(font *opentype.Font, fopts *opentype.FaceOptions) font.Face {
	face, err := opentype.NewFace(font, fopts)
	if err != nil {
		panic(err) // TODO: opentype.newface() doesn't return errors for now
	}
	return face
}

//----------

// avoid zero value in size/dpi by forcing set funcs; use existing copies
type FaceOptions struct {
	opts opentype.FaceOptions
}

func NewFaceOptions(size, dpi float64) FaceOptions {
	o := FaceOptions{}
	o.SetSize(size)
	o.SetDPI(dpi)
	return o
}
func (o *FaceOptions) Size() float64 {
	return o.opts.Size
}
func (o *FaceOptions) DPI() float64 {
	return o.opts.DPI
}
func (o *FaceOptions) SetSize(v float64) {
	o.opts.Size = max(0.1, v)
}
func (o *FaceOptions) SetDPI(v float64) {
	o.opts.DPI = max(0.1, v)
}
func (o *FaceOptions) SetHinting(h font.Hinting) {
	o.opts.Hinting = h
}
