package fontutil

import (
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var FontsMan = NewFontsManager()

//----------

type FontsManager struct {
	fcmu       sync.Mutex
	fontsCache map[string]*Font
}

func NewFontsManager() *FontsManager {
	fm := &FontsManager{}
	fm.ClearFontsCache()
	return fm
}

func (fm *FontsManager) ClearFontsCache() {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
	fm.fontsCache = map[string]*Font{}
}

func (fm *FontsManager) Font(ttf []byte) (*Font, error) {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
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

func (fm *FontsManager) mustFont(ttf []byte) *Font {
	f, err := fm.Font(ttf)
	if err != nil {
		panic(err)
	}
	return f
}

//----------

type Font struct {
	Font *sfnt.Font

	fcmu       sync.Mutex
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
	f.fcmu.Lock()
	defer f.fcmu.Unlock()
	f.facesCache = map[opentype.FaceOptions]*FontFace{}
}
func (f *Font) FontFace(fopts FaceOptions) *FontFace {
	f.fcmu.Lock()
	defer f.fcmu.Unlock()
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

	if emojiFont := DefaultEmojiFont(); font != emojiFont {
		emojiFace := newEmojiFace(face, emojiFont, fopts)
		face = NewFaceEmoji(face, emojiFace)
	}

	face = NewFaceRunes(face)
	// TODO: allow cache choice
	//face = NewFaceCache(face) // safe for ui loop thread only (read)
	face = NewFaceCacheL(face) // safe for concurrent calls
	//face = NewFaceCacheL2(face)

	ff := &FontFace{Font: font, Face: face, Opts: fopts}
	m := face.Metrics()
	ff.Metrics = &m

	ff.lineHeight, ff.baselineY = faceLineHeightBaseline(face)

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

func (ff *FontFace) AvgGlyphAdvance() fixed.Int26_6 {
	const sample = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sum fixed.Int26_6
	n := 0
	for _, ru := range sample {
		adv, ok := ff.Face.GlyphAdvance(ru)
		if !ok {
			continue
		}
		sum += adv
		n++
	}
	if n == 0 {
		return 1
	}
	return sum / fixed.Int26_6(n)
}

func (ff *FontFace) TestIsMono() bool {
	return testIsMonoFace(ff.Face)
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

func newEmojiFace(face font.Face, emojiFont *Font, fopts FaceOptions) font.Face {
	emojiFace := mustNewFace(emojiFont.Font, &fopts.opts)
	maxAdv, ok := face.GlyphAdvance('W')
	if !ok {
		maxAdv = fixed.I(2)
	}
	maxHeight, _ := faceLineHeightBaseline(face)

	for _, p := range []float64{1.0, 0.95, 0.9, 0.85, 0.8, 0.75, 0.7, 0.65, 0.6, 0.55, 0.5, 0.45, 0.4} {
		fopts2 := fopts
		fopts2.SetSize(fopts.Size() * p)
		face2 := mustNewFace(emojiFont.Font, &fopts2.opts)
		if ok := emojiFaceFits(face2, maxAdv, maxHeight); ok {
			return face2
		}
	}
	return emojiFace
}

func emojiFaceFits(emojiFace font.Face, maxAdv, maxHeight fixed.Int26_6) bool {
	for _, ru := range []rune{'🙂', '👍', '🔥', '✅'} {
		bounds, adv, ok := emojiFace.GlyphBounds(ru)
		if !ok {
			continue
		}
		if adv > maxAdv {
			return false
		}
		h := bounds.Max.Y - bounds.Min.Y
		if h > maxHeight {
			return false
		}
	}
	return true
}

//----------

func faceLineHeightBaseline(face font.Face) (fixed.Int26_6, fixed.Int26_6) {
	m := face.Metrics()
	lineHeight := max(
		m.Ascent+m.Descent,
		m.Height)
	baselineY := min(
		m.Ascent,
		lineHeight-m.Descent)
	return lineHeight, baselineY
}

func testIsMonoFace(face font.Face) bool {
	adv1, ok1 := face.GlyphAdvance('W')
	adv2, ok2 := face.GlyphAdvance('i')
	return ok1 && ok2 && adv1 == adv2
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
