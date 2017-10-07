package widget

import "testing"

func TestNode1(t *testing.T) {
	r1 := NewRectangle(nil)
	r2 := NewRectangle(nil)
	r3 := NewRectangle(nil)

	AppendChilds(r1, r2, r3)

	r3.SetHidden(true)

	if r1.NChilds() != 1 {
		t.Fatalf("%v", r1.NChilds())
	}
}
