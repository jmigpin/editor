package loopers

//type LineDataLooper struct {
//	EmbedLooper

//	strl   *StringLooper
//	wlinel *WrapLineLooper

//	notLineStart bool
//	lineStartPen fixed.Point26_6
//	lineLen      fixed.Int26_6
//	lines        []*LineDataData
//}

//func NewLineDataLooper(strl *StringLooper, wlinel *WrapLineLooper) *LineDataLooper {
//	lpr := &LineDataLooper{strl: strl, wlinel: wlinel}
//	return lpr
//}
//func (lpr *LineDataLooper) Loop(fn func() bool) {
//	lpr.OuterLooper().Loop(func() bool {
//		lpr.positionalData()
//		return true
//	})
//}
//func (lpr *LineDataLooper) positionalData() {
//	if !lpr.notLineStart {
//		lpr.notLineStart = false
//		lpr.lineStartPen = lpr.strl.Pen
//	}
//	lpr.lineLen += lpr.strl.Kern + lpr.strl.Advance
//	if lpr.strl.Ru != '\n' {
//		return
//	}

//	// TODO: this code is duplicated - unify with func
//	wlrAdv := lpr.wlinel.wrapLineRuneAdvance(WrapLineRune)
//	fixedAdv := wlrAdv + lpr.wlinel.advance()

//	e := &LineDataData{
//		Pen:    lpr.lineStartPen,
//		Len:    lpr.lineLen,
//		Indent: lpr.wlinel.data.PenX + fixedAdv,
//	}
//	lpr.lines = append(lpr.lines, e)

//	lpr.notLineStart = true
//	lpr.lineLen = 0
//}

////// Implements PosDataKeeper
////func (lpr *WrapLineLooper) UpdatePosData() {
////	//var lines []*WrapLine2LineData
////	first := true
////	var pen fixed.Point26_6
////	for _, ld := range lpr.WrapData.Lines {
////		if first {
////			first = false
////			pen = ld.lineStartPen
////		}
////		pen.Y += lpr.strl.LineHeight()
////		if ld.Len > lpr.MaxX {
////			len := ld.Len

////			// remove the first line
////			len -= lpr.MaxX

////			// wrap lines with the indented max size
////			maxX := lpr.MaxX - ld.Indent

////			// divide to get n lines
////			n := int(len / maxX)

////			// remove n lines
////			len -= fixed.Int26_6(n) * maxX

////			// TODO: need the rune always visible code

////			// new position
////			pen.Y += fixed.Int26_6(n) * lpr.strl.LineHeight()
////			pen.X = ld.Indent + len

////			lpr.strl.Pen = pen
////		}
////	}
////}

//type LineDataData struct {
//	Pen    fixed.Point26_6
//	Len    fixed.Int26_6
//	Indent fixed.Int26_6
//}
