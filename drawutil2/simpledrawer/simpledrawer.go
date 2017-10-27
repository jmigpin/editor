package simpledrawer

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/drawutil2/loopers"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func Measure(face font.Face, str string, max image.Point) image.Point {
	start := &loopers.EmbedLooper{}
	var strl loopers.StringLooper
	strl.Init(face, str)
	linel := loopers.NewLineLooper(&strl)
	ml := loopers.NewMeasureLooper(&strl)

	// iterator order
	strl.SetOuterLooper(start)
	linel.SetOuterLooper(&strl)
	ml.SetOuterLooper(linel)

	ml.Loop(func() bool { return true })

	// truncate measure
	m := image.Point{ml.M.X.Ceil(), ml.M.Y.Ceil()}
	if m.X > max.X {
		m.X = max.X
	}
	if m.Y > max.Y {
		m.Y = max.Y
	}

	return m
}

func Draw(img draw.Image, face font.Face, str string, bounds *image.Rectangle, fg color.Color) {
	max := bounds.Size()
	fmax := fixed.P(max.X, max.Y)

	start := &loopers.EmbedLooper{}
	var strl loopers.StringLooper
	strl.Init(face, str)
	linel := loopers.NewLineLooper(&strl)
	var dl loopers.DrawLooper
	dl.Init(&strl, img, bounds)
	var eel loopers.EarlyExitLooper
	eel.Init(&strl, fmax.Y)

	dl.Fg = fg

	// iterator order
	strl.SetOuterLooper(start)
	linel.SetOuterLooper(&strl)
	dl.SetOuterLooper(linel)
	eel.SetOuterLooper(&dl)

	// draw runes
	eel.Loop(func() bool { return true })
}
