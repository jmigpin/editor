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
		s2 := &topShadow2{s}
		return s2
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

//----------

func WrapInBottomShadowOrNone(ctx widget.ImageContext, content widget.Node) widget.Node {
	if !ShadowsOn {
		return content
	}
	s := widget.NewBottomShadow(ctx, content)
	s.MaxDiff = shadowMaxDiff
	s2 := &bottomShadow2{s}
	return s2
}

//----------

type topShadow2 struct {
	*widget.TopShadow
}

func (s *topShadow2) OnThemeChange() {
	ff := s.FirstChild().Embed().TreeThemeFontFace()
	s.TopShadow.Height = UIThemeUtil.ShadowHeight(ff)
}

//----------

type bottomShadow2 struct {
	*widget.BottomShadow
}

func (s *bottomShadow2) OnThemeChange() {
	ff := s.FirstChild().Embed().TreeThemeFontFace()
	s.BottomShadow.Height = UIThemeUtil.ShadowHeight(ff)
}
