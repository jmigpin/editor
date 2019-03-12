package drawer3

import (
	"image"
	"testing"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

func Test1(t *testing.T) {
	s := "Ai"
	for i := 0; i < 10; i++ {
		s += s
	}
	//fmt.Printf("len s=%v\n", len(s))

	d := NewPosDrawer()
	d.WrapLine.SetOn(true)

	r := iorw.NewBytesReadWriter([]byte(s))

	face := drawutil.GetTestFace()
	d.SetFace(face)

	d.SetReader(r)

	maxX := 70
	d.SetBounds(image.Rect(0, 0, maxX, 70))

	d.SetOffset(image.Point{0, 30})

	//i := d.IndexOf(d.Offset())
	//p := d.PointOf(i)
	//u := mathutil.Point64{12, 28}
	//if p != u {
	//	t.Fatalf("%v", p)
	//}

	_ = d.Measure()

	// reducing x bound
	for x := 0; x <= maxX; x++ {
		o := d.Offset()
		oi := d.IndexOf(o)
		oip := d.PointOf(oi)
		diff := o.Y - oip.Y

		d.SetBounds(image.Rect(0, 0, maxX-x, 70))

		_ = d.Measure()

		o2 := d.Offset()
		oip2 := d.PointOf(oi)
		diff2 := o2.Y - oip2.Y

		if diff != diff2 {
			t.Fatalf("x=%v oip=%v oip2=%v", x, oip, oip2)
		}
	}
}
