package drawer3

import "image/color"

// Current colors
type CurColors struct {
	EExt
	Fg, Bg  color.Color
	startFg color.Color
}

func (cc *CurColors) setup(startFg color.Color) {
	cc.startFg = startFg
}

func (cc *CurColors) Start(r *ExtRunner) {
	// also sets if the reader is empty (ex: cursor might need it at the end with len=0)
	cc.Fg = cc.startFg
	cc.Bg = nil
}

func (cc *CurColors) Iterate(r *ExtRunner) {
	cc.Fg = cc.startFg
	cc.Bg = nil
	r.NextExt()
}
