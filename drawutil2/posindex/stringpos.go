package posindex

//type StringPos struct {
//	Iterator loopers.Looper
//	strl     *loopers.StringIterator
//}

//func NewStringPos(strl *loopers.StringLooper) *StringPos {
//	return &StringPos{strl: strl}
//}
//func (sp *StringPos) GetPoint(index int) *fixed.Point26_6 {
//	sp.Iterator.Loop(func() bool {
//		if sp.strl.Ri >= index {
//			return false
//		}
//		return true
//	})
//	pb := sp.strl.PenBounds()
//	return &pb.Min
//}
//func (sp *StringPos) GetIndex(p *fixed.Point26_6) int {
//	found := false
//	foundLine := false
//	lineRuneIndex := 0

//	sp.Iterator.Loop(func() bool {
//		pb := sp.strl.PenBounds()

//		// before the start or already passed the line
//		if p.Y < pb.Min.Y {
//			if !foundLine {
//				// before the start, returns first index
//				found = true
//			}
//			return false
//		}
//		// in the line
//		if p.Y >= pb.Min.Y && p.Y < pb.Max.Y {
//			// before the first rune of the line
//			if p.X < pb.Min.X {
//				found = true
//				return false
//			}
//			// in a rune
//			if p.X < pb.Max.X {
//				found = true
//				return false
//			}
//			// after last rune of the line
//			foundLine = true
//			lineRuneIndex = sp.strl.Ri
//		}
//		return true
//	})
//	if found {
//		return sp.strl.Ri
//	}
//	if foundLine {
//		return lineRuneIndex
//	}
//	return len(sp.strl.Str)
//}
