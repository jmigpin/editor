package cmdutil

import (
	"image"
	"net/url"
	"strings"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xutil/dragndrop"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

func SetupDragNDrop(ed Editori) {
	h := &dndHandler{ed}
	ui := ed.UI()
	fn := &xgbutil.ERCallback{h.onError}
	ui.Win.EvReg.Add(dragndrop.ErrorEventId, fn)
	fn = &xgbutil.ERCallback{h.onPosition}
	ui.Win.EvReg.Add(dragndrop.PositionEventId, fn)
	fn = &xgbutil.ERCallback{h.onDrop}
	ui.Win.EvReg.Add(dragndrop.DropEventId, fn)
}

type dndHandler struct {
	ed Editori
}

func (h *dndHandler) onError(ev0 xgbutil.EREvent) {
	err := ev0.(error)
	h.ed.Error(err)
}
func (h *dndHandler) onPosition(ev0 xgbutil.EREvent) {
	ev := ev0.(*dragndrop.PositionEvent)
	// get event point
	p, err := ev.WindowPoint()
	if err != nil {
		h.ed.Error(err)
		return
	}
	// only supporting dnd on columns
	// find column that matches
	_, ok := h.columnAtPoint(p)
	if !ok {
		// dnd position must receive a reply
		ev.ReplyDeny()
		return
	}
	// supported types
	ok = false
	types := []xproto.Atom{dragndrop.DropTypeAtoms.TextURLList}
	for _, t := range types {
		if ev.SupportsType(t) {
			ok = true
			break
		}
	}
	if ok {
		// TODO: if ctrl is pressed, set to XdndActionLink
		// reply accept with action
		action := dragndrop.DndAtoms.XdndActionCopy
		ev.ReplyAccept(action)
	}
}
func (h *dndHandler) columnAtPoint(p *image.Point) (*ui.Column, bool) {
	for _, col := range h.ed.UI().Layout.Cols.Cols {
		if p.In(col.C.Bounds) {
			return col, true
		}
	}
	return nil, false
}
func (h *dndHandler) onDrop(ev0 xgbutil.EREvent) {
	ev := ev0.(*dragndrop.DropEvent)
	// get event point
	p, err := ev.WindowPoint()
	if err != nil {
		h.ed.Error(err)
		return
	}
	// find column that matches
	col, ok := h.columnAtPoint(p)
	if !ok {
		// dnd position must receive a reply
		ev.ReplyDeny()
		return
	}
	// get data in required format
	data, err := ev.RequestData(dragndrop.DropTypeAtoms.TextURLList)
	if err != nil {
		ev.ReplyDeny()
		h.ed.Error(err)
		return
	}
	// parse data
	urls, err := parseAsTextURLList(data)
	if err != nil {
		ev.ReplyDeny()
		h.ed.Error(err)
		return
	}

	h.handleDroppedURLs(col, p, urls)
	ev.ReplyAccepted()
}
func (h *dndHandler) handleDroppedURLs(col *ui.Column, p *image.Point, urls []*url.URL) {
	for _, u := range urls {
		if u.Scheme == "file" {
			// calculate position before the row is inserted if the row doesn't exist
			var c *ui.Column
			var i int
			posCalc := false
			_, ok := h.ed.FindRow(u.Path)
			if !ok {
				c, i, ok = col.Cols.PointRowPosition(nil, p)
				if !ok {
					continue
				}
				posCalc = true
			}
			// find/create
			row, err := h.ed.FindRowOrCreateInColFromFilepath(u.Path, col)
			if err != nil {
				h.ed.Error(err)
				continue
			}
			// calculate if not calculated yet
			if !posCalc {
				c, i, ok = col.Cols.PointRowPosition(row, p)
				if !ok {
					continue
				}
			}
			// move row
			col.Cols.MoveRowToColumn(row, c, i)
		}
	}
}

func parseAsTextURLList(data []byte) ([]*url.URL, error) {
	s := string(data)
	entries := strings.Split(s, "\n")
	var urls []*url.URL
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		u, err := url.Parse(e)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}
