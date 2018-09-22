package drawer3

import "image/color"

type Segments struct {
	EExt
	Opt SegmentsOpt
	cc  *CurColors
	d   Drawer // needed by SetOn

	// setup values
	sgi []int
}

func Segments1(cc *CurColors, d Drawer) Segments {
	return Segments{cc: cc, d: d}
}

func (s *Segments) SetOn(v bool) {
	if v != s.EExt.On() {
		s.d.SetNeedMeasure(true)
	}
	if !v {
		// clear data
		// commented: need to disable segments where groups are being set
		//s.Opt = SegmentsOpt{}
	}
	s.EExt.SetOn(v)
}

func (s *Segments) Start(r *ExtRunner) {
	s.sgi = make([]int, len(s.Opt.Groups))
}

func (s *Segments) Iterate(r *ExtRunner) {
	if r.RR.RiClone() {
		r.NextExt()
		return
	}

	ri := r.RR.Ri

	for k, sg := range s.Opt.Groups {
		if !sg.On {
			continue
		}
		i := &s.sgi[k]
		for ; *i < len(sg.Segs); (*i)++ {
			seg := sg.Segs[*i]
			if ri < seg.Pos {
				// have not reached this segment yet
				break
			} else if ri < seg.End {
				// colorize
				if sg.Fg != nil {
					s.cc.Fg = sg.Fg
				}
				if sg.Bg != nil {
					s.cc.Bg = sg.Bg
				}
				if sg.ProcColor != nil {
					s.cc.Fg, s.cc.Bg = sg.ProcColor(s.cc.Fg, s.cc.Bg)
				}
				break
			}
		}
	}

	r.NextExt()
}

//----------

type SegmentsOpt struct {
	Groups []*SegGroup
}

func (sopt *SegmentsOpt) SetupNGroups(n int) {
	sopt.Groups = make([]*SegGroup, n)
	for k := range sopt.Groups {
		sopt.Groups[k] = &SegGroup{}
	}
}

type SegGroup struct {
	On        bool
	Segs      []*Segment // assumed to be ordered by Pos
	Fg, Bg    color.Color
	ProcColor func(fg, bg color.Color) (fg2, bg2 color.Color) // optional
}

type Segment struct {
	Pos, End int
}
