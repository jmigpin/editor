package loopers

import (
	"sort"

	"golang.org/x/image/math/fixed"
)

// Cached Position Data looper with getpoint/getindex.
type PosData struct {
	EmbedLooper
	strl    *String
	keepers []PosDataKeeper
	Data    []*PosDataData
	jump    int
}

func MakePosData(strl *String, keepers []PosDataKeeper, jump int, prevData []*PosDataData) PosData {
	return PosData{strl: strl, keepers: keepers, Data: prevData, jump: jump}
}

func (lpr *PosData) Loop(fn func() bool) {
	// keep values of first iteration, if string empty it's ok to keep nothing
	count := 0
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.IsRiClone() {
			return fn()
		}
		if count%lpr.jump == 0 {
			lpr.keep()
		}
		count++
		return fn()
	})
}

func (lpr *PosData) keep() {
	kd := make([]interface{}, len(lpr.keepers))
	for i, k := range lpr.keepers {
		if k != nil {
			kd[i] = k.KeepPosData()
		}
	}
	pd := &PosDataData{
		ri:            lpr.strl.Ri,
		penBoundsMaxY: lpr.strl.LineY1(),
		keepersData:   kd,
	}
	lpr.Data = append(lpr.Data, pd)
}
func (lpr *PosData) restore(pd *PosDataData) {
	for i, kd := range pd.keepersData {
		if lpr.keepers[i] != nil && kd != nil {
			lpr.keepers[i].RestorePosData(kd)
		}
	}
}

func (lpr *PosData) RestorePosDataCloseToIndex(index int) {
	pd, ok := lpr.PosDataCloseToIndex(index)
	if ok {
		lpr.restore(pd)
	}
}
func (lpr *PosData) RestorePosDataCloseToPoint(p *fixed.Point26_6) {
	pd, ok := lpr.PosDataCloseToPoint(p)
	if ok {
		lpr.restore(pd)
	}
}

func (lpr *PosData) PosDataCloseToIndex(index int) (*PosDataData, bool) {
	n := len(lpr.Data)
	if n == 0 {
		return nil, false
	}
	j := sort.Search(n, func(i int) bool {
		return lpr.Data[i].ri > index
	})
	// get previous entry before p
	if j > 0 {
		j--
	}
	return lpr.Data[j], true
}
func (lpr *PosData) PosDataCloseToPoint(p *fixed.Point26_6) (*PosDataData, bool) {
	n := len(lpr.Data)
	if n == 0 {
		return nil, false
	}
	j := sort.Search(n, func(i int) bool {
		// has to be in a previous y or it won't draw all runes in a previous x position of the kept data
		by := lpr.Data[i].penBoundsMaxY
		return by > p.Y
	})
	// get previous entry before p
	if j > 0 {
		j--
	}
	return lpr.Data[j], true
}

func (lpr *PosData) GetPoint(index int) *fixed.Point26_6 {
	strl := lpr.strl
	lpr.OuterLooper().Loop(func() bool {
		if strl.IsRiClone() {
			return true
		}
		if strl.Ri >= index {
			return false
		}
		return true
	})
	pb := strl.PenBounds()
	return &pb.Min
}
func (lpr *PosData) GetIndex(p *fixed.Point26_6) int {
	strl := lpr.strl

	found := false
	foundLine := false
	lineRuneIndex := 0

	lpr.OuterLooper().Loop(func() bool {
		if strl.IsRiClone() {
			return true
		}

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

			//// in first half of a rune
			//half := (pb.Max.X - pb.Min.X) / 2
			//if p.X < pb.Max.X-half {
			//	found = true
			//	return false
			//}

			// in the rune
			if p.X < pb.Max.X {
				found = true
				return false
			}

			// after this rune - keep to have last rune of the line
			foundLine = true
			lineRuneIndex = strl.Ri
		}
		return true
	})

	//log.Printf("**p %v pen %v", p, pdl.strl.Pen)

	if found {
		return strl.Ri
	}
	if foundLine {
		return lineRuneIndex
	}
	return len(strl.Str)
}

type PosDataKeeper interface {
	KeepPosData() interface{}
	RestorePosData(interface{})
}

type PosDataData struct {
	ri            int
	penBoundsMaxY fixed.Int26_6
	keepersData   []interface{}
}
