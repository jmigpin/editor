package widget

// TODO: test insertion that push rows below and if there is not space it should push rows above

import (
	"image"
	"math"
	"testing"
)

func TestStartPercentLayout1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)

	l1 := NewStartPercentLayout()

	logNodes := func() {
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3)
	logNodes()
	tpv([]float64{0.0, 0.5, 0.75})

	l1.Remove(r2)
	tpv([]float64{0.0, 0.75})

	l1.InsertBefore(r2, r3)
	tpv([]float64{0.0, 0.375, 0.75})

	l1.Remove(r3)
	tpv([]float64{0.0, 0.375})

	l1.Remove(r1)
	tpv([]float64{0.375})
}

func TestStartPercentLayout2(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)

	l1 := NewStartPercentLayout()

	logNodes := func() {
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3)
	logNodes()
	tpv([]float64{0.0, 0.5, 0.75})

	l1.Remove(r1)
	tpv([]float64{0.5, 0.75})
}

func TestStartPercentLayout3(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l1 := NewStartPercentLayout()
	l1.minp = 0.05

	logNodes := func() {
		t.Helper()
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3)
	tpv([]float64{0.0, 0.5, 0.75})

	// insert before first, becomes second
	logNodes()
	l1.InsertBefore(r4, r1)
	tpv([]float64{0.0, 0.25, 0.5, 0.75})

	// insert before first with small space (smaller then min), insert at first
	l1.Remove(r4)
	l1.Resize(r1, 0.1)
	l1.InsertBefore(r4, r1)
	tpv([]float64{0.0, 0.1, 0.5, 0.75})
}

func TestStartPercentLayoutResize1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)

	l1 := NewStartPercentLayout()
	l1.minp = 0.05

	logNodes := func() {
		t.Helper()
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3)
	logNodes()

	l1.Resize(r1, 0.90)
	tpv([]float64{0.45, 0.5, 0.75})

	l1.Resize(r1, 0.20)
	l1.Resize(r2, 0.10)
	tpv([]float64{0.20, 0.25, 0.75})

	l1.Resize(r1, 0.0)
	tpv([]float64{0.0, 0.25, 0.75})
}

func TestStartPercentLayoutResizeWithPush1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)

	l1 := NewStartPercentLayout()
	l1.minp = 0.05

	logNodes := func() {
		t.Helper()
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3)
	logNodes()
	l1.ResizeWithPush(r1, 0.80)
	tpv([]float64{0.8, 0.85, 0.90})

	l1.ResizeWithPush(r1, 0.90)
	tpv([]float64{0.85, 0.90, 0.95})

	l1.ResizeWithPush(r3, 0.5)
	tpv([]float64{0.4, 0.45, 0.5})

	l1.ResizeWithPush(r2, 0.07)
	tpv([]float64{0.02, 0.07, 0.5})

	l1.ResizeWithPush(r2, 0.4)
	tpv([]float64{0.02, 0.40, 0.5})
}

func TestStartPercentLayoutResizeWithPush2(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)

	l1 := NewStartPercentLayout()
	l1.minp = 0.05

	logNodes := func() {
		t.Helper()
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3, r4)
	l1.Resize(r3, 0.81)
	l1.Resize(r2, 0.75)
	l1.Resize(r1, 0.5)
	logNodes()
	l1.ResizeWithPush(r3, 0.5)
	tpv([]float64{0.4, 0.45, 0.5, 0.875})
}

func TestStartPercentLayoutResizeWithMove1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)
	r5 := NewRectangle(nil)

	l1 := NewStartPercentLayout()
	l1.minp = 0.05

	logNodes := func() {
		t.Helper()
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3, r4, r5)
	logNodes()
	l1.ResizeWithMove(r1, 0.81)
	tpv([]float64{0.5, 0.75, 0.81, 0.875, 0.9375})

	logNodes()
	l1.ResizeWithPush(r1, 0.5)
	tpv([]float64{0.4, 0.45, 0.5, 0.875, 0.9375})

	l1.ResizeWithPush(r2, 0.5)
	tpv([]float64{0.5, 0.55, 0.6, 0.875, 0.9375})

	l1.ResizeWithMove(r1, 0.3)
	tpv([]float64{0.3, 0.5, 0.55, 0.875, 0.9375})
}

func TestStartPercentLayoutResizeWithMove2(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)
	r5 := NewRectangle(nil)

	l1 := NewStartPercentLayout()
	l1.minp = 0.05

	logNodes := func() {
		t.Helper()
		logSPNodes(t, l1)
	}
	tpv := func(v []float64) {
		t.Helper()
		testSPNodes(t, l1, v)
	}

	l1.Append(r1, r2, r3, r4, r5)
	logNodes()
	l1.ResizeWithMove(r2, 0.0)
	tpv([]float64{0.0, 0.05, 0.75, 0.875, 0.9375})
}

func TestStartPercentLayoutBounds1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)
	r4 := NewRectangle(nil)
	r5 := NewRectangle(nil)

	l1 := NewStartPercentLayout()

	logNodes := func() {
		t.Helper()
		logSPNodes(t, l1)
	}

	r := image.Rect(0, 0, 100, 100)
	l1.Bounds = r
	l1.Append(r1, r2, r3, r4)
	logNodes()
	l1.setsp(r1, 0.10)
	l1.setsp(r3, 0.80)
	logNodes()
	l1.CalcChildsBounds()
	if r4.Bounds != image.Rect(87, 0, 100, 100) {
		t.Log(r4.Bounds)
		t.Fatal()
	}

	l1.Append(r5)
	l1.CalcChildsBounds()
	if !(r4.Bounds == image.Rect(87, 0, 93, 100) && r5.Bounds == image.Rect(93, 0, 100, 100)) {
		t.Log(r4.Bounds, r5.Bounds)
		t.Fatal()
	}
}

func logSPNodes(t *testing.T, sp *StartPercentLayout) {
	t.Helper()
	u := []float64{}
	sp.IterChilds(func(c Node) {
		u = append(u, sp.spm[c])
	})
	t.Log(u)
}

func testSPNodes(t *testing.T, sp *StartPercentLayout, values []float64) {
	t.Helper()
	fail := false
	i := 0
	if len(values) != sp.ChildsLen() {
		fail = true
	} else {
		sp.IterChilds(func(c Node) {
			if !testPercentValue(sp, c, values[i]) {
				fail = true
			}
			i++
		})
	}
	// fail outside of iterchilds to get correct source line on error (not helper func)
	if fail {
		logSPNodes(t, sp)
		t.Fatal()
	}
}

func testPercentValue(l *StartPercentLayout, n Node, v float64) bool {
	return feq(l.spm[n], v)
}

var eps = 0.00000001

func feq(a, b float64) bool {
	if math.Abs(a-b) < eps {
		return true
	}
	return false
}
