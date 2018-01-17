package loopers

import (
	"image"
	"testing"

	"github.com/jmigpin/editor/util/drawutil"
	"golang.org/x/image/math/fixed"
)

var loremStr2 = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

var testStr3 = "a\na\na\na\na\na\na\na\na\na\na\na\na\na\n"

var testStr4 = "aaaaa\nbbbbb\nccccc\n"
var testStr5 = `
aaaaa
	bbbbb
	ccccc
`

var testStr6 = "abcde abcde abcde abcde abcde"
var testStr7 = `
abcde
abcde
abcde
abcde
abcde
abcde
abcde
abcde
abcde
abcde
abcde`

func TestPosData1(t *testing.T) {
	f1 := drawutil.GetTestFace()
	f2 := drawutil.NewFaceRunes(f1)
	f3 := drawutil.NewFaceCache(f2)
	face := f3

	bounds := image.Rect(0, 0, 1000, 1000)
	max := fixed.P(bounds.Dx(), bounds.Dy())

	start := &EmbedLooper{}
	strl := MakeString(face, testStr7)
	linel := MakeLine(&strl, 0)
	wlinel := MakeWrapLine(&strl, &linel, max.X)
	keepers := []PosDataKeeper{&strl, &wlinel}
	pdl := MakePosData(&strl, keepers, 5, nil)

	strl.SetOuterLooper(start)
	linel.SetOuterLooper(&strl)
	wlinel.SetOuterLooper(&linel)
	pdl.SetOuterLooper(&wlinel)

	// run
	pdl.Loop(func() bool { return true })

	if len(pdl.Data) != 14 {
		t.Logf("data len %v", len(pdl.Data))
		t.Fatal()
	}

	// test position
	p := fixed.P(10, 0)
	pd, ok := pdl.PosDataCloseToPoint(&p)
	if ok {
		pdl.restore(pd)
	}
	i := pdl.GetIndex(&p)
	if i != 0 {
		t.Log(i)
		t.Fatal()
	}

	// test position
	p = fixed.P(20, 40)
	pd, ok = pdl.PosDataCloseToPoint(&p)
	if ok {
		pdl.restore(pd)
	}
	i = pdl.GetIndex(&p)
	if i != 10 {
		t.Log(i)
		t.Fatal()
	}
}
