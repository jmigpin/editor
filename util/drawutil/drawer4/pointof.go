package drawer4

type PointOf struct {
	d *Drawer
}

func (po *PointOf) Init() {}

func (po *PointOf) Iter() {
	if !po.d.iters.runeR.isRiExtra() {
		if po.d.st.runeR.ri >= po.d.st.pointOf.index {
			po.d.iterStop()
			return
		}
	}
	_ = po.d.iterNext()
}

func (po *PointOf) End() {
	// pen is top/left, use what penbounds is using
	penb := po.d.iters.runeR.penBounds()
	po.d.st.pointOf.p = penb.Min.ToPointFloor()
}
