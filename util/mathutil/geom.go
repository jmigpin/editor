package mathutil

import (
	"image"

	"golang.org/x/image/math/fixed"
)

// Integer based float. Based on fixed.Int52_12.
type Intf int64

func Intf1(x int) Intf           { return Intf(x) << 12 }
func Intf2(x fixed.Int26_6) Intf { return Intf(x) << 6 }

func (x Intf) Floor() int { return int(x >> 12) }
func (x Intf) Ceil() int  { return int((x + 0xfff) >> 12) }

func (x Intf) Mul(y Intf) Intf {
	a := fixed.Int52_12(x)
	b := fixed.Int52_12(y)
	return Intf(a.Mul(b))
}

func (x Intf) String() string {
	return fixed.Int52_12(x).String()
}

//----------

type PointIntf struct {
	X, Y Intf
}

func PIntf1(x, y int) PointIntf {
	x2 := Intf1(x)
	y2 := Intf1(y)
	return PointIntf{x2, y2}
}
func PIntf2(p image.Point) PointIntf {
	return PIntf1(p.X, p.Y)
}

func (p PointIntf) Add(q PointIntf) PointIntf {
	return PointIntf{p.X + q.X, p.Y + q.Y}
}
func (p PointIntf) Sub(q PointIntf) PointIntf {
	return PointIntf{p.X - q.X, p.Y - q.Y}
}

func (p PointIntf) In(r RectangleIntf) bool {
	return r.Min.X <= p.X && p.X < r.Max.X && r.Min.Y <= p.Y && p.Y < r.Max.Y
}

func (p PointIntf) ToPointCeil() image.Point {
	return image.Point{p.X.Ceil(), p.Y.Ceil()}
}
func (p PointIntf) ToPointFloor() image.Point {
	return image.Point{p.X.Floor(), p.Y.Floor()}
}

//----------

type RectangleIntf struct {
	Min, Max PointIntf
}

func RIntf(r image.Rectangle) RectangleIntf {
	min := PIntf2(r.Min)
	max := PIntf2(r.Max)
	return RectangleIntf{min, max}
}

func (r RectangleIntf) Add(p PointIntf) RectangleIntf {
	return RectangleIntf{r.Min.Add(p), r.Max.Add(p)}
}
func (r RectangleIntf) Sub(p PointIntf) RectangleIntf {
	return RectangleIntf{r.Min.Sub(p), r.Max.Sub(p)}
}

func (r RectangleIntf) Intersect(s RectangleIntf) RectangleIntf {
	if r.Min.X < s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y < s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X > s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y > s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	if r.Empty() {
		return RectangleIntf{}
	}
	return r
}

func (r RectangleIntf) Empty() bool {
	return r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y
}

func (r RectangleIntf) ToRectFloorCeil() image.Rectangle {
	min := r.Min.ToPointFloor()
	max := r.Max.ToPointCeil()
	return image.Rectangle{min, max}
}
