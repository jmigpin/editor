package drawutil

import (
	"image/color"
	"sync"
)

var WrapLineRune = rune('←') // positioned at the start of wrapped line (left)
var WrapLineIndentTabs = 0.0
var WrapWordLimit = 0
var CursorHalfHit = false

type TextDrawerOptions struct {
	LineWrap struct {
		On     bool
		Fg, Bg color.Color
	}
	Cursor struct {
		On         bool
		Fg         color.Color
		AddedWidth int
	}
	IndexOf struct {
		HalfHit bool
	}
	Colorize struct {
		Groups []*ColorizeGroup
	}
	Decorations struct {
		Groups []*DecorationGroup
	}
	Annotations struct {
		On       bool
		Fg, Bg   color.Color
		Selected struct {
			EntryIndex int
			Fg, Bg     color.Color
		}
		Entries *AnnotationGroup // must be ordered by offset
	}
	WordHighlight struct {
		On     bool
		Fg, Bg color.Color
		Group  ColorizeGroup
	}
	ParenthesisHighlight struct {
		On     bool
		Fg, Bg color.Color
		Group  ColorizeGroup
	}
	ContentColorize struct {
		Git struct {
			On       bool
			AddFg    color.Color
			DeleteFg color.Color
		}
		Group ColorizeGroup
	}
	SyntaxHighlight struct {
		On      bool
		Comment struct {
			SCs    []*SyntaxComment
			Fg, Bg color.Color
		}
		String struct {
			Fg, Bg color.Color
		}
		Group ColorizeGroup
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

//----------

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

//----------

type AnnotationGroup struct {
	sync.RWMutex
	Anns []*Annotation
}

func NewAnnotationGroup(n int) *AnnotationGroup {
	ag := &AnnotationGroup{}
	ag.Anns = make([]*Annotation, n)
	// Allocate contiguous memory.
	w := make([]Annotation, n)
	for i := range w {
		ag.Anns[i] = &w[i]
	}
	return ag
}

func (ag *AnnotationGroup) On() bool {
	return ag != nil && len(ag.Anns) > 0
}

type Annotation struct {
	Offset     int
	Bytes      []byte
	NotesBytes []byte // used for arrival index
}
