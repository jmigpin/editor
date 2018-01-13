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
	bui, err := uiutil.NewBasicUI(events, winName)
	if err != nil {
		return nil, err
	}

	ui := &UI{
		BasicUI: bui,
		OnError: func(error) {},
	}

	SetupRoot(ui)

	return ui, nil
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
