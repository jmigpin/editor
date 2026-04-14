package fontutil

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var FontsMan = NewFontsManager()

func init() {
	FontsMan.mustFont(goregular.TTF, "embedded:regular")
	FontsMan.mustFont(gomedium.TTF, "embedded:medium")
	FontsMan.mustFont(gomono.TTF, "embedded:mono")

	FontsMan.RegisterAlias("regular", "go_regular")
	FontsMan.RegisterAlias("medium", "go_medium")
	FontsMan.RegisterAlias("mono", "go_mono")
}

//----------

type FontsManager struct {
	fcmu       sync.Mutex
	fontsCache map[string]*Font
	aliases    map[string]string

	fallbackFonts []*Font
}

func NewFontsManager() *FontsManager {
	fm := &FontsManager{
		fontsCache: make(map[string]*Font),
		aliases:    make(map[string]string),
	}
	return fm
}

func (fm *FontsManager) RegisterAlias(alias, targetName string) {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
	fm.aliases[sanitizeFontName(alias)] = sanitizeFontName(targetName)
}

func (fm *FontsManager) AddFallbackFont(f *Font) {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
	for _, f2 := range fm.fallbackFonts {
		if f2 == f {
			return
		}
	}
	fm.fallbackFonts = append(fm.fallbackFonts, f)
}

func (fm *FontsManager) ClearFontsCache() {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
	fm.fontsCache = map[string]*Font{}
}

func (fm *FontsManager) Font(ttf []byte, srcName string) (*Font, error) {
	hash0 := sha256.Sum256(ttf)
	hash := hex.EncodeToString(hash0[:])

	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
	f, ok := fm.fontsCache[hash]
	if ok {
		return f, nil
	}
	f, err := NewFont(fm, ttf, srcName)
	if err != nil {
		return nil, err
	}
	fm.fontsCache[hash] = f
	return f, nil
}

func (fm *FontsManager) mustFont(ttf []byte, srcName string) *Font {
	f, err := fm.Font(ttf, srcName)
	if err != nil {
		panic(err)
	}
	return f
}

//----------

type Font struct {
	Font    *sfnt.Font
	fm      *FontsManager
	SrcName string

	fcmu       sync.Mutex
	facesCache map[opentype.FaceOptions]*FontFace
}

func NewFont(fm *FontsManager, ttf []byte, srcName string) (*Font, error) {
	font, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	f := &Font{Font: font, fm: fm, SrcName: srcName}
	f.ClearFacesCache()

	return f, nil
}
func (f *Font) ClearFacesCache() {
	f.fcmu.Lock()
	defer f.fcmu.Unlock()
	f.facesCache = map[opentype.FaceOptions]*FontFace{}
}

func (f *Font) Name() string {
	s, err := f.Font.Name(nil, sfnt.NameIDFull)
	if err != nil {
		return ""
	}
	return s
}

func (f *Font) NameID() string {
	return sanitizeFontName(f.Name())
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

func (fm *FontsManager) FontByName(name string) *Font {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()

	name = sanitizeFontName(name)

	// follow aliases
	for {
		target, ok := fm.aliases[name]
		if !ok {
			break
		}
		name = target
	}

	for _, f := range fm.fontsCache {
		if f.NameID() == name {
			return f
		}
	}
	return nil
}

func (fm *FontsManager) Aliases(targetName string) []string {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
	targetName = sanitizeFontName(targetName)
	res := []string{}
	for alias, target := range fm.aliases {
		if target == targetName {
			res = append(res, alias)
		}
	}
	return res
}

func sanitizeFontName(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ToLower(s)
	// replace multiple underscores with one
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}

func (fm *FontsManager) LoadedFonts() []*Font {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()

	isFallback := func(f *Font) bool {
		for _, f2 := range fm.fallbackFonts {
			if f2 == f {
				return true
			}
		}
		return false
	}

	w := []*Font{}
	for _, f := range fm.fontsCache {
		if !isFallback(f) {
			w = append(w, f)
		}
	}
	return w
}

func (fm *FontsManager) FallbackFonts() []*Font {
	fm.fcmu.Lock()
	defer fm.fcmu.Unlock()
	w := make([]*Font, len(fm.fallbackFonts))
	copy(w, fm.fallbackFonts)
	return w
}

//----------

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
	isMono := testIsMonoFace(face)
	monoAdv, _ := face.GlyphAdvance('W')

	// User fallback fonts (from cmd line)
	for _, ff := range font.fm.fallbackFonts {
		if font != ff {
			fallbackFaces := NewFallbackFaces(ff, fopts)
			face = NewFaceFallback(face, fallbackFaces, isMono, monoAdv)
		}
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
