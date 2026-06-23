package xcursors

import (
	"image/color"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/render"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/driver/xdriver/xcursors/xcur"
	"github.com/jmigpin/editor/util/imageutil"
)

// https://tronche.com/gui/x/xlib/appendix/b/

type Cursors struct {
	conn         *xgb.Conn
	win          xproto.Window
	m            map[Cursor]xproto.Cursor
	hiddenCursor xproto.Cursor
	theme        *xcur.Theme
	themeSize    int
	themeM       map[Cursor]xproto.Cursor
	pictFormat   render.Pictformat
}

func NewCursors(conn *xgb.Conn, win xproto.Window) (*Cursors, error) {
	cs := &Cursors{
		conn:   conn,
		win:    win,
		m:      make(map[Cursor]xproto.Cursor),
		themeM: make(map[Cursor]xproto.Cursor),
	}
	cs.initTheme()
	return cs, nil
}
func (cs *Cursors) SetCursor(c Cursor) error {
	xc, err := cs.cursor(c)
	if err != nil {
		return err
	}
	return cs.setCursorId(xc)
}

func (cs *Cursors) SetDefaultCursor() error {
	return cs.setCursorId(0)
}

func (cs *Cursors) SetHiddenCursor() error {
	cursor, err := cs.hiddenCursorId()
	if err != nil {
		return err
	}
	return cs.setCursorId(cursor)
}

func (cs *Cursors) setCursorId(cursor xproto.Cursor) error {
	mask := uint32(xproto.CwCursor)
	values := []uint32{uint32(cursor)}
	_ = xproto.ChangeWindowAttributes(cs.conn, cs.win, mask, values)
	return nil
}

//----------

func (cs *Cursors) cursor(c Cursor) (xproto.Cursor, error) {
	if xc, ok := cs.themeM[c]; ok {
		return xc, nil
	}
	if xc, err := cs.loadThemeCursor(c); err == nil {
		cs.themeM[c] = xc
		return xc, nil
	}

	xc, ok := cs.m[c]
	if !ok {
		xc2, err := cs.loadCursor(c)
		if err != nil {
			return 0, err
		}
		cs.m[c] = xc2
		xc = xc2
	}
	return xc, nil
}

func (cs *Cursors) loadCursor(c Cursor) (xproto.Cursor, error) {
	return cs.loadCursor2(c, color.Black, color.White)
}
func (cs *Cursors) loadCursor2(c Cursor, fg, bg color.Color) (xproto.Cursor, error) {
	fontId, err := xproto.NewFontId(cs.conn)
	if err != nil {
		return 0, err
	}
	cursor, err := xproto.NewCursorId(cs.conn)
	if err != nil {
		return 0, err
	}
	name := "cursor"
	err = xproto.OpenFontChecked(cs.conn, fontId, uint16(len(name)), name).Check()
	if err != nil {
		return 0, err
	}

	// colors
	ur, ug, ub, _ := imageutil.ColorUint16s(fg)
	vr, vg, vb, _ := imageutil.ColorUint16s(bg)

	err = xproto.CreateGlyphCursorChecked(
		cs.conn, cursor,
		fontId, fontId,
		uint16(c), uint16(c)+1,
		ur, ug, ub,
		vr, vg, vb).Check()
	if err != nil {
		return 0, err
	}

	err = xproto.CloseFontChecked(cs.conn, fontId).Check()
	if err != nil {
		return 0, err
	}

	return cursor, nil
}

func (cs *Cursors) hiddenCursorId() (xproto.Cursor, error) {
	if cs.hiddenCursor != 0 {
		return cs.hiddenCursor, nil
	}
	cursor, err := cs.loadHiddenCursor()
	if err != nil {
		return 0, err
	}
	cs.hiddenCursor = cursor
	return cursor, nil
}

func (cs *Cursors) loadHiddenCursor() (xproto.Cursor, error) {
	pixmap, err := xproto.NewPixmapId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := xproto.CreatePixmapChecked(cs.conn, 1, pixmap, xproto.Drawable(cs.win), 1, 1).Check(); err != nil {
		return 0, err
	}
	defer xproto.FreePixmap(cs.conn, pixmap)

	gc, err := xproto.NewGcontextId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := xproto.CreateGCChecked(cs.conn, gc, xproto.Drawable(pixmap), xproto.GcForeground, []uint32{0}).Check(); err != nil {
		return 0, err
	}
	defer xproto.FreeGC(cs.conn, gc)

	rect := xproto.Rectangle{X: 0, Y: 0, Width: 1, Height: 1}
	if err := xproto.PolyFillRectangleChecked(cs.conn, xproto.Drawable(pixmap), gc, []xproto.Rectangle{rect}).Check(); err != nil {
		return 0, err
	}

	cursor, err := xproto.NewCursorId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := xproto.CreateCursorChecked(cs.conn, cursor, pixmap, pixmap, 0, 0, 0, 0, 0, 0, 0, 0).Check(); err != nil {
		return 0, err
	}
	return cursor, nil
}

func (c Cursor) xcursorNames() []string {
	switch c {
	case SBVDoubleArrow:
		return []string{"sb_v_double_arrow", "ns-resize", "size_ver"}
	case SBHDoubleArrow:
		return []string{"sb_h_double_arrow", "ew-resize", "size_hor"}
	case XCursor:
		return []string{"X_cursor", "cross", "crosshair"}
	case Fleur:
		return []string{"fleur", "move", "all-scroll"}
	case Hand2:
		return []string{"hand2", "pointer", "hand1"}
	case XTerm:
		return []string{"xterm", "text"}
	case Watch:
		return []string{"watch", "wait"}
	default:
		return nil
	}
}

//----------

type Cursor uint16

const (
	XCursor           = 0
	Arrow             = 2
	BasedArrowDown    = 4
	BasedArrowUp      = 6
	Boat              = 8
	Bogosity          = 10
	BottomLeftCorner  = 12
	BottomRightCorner = 14
	BottomSide        = 16
	BottomTee         = 18
	BoxSpiral         = 20
	CenterPtr         = 22
	Circle            = 24
	Clock             = 26
	CoffeeMug         = 28
	Cross             = 30
	CrossReverse      = 32
	Crosshair         = 34
	DiamondCross      = 36
	Dot               = 38
	DotBoxMask        = 40
	DoubleArrow       = 42
	DraftLarge        = 44
	DraftSmall        = 46
	DrapedBox         = 48
	Exchange          = 50
	Fleur             = 52
	Gobbler           = 54
	Gumby             = 56
	Hand1             = 58
	Hand2             = 60
	Heart             = 62
	Icon              = 64
	IronCross         = 66
	LeftPtr           = 68
	LeftSide          = 70
	LeftTee           = 72
	LeftButton        = 74
	LLAngle           = 76
	LRAngle           = 78
	Man               = 80
	MiddleButton      = 82
	Mouse             = 84
	Pencil            = 86
	Pirate            = 88
	Plus              = 90
	QuestionArrow     = 92
	RightPtr          = 94
	RightSide         = 96
	RightTee          = 98
	RightButton       = 100
	RtlLogo           = 102
	Sailboat          = 104
	SBDownArrow       = 106
	SBHDoubleArrow    = 108
	SBLeftArrow       = 110
	SBRightArrow      = 112
	SBUpArrow         = 114
	SBVDoubleArrow    = 116
	Shuttle           = 118
	Sizing            = 120
	Spider            = 122
	Spraycan          = 124
	Star              = 126
	Target            = 128
	TCross            = 130
	TopLeftArrow      = 132
	TopLeftCorner     = 134
	TopRightCorner    = 136
	TopSide           = 138
	TopTee            = 140
	Trek              = 142
	ULAngle           = 144
	Umbrella          = 146
	URAngle           = 148
	Watch             = 150
	XTerm             = 152
)
