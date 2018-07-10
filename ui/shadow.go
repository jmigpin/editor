package ui

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

var (
	ShadowsOn     = true
	shadowMaxDiff = 0.25
)

func WrapInTopShadowOrSeparator(ctx widget.ImageContext, content widget.Node) widget.Node {
	if ShadowsOn {
		s := widget.NewTopShadow(ctx, content)
		s.MaxDiff = shadowMaxDiff
		tf := content.Embed().TreeThemeFont()
		s.Height = UIThemeUtil.ShadowHeight(tf)
		return s
	} else {
		bl := widget.NewBoxLayout()
		bl.YAxis = true
		rect := widget.NewRectangle(ctx)
		rect.SetThemePaletteNamePrefix("shadowsep_")
		rect.Size.Y = separatorWidth
		bl.Append(rect, content)
		bl.SetChildFlex(content, true, true)
		bl.SetChildFill(rect, true, true)
		return bl
	}
}

func WrapInBottomShadowOrNone(ctx widget.ImageContext, content widget.Node) widget.Node {
	if !ShadowsOn {
		return content
	}
	s := widget.NewBottomShadow(ctx, content)
	s.MaxDiff = shadowMaxDiff
	tf := content.Embed().TreeThemeFont()
	s.Height = UIThemeUtil.ShadowHeight(tf)
	return s
}
