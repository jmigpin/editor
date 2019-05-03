package drawer4

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

func TestEmpty(t *testing.T) {
	d := New()
	d.SetFace(drawutil.GetTestFace())
	d.SetBounds(image.Rect(0, 0, 100, 100))

	s := ""
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	w := d.wlineStartIndex(true, 10, 0, nil)
	if w != 0 {
		t.Fatal()
	}
}

func TestNLinesStartIndex1(t *testing.T) {
	d := New()
	d.SetFace(drawutil.GetTestFace())
	d.SetBounds(image.Rect(0, 0, 100, 100))

	s := "111\n222\n333"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)
	pos := r.Max()
	d.SetRuneOffset(pos)
	w := d.iters.lineStart.lineStartIndex(pos, 0)
	if w != 8 {
		t.Fatal()
	}
	w = d.iters.lineStart.lineStartIndex(pos, 1)
	if w != 4 {
		t.Fatal()
	}
	w = d.iters.lineStart.lineStartIndex(pos, 2)
	if w != 0 {
		t.Fatal()
	}
	w = d.iters.lineStart.lineStartIndex(pos, 100)
	if w != 0 {
		t.Fatal()
	}
}

//----------

func TestImg01(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Draw(img)
	cmpResult(t, img, "img01")
}

func TestImg01bDrawFullLineAtEndOfLineOffset(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.SetRuneOffset(5)

	d.Draw(img)
	cmpResult(t, img, "img01b")
}

func TestImg02WrapLine(t *testing.T) {
	d, img := newTestDrawer()

	s := "1111111\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Draw(img)
	cmpResult(t, img, "img02")
}

func TestImg03Ident(t *testing.T) {
	d, img := newTestDrawer()

	s := "  1111111\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Draw(img)
	cmpResult(t, img, "img03")
}

func TestImg04Offset1(t *testing.T) {
	d, img := newTestDrawer()

	s := "  1111111\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)
	d.SetRuneOffset(6)

	d.Draw(img)
	cmpResult(t, img, "img04")
}

func TestImg04bOffset2(t *testing.T) {
	d, img := newTestDrawer()

	s := "  1111111\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)
	d.SetRuneOffset(8)

	d.Draw(img)
	cmpResult(t, img, "img04b")
}

func TestImg05RunePerLine(t *testing.T) {
	rect := image.Rect(0, 0, 14, 100)
	d, img := newTestDrawerRect(rect)

	s := "WWW"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Draw(img)
	cmpResult(t, img, "img05")
}

func TestImg06Scroll1(t *testing.T) {
	d, img := newTestDrawer()

	s := "  1111111\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	sy := d.scrollSizeYDown(3)
	d.SetScrollOffset(image.Point{0, sy})

	d.Draw(img)
	cmpResult(t, img, "img06")
}

func TestImg07Scroll2(t *testing.T) {
	d, img := newTestDrawer()

	s := "  1111221\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	sy := d.scrollSizeYDown(1)
	d.SetRuneOffset(sy)

	sy = d.scrollSizeYDown(1) // 2nd line
	d.SetRuneOffset(sy)

	d.Draw(img)
	cmpResult(t, img, "img07")
}

func TestImg08Scroll3(t *testing.T) {
	d, img := newTestDrawer()

	s := "  1111221\n22222\n33333\n44444"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.SetRuneOffset(10)

	sy := d.scrollSizeYUp(1)
	d.SetRuneOffset(sy)

	sy = d.scrollSizeYUp(1) // 2nd line
	d.SetRuneOffset(sy)

	d.Draw(img)
	cmpResult(t, img, "img08")
}

func TestImg09Visible(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444\n55555\n66666\n77777\n88888"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	o := d.RangeVisibleOffset(r.Max(), 0)
	d.SetRuneOffset(o)

	d.Draw(img)
	cmpResult(t, img, "img09")
}

func TestImg10Visible(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444\n55555\n66666\n77777\n88888"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	o := d.RangeVisibleOffset(19, 4) // line with 4's
	d.SetRuneOffset(o)

	d.Draw(img)
	cmpResult(t, img, "img10")
}

func TestImg11Visible(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444\n55555\n66666\n77777\n88888"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	o := d.RangeVisibleOffset(19, 7) // line with 4's
	d.SetRuneOffset(o)

	d.Draw(img)
	cmpResult(t, img, "img11")
}

func TestImg12Cursor(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444\n55555\n66666\n77777\n88888"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Opt.Cursor.On = true

	c := 17
	d.SetRuneOffset(c)
	d.SetCursorOffset(c)

	p := d.LocalPointOf(c)
	p.Y -= d.LineHeight() - 1
	k := d.LocalIndexOf(p)

	d.SetRuneOffset(k)
	d.SetCursorOffset(k)

	d.Draw(img)
	cmpResult(t, img, "img12")
}

func TestImg13Cursor(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444\n55555\n66666\n77777\n88888"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Opt.Cursor.On = true

	c := r.Max()
	d.SetRuneOffset(c - 3)
	d.SetCursorOffset(c)

	// range visible when offset was eof was causing draw at bottom
	u := d.RangeVisibleOffset(c, 1)
	d.SetRuneOffset(u)

	d.Draw(img)
	cmpResult(t, img, "img13")
}

func TestImg14Cursor(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333\n44444\n55555\n66666\n77777\n88888"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Opt.Cursor.On = true

	c := r.Max()
	d.SetRuneOffset(c)

	l := 8
	r.Delete(r.Max()-l, l)

	d.Draw(img)
	cmpResult(t, img, "img14")
}

func TestImg15Visible(t *testing.T) {
	d, img := newTestDrawer()

	s := "11111\n22222\n33333"
	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Opt.Cursor.On = true

	c := r.Max()
	d.SetRuneOffset(c)

	r.Delete(r.Min(), iorw.MMLen(r))
	b, _ := r.ReadNSliceAt(r.Min(), iorw.MMLen(r))
	_ = string(b)

	o := d.RangeVisibleOffset(0, 0)
	d.SetRuneOffset(o)

	r.Insert(r.Min(), []byte("44444\n"))

	d.Draw(img)
	cmpResult(t, img, "img15")
}

func TestImg16Select(t *testing.T) {
	rect := image.Rect(0, 0, 100, 100)
	d, img := newTestDrawerRect(rect)

	tmp := limitedReaderPadding
	defer func() { limitedReaderPadding = tmp }()
	limitedReaderPadding = 3

	s := ""
	for i := 0; i < 10; i++ {
		s += fmt.Sprintf("%v", i%10)
	}

	r := iorw.NewBytesReadWriter([]byte(s))
	d.SetReader(r)

	d.Opt.Cursor.On = true

	d.SetRuneOffset(7)
	d.SetCursorOffset(7)

	d.Draw(img)
	cmpResult(t, img, "img16")
}

//----------

func newTestDrawer() (*Drawer, draw.Image) {
	rect := image.Rect(0, 0, 50, 70)
	return newTestDrawerRect(rect)
}

func newTestDrawerRect(rect image.Rectangle) (*Drawer, draw.Image) {
	face := newTestFace()
	d := New()
	d.SetFace(face)
	d.SetBounds(rect)
	d.SetFg(color.Black)

	d.smoothScroll = false
	d.Opt.LineWrap.Bg = colornames.Red

	img := image.NewRGBA(rect)
	return d, img
}

func newTestFace() font.Face {
	ttf := goregular.TTF
	f, err := truetype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	return drawutil.NewFace(f, &truetype.Options{DPI: 100})
}

var testImg0Dir = "testimgs"

func imgFilename(name string) string {
	return filepath.Join(testImg0Dir, name+".png")
}

func cmpResult(t *testing.T, img image.Image, name string) {
	t.Helper()

	// auto save if file doesn't exit
	fname := imgFilename(name)
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		saveResult(img, name)
		return
	}

	compareResult(t, img, name)
}

func saveResult(img image.Image, name string) {
	fname := filepath.Join(testImg0Dir, name+".png")
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func openResult(name string) image.Image {
	fname := filepath.Join(testImg0Dir, name+".png")
	f, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	return img
}

func compareResult(t *testing.T, img image.Image, name string) {
	t.Helper()
	img2 := openResult(name)
	if img.Bounds() != img2.Bounds() {
		saveResult(img, name+"_err")
		t.Fatal("different bounds")
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y))
			c2 := color.RGBAModel.Convert(img2.At(x, y))
			if c != c2 {
				saveResult(img, name+"_err")
				t.Fatalf("different color value: %vx%v: %v %v", x, y, c, c2)
			}
		}
	}
}
