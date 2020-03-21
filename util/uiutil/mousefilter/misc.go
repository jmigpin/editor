package mousefilter

import "image"

func DetectMove(press, p image.Point) bool {
	r := image.Rectangle{press, press}
	// padding to detect intention to move/drag
	v := 3
	r = r.Inset(-v) // negative inset (outset)
	return !p.In(r)
}

func DetectMovePad(p, press, ref image.Point) image.Point {
	u := ref.Sub(p)
	v := 3 + 2 // matches value in DetectMove()+2
	if u.X > v || u.X < -v {
		u.X = 0
	}
	if u.Y > v || u.Y < -v {
		u.Y = 0
	}
	return u
}
