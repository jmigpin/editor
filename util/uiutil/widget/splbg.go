package widget

// Wraps a StartPercentLayout with a node that fills the first space background.
type SplBg struct {
	ENode
	Spl *StartPercentLayout
	Bg  Node
}

func NewSplBg(bg Node) *SplBg {
	l := &SplBg{
		Spl: NewStartPercentLayout(),
		Bg:  bg,
	}
	l.Append(l.Bg, l.Spl)
	return l
}

func (l *SplBg) OnChildMarked(child Node, newMarks Marks) {
	if child == l.Spl {
		// need layout to adjust the bg node
		if newMarks.HasAny(MarkNeedsLayout) {
			l.MarkNeedsLayout()
		}
	}
}

func (l *SplBg) Layout() {
	if l.Spl.ChildsLen() > 0 {
		// layout SPL first to calc "Bg" size based on SPL first child
		l.Spl.Layout()

		// redimension "Bg" to match first row start
		min := &l.Spl.FirstChild().Embed().Bounds.Min
		max := &l.Bg.Embed().Bounds.Max
		if l.Spl.YAxis {
			max.Y = min.Y
		} else {
			max.X = min.X
		}
	}
}
