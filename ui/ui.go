package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil"
)

type UI struct {
	*uiutil.BasicUI
	Root    *Root
	OnError func(error)
}

func NewUI(events chan<- interface{}, winName string) (*UI, error) {
	ui := &UI{OnError: func(error) {}}
	ui.Root = NewRoot(ui)

	bui, err := uiutil.NewBasicUI(events, winName, ui.Root)
	if err != nil {
		return nil, err
	}
	ui.BasicUI = bui

	// set theme before root init
	c1 := &ColorThemeCycler
	c1.Set(c1.CurName, ui.Root)
	c2 := &FontThemeCycler
	c2.Set(c2.CurName, ui.Root)

	// build ui - needs ui.BasicUI to be set
	ui.Root.Init()

	return ui, nil
}

func (ui *UI) WarpPointerToRectanglePad(r image.Rectangle) {
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

	set(&p.X, r.Min.X, r.Max.X)
	set(&p.Y, r.Min.Y, r.Max.Y)

	ui.WarpPointer(p)
}
