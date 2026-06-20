package drawer4

import "github.com/jmigpin/editor/util/drawutil"

type Decorations struct {
	d *Drawer
}

func (dc *Decorations) Init() {
	dc.d.st.decorations.indexes = make([]int, len(dc.d.Opt.Decorations.Groups))
}

func (dc *Decorations) Iter() {
	if !dc.d.iterNext() {
		return
	}
}

func (dc *Decorations) End() {}

func (dc *Decorations) isValidLineStartOffset(offset int) bool {
	if offset == 0 {
		return true
	}
	if offset < 0 {
		return false
	}
	b, err := dc.d.Reader().ReadFastAt(offset-1, 1)
	if err != nil || len(b) == 0 {
		return false
	}
	return b[0] == '\n'
}

type DecorationGroup = drawutil.DecorationGroup
type Decoration = drawutil.Decoration
type DecorationKind = drawutil.DecorationKind

const DecorationHorizontalRule = drawutil.DecorationHorizontalRule
