package ui

import (
	"image"
	"strings"
	"sync"
	"testing"

	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestRowWheelResizeDoesNotScrollPreviousRow(t *testing.T) {
	colorTheme, fontTheme := ColorThemeCycler.CurName, FontThemeCycler.CurName
	ColorThemeCycler.CurName, FontThemeCycler.CurName = "light", "regular"
	defer func() { ColorThemeCycler.CurName, FontThemeCycler.CurName = colorTheme, fontTheme }()

	win := &rowWheelTestWindow{img: image.NewRGBA(image.Rect(0, 0, 800, 900)), closed: make(chan struct{})}
	ui, err := NewUIWithWindow("row wheel test", win)
	if err != nil {
		t.Fatal(err)
	}
	defer ui.Close()
	ui.HandleEvent(&event.WindowResize{Rect: win.img.Bounds()})
	ui.LayoutMarkedAndSchedulePaint()

	col := ui.Root.Cols.NewColumn()
	upper := col.NewRowBefore(nil)
	lower := col.NewRowBefore(nil)
	if err := upper.TextArea.SetStr(strings.Repeat("scrollable line\n", 200)); err != nil {
		t.Fatal(err)
	}
	rootFace := ui.Root.TreeThemeFontFace()
	localOpts := rootFace.Opts
	localOpts.SetSize(localOpts.Size() / 2)
	lower.TextArea.SetThemeFontFace(rootFace.Font.FontFace(localOpts))
	col.RowsLayout.Spl.Resize(lower, 0.5)
	ui.LayoutMarkedAndSchedulePaint()

	p := lower.Toolbar.Square.Bounds.Min.Add(image.Pt(6, 2))
	if !p.In(lower.Toolbar.Square.Bounds) {
		t.Fatalf("square bounds %v, point %v", lower.Toolbar.Square.Bounds, p)
	}
	initialBoundary := lower.Bounds.Min.Y
	initialScroll := upper.TextArea.ScrollOffset()
	jump := rootFace.LineHeightInt() * 4
	if jump == lower.TextArea.LineHeight()*4 {
		t.Fatal("local textarea font did not differ from root font")
	}
	wheel := func() {
		md := &event.MouseDown{Point: p, Button: event.ButtonWheelDown}
		ui.HandleEvent(&event.WindowInput{Point: p, Event: md})
		mu := &event.MouseUp{Point: p, Button: event.ButtonWheelDown}
		ui.HandleEvent(&event.WindowInput{Point: p, Event: mu})
		ui.LayoutMarkedAndSchedulePaint()
	}
	wheel()
	if lower.Bounds.Min.Y <= initialBoundary {
		t.Fatalf("first wheel did not move boundary: %d -> %d", initialBoundary, lower.Bounds.Min.Y)
	}
	nearOldPoint := p.Add(image.Pt(2, 0))
	ui.HandleEvent(&event.WindowInput{Point: nearOldPoint, Event: &event.MouseMove{Point: nearOldPoint}})
	if ui.rowWheel.state == nil {
		t.Fatal("small movement before warp ended wheel resize")
	}
	wheel()
	wheel()
	if got := lower.Bounds.Min.Y - initialBoundary; got < 3*jump-1 || got > 3*jump+1 {
		t.Errorf("three wheels moved boundary: %d -> %d, want +%d", initialBoundary, lower.Bounds.Min.Y, 3*jump)
	}
	if got := upper.TextArea.ScrollOffset(); got != initialScroll {
		t.Errorf("upper row scrolled: %v -> %v", initialScroll, got)
	}

	win.mu.Lock()
	warpPoint := win.warpPoint
	win.mu.Unlock()
	ui.HandleEvent(&event.WindowInput{Point: warpPoint, Event: &event.MouseMove{Point: warpPoint}})
	if ui.rowWheel.state == nil {
		t.Fatal("warp-generated movement ended wheel resize")
	}
	currentSquare := lower.Toolbar.Square.Bounds.Min.Add(image.Pt(6, 2))
	boundary := lower.Bounds.Min.Y
	ui.HandleEvent(&event.WindowInput{Point: currentSquare, Event: &event.MouseDown{Point: currentSquare, Button: event.ButtonWheelDown}})
	ui.LayoutMarkedAndSchedulePaint()
	if got := lower.Bounds.Min.Y - boundary; got < jump-1 || got > jump+1 {
		t.Errorf("wheel at warped position moved boundary: %d -> %d, want +%d", boundary, got, jump)
	}
	ui.HandleEvent(&event.WindowInput{Point: p, Event: &event.MouseMove{Point: p}})
	if ui.rowWheel.state != nil {
		t.Fatal("physical movement did not end wheel resize")
	}
	boundary = lower.Bounds.Min.Y
	wheel()
	if lower.Bounds.Min.Y != boundary {
		t.Errorf("wheel after physical movement resized row: %d -> %d", boundary, lower.Bounds.Min.Y)
	}
	if upper.TextArea.ScrollOffset() == initialScroll {
		t.Error("wheel after physical movement did not scroll upper row")
	}

	currentSquare = lower.Toolbar.Square.Bounds.Min.Add(image.Pt(6, 2))
	ui.HandleEvent(&event.WindowInput{Point: currentSquare, Event: &event.MouseDown{Point: currentSquare, Button: event.ButtonWheelDown}})
	ui.LayoutMarkedAndSchedulePaint()
	if ui.rowWheel.state == nil {
		t.Fatal("wheel on current square did not start another resize")
	}
	clickPoint := upper.TextArea.Bounds.Min.Add(image.Pt(30, 30))
	ui.HandleEvent(&event.WindowInput{Point: clickPoint, Event: &event.MouseDown{Point: clickPoint, Button: event.ButtonLeft}})
	ui.HandleEvent(&event.WindowInput{Point: clickPoint, Event: &event.MouseUp{Point: clickPoint, Button: event.ButtonLeft}})
	if ui.rowWheel.state != nil {
		t.Error("left button did not end wheel resize")
	}
	ui.LayoutMarkedAndSchedulePaint()

	sepPoint := image.Pt(20, lower.Bounds.Min.Y)
	boundary = lower.Bounds.Min.Y
	for range 2 {
		ui.HandleEvent(&event.WindowInput{Point: sepPoint, Event: &event.MouseDown{Point: sepPoint, Button: event.ButtonWheelDown}})
		ui.LayoutMarkedAndSchedulePaint()
	}
	if got := lower.Bounds.Min.Y - boundary; got < 2*jump-1 || got > 2*jump+1 {
		t.Errorf("separator wheels moved boundary: %d -> %d, want +%d", boundary, got, 2*jump)
	}
	if ui.rowWheel.state == nil {
		t.Fatal("separator resize mode ended unexpectedly")
	}
	outside := image.Pt(30, upper.TextArea.Bounds.Min.Y+30)
	beforeOutsideScroll := upper.TextArea.ScrollOffset()
	ui.HandleEvent(&event.WindowInput{Point: outside, Event: &event.MouseDown{Point: outside, Button: event.ButtonWheelDown}})
	if ui.rowWheel.state != nil {
		t.Error("wheel outside resize zone did not end mode")
	}
	if upper.TextArea.ScrollOffset() == beforeOutsideScroll {
		t.Error("wheel outside resize zone did not scroll upper row")
	}
}

type rowWheelTestWindow struct {
	mu        sync.Mutex
	img       *image.RGBA
	warpPoint image.Point
	closed    chan struct{}
	once      sync.Once
}

func (win *rowWheelTestWindow) NextEvent() (event.Event, bool) {
	<-win.closed
	return nil, false
}

func (win *rowWheelTestWindow) Request(req event.Request) error {
	win.mu.Lock()
	defer win.mu.Unlock()
	switch r := req.(type) {
	case *event.ReqImageResize:
		win.img = image.NewRGBA(r.Rect)
	case *event.ReqImage:
		r.ReplyImg = win.img
	case *event.ReqPointerWarp:
		win.warpPoint = r.P
	case *event.ReqClose:
		win.once.Do(func() { close(win.closed) })
	}
	return nil
}
