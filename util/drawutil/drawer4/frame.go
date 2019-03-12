package drawer4

// TODO: iteration later must use readrune, can't use a loop with ints (i++)

//type Frame struct {
//	m map[int]*RuneInfo
//}

//func NewFrame() *Frame {
//	fr := &Frame{}
//	fr.m = map[int]*RuneInfo{}
//	return fr
//}

//func (fr *Frame) get(ri int) *RuneInfo {
//	info, ok := fr.m[ri]
//	if !ok {
//		info = &RuneInfo{}
//		fr.m[ri] = info
//	}
//	return info
//}

////----------

//type RuneInfo struct {
//	lineStart bool
//	penBounds mathutil.RectangleIntf
//}

//----------

//func frameInfo(d *Drawer, offset int) *Frame {
//	fr := NewFrame()
//	maxY := mathutil.Intf(math.MaxInt64)
//	//first := true
//	frameLoop(d, true, 0, offset,
//		func() {
//			//info := fr.get(d.st.runeR.ri)
//			//info.lineStart = true
//		},
//		func() {
//			pb := d.iters.runeR.penBounds()

//			// set early exit
//			if d.st.runeR.ri == offset {
//				bdy := mathutil.Intf1(d.bounds.Dy())
//				maxY = pb.Min.Y + bdy
//			}

//			// early exit
//			if pb.Min.Y >= maxY {
//				d.iterStop()
//				return
//			}

//			if d.iters.runeR.isNormal() {
//				info := fr.get(d.st.runeR.ri)
//				info.penBounds = pb
//				info.lineStart = d.st.line.lineStart || d.st.lineWrap.lineStart
//				//// first ri line start special case
//				//if first {
//				//	first = false
//				//	if !info.lineStart {
//				//		if d.st.runeR.ri == offset {
//				//			//info.lineStart = true
//				//		}
//				//	}
//				//}
//			}

//			if !d.iterNext() {
//				return
//			}
//		})
//	return fr
//}

//----------
