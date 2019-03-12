package drawer4

type Annotations struct {
	d *Drawer
}

func (ann *Annotations) Init() {}

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
	entries := ann.d.Opt.Annotations.Entries // ordered by offset
	i := &ann.d.st.annotations.cei
	q := &ann.d.st.annotations.indexQ
	// keep track of annotations to be added
	for ; *i < len(entries); *i++ {
		e := entries[*i]
		if e == nil {
			continue
		}
		// already passed the annotation
		if ann.d.st.runeR.ri > e.Offset {
			continue
		}
		// next annotation is far away
		if ann.d.st.runeR.ri < e.Offset {
			break
		}
		*q = append(*q, *i)
	}
	// add annotations after newline
	if len(*q) > 0 {
		switch ann.d.st.runeR.ru {
		case '\n', 0: // insert annotations at newline or EOF
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

		if !ann.d.iters.runeR.insertExtraString("   \t") {
			return
		}

		space := ann.d.iters.runeR.glyphAdvance(' ')
		min := space * (8 * 10)
		margin := space * 5
		maxX := ann.d.iters.runeR.maxX()
		if pen.X < min {
			pen.X = min
		}
		if pen.X > maxX-margin {
			pen.X = maxX - margin
		}
		if pen.X < startX {
			pen.X = startX
		}
	}

	// annotations
	for i, index := range ann.d.st.annotations.indexQ {
		// space separator between entries on the same line
		if i > 0 {
			if !ann.d.iters.runeR.insertExtraString(" ") {
				return
			}
		}

		entry := ann.d.Opt.Annotations.Entries[index]
		if entry == nil {
			continue
		}

		if !ann.insertAnnotationString(string(entry.Bytes), index) {
			return
		}
	}
}

func (ann *Annotations) insertAnnotationString(s string, eindex int) bool {
	// keep color state
	keep := ann.d.st.curColors
	defer func() { ann.d.st.curColors = keep }()
	// set colors
	opt := &ann.d.Opt.Annotations
	if eindex == opt.Selected.EntryIndex {
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

//----------

type Annotation struct {
	Offset int
	Bytes  []byte
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
