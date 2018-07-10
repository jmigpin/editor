package drawer3

import (
	"sort"

	"github.com/jmigpin/editor/util/mathutil"
)

type PosData struct {
	EExt
	jump    int
	keepers []PosDataKeeper

	// start values
	count int
	data  []*PosDataData
}

func PosData1(jump int, keepers []PosDataKeeper) PosData {
	return PosData{jump: jump, keepers: keepers}
}

func (pd *PosData) Start(r *ExtRunner) {
	pd.count = 0
	pd.data = nil
}

func (pd *PosData) Iterate(r *ExtRunner) {
	if r.RR.RiClone() {
		r.NextExt()
	}
	if pd.count%pd.jump == 0 {
		pd.keep(r)
	}
	pd.count++

	r.NextExt()
}

//----------

func (pd *PosData) keep(r *ExtRunner) {
	kd := make([]interface{}, len(pd.keepers))
	for i, k := range pd.keepers {
		if k != nil {
			kd[i] = k.KeepPosData()
		}
	}
	pdd := &PosDataData{
		ri:          r.RR.Ri,
		pen:         r.RR.Pen,
		penMaxY:     r.RR.Pen.Y + r.RR.LineHeight,
		keepersData: kd,
	}
	pd.data = append(pd.data, pdd)
}

func (pd *PosData) restore(pdd *PosDataData) {
	for i, kd := range pdd.keepersData {
		if pd.keepers[i] != nil && kd != nil {
			pd.keepers[i].RestorePosData(kd)
		}
	}
}

//----------

func (pd *PosData) RestoreCloseToIndex(index int) {
	n := len(pd.data)
	if n == 0 {
		return
	}
	j := sort.Search(n, func(i int) bool {
		return pd.data[i].ri > index
	})
	if j > 0 {
		j--
	}
	pd.restore(pd.data[j])
}

func (pd *PosData) RestoreCloseToPoint(p mathutil.PointIntf) {
	n := len(pd.data)
	if n == 0 {
		return
	}
	j := sort.Search(n, func(i int) bool {
		pdd := pd.data[i]
		return pdd.penMaxY > p.Y ||
			(pdd.pen.Y > p.Y && p.Y > pdd.penMaxY &&
				pdd.pen.X > p.X)
	})
	if j > 0 {
		j--
	}
	pd.restore(pd.data[j])
}

//----------

type PosDataData struct {
	ri          int
	pen         mathutil.PointIntf
	penMaxY     mathutil.Intf
	keepersData []interface{}
}

//----------

type PosDataKeeper interface {
	KeepPosData() interface{}
	RestorePosData(interface{})
}
