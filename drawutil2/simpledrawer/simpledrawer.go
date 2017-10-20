package simpledrawer

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/drawutil2/loopers"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func Measure(face font.Face, str string, max *image.Point) *fixed.Point26_6 {
	max2 := fixed.P(max.X, max.Y)
	var bounds image.Rectangle
	bounds.Max = *max
	strl := loopers.NewStringLooper(face, str)
	linel := loopers.NewLineLooper(strl, max2.Y)
	ml := loopers.NewMeasureLooper(strl, &max2)
	eel := loopers.NewEarlyExitLooper(strl, &bounds)

	// iterator order
	linel.SetOuterLooper(strl)
	ml.SetOuterLooper(linel)
	eel.SetOuterLooper(ml)

	eel.Loop(func() bool { return true })

	return ml.M
}

func Draw(img draw.Image, face font.Face, str string, bounds *image.Rectangle, fg color.Color) {
	max := bounds.Max
	max2 := fixed.P(max.X, max.Y)

	strl := loopers.NewStringLooper(face, str)
	linel := loopers.NewLineLooper(strl, max2.Y)
	dl := loopers.NewDrawLooper(strl, img, bounds)
	eel := loopers.NewEarlyExitLooper(strl, bounds)

	dl.Fg = fg

	// iterator order
	linel.SetOuterLooper(strl)
	dl.SetOuterLooper(linel)
	eel.SetOuterLooper(dl)

	// draw runes
	eel.Loop(func() bool { return true })
}
