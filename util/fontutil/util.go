package fontutil

import (
	"image"

	"golang.org/x/image/math/fixed"
)

func Rect266MinFloorMaxCeil(r fixed.Rectangle26_6) image.Rectangle {
	min := image.Point{r.Min.X.Floor(), r.Min.Y.Floor()}
	max := image.Point{r.Max.X.Ceil(), r.Max.Y.Ceil()}
	return image.Rectangle{min, max}
}

func Float32ToFixed266(v float32) fixed.Int26_6 {
	return fixed.Int26_6(v * 64)
}
func Float64ToFixed266(v float64) fixed.Int26_6 {
	return fixed.Int26_6(v * 64)
}
func Fixed266ToFloat64(v fixed.Int26_6) float64 {
	return float64(v) / float64(64)
}
