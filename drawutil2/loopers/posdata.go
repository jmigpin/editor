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
	Jump    int
	data    []*PosDataData
}

func MakePosData() PosData {
	return PosData{Jump: 250}
}
func (pdl *PosData) Setup(strl *String, keepers []PosDataKeeper) {
	pdl.strl = strl
	pdl.keepers = keepers
}
func (pdl *PosData) Loop(fn func() bool) {
	// keep values of first iteration, if string empty it's ok to not keep anything
	count := 0
	pdl.OuterLooper().Loop(func() bool {
		if pdl.strl.RiClone {
			return fn()
		}
		if count%pdl.Jump == 0 {
			pdl.keep()
		}
		count++
		return fn()
	})
}

func (pdl *PosData) keep() {
	kd := make([]interface{}, len(pdl.keepers))
	for i, k := range pdl.keepers {
		kd[i] = k.KeepPosData()
	}
	pd := &PosDataData{
		ri:            pdl.strl.Ri,
		penBoundsMaxY: pdl.strl.LineY1(),
		keepersData:   kd,
	}
	pdl.data = append(pdl.data, pd)
}
func (pdl *PosData) restore(pd *PosDataData) {
	for i, kd := range pd.keepersData {
		pdl.keepers[i].RestorePosData(kd)
	}
}

func (pdl *PosData) RestorePosDataCloseToIndex(index int) {
	pd, ok := pdl.PosDataCloseToIndex(index)
	if ok {
		pdl.restore(pd)
	}
}
func (pdl *PosData) RestorePosDataCloseToPoint(p *fixed.Point26_6) {
	pd, ok := pdl.PosDataCloseToPoint(p)
	if ok {
		pdl.restore(pd)
	}
}
func (pdl *PosData) PosDataCloseToIndex(index int) (*PosDataData, bool) {
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
func (pdl *PosData) PosDataCloseToPoint(p *fixed.Point26_6) (*PosDataData, bool) {
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

func (pdl *PosData) GetPoint(index int) *fixed.Point26_6 {
	strl := pdl.strl
	pdl.OuterLooper().Loop(func() bool {
		if strl.RiClone {
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
func (pdl *PosData) GetIndex(p *fixed.Point26_6) int {
	strl := pdl.strl

	found := false
	foundLine := false
	lineRuneIndex := 0

	pdl.OuterLooper().Loop(func() bool {
		if strl.RiClone {
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
		// clicking beyond the last rune should return len(str), but if a foundLine was triggered, it will return the lineRuneIndex with the last rune. Which is ok if it is a newline.
		if strl.Ri == len(strl.Str) && strl.PrevRu != '\n' {
			return len(strl.Str)
		}

		return lineRuneIndex
	}
	return len(strl.Str)
}

func (pdl *PosData) Update() {
	for _, pd := range pdl.data {
		pdl.restore(pd)
		for _, k := range pdl.keepers {
			k.UpdatePosData()
		}
		pdl.keep()
	}
}

type PosDataKeeper interface {
	KeepPosData() interface{}
	RestorePosData(interface{})
	UpdatePosData()
}

type PosDataData struct {
	ri            int
	penBoundsMaxY fixed.Int26_6
	keepersData   []interface{}
}
