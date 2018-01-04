package widget

import (
	"image"
	"testing"
)

func TestEndPercentLayout1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)
	r5 := NewRectangle(nil)

	l1 := &EndPercentLayout{YAxis: false}
	AppendChilds(l1, r1, r2, r3, r4, r5)
	l1.SetChildEndPercent(l1.FirstChild(), 0.10)
	l1.SetChildEndPercent(l1.FirstChild().Next().Next(), 0.80)

	r := image.Rect(0, 0, 100, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r5.Bounds() != image.Rect(90, 0, 100, 100) {
		t.Fatalf("last child bounds: %v", r5.Bounds())
	}

	r6 := NewRectangle(nil)
	PushBack(l1, r6)

	l1.CalcChildsBounds()

	if r6.Bounds() != image.Rect(95, 0, 100, 100) {
		t.Fatalf("last child bounds: %v", r6.Bounds())
	}
}

//for _, child := range l1.Childs() {
//	t.Logf("%v", child.Bounds())
//}
