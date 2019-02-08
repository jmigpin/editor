package drawer4

import "image/color"

type Colorize struct {
	d *Drawer
}

func (c *Colorize) Init() {
	c.d.st.colorize.indexes = make([]int, len(c.d.Opt.Colorize.Groups))
}

func (c *Colorize) Iter() {
	c.colorize()
	_ = c.d.iterNext()
}

func (c *Colorize) End() {}

//----------

func (c *Colorize) colorize() {
	ri := c.d.st.runeR.ri
	for k, g := range c.d.Opt.Colorize.Groups {
		if g == nil || g.Off {
			continue
		}
		i := &c.d.st.colorize.indexes[k]
		var op *ColorizeOp
		for ; *i < len(g.Ops); *i++ {
			op2 := g.Ops[*i]
			if ri >= op2.Offset {
				op = op2
			} else if ri < op2.Offset {
				// next offset passes the ri, went too far
				if *i > 0 {
					*i--
				}
				break
			}
		}
		if op != nil {
			// colorize
			if op.Fg != nil {
				c.d.st.curColors.fg = op.Fg
			}
			if op.Bg != nil {
				c.d.st.curColors.bg = op.Bg
			}
			if op.ProcColor != nil {
				st := &c.d.st.curColors
				st.fg, st.bg = op.ProcColor(st.fg, st.bg)
			}
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
}
