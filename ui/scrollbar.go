package ui

//// Scrollbar for the Textarea.
//type Scrollbar struct {
//	C             uiutil.Container
//	ta            *TextArea
//	buttonPressed bool
//	bar           struct { // inner rectangle
//		sizePercent     float64
//		positionPercent float64
//		bounds          image.Rectangle
//		origPad         image.Point
//	}
//	evUnreg evreg.Unregister
//}

//func NewScrollbar(ta *TextArea) *Scrollbar {
//	sb := &Scrollbar{ta: ta}
//	width := ScrollbarWidth
//	sb.C.Style.MainSize = &width
//	sb.C.PaintFunc = sb.paint

//	r1 := sb.ta.ui.EvReg.Add(xinput.ButtonPressEventId,
//		&evreg.Callback{sb.onButtonPress})
//	r2 := sb.ta.ui.EvReg.Add(xinput.ButtonReleaseEventId,
//		&evreg.Callback{sb.onButtonRelease})
//	r3 := sb.ta.ui.EvReg.Add(xinput.MotionNotifyEventId,
//		&evreg.Callback{sb.onMotionNotify})
//	sb.evUnreg.Add(r1, r2, r3)

//	// textarea set text
//	sb.ta.EvReg.Add(TextAreaSetStrEventId,
//		&evreg.Callback{func(ev0 interface{}) {
//			sb.calcPositionAndSize()
//			sb.C.NeedPaint()
//		}})
//	// textarea y jump
//	sb.ta.EvReg.Add(TextAreaSetOffsetYEventId,
//		&evreg.Callback{func(ev0 interface{}) {
//			sb.calcPositionAndSize()
//			sb.C.NeedPaint()
//		}})
//	// textarea bounds change
//	sb.ta.EvReg.Add(TextAreaBoundsChangeEventId,
//		&evreg.Callback{func(ev0 interface{}) {
//			sb.calcPositionAndSize()
//			sb.C.NeedPaint()
//		}})
//	//// textarea set cursor index
//	//sb.ta.EvReg.Add(TextAreaSetCursorIndexEventId,
//	//&evreg.Callback{func(ev0 interface{}) {
//	//sb.C.NeedPaint()
//	//}})

//	return sb
//}
//func (sb *Scrollbar) Close() {
//	sb.evUnreg.UnregisterAll()
//}
//func (sb *Scrollbar) calcPositionAndSize() {
//	// size and position percent (from textArea)
//	ta := sb.ta
//	sp := 1.0
//	pp := 0.0
//	h := ta.StrHeight()
//	if h > 0 {
//		dy := fixed.I(ta.Bounds().Dy())
//		sp = float64(dy) / float64(h)
//		if sp > 1 {
//			sp = 1
//		}
//		y := sb.ta.OffsetY()
//		pp = float64(y) / float64(h)
//	}
//	sb.bar.sizePercent = sp
//	sb.bar.positionPercent = pp
//}

//// Dragging the scrollbar
//func (sb *Scrollbar) calcPositionFromPoint(p *image.Point) {
//	r := *sb.Bounds()
//	height := r.Dy()
//	py := p.Add(sb.bar.origPad).Y - r.Min.Y
//	if py < 0 {
//		py = 0
//	} else if py > height {
//		py = height
//	}
//	sb.bar.positionPercent = float64(py) / float64(height)
//}

//func (sb *Scrollbar) setTextareaOffset() {
//	pp := sb.bar.positionPercent
//	h := sb.ta.StrHeight()
//	py := fixed.Int26_6(pp * float64(h))
//	sb.ta.SetOffsetY(py)
//}

//func (sb *Scrollbar) paint() {
//	// background
//	sb.ta.ui.FillRectangle(&sb.Bounds(), ScrollbarBgColor)
//	// bar
//	r := sb.Bounds()
//	size := int(float64(r.Dy()) * sb.bar.sizePercent)
//	if size < 7 { // minimum size
//		size = 7
//	}
//	r2 := r
//	r2.Min.Y += int(float64(r.Dy()) * sb.bar.positionPercent)
//	r2.Max.Y = r2.Min.Y + size
//	r2 = r2.Intersect(sb.Bounds())
//	sb.ta.ui.FillRectangle(&r2, ScrollbarFgColor)
//	sb.bar.bounds = r2

//	//// cursor index
//	//cip := sb.ta.stringCache.GetPoint(sb.ta.CursorIndex())
//	//h := sb.ta.StrHeight()
//	//percent := float64(cip.Y) / float64(h)
//	//sy := int(percent * float64(sb.Bounds().Dy()))
//	//r3 := sb.Bounds()
//	//r3.Min.Y += sy
//	//r3.Max.Y = r3.Min.Y + 1
//	//r3.Min.X += r3.Dx() / 2
//	//r3 = r3.Intersect(sb.Bounds())
//	//sb.ta.ui.FillRectangle(&r3, color.Black)
//}
//func (sb *Scrollbar) onButtonPress(ev0 interface{}) {
//	ev := ev0.(*xinput.ButtonPressEvent)
//	if !ev.Point.In(sb.Bounds()) {
//		return
//	}
//	sb.buttonPressed = true
//	switch {
//	case ev.Button.Button(1):
//		sb.setOrigPad(ev.Point) // keep pad for drag calc
//		sb.calcPositionFromPoint(ev.Point)
//		sb.setTextareaOffset()
//		sb.C.NeedPaint()
//	case ev.Button.Button(4): // wheel up
//		sb.ta.PageUp()
//	case ev.Button.Button(5): // wheel down
//		sb.ta.PageDown()
//	}
//}
//func (sb *Scrollbar) onMotionNotify(ev0 interface{}) {
//	if !sb.buttonPressed {
//		return
//	}
//	ev := ev0.(*xinput.MotionNotifyEvent)
//	switch {
//	case ev.Mods.HasButton(1):
//		sb.calcPositionFromPoint(ev.Point)
//		sb.setTextareaOffset()
//		sb.C.NeedPaint()
//	}
//}
//func (sb *Scrollbar) onButtonRelease(ev0 interface{}) {
//	if !sb.buttonPressed {
//		return
//	}
//	sb.buttonPressed = false
//	ev := ev0.(*xinput.ButtonReleaseEvent)
//	if ev.Button.Button(1) {
//		sb.calcPositionFromPoint(ev.Point)
//		sb.setTextareaOffset()
//		sb.C.NeedPaint()
//	}
//}
//func (sb *Scrollbar) setOrigPad(p *image.Point) {
//	if p.In(sb.bar.bounds) {
//		// set position relative to the bar top
//		r := &sb.bar.bounds
//		sb.bar.origPad.X = r.Max.X - p.X
//		sb.bar.origPad.Y = r.Min.Y - p.Y
//	} else {
//		// set position in the middle of the bar
//		r := &sb.bar.bounds
//		sb.bar.origPad.X = r.Dx() / 2
//		sb.bar.origPad.Y = -r.Dy() / 2
//	}
//}
