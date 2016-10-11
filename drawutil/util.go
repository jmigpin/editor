package drawutil

import (
	"image"
	"io/ioutil"

	//"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/fixed"
)

func ParseFont(filename string) (*truetype.Font, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	//return freetype.ParseFont(b)
	return truetype.Parse(b)
}

func Point266ToPoint(p *fixed.Point26_6) *image.Point {
	return &image.Point{p.X.Round(), p.Y.Round()}
}
func PointToPoint266(p *image.Point) *fixed.Point26_6 {
	p2 := fixed.P(p.X, p.Y)
	return &p2
}
func Rect266ToRect(r *fixed.Rectangle26_6) *image.Rectangle {
	var r2 image.Rectangle
	r2.Min = *Point266ToPoint(&r.Min)
	r2.Max = *Point266ToPoint(&r.Max)
	return &r2
}
