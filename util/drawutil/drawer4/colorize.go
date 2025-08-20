package drawer4

import (
	"image/color"
)

type Colorize struct {
	d *Drawer
}

func (c *Colorize) Init() {
	c.d.st.colorize.indexes = make([]int, len(c.d.Opt.Colorize.Groups))
}

func (c *Colorize) Iter() {
	c.colorize()
	if !c.d.iterNext() {
		return
	}
}

func (c *Colorize) End() {}

//----------

func (c *Colorize) colorize() {
	ri := c.d.st.runeR.ri
	for k, g := range c.d.Opt.Colorize.Groups {
		if g == nil || g.Off {
			continue
		}
		var w *ColorizeOp
		i := &c.d.st.colorize.indexes[k]
		for k := *i; k < len(g.Ops); k++ {
			op := g.Ops[k]
			if ri >= op.Offset {
				w = op
				*i = k
			} else if ri < op.Offset {
				break
			}
		}
		if w != nil {
			c.applyOp(w)
		}
	}
}

func (c *Colorize) applyOp(op *ColorizeOp) {
	st := &c.d.st.curColors
	if op.Fg != nil {
		st.fg = op.Fg
	} else if op.SetNil {
		st.fg = c.d.fg // default drawer color
	}
	if op.Bg != nil || op.SetNil {
		st.bg = op.Bg
	}
	if op.ProcColor != nil {
		st.fg, st.bg = op.ProcColor(st.fg, st.bg)
	}
	if op.Line {
		// run only once or will paint over runes
		if op.Offset == c.d.st.runeR.ri {
			st.lineBg = c.d.st.curColors.bg
		}
	}
}

//----------

type ColorizeGroup struct {
	Off bool
	Ops []*ColorizeOp
}

type ColorizeOp struct {
	Offset    int
	Fg, Bg    color.Color
	ProcColor func(fg, bg color.Color) (fg2, bg2 color.Color)
	Line      bool
	SetNil    bool
}
