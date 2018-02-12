package imageutil

import (
	"image/color"
	"log"
	"testing"
)

func init() {
	//log.SetFlags(log.Llongfile)
	log.SetFlags(0)
}

func cmpRgbaHsv(t *testing.T, i int, rgba color.RGBA, hsv HSV) {
	t.Helper()
	c1 := HSVModel.Convert(rgba).(HSV)
	//if c1 != hsv {
	if !similarHsv(c1, hsv) {
		t.Logf("\ninput %v:\n%v\nexpecting:\n%v\ngot:\n%v\n", i, rgba, hsv, c1)
		t.Fatal()
	}
}

func cmpHsvRgba(t *testing.T, i int, hsv HSV, rgba color.RGBA) {
	t.Helper()
	c1 := color.RGBAModel.Convert(hsv).(color.RGBA)
	//if c1 != rgba {
	if !similarRgba(c1, rgba) {
		t.Logf("\ninput %v:\n%v\nexpecting:\n%v\ngot:\n%v\n", i, hsv, rgba, c1)
		t.Fatal()
	}
}

func similarHsv(v1, v2 HSV) bool {
	u1 := dabs(int(v1.H), int(v2.H))
	u2 := dabs(int(v1.S), int(v2.S))
	u3 := dabs(int(v1.V), int(v2.V))
	d := 3
	return u1 < d && u2 < d && u3 < d
}
func similarRgba(v1, v2 color.RGBA) bool {
	u1 := dabs(int(v1.R), int(v2.R))
	u2 := dabs(int(v1.G), int(v2.G))
	u3 := dabs(int(v1.B), int(v2.B))
	d := 1
	return u1 < d && u2 < d && u3 < d
}
func dabs(a, b int) int {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d
}

//------------

func TestHSV1(t *testing.T) {
	u := []interface{}{
		MakeHSV(0, 0, 20),
		color.RGBA{51, 51, 51, 255},

		MakeHSV(300, 27, 20),
		color.RGBA{51, 37, 51, 255},

		MakeHSV(300, 84, 63),
		//color.RGBA{161, 26, 161, 255},
		color.RGBA{160, 25, 160, 255},

		MakeHSV(130, 61, 89),
		//color.RGBA{89, 227, 112, 255},
		color.RGBA{88, 226, 111, 255},

		MakeHSV(310, 100, 100),
		//color.RGBA{255, 0, 212, 255},
		color.RGBA{255, 0, 213, 255},
	}

	for i := 0; i < len(u); i += 2 {
		c1 := u[i].(HSV)
		c2 := u[i+1].(color.RGBA)

		cmpHsvRgba(t, i/2, c1, c2)
		cmpRgbaHsv(t, i/2, c2, c1)
	}
}
