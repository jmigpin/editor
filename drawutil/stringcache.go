package drawutil

import (
	"fmt"
	"image"
	"image/draw"
	"sort"

	"golang.org/x/image/math/fixed"
)

// Keeps info data every x runes for faster jump to the state of the text.
type StringCache struct {
	Face   *Face
	str    string
	width  int
	height fixed.Int26_6
	rdata  []*SCRuneData
}

type SCRuneData struct {
	liner struct {
		iter struct {
			ri  int // rune index
			pen fixed.Point26_6
		}
		wrapIndent StringLinerWrapIndent
		states     StringLinerStates
	}
}

func NewStringCache(face *Face) *StringCache {
	sc := &StringCache{Face: face}
	sc.calcRuneData()
	return sc
}
func (sc *StringCache) Update(str string, width int) {
	if sc.str == str && sc.width == width {
		return
	}
	sc.str = str
	sc.width = width
	sc.calcRuneData()
}
func (sc *StringCache) calcRuneData() {
	jump := 250 // keep data every x runes

	// can't allocate since it's unknown the number of runes in a string - using append instead
	sc.rdata = []*SCRuneData{}

	count := 0
	liner := NewStringLiner(sc.Face, sc.str, sc.maxPoint())

	keep := func() {
		var rd SCRuneData
		rd.liner.iter.ri = liner.iter.ri
		rd.liner.iter.pen = liner.iter.pen
		rd.liner.wrapIndent = liner.wrapIndent
		rd.liner.states = liner.states
		sc.rdata = append(sc.rdata, &rd)
	}

	// always keep starting point, even for empty text, to
	// keep initialized data from newstringliner
	keep()

	liner.Loop(func() bool {
		count++
		if count%jump == 0 {
			keep()
		}
		return true
	})

	// cache text height
	sc.height = LineY1(liner.iter.pen.Y, liner.iter.fm)
	if sc.str == "" {
		sc.height = 0
	}
}
func (sc *StringCache) maxPoint() *fixed.Point26_6 {
	p := fixed.P(sc.width, 1000000)
	return &p
}
func (sc *StringCache) restoreRuneData(rd *SCRuneData, liner *StringLiner) {
	liner.iter.ri = rd.liner.iter.ri
	liner.iter.pen = rd.liner.iter.pen
	liner.wrapIndent = rd.liner.wrapIndent
	liner.states = rd.liner.states
}
func (sc *StringCache) Height() fixed.Int26_6 {
	return sc.height
}
func (sc *StringCache) GetIndex(p *fixed.Point26_6) int {
	rd := sc.getRuneDataCloseToPoint(p)
	return sc.getIndexFromRuneData(rd, p)
}
func (sc *StringCache) GetPoint(index int) *fixed.Point26_6 {
	rd := sc.getRuneDataCloseToIndex(index)
	return sc.getPointFromRuneData(rd, index)
}
func (sc *StringCache) Draw(
	img draw.Image,
	bounds *image.Rectangle,
	cursorIndex int,
	offsetY fixed.Int26_6,
	colors *Colors,
	selection *Selection,
	highlight bool) error {

	if bounds.Dx() != sc.width {
		err := fmt.Errorf("bounds.dx doesn't match stringcache.width: %d, %d", bounds.Dx(), sc.width)
		return err
	}

	sdc := NewStringDrawColors(img, bounds, sc.Face, sc.str, colors)

	p := &fixed.Point26_6{0, offsetY}
	rd := sc.getRuneDataCloseToPoint(p)
	sc.restoreRuneData(rd, sdc.sd.liner)
	sdc.sd.liner.iter.pen.Y -= offsetY

	sdc.highlight = highlight
	sdc.selection = selection
	sdc.sd.cursorIndex = cursorIndex

	sdc.Loop()

	return nil
}

func (sc *StringCache) getRuneDataCloseToPoint(p *fixed.Point26_6) *SCRuneData {
	// binary search first entry after p
	fm := sc.Face.Face.Metrics()
	j := sort.Search(len(sc.rdata), func(i int) bool {
		pen0 := sc.rdata[i].liner.iter.pen
		ly1 := LineY1(pen0.Y, &fm)
		// rune data has to be in a previous y or it won't draw
		// all runes in a previous x position
		return ly1 > p.Y
	})
	// get previous entry, first before p
	if j > 0 {
		j--
	}
	return sc.rdata[j]
}
func (sc *StringCache) getRuneDataCloseToIndex(index int) *SCRuneData {
	// binary search first entry after index
	j := sort.Search(len(sc.rdata), func(i int) bool {
		return sc.rdata[i].liner.iter.ri > index
	})
	// get previous entry, first before index
	if j > 0 {
		j--
	}
	//println("get rune data close to index j", j, "len", len(sc.rdata))
	return sc.rdata[j]
}

func (sc *StringCache) getIndexFromRuneData(rd *SCRuneData, p *fixed.Point26_6) int {
	liner := NewStringLiner(sc.Face, sc.str, sc.maxPoint())
	sc.restoreRuneData(rd, liner)

	found := false
	foundLine := false
	lineRuneIndex := 0

	liner.Loop(func() bool {
		pb := liner.iter.PenBounds()

		// before the start or already passed the line
		if p.Y < pb.Min.Y {
			if !foundLine {
				// before the start, returns first index
				found = true
			}
			return false
		}
		// in the line
		if p.Y >= pb.Min.Y && p.Y < pb.Max.Y {
			// before the first rune of the line
			if p.X < pb.Min.X {
				found = true
				return false
			}
			// in a rune
			//if p.X < pb.Min.X+(pb.Max.X-pb.Min.X)/2 {
			if p.X < pb.Max.X {
				found = true
				return false
			}
			// after last rune of the line
			foundLine = true
			lineRuneIndex = liner.iter.ri
		}
		return true
	})
	if found {
		return liner.iter.ri
	}
	if foundLine {
		return lineRuneIndex
	}
	return len(sc.str)
}
func (sc *StringCache) getPointFromRuneData(rd *SCRuneData, index int) *fixed.Point26_6 {
	liner := NewStringLiner(sc.Face, sc.str, sc.maxPoint())
	sc.restoreRuneData(rd, liner)
	liner.Loop(func() bool {
		if liner.iter.ri >= index {
			return false
		}
		return true
	})
	ly0 := LineY0(liner.iter.pen.Y, liner.iter.fm)
	return &fixed.Point26_6{liner.iter.pen.X, ly0}
}
