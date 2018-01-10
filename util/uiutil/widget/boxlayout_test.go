package widget

import (
	"image"
	"testing"
)

func TestBoxLayout1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	s1 := image.Point{5, 5}
	r1.Size, r2.Size, r3.Size, r4.Size = s1, s1, s1, s1

	l1 := NewBoxLayout()
	l1.Bounds = image.Rect(0, 0, 100, 100)
	l1.Append(r1, r2, r3, r4)
	l1.CalcChildsBounds()
	if !(r1.Bounds == image.Rect(0, 0, 5, 5) &&
		r2.Bounds == image.Rect(5, 0, 10, 5) &&
		r3.Bounds == image.Rect(10, 0, 15, 5) &&
		r4.Bounds == image.Rect(15, 0, 20, 5)) {
		t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
		t.Fatal()
	}

	l1.SetChildFlex(r2, true, false)
	l1.CalcChildsBounds()
	if !(r1.Bounds == image.Rect(0, 0, 5, 5) &&
		r2.Bounds == image.Rect(5, 0, 90, 5) &&
		r3.Bounds == image.Rect(90, 0, 95, 5) &&
		r4.Bounds == image.Rect(95, 0, 100, 5)) {
		t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
		t.Fatal()
	}

	l1.SetChildFlex(r1, true, false)
	l1.SetChildFlex(r2, true, false)
	l1.SetChildFlex(r3, true, false)
	l1.SetChildFlex(r4, true, false)
	l1.CalcChildsBounds()
	if !(r1.Bounds == image.Rect(0, 0, 25, 5) &&
		r2.Bounds == image.Rect(25, 0, 50, 5) &&
		r3.Bounds == image.Rect(50, 0, 75, 5) &&
		r4.Bounds == image.Rect(75, 0, 100, 5)) {
		t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
		t.Fatal()
	}
}

func TestBoxLayout2(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	s1 := image.Point{10, 10}
	r1.Size, r2.Size, r3.Size, r4.Size = s1, s1, s1, s1
	r2.Size = image.Point{5, 5}

	l1 := NewBoxLayout()
	l1.Bounds = image.Rect(0, 0, 100, 100)
	l1.Append(r1, r2, r3, r4)
	l1.CalcChildsBounds()
	t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)

	l1.SetChildFill(r2, false, true)
	// r2 fill y is set to true, but only measuring, should not affect anything
	m := l1.Measure(image.Point{50, 50})
	p := image.Point{35, 10}
	if m != p {
		t.Log(m)
		t.Fatal()
	}

	// r2 fill y is set to true, should affect r2.max.y
	t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
	l1.CalcChildsBounds()
	if !(r1.Bounds == image.Rect(0, 0, 10, 10) &&
		r2.Bounds == image.Rect(10, 0, 15, 100) &&
		r3.Bounds == image.Rect(15, 0, 25, 10) &&
		r4.Bounds == image.Rect(25, 0, 35, 10)) {
		t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
		t.Fatal()
	}

	// should divide space among "fillables"
	t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
	l1.SetChildFill(r2, true, false)
	l1.SetChildFill(r4, true, false)
	l1.CalcChildsBounds()
	if !(r1.Bounds == image.Rect(0, 0, 10, 10) &&
		r2.Bounds == image.Rect(10, 0, 50, 5) &&
		r3.Bounds == image.Rect(50, 0, 60, 10) &&
		r4.Bounds == image.Rect(60, 0, 100, 10)) {
		t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
		t.Fatal()
	}

	// should ignore the fillables
	l1.SetChildFill(r2, true, true)
	l1.SetChildFill(r4, true, true)
	m = l1.Measure(image.Point{50, 50})
	p = image.Point{35, 10}
	if m != p {
		t.Log(m)
		t.Fatal()
	}

	// have no flex in y, should get maximum value in y
	l1.SetChildFlex(r1, false, true)
	m = l1.Measure(image.Point{50, 50})
	p = image.Point{35, 50}
	if m != p {
		t.Log(m)
		t.Fatal()
	}

	// flex should have priority over the fill, but in y they should all get the value
	// r1,r2,r4 should have maximum y
	t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
	l1.CalcChildsBounds()
	if !(r1.Bounds == image.Rect(0, 0, 10, 100) &&
		r2.Bounds == image.Rect(10, 0, 50, 100) &&
		r3.Bounds == image.Rect(50, 0, 60, 10) &&
		r4.Bounds == image.Rect(60, 0, 100, 100)) {
		t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
		t.Fatal()
	}

	// flex in x
	l1.SetChildFlex(r1, true, false)
	l1.SetChildFill(r2, true, true)
	l1.SetChildFill(r4, true, true)
	m = l1.Measure(image.Point{50, 50})
	p = image.Point{50, 10}
	if m != p {
		t.Log(m)
		t.Fatal()
	}

	// flex in x, flex has priority over fill
	t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
	l1.CalcChildsBounds()
	if !(r1.Bounds == image.Rect(0, 0, 75, 10) &&
		r2.Bounds == image.Rect(75, 0, 80, 100) &&
		r3.Bounds == image.Rect(80, 0, 90, 10) &&
		r4.Bounds == image.Rect(90, 0, 100, 100)) {
		t.Log(r1.Bounds, r2.Bounds, r3.Bounds, r4.Bounds)
		t.Fatal()
	}
}
