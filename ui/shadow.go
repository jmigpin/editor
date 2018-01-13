package ui

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

var (
	ShadowsOn     = true
	shadowMaxDiff = 0.25
)

func WrapInShadowTop(ctx widget.ImageContext, child widget.Node) widget.Node {
	if ShadowsOn {
		s := widget.NewShadow(ctx, child)
		s.MaxDiff = shadowMaxDiff
		s.Top = ShadowHeight()

		//s.Tint = true // TODO: darker colors

		return s
	}
	return child
}
func WrapInShadowBottom(ctx widget.ImageContext, child widget.Node) widget.Node {
	if ShadowsOn {
		s := widget.NewShadow(ctx, child)
		s.MaxDiff = shadowMaxDiff
		s.Bottom = ShadowHeight()
		return s
	}
	return child
}
