package drawer4

import (
	"image/color"
)

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

type DecorationGroup struct {
	Off     bool
	Entries []*Decoration
}

type Decoration struct {
	Offset    int
	Kind      DecorationKind
	Fg        color.Color
	Thickness int
}

type DecorationKind uint8

const (
	DecorationHorizontalRule DecorationKind = iota + 1
)
