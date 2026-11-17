package ui

import (
	"image"
	"testing"

	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestPotentialTextAreaInnerWidth(t *testing.T) {
	colorTheme, fontTheme, shadows := ColorThemeCycler.CurName, FontThemeCycler.CurName, ShadowsOn
	defer func() {
		ColorThemeCycler.CurName, FontThemeCycler.CurName, ShadowsOn = colorTheme, fontTheme, shadows
	}()
	ColorThemeCycler.CurName, FontThemeCycler.CurName = "light", "regular"

	for _, shadowsOn := range []bool{true, false} {
		ShadowsOn = shadowsOn
		win := &rowWheelTestWindow{img: image.NewRGBA(image.Rect(0, 0, 800, 600)), closed: make(chan struct{})}
		ui, err := NewUIWithWindow("row width test", win)
		if err != nil {
			t.Fatal(err)
		}
		ui.HandleEvent(&event.WindowResize{Rect: win.img.Bounds()})
		ui.LayoutMarkedAndSchedulePaint()

		col := ui.Root.Cols.NewColumn()
		row := col.NewRowBefore(nil)
		ui.LayoutMarkedAndSchedulePaint()

		d := row.TextArea.Drawer.(*drawer4.Drawer)
		want := d.InnerBounds().Dx()
		if want <= 0 {
			t.Fatalf("visible textarea width=%d", want)
		}
		if got := row.PotentialTextAreaInnerWidth(); got != want {
			t.Errorf("shadows=%v: got width=%d, want %d", shadowsOn, got, want)
		}

		row.TextArea.Bounds = image.ZR
		d.SetBounds(image.ZR)
		if got := row.PotentialTextAreaInnerWidth(); got != want {
			t.Errorf("shadows=%v, collapsed: got width=%d, want %d", shadowsOn, got, want)
		}
		ui.Close()
	}
}

func TestGoodRowPosAvoidsExecutingRowWhenSpaceAvailable(t *testing.T) {
	colorTheme, fontTheme := ColorThemeCycler.CurName, FontThemeCycler.CurName
	defer func() { ColorThemeCycler.CurName, FontThemeCycler.CurName = colorTheme, fontTheme }()
	ColorThemeCycler.CurName, FontThemeCycler.CurName = "light", "regular"

	win := &rowWheelTestWindow{img: image.NewRGBA(image.Rect(0, 0, 800, 600)), closed: make(chan struct{})}
	ui, err := NewUIWithWindow("row placement test", win)
	if err != nil {
		t.Fatal(err)
	}
	defer ui.Close()
	ui.HandleEvent(&event.WindowResize{Rect: win.img.Bounds()})
	ui.LayoutMarkedAndSchedulePaint()

	runningCol := ui.Root.Cols.NewColumn()
	otherCol := ui.Root.Cols.NewColumn()
	ui.Root.Cols.ColsLayout.Spl.Resize(otherCol, 0.8)
	running := runningCol.NewRowBefore(nil)
	running.SetState(RowStateExecuting, true)
	ui.LayoutMarkedAndSchedulePaint()
	b := ui.rowInsertionBounds(running, true)
	runningArea := b.Dx() * b.Dy()
	otherArea := otherCol.Bounds.Dx() * otherCol.Bounds.Dy()
	if runningArea <= otherArea {
		t.Fatalf("test setup needs the executing area to be larger: %d <= %d", runningArea, otherArea)
	}

	if got := ui.GoodRowPos().Column; got != otherCol {
		t.Fatalf("selected executing column %p instead of available column %p", got, otherCol)
	}

	otherCol.Close()
	ui.LayoutMarkedAndSchedulePaint()
	if got := ui.GoodRowPos().Column; got != runningCol {
		t.Fatalf("fallback selected column %p, want %p", got, runningCol)
	}
}
