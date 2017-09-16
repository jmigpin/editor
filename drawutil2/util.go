package drawutil2

// glyph metrics
// https://developer.apple.com/library/content/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png

//func LineBaseline(fm *font.Metrics) fixed.Int26_6 {
//return fm.Ascent
//}
//func LineHeight(fm *font.Metrics) fixed.Int26_6 {
//// make it fixed to an int to avoid round errors between lines
//lh := LineBaseline(fm) + fm.Descent
//return fixed.I(lh.Ceil())
//}
//func LineY0(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
//return penY - LineBaseline(fm)
//}
//func LineY1(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
//return LineY0(penY, fm) + LineHeight(fm)
//}

//func Point266ToPoint(p *fixed.Point26_6) *image.Point {
//return &image.Point{p.X.Round(), p.Y.Round()}
//}
//func PointToPoint266(p *image.Point) *fixed.Point26_6 {
//p2 := fixed.P(p.X, p.Y)
//return &p2
//}
//func Rect266ToRect(r *fixed.Rectangle26_6) *image.Rectangle {
//var r2 image.Rectangle
//r2.Min = *Point266ToPoint(&r.Min)
//r2.Max = *Point266ToPoint(&r.Max)
//return &r2
//}
