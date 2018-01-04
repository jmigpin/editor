package ui

import (
	"image"
	"time"

	"github.com/jmigpin/editor/uiutil"
	"golang.org/x/image/font"
)

const (
	DrawFrameRate = 35
	FlashDuration = 500 * time.Millisecond
)

var (
	SeparatorWidth = 1
	ScrollbarWidth = 10
	ScrollbarLeft  = false
	ShadowsOn      = true
	ShadowMaxShade = 0.25
	ShadowSteps    = 8
)

type UI struct {
	*uiutil.BasicUI
	Layout  Layout
	OnError func(error)
}

func NewUI(events chan<- interface{}, winName string) (*UI, error) {
	bui, err := uiutil.NewBasicUI(events, winName)
	if err != nil {
		return nil, err
	}

	ui := &UI{
		BasicUI: bui,
		OnError: func(error) {},
	}
	ui.DrawFrameRate = DrawFrameRate

	ui.Layout.Init(ui)
	ui.BasicUI.RootNode = &ui.Layout

	return ui, nil
}

// Implements widget.Context
func (ui *UI) FontFace1() font.Face {
	return FontFace
}

func (ui *UI) WarpPointerToRectanglePad(r0 *image.Rectangle) {
	p, err := ui.QueryPointer()
	if err != nil {
		return
	}

	pad := 5

	set := func(v *int, min, max int) {
		if max-min < pad*2 {
			*v = min + (max-min)/2
		} else {
			if *v < min+pad {
				*v = min + pad
			} else if *v > max-pad {
				*v = max - pad
			}
		}
	}

	r := *r0
	set(&p.X, r.Min.X, r.Max.X)
	set(&p.Y, r.Min.Y, r.Max.Y)

	ui.WarpPointer(p)
}
