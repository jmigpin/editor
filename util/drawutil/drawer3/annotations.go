package drawer3

import (
	"image/color"
	"sync"

	"github.com/jmigpin/editor/util/mathutil"
)

type Annotations struct {
	EExt
	Opt AnnotationsOpt

	data              AnnotationsData // positional data
	state             AnnState
	stateOnEntryIndex int
	entryOffset       int
}

func Annotations1() Annotations {
	return Annotations{}
}

func (ann *Annotations) Start(r *ExtRunner) {
	ann.data = AnnotationsData{}
	ann.Opt.EntriesMu.RLock()
}

func (ann *Annotations) Iterate(r *ExtRunner) {
	if r.RR.RiClone() {
		r.NextExt()
		return
	}

	// indexes to add
	// NOTE: must be true: i<j => entry[i].Offset <= entry[j].Offset
	entries := ann.Opt.Entries
	for ; ann.data.index < len(entries); ann.data.index++ {
		e := entries[ann.data.index]
		if e == nil {
			continue
		}
		if r.RR.Ri < e.Offset {
			break
		}
		ann.data.indexesToAdd = append(ann.data.indexesToAdd, ann.data.index)
	}

	// add annotations after newline
	if (r.RR.Ru == '\n' || r.RR.Ru == 0) && len(ann.data.indexesToAdd) > 0 {
		ann.insertAnnotations(r)
	}

	r.NextExt()
}

func (ann *Annotations) End(r *ExtRunner) {
	ann.Opt.EntriesMu.RUnlock()
}

//----------

func (ann *Annotations) insertAnnotations(r *ExtRunner) {
	// clear
	defer func() { ann.data.indexesToAdd = []int{} }()

	// keep to restore later
	origRR := *r.RR
	// restore and continue
	defer func() { *r.RR = origRR }()

	// set clone flag to avoid selections and state keepers
	r.RR.PushRiClone()
	defer r.RR.PopRiClone()

	// helper functions to iterate annotations runes
	iterRuneClone := func(ru rune) bool {
		return r.RR.Iterate2(r, ru, len(string(ru)))
	}
	iterString := func(s string) bool {
		for i, ru := range s {
			ann.entryOffset = i
			if !iterRuneClone(ru) {
				return false
			}
		}
		return true
	}
	iterAnnotationString := func(index int, s string) bool {
		ann.state = AnnStateOn
		ann.stateOnEntryIndex = index
		ann.entryOffset = 0
		defer func() { ann.state = AnnStateNormal }()
		return iterString(s)
	}

	// separator between content and annotation
	{
		// fixed minimum or a space string
		min := r.RR.GlyphAdvance('\t') * 7
		if r.RR.Pen.X < min {
			r.RR.Pen.X = min
		}
		if !iterString("   \t") {
			return
		}
	}

	// annotations
	for i, index := range ann.data.indexesToAdd {
		// space separator between entries on the same line
		if i > 0 {
			if !iterString(" ") {
				return
			}
		}

		entry := ann.Opt.Entries[index]
		if entry == nil {
			continue
		}

		// iterate annotation runes
		if !iterAnnotationString(index, string(entry.Bytes)) {
			return
		}
	}
}

//----------

// Implements PosDataKeeper
func (ann *Annotations) KeepPosData() interface{} {
	// copy
	u := ann.data
	// copy slice
	u.indexesToAdd = make([]int, len(ann.data.indexesToAdd))
	copy(u.indexesToAdd, ann.data.indexesToAdd)
	return &u
}

// Implements PosDataKeeper
func (ann *Annotations) RestorePosData(data interface{}) {
	ann.data = *(data.(*AnnotationsData))
}

//----------

type AnnState int

const (
	AnnStateNormal AnnState = iota
	AnnStateOn
)

//----------

type AnnotationsData struct {
	index        int // entry being tested to add to indexesToAdd
	indexesToAdd []int
}

//----------

type AnnotationsOpt struct {
	Entries   []*Annotation // ordered
	EntriesMu sync.RWMutex  // allow external processes to lock for update

	Fg, Bg color.Color

	Select struct {
		Line   int
		Fg, Bg color.Color
	}
}

//----------

type Annotation struct {
	Offset int
	Bytes  []byte
}

//----------

type AnnotationsColor struct {
	EExt
	ann *Annotations
	cc  *CurColors
}

func AnnotationsColor1(ann *Annotations, cc *CurColors) AnnotationsColor {
	return AnnotationsColor{ann: ann, cc: cc}
}

func (annc *AnnotationsColor) Iterate(r *ExtRunner) {
	if !annc.ann.On() {
		r.NextExt()
		return
	}

	switch annc.ann.state {
	case AnnStateOn:
		if annc.ann.Opt.Fg != nil {
			annc.cc.Fg = annc.ann.Opt.Fg
		}
		if annc.ann.Opt.Bg != nil {
			annc.cc.Bg = annc.ann.Opt.Bg
		}
		if annc.ann.stateOnEntryIndex == annc.ann.Opt.Select.Line {
			if annc.ann.Opt.Select.Fg != nil {
				annc.cc.Fg = annc.ann.Opt.Select.Fg
			}
			if annc.ann.Opt.Select.Bg != nil {
				annc.cc.Bg = annc.ann.Opt.Select.Bg
			}
		}
	}
	r.NextExt()
}

//----------

type AnnotationsIndexOf struct {
	EExt
	entryIndex  int // result
	entryOffset int // result
	point       mathutil.PointIntf
	ann         *Annotations
}

func MakeAnnotationsIndexOf(ann *Annotations, p mathutil.PointIntf) AnnotationsIndexOf {
	return AnnotationsIndexOf{ann: ann, point: p}
}

func (aiof *AnnotationsIndexOf) Start(r *ExtRunner) {
	aiof.entryIndex = -1
}

func (aiof *AnnotationsIndexOf) Iterate(r *ExtRunner) {

	// must be inside state "on"
	if aiof.ann.state != AnnStateOn {
		r.NextExt()
		return
	}

	p := &aiof.point
	pb := r.RR.PenBounds()

	// before the start or already passed the line
	if p.Y < pb.Min.Y {
		r.Stop()
		return
	}

	// in the pen bounds
	if p.In(pb) {
		aiof.entryIndex = aiof.ann.stateOnEntryIndex
		aiof.entryOffset = aiof.ann.entryOffset
		r.Stop()
		return
	}

	r.NextExt()
}
