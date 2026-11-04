package widget

import (
	"image"
	"testing"
)

func TestFloatBoxButtonMovesOpenMenuAndRepaintsOldBounds(t *testing.T) {
	ml := NewMultiLayer()
	ml.SetWrapperForRoot(ml)
	viewport := image.Rect(0, 0, 100, 100)
	ml.Bounds = viewport
	ml.MenuLayer.Bounds = viewport
	ml.BgLayer.Bounds = viewport

	bg := NewRectangle(nil)
	ml.BgLayer.Append(bg)
	bg.Bounds = viewport

	content := NewRectangle(nil)
	content.Size = image.Pt(20, 10)
	fb := NewFloatBox(ml, content)
	ml.MenuLayer.Append(fb)
	fbb := &FloatBoxButton{Button: &Button{}, floatBox: fb}
	fb.Show()
	fbb.Bounds = image.Rect(0, 0, 10, 10)
	fbb.Layout()
	ml.MenuLayer.LayoutTree()
	oldBounds := image.Rect(0, 10, 20, 20)
	if fb.Bounds != oldBounds {
		t.Fatalf("initial menu bounds=%v, want %v", fb.Bounds, oldBounds)
	}

	bg.marks = 0
	fbb.Bounds = image.Rect(0, 0, 10, 30)
	fbb.Layout()
	if !fb.HasAnyMarks(MarkNeedsLayout) {
		t.Fatal("open menu was not marked for layout")
	}
	ml.MenuLayer.LayoutTree()
	wantBounds := image.Rect(0, 30, 20, 40)
	if fb.Bounds != wantBounds {
		t.Fatalf("moved menu bounds=%v, want %v", fb.Bounds, wantBounds)
	}
	ml.markAddedRects()
	if !bg.HasAnyMarks(MarkNeedsPaint) {
		t.Fatal("background beneath old menu bounds was not marked for paint")
	}
}

func TestFloatBoxButtonOpensAtCurrentPositionAfterHiddenMove(t *testing.T) {
	ml := NewMultiLayer()
	ml.SetWrapperForRoot(ml)
	ml.MenuLayer.Bounds = image.Rect(0, 0, 100, 100)

	content := NewRectangle(nil)
	content.Size = image.Pt(20, 10)
	fb := NewFloatBox(ml, content)
	ml.MenuLayer.Append(fb)
	fbb := &FloatBoxButton{Button: &Button{}, floatBox: fb}
	fb.Hide()
	ml.MenuLayer.LayoutTree()

	fbb.Bounds = image.Rect(0, 0, 10, 30)
	fbb.Layout()
	if fb.HasAnyMarks(MarkNeedsLayout) {
		t.Fatal("hidden menu was marked for layout by button movement")
	}

	fb.Show()
	if !fb.HasAnyMarks(MarkNeedsLayout) {
		t.Fatal("show did not mark menu for layout")
	}
	ml.MenuLayer.LayoutTree()
	want := image.Rect(0, 30, 20, 40)
	if fb.Bounds != want {
		t.Fatalf("opened menu bounds=%v, want %v", fb.Bounds, want)
	}
}
