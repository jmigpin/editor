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
