package xcursors

import (
	"fmt"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/render"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/driver/xdriver/xcursors/xcur"
)

func (cs *Cursors) initTheme() {
	if err := render.Init(cs.conn); err != nil {
		return
	}
	pictFormat, err := findARGB32PictFormat(cs.conn)
	if err != nil {
		return
	}
	theme, err := xcur.LoadThemeFromEnv()
	if err != nil {
		return
	}
	cs.pictFormat = pictFormat
	cs.theme = theme
	cs.themeSize = xcur.SizeFromEnv()
}

func (cs *Cursors) loadThemeCursor(c Cursor) (xproto.Cursor, error) {
	if c == XCNone {
		return 0, nil
	}
	if cs.theme == nil {
		return 0, fmt.Errorf("xcursor theme not available")
	}
	imgs := cs.theme.Images(c.xcursorNames(), cs.themeSize)
	if len(imgs) == 0 {
		return 0, fmt.Errorf("xcursor theme cursor not found: %v", c)
	}
	if len(imgs) == 1 {
		return cs.loadThemeImage(imgs[0])
	}
	return cs.loadThemeAnimImages(imgs)
}

func (cs *Cursors) loadThemeAnimImages(imgs []*xcur.Image) (xproto.Cursor, error) {
	elts := make([]render.Animcursorelt, 0, len(imgs))
	for _, img := range imgs {
		cursor, err := cs.loadThemeImage(img)
		if err != nil {
			return 0, err
		}
		elts = append(elts, render.Animcursorelt{
			Cursor: cursor,
			Delay:  uint32(img.Delay / time.Millisecond),
		})
	}

	cursor, err := xproto.NewCursorId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := render.CreateAnimCursorChecked(cs.conn, cursor, elts).Check(); err != nil {
		return 0, err
	}
	return cursor, nil
}

func (cs *Cursors) loadThemeImage(img *xcur.Image) (xproto.Cursor, error) {
	b := img.Bounds
	w, h := b.Dx(), b.Dy()

	pixmap, err := xproto.NewPixmapId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := xproto.CreatePixmapChecked(cs.conn, 32, pixmap, xproto.Drawable(cs.win), uint16(w), uint16(h)).Check(); err != nil {
		return 0, err
	}
	defer xproto.FreePixmap(cs.conn, pixmap)

	gc, err := xproto.NewGcontextId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := xproto.CreateGCChecked(cs.conn, gc, xproto.Drawable(pixmap), 0, nil).Check(); err != nil {
		return 0, err
	}
	defer xproto.FreeGC(cs.conn, gc)

	if err := xproto.PutImageChecked(
		cs.conn,
		xproto.ImageFormatZPixmap,
		xproto.Drawable(pixmap),
		gc,
		uint16(w), uint16(h),
		0, 0,
		0,
		32,
		img.PixARGB).Check(); err != nil {
		return 0, err
	}

	picture, err := render.NewPictureId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := render.CreatePictureChecked(cs.conn, picture, xproto.Drawable(pixmap), cs.pictFormat, 0, nil).Check(); err != nil {
		return 0, err
	}
	defer render.FreePicture(cs.conn, picture)

	cursor, err := xproto.NewCursorId(cs.conn)
	if err != nil {
		return 0, err
	}
	if err := render.CreateCursorChecked(cs.conn, cursor, picture, uint16(img.Hot.X), uint16(img.Hot.Y)).Check(); err != nil {
		return 0, err
	}
	return cursor, nil
}

func findARGB32PictFormat(conn *xgb.Conn) (render.Pictformat, error) {
	reply, err := render.QueryPictFormats(conn).Reply()
	if err != nil {
		return 0, err
	}
	for _, f := range reply.Formats {
		if isARGB32PictFormat(f) {
			return f.Id, nil
		}
	}
	return 0, fmt.Errorf("argb32 pict format not found")
}

func isARGB32PictFormat(f render.Pictforminfo) bool {
	return f.Type == render.PictTypeDirect &&
		f.Depth == 32 &&
		f.Direct.RedShift == 16 &&
		f.Direct.RedMask == 0xff &&
		f.Direct.GreenShift == 8 &&
		f.Direct.GreenMask == 0xff &&
		f.Direct.BlueShift == 0 &&
		f.Direct.BlueMask == 0xff &&
		f.Direct.AlphaShift == 24 &&
		f.Direct.AlphaMask == 0xff
}
