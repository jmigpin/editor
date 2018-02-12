package loopers

import (
	"image/color"

	"golang.org/x/image/math/fixed"
)

type Annotations struct {
	EmbedLooper
	strl  *String
	linel *Line
	opt   *AnnotationsOpt

	data AnnotationsData

	// only valid on state AnnStateOn
	entryIndex       int
	entryIndexOffset int
}

func MakeAnnotations(strl *String, linel *Line, opt *AnnotationsOpt) Annotations {
	ann := Annotations{strl: strl, linel: linel, opt: opt}
	return ann
}
func (lpr *Annotations) Loop(fn func() bool) {
	entries := lpr.opt.OrderedEntries

	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.IsRiClone() {
			return fn()
		}

		// indexes to add
		for ; lpr.data.index < len(entries); lpr.data.index++ {
			e := entries[lpr.data.index]
			if e == nil {
				continue
			}
			if lpr.strl.Ri < e.Offset {
				break
			}
			lpr.data.indexesToAdd = append(lpr.data.indexesToAdd, lpr.data.index)
		}

		// add annotations after newline
		if (lpr.strl.Ru == '\n' || lpr.strl.Ru == 0) && len(lpr.data.indexesToAdd) > 0 {
			// keep to restore later
			origRu := lpr.strl.Ru
			origPrevRu := lpr.strl.PrevRu
			origPen := lpr.strl.Pen
			origAdvance := lpr.strl.Advance

			// set clone flag to avoid selections and state keepers
			lpr.strl.PushRiClone()

			// space separator
			for _, ru := range "\t" {
				lpr.strl.Ru = ru
				if !lpr.strl.Iterate(fn) {
					return false
				}
			}

			// annotations
			//lpr.data.state = AnnStateOn
			lpr.entryIndex = -1 // reset
			for i, index := range lpr.data.indexesToAdd {
				if i > 0 {
					// space separator between entries on the same line
					for _, ru := range " " {
						lpr.strl.Ru = ru
						if !lpr.strl.Iterate(fn) {
							return false
						}
					}
				}

				// iterate annotation runes
				// ensure valid index incase entries have changed
				if index < len(entries) {
					// allow other loopers to know which entry index is being iterated
					lpr.entryIndex = index

					lpr.data.state = AnnStateOn
					for i, ru := range entries[index].Str {
						// allow other loopers to know which rune index is being iterated
						lpr.entryIndexOffset = i

						lpr.strl.Ru = ru
						if !lpr.strl.Iterate(fn) {
							return false
						}
					}
					lpr.data.state = AnnStateNormal
				}
			}
			//lpr.data.state = AnnStateNormal

			// clear clone flag
			lpr.strl.PopRiClone()

			// clear
			lpr.data.indexesToAdd = []int{}

			// restore and continue
			lpr.strl.Ru = origRu
			lpr.strl.PrevRu = origPrevRu
			lpr.strl.Pen = origPen
			lpr.strl.Advance = origAdvance
		}

		return fn()
	})
}

// Implements PosDataKeeper
func (lpr *Annotations) KeepPosData() interface{} {
	// copy
	u := lpr.data
	// copy slice
	u.indexesToAdd = make([]int, len(lpr.data.indexesToAdd))
	copy(u.indexesToAdd, lpr.data.indexesToAdd)
	return u
}

// Implements PosDataKeeper
func (lpr *Annotations) RestorePosData(data interface{}) {
	lpr.data = data.(AnnotationsData)
}

type AnnState int

const (
	AnnStateNormal AnnState = iota
	AnnStateOn
)

type AnnotationsData struct {
	state        AnnState
	index        int // current entry
	indexesToAdd []int
}

type AnnotationsOpt struct {
	Fg             color.Color
	Bg             color.Color
	OrderedEntries []*AnnotationsEntry // ordered by offset
}

type AnnotationsEntry struct {
	Offset int
	Str    string
}

type AnnotationsColor struct {
	EmbedLooper
	ann  *Annotations
	strl *String
	dl   *Draw
	bgl  *Bg
	opt  *AnnotationsOpt
}

func MakeAnnotationsColor(ann *Annotations, strl *String, dl *Draw, bgl *Bg, opt *AnnotationsOpt) AnnotationsColor {
	return AnnotationsColor{ann: ann, strl: strl, dl: dl, bgl: bgl, opt: opt}
}
func (lpr *AnnotationsColor) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.ann.data.state {
		case AnnStateOn:
			if lpr.opt.Fg != nil {
				lpr.dl.Fg = lpr.opt.Fg
			}
			if lpr.opt.Bg != nil {
				lpr.bgl.Bg = lpr.opt.Bg
			}
		}
		return fn()
	})
}

func GetAnnotationsIndex(lpr Looper, al *Annotations, p *fixed.Point26_6) (int, int, bool) {
	strl := al.strl
	index := -1
	ioffset := -1
	lpr.OuterLooper().Loop(func() bool {
		if al.data.state == AnnStateOn {
			pb := strl.PenBounds()
			if p.In(*pb) {
				index = al.entryIndex
				ioffset = al.entryIndexOffset
				return false
			}
		}
		return true
	})
	return index, ioffset, index >= 0
}
