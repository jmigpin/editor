package loopers

import (
	"image"
	"testing"

	"github.com/jmigpin/editor/drawutil2"
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
	f1 := drawutil2.GetTestFace()
	f2 := drawutil2.NewFaceRunes(f1)
	f3 := drawutil2.NewFaceCache(f2)
	face := f3

	bounds := image.Rect(0, 0, 1000, 1000)
	max := fixed.P(bounds.Dx(), bounds.Dy())

	start := &EmbedLooper{}
	var strl StringLooper
	strl.Init(face, testStr7)
	linel := NewLineLooper(&strl)
	var wlinel WrapLineLooper
	wlinel.Init(&strl, linel, max.X)
	var pdl PosDataLooper
	pdl.Init()
	pdl.Setup(&strl, []PosDataKeeper{&strl, &wlinel})
	pdl.Jump = 5

	strl.SetOuterLooper(start)
	linel.SetOuterLooper(&strl)
	wlinel.SetOuterLooper(linel)
	pdl.SetOuterLooper(&wlinel)

	pdl.Loop(func() bool { return true })

	t.Logf("pdl has %v points", len(pdl.data))

	t.Logf("ri %v", strl.Ri)

	//for _, d := range pdl.data {
	//log.Printf("pdl data ri %v %v", d.ri, d.penBoundsMaxY)
	//}

	p := fixed.P(10, 0)
	//pdl.RestorePosDataCloseToPoint(&p)
	pd, ok := pdl.PosDataCloseToPoint(&p)
	if ok {
		t.Logf("restoring %+v", pd)
		pdl.restore(pd)
	}
	i := pdl.GetIndex(&p, &wlinel)

	t.Logf("ri %v", strl.Ri)
	t.Logf("i %v", i)

	//------

	//// update

	//bounds = image.Rect(0, 0, 200, 20)
	//max = fixed.P(bounds.Dx(), bounds.Dy())

	//wlinel.MaxX = max.X
	//pdl.Update()

	//log.Printf("pen max %v", max)
	//log.Printf("pen %v", strl.Pen)
	//for _, d := range pdl.data {
	//	log.Printf("%v", spew.Sdump(d))
	//}
}
