package posindex

import (
	"sort"

	"github.com/jmigpin/editor/drawutil2/loopers"

	"golang.org/x/image/math/fixed"
)

// Position Data Index.
type PosDataIndex struct {
	PosLooper PosLooper
	data      []*PosData
}

func NewPosDataIndex(PosLooper PosLooper) *PosDataIndex {
	return &PosDataIndex{PosLooper: posLooper}
}
func (pdi *PosDataIndex) Calc() {
	jump := 250 // experimental value
	pdi.data = []*PosData{}
	count := 0 // count at zero keeps values of first iteration
	pdi.PosLooper.Loop(func() bool {
		if count%jump == 0 {
			// keep
			pd := pdi.posLooper.KeepPosData()
			pdi.data = append(pdi.data, pd)
		}
		return true
	})
}
func (pdi *PosDataIndex) RestorePosDataCloseToIndex(index int) {
	pd, ok := pdi.PosDataCloseToIndex(index)
	if ok {
		pdi.posLooper.RestorePosData(pd)
	}
}
func (pdi *PosDataIndex) RestorePosDataCloseToPoint(p *fixed.Point26_6) {
	pd, ok := pdi.PosDataCloseToPoint(p)
	if ok {
		pdi.posLooper.RestorePosData(pd)
	}
}
func (pdi *PosDataIndex) PosDataCloseToIndex(index int) (*PosData, bool) {
	n := len(pdi.data)
	if n == 0 {
		return nil, false
	}
	j := sort.Search(n, func(i int) bool {
		return pdi.data[i].Index >= index
	})
	if j == n {
		// not found, use last
		return pdi.data[n-1], true
	}
	return pdi.data[j], true
}
func (pdi *PosDataIndex) PosDataCloseToPoint(p *fixed.Point26_6) (*PosData, bool) {
	n := len(pdi.data)
	if n == 0 {
		return nil, false
	}
	j := sort.Search(n, func(i int) bool {
		m := pdi.data[i].PenBoundsMin
		return p.X >= m.X && p.Y >= m.Y
	})
	if j == n {
		// not found, use last
		return pdi.data[n-1], true
	}
	return pdi.data[j], true
}

type PosLooper interface {
	loopers.Looper
	KeepPosData() *PosData
	RestorePosData(*PosData)
}

type PosData struct {
	Index        int
	PenBoundsMin fixed.Point26_6 // upper left corner of pen
	Data         interface{}
}
