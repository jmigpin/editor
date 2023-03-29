package drawer4

import (
	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/mathutil"
)

type Annotations struct {
	d          *Drawer
	notesFFace *fontutil.FontFace
}

func (ann *Annotations) Init() {
	size2 := ann.d.st.runeR.fface.Size * 0.70
	ann.notesFFace = ann.d.st.runeR.fface.Font.FontFace2(size2)
}

func (ann *Annotations) Iter() {
	if ann.d.Opt.Annotations.On {
		if ann.d.iters.runeR.isNormal() {
			ann.iter2()
		}
	}
	if !ann.d.iterNext() {
		return
	}
}

func (ann *Annotations) iter2() {
	entries := ann.d.Opt.Annotations.Entries // *mostly* ordered by offset
	i := &ann.d.st.annotations.cei
	q := &ann.d.st.annotations.indexQ
	// add annotations up to the first entry offset, only need to check next entries with offsets smaller then the first entry (ex: function literals with inner annotations that have higher entry index, but lower offsets).
	var first *Annotation
	for k := *i; k < len(entries); k++ {
		e := entries[k]
		if e == nil {
			continue
		}
		// past annotation
		if e.Offset < ann.d.st.runeR.ri {
			continue
		}

		if first == nil {
			first = e
		}

		// annotation match
		if e.Offset == ann.d.st.runeR.ri {
			*q = append(*q, k)
			if k == *i { // handled next entry
				*i++
				first = nil
				continue
			}
		}

		// future annotation
		// Commented: need to handle next entries with earlier offsets
		//if e.Offset > ann.d.st.runeR.ri {
		//break
		//}
		// future annotation after the first entry
		if e.Offset > first.Offset {
			break
		}
	}

	// add annotations after newline
	if len(*q) > 0 {
		switch ann.d.st.runeR.ru {
		case '\n', eofRune: // insert annotations at newline or EOF
			ann.insertAnnotations()
		}
	}
}

func (ann *Annotations) End() {}

//----------

func (ann *Annotations) insertAnnotations() {
	tmp := ann.d.st.runeR                   // keep state
	defer func() { ann.d.st.runeR = tmp }() // restore state
	ann.insertAnnotations2()
}

func (ann *Annotations) insertAnnotations2() {
	// clear at the end
	defer func() { ann.d.st.annotations.indexQ = []int{} }()

	// separator between content and annotation
	{
		pen := &ann.d.st.runeR.pen
		startX := pen.X + ann.d.st.runeR.advance

		//if !ann.insertSeparatorString("\t") {
		//	return
		//}

		space := ann.d.iters.runeR.glyphAdvance(' ')
		boundsMinX := mathutil.Intf1(ann.d.bounds.Min.X)
		min := boundsMinX + space*(8*10)
		margin := space * 10
		max := ann.d.iters.runeR.maxX() - margin
		if pen.X < min {
			pen.X = min
		}
		if pen.X > max {
			pen.X = max
		}
		if pen.X < startX {
			pen.X = startX
		}
	}

	// annotations
	c := 0
	for _, index := range ann.d.st.annotations.indexQ {
		entry := ann.d.Opt.Annotations.Entries[index]
		if entry == nil {
			continue
		}

		// space separator between entries on the same line
		c++
		if c >= 2 {
			if !ann.insertSeparatorString(" ") {
				return
			}
		}

		s1 := string(entry.Bytes)
		if !ann.insertAnnotationString(s1, index, true) {
			return
		}

		// entry.notes (used for arrival index)
		s2 := string(entry.NotesBytes)
		if !ann.insertNotesString(ann.notesFFace, s2) {
			return
		}
	}
}

func (ann *Annotations) insertAnnotationString(s string, eindex int, colorizeIfIndex bool) bool {
	// keep/restore color state
	keep := ann.d.st.curColors
	defer func() { ann.d.st.curColors = keep }()
	// set colors
	opt := &ann.d.Opt.Annotations
	if colorizeIfIndex && eindex == opt.Selected.EntryIndex {
		assignColor(&ann.d.st.curColors.fg, opt.Selected.Fg)
		assignColor(&ann.d.st.curColors.bg, opt.Selected.Bg)
	} else {
		assignColor(&ann.d.st.curColors.fg, opt.Fg)
		assignColor(&ann.d.st.curColors.bg, opt.Bg)
	}

	// update annotationsindexof state
	ann.d.st.annotationsIndexOf.inside.on = true
	ann.d.st.annotationsIndexOf.inside.ei = eindex
	ann.d.st.annotationsIndexOf.inside.soffset = ann.d.st.runeR.ri
	defer func() { ann.d.st.annotationsIndexOf.inside.on = false }()

	return ann.d.iters.runeR.insertExtraString(s)
}

func (ann *Annotations) insertNotesString(fface *fontutil.FontFace, s string) bool {
	// keep/restore color state
	keep := ann.d.st.curColors
	defer func() { ann.d.st.curColors = keep }()
	// set colors
	ann.d.st.curColors.fg = ann.d.fg
	ann.d.st.curColors.bg = nil

	// keep/restore face
	keepf := ann.d.st.runeR.fface
	ann.d.st.runeR.fface = fface
	defer func() { ann.d.st.runeR.fface = keepf }()

	return ann.d.iters.runeR.insertExtraString(" " + s)
}

func (ann *Annotations) insertSeparatorString(s string) bool {
	// keep/restore color state
	keep := ann.d.st.curColors
	defer func() { ann.d.st.curColors = keep }()
	// set colors
	ann.d.st.curColors.fg = ann.d.fg
	ann.d.st.curColors.bg = nil
	return ann.d.iters.runeR.insertExtraString(s)
}

//----------

type Annotation struct {
	Offset     int
	Bytes      []byte
	NotesBytes []byte // used for arrival index
}

//----------

type AnnotationsIndexOf struct {
	d *Drawer
}

func (aio *AnnotationsIndexOf) Init() {
	aio.d.st.annotationsIndexOf.eindex = -1
}

func (aio *AnnotationsIndexOf) Iter() {
	if aio.d.st.annotationsIndexOf.inside.on {
		aio.iter2()
	}
	_ = aio.d.iterNext()
}

func (aio *AnnotationsIndexOf) End() {}

//----------

func (aio *AnnotationsIndexOf) iter2() {
	p := &aio.d.st.annotationsIndexOf.p
	pb := aio.d.iters.runeR.penBounds()

	// before the y start
	if p.Y < pb.Min.Y {
		aio.d.iterStop()
		return
	}
	// in the line
	if p.Y < pb.Max.Y {
		// before the x start
		if p.X < pb.Min.X {
			aio.d.iterStop()
			return
		}
		// inside
		if p.X < pb.Max.X {
			st := &aio.d.st.annotationsIndexOf
			st.eindex = st.inside.ei
			st.offset = aio.d.st.runeR.ri - st.inside.soffset
			aio.d.iterStop()
			return
		}
	}
}
