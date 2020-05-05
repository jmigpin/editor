package widget

import "testing"

func TestStableOffsetScroll1(t *testing.T) {
	// 0123456789
	o := StableOffsetScroll(3, 4, 2, 1)
	if o != 3 {
		t.Fatal(o)
	}
	o = StableOffsetScroll(4, 4, 0, 1)
	if o != 4 {
		t.Fatal(o)
	}
	o = StableOffsetScroll(4, 4, 1, 0)
	if o != 4 {
		t.Fatal(o)
	}
	o = StableOffsetScroll(5, 4, 1, 0)
	if o != 4 {
		t.Fatal(o)
	}
	o = StableOffsetScroll(5, 4, 0, 1)
	if o != 5 {
		t.Fatal(o)
	}
	o = StableOffsetScroll(4, 3, 0, 1)
	if o != 4 {
		t.Fatal(o)
	}
	o = StableOffsetScroll(4, 3, 1, 0)
	if o != 3 {
		t.Fatal(o)
	}
	o = StableOffsetScroll(4, 3, 1, 1)
	if o != 4 {
		t.Fatal(o)
	}
}

func TestStableOffsetScroll2(t *testing.T) {
	// 0123456789
	var o int
	o = StableOffsetScroll(4, 4, 1, 0)
	if o != 4 {
		t.Fatal(o)
	}
}
