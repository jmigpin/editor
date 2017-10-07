package widget

import (
	"image"
	"testing"
)

func TestFlowLayout1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l1 := &FlowLayout{YAxis: false}
	AppendChilds(l1, r1, r2, r3, r4)

	r := image.Rect(0, 0, 100, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r4.Bounds() != image.Rect(30, 0, 40, 10) {
		t.Fatalf("%v", r4.Bounds())
	}
}

func TestFlowLayout2(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l1 := &FlowLayout{YAxis: true}
	AppendChilds(l1, r1, r2, r3, r4)

	r := image.Rect(0, 0, 100, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r4.Bounds() != image.Rect(0, 30, 10, 40) {
		t.Fatalf("%v", r4.Bounds())
	}
}

func TestFlowLayout3(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l1 := &FlowLayout{YAxis: false}
	AppendChilds(l1, r1, r2, r3, r4)

	r2.SetExpand(false, true)
	r3.SetExpand(true, false)

	r := image.Rect(0, 0, 100, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r2.Bounds() != image.Rect(10, 0, 20, 100) {
		t.Fatalf("%v", r2.Bounds())
	}
	if r3.Bounds() != image.Rect(20, 0, 90, 10) {
		t.Fatalf("%v", r3.Bounds())
	}
}

func TestFlowLayout4(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l1 := &FlowLayout{YAxis: false}
	AppendChilds(l1, r1, r2, r3, r4)

	r1.SetExpand(true, false)

	r := image.Rect(0, 0, 20, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r2.Bounds() != image.Rect(0, 0, 10, 10) {
		t.Fatalf("%v", r2.Bounds())
	}
}

func TestFlowLayout5(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l1 := &FlowLayout{YAxis: true}
	AppendChilds(l1, r1, r2, r3, r4)

	r1.SetExpand(false, true)

	r := image.Rect(0, 0, 100, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r1.Bounds() != image.Rect(0, 0, 10, 70) {
		t.Fatalf("%v", r1.Bounds())
	}
}

func TestFlowLayout6(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)

	l1 := &FlowLayout{YAxis: false}
	AppendChilds(l1, r1, r2)
	r2.SetExpand(true, false)

	r3 := NewRectangle(nil)
	l2 := &FlowLayout{YAxis: true}
	AppendChilds(l2, l1, r3)
	r := image.Rect(0, 0, 100, 100)
	l2.SetBounds(&r)
	l2.CalcChildsBounds()

	if r2.Bounds() != image.Rect(10, 0, 100, 10) {
		t.Fatalf("%v", r2.Bounds())
	}
}

func TestFlowLayout7(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)

	l1 := &FlowLayout{YAxis: false}
	AppendChilds(l1, r1, r2)
	r2.SetFill(true, false)
	l1.SetExpand(true, false)

	r3 := NewRectangle(nil)
	l2 := &FlowLayout{YAxis: true}
	AppendChilds(l2, l1, r3)
	r := image.Rect(0, 0, 100, 100)
	l2.SetBounds(&r)
	l2.CalcChildsBounds()

	if r2.Bounds() != image.Rect(10, 0, 100, 10) {
		t.Fatalf("%v", r2.Bounds())
	}
}

func TestFlowLayoutExpandX(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)

	r2.SetExpand(true, false)

	l2 := &FlowLayout{}
	AppendChilds(l2, r1, r2)

	l1 := &FlowLayout{}
	AppendChilds(l1, l2, r3)

	r := image.Rect(0, 0, 100, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r3.Bounds() != image.Rect(90, 0, 100, 10) {
		t.Fatalf("%v", r3.Bounds())
	}
}

func TestFlowLayoutFillX(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l3 := &FlowLayout{}
	AppendChilds(l3, r1, r2)
	r2.SetFill(true, false)

	l2 := &FlowLayout{YAxis: true}
	AppendChilds(l2, l3, r3)

	r3.Size.X = 50

	l1 := &FlowLayout{}
	AppendChilds(l1, l2, r4)
	r4.SetExpand(true, false)

	r := image.Rect(0, 0, 100, 100)
	l1.SetBounds(&r)
	l1.CalcChildsBounds()

	if r2.Bounds() != image.Rect(10, 0, 50, 10) {
		t.Fatalf("%v", r2.Bounds())
	}
}

//for _, child := range l1.Childs() {
//	t.Logf("%v", child.Bounds())
//}
