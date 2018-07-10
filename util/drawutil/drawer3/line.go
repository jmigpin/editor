package drawer3

type Line struct {
	EExt
}

func (l *Line) Iterate(r *ExtRunner) {
	if r.RR.RiClone() {
		r.NextExt()
		return
	}
	if !r.NextExt() {
		return
	}
	if r.RR.Ru == '\n' {
		l.NewLine(r)
	}
}

func (l *Line) NewLine(r *ExtRunner) {
	r.RR.Pen.X = 0
	r.RR.Pen.Y += r.RR.LineHeight
	r.RR.PrevRu = 0
	r.RR.Advance = 0
}
