package loopers

import (
	"sort"

	"golang.org/x/image/math/fixed"
)

// Cached Position Data looper with getpoint/getindex.
type PosDataLooper struct {
	EmbedLooper
	data []*PosData
	pdk  PosDataKeeper
	strl *StringLooper
}

func NewPosDataLooper(pdk PosDataKeeper, strl *StringLooper) *PosDataLooper {
	return &PosDataLooper{pdk: pdk, strl: strl}
}
func (pdl *PosDataLooper) Loop(fn func() bool) {
	jump := 250 // experimental value
	pdl.data = []*PosData{}

	// keep values of first iteration, if string empty it's ok to not keep anything
	count := 0
	pdl.OuterLooper().Loop(func() bool {
		if count%jump == 0 {
			pdl.keep()
		}
		count++
		return fn()
	})
}

func (pdl *PosDataLooper) keep() {
	data := pdl.pdk.KeepPosData()
	pd := &PosData{
		ri:            pdl.strl.Ri,
		pen:           pdl.strl.Pen,
		penBoundsMaxY: pdl.strl.PenBounds().Max.Y,
		data:          data,
	}
	pdl.data = append(pdl.data, pd)
}
func (pdl *PosDataLooper) restore(pd *PosData) {
	pdl.strl.Ri = pd.ri
	pdl.strl.Pen = pd.pen
	pdl.pdk.RestorePosData(pd.data)
}

func (pdl *PosDataLooper) RestorePosDataCloseToIndex(index int) {
	pd, ok := pdl.PosDataCloseToIndex(index)
	if ok {
		pdl.restore(pd)
	}
}
func (pdl *PosDataLooper) RestorePosDataCloseToPoint(p *fixed.Point26_6) {
	pd, ok := pdl.PosDataCloseToPoint(p)
	if ok {
		pdl.restore(pd)
		//pdl.restore(pdl.data[0])
	}
}
func (pdl *PosDataLooper) PosDataCloseToIndex(index int) (*PosData, bool) {
	n := len(pdl.data)
	if n == 0 {
		return nil, false
	}
	j := sort.Search(n, func(i int) bool {
		return pdl.data[i].ri > index
	})
	// get previous entry before p
	if j > 0 {
		j--
	}
	return pdl.data[j], true
}
func (pdl *PosDataLooper) PosDataCloseToPoint(p *fixed.Point26_6) (*PosData, bool) {
	n := len(pdl.data)
	if n == 0 {
		return nil, false
	}
	j := sort.Search(n, func(i int) bool {
		// has to be in a previous y or it won't draw
		// all runes in a previous x position of the kept data
		by := pdl.data[i].penBoundsMaxY
		return by > p.Y
	})
	// get previous entry before p
	if j > 0 {
		j--
	}
	return pdl.data[j], true
}

type PosDataKeeper interface {
	KeepPosData() interface{}
	RestorePosData(interface{})
}

type PosData struct {
	ri            int
	pen           fixed.Point26_6
	penBoundsMaxY fixed.Int26_6 // upper left corner of pen
	data          interface{}
}

func PosDataGetPoint(index int, strl *StringLooper, looper Looper) *fixed.Point26_6 {
	looper.Loop(func() bool {
		if strl.Ri >= index {
			return false
		}
		return true
	})
	pb := strl.PenBounds()
	return &pb.Min
}

func PosDataGetIndex(p *fixed.Point26_6, strl *StringLooper, looper Looper) int {
	found := false
	foundLine := false
	lineRuneIndex := 0

	looper.Loop(func() bool {
		pb := strl.PenBounds()

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
			if p.X < pb.Max.X {
				found = true
				return false
			}
			// after last rune of the line
			foundLine = true
			lineRuneIndex = strl.Ri
		}
		return true
	})
	if found {
		return strl.Ri
	}
	if foundLine {
		// position at end of string if last line
		if strl.Ri == len(strl.Str) {
			return len(strl.Str)
		}

		return lineRuneIndex
	}
	return len(strl.Str)
}
