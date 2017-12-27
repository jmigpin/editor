package cmdutil

import (
	"image"
	"net/url"
	"strings"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/uiutil/event"
)

type DndHandler struct {
	ed Editorer
}

func NewDndHandler(ed Editorer) *DndHandler {
	return &DndHandler{ed}
}
func (h *DndHandler) OnPosition(ev *event.DndPosition) {
	// dnd position must receive a reply
	ev.Reply(h.onPosition3(ev))
}
func (h *DndHandler) onPosition3(ev *event.DndPosition) event.DndAction {
	// must drop on a column
	_, ok := h.columnAtPoint(&ev.Point)
	if !ok {
		return event.DenyDndA
	}
	// supported types
	for _, t := range ev.Types {
		if t == event.TextURLListDndT {
			return event.PrivateDndA
		}
	}
	return event.DenyDndA
}

func (h *DndHandler) OnDrop(ev *event.DndDrop) {
	// The drop event might need to request data (send and then receive an event). To receive that event, the main eventloop can't be blocking with this procedure
	go func() {
		v := h.onDrop3(ev)
		ev.ReplyAccept(v)
		if v {
			h.ed.UI().RequestPaint()
		}
	}()
}
func (h *DndHandler) onDrop3(ev *event.DndDrop) bool {
	// find column that matches
	col, ok := h.columnAtPoint(&ev.Point)
	if !ok {
		return false
	}
	// get data in required format
	data, err := ev.RequestData(event.TextURLListDndT)
	if err != nil {
		h.ed.Error(err)
		return false
	}
	// parse data
	urls, err := parseAsTextURLList(data)
	if err != nil {
		h.ed.Error(err)
		return false
	}

	h.handleDroppedURLs(col, &ev.Point, urls)
	return true
}

//func SetupDragNDrop(ed Editorer) {
//	h := &DndHandler{ed}
//	ui := ed.UI()
//	ui.EvReg.Add(dragndrop.ErrorEventId, h.onError)
//	ui.EvReg.Add(dragndrop.PositionEventId, h.onPosition)
//	ui.EvReg.Add(dragndrop.DropEventId, h.onDrop)
//}
//func (h *DndHandler) onPosition(ev0 interface{}) {
//	ev := ev0.(*dragndrop.PositionEvent)
//	// dnd position must receive a reply
//	action, ok := h.onPosition2(ev)
//	//log.Printf("position %v %v\n", action, ok)
//	if !ok {
//		ev.ReplyDeny()
//	} else {
//		ev.ReplyAccept(action)
//	}
//}

//func (h *DndHandler) onPosition2(ev *dragndrop.PositionEvent) (xproto.Atom, bool) {
//	// get event point
//	p, err := ev.WindowPoint()
//	if err != nil {
//		return 0, false
//	}
//	// only supporting dnd on columns
//	// find column that matches
//	_, ok := h.columnAtPoint(p)
//	if !ok {
//		return 0, false
//	}
//	// supported types
//	ok = false
//	types := []xproto.Atom{dragndrop.DropTypeAtoms.TextURLList}
//	for _, t := range types {
//		if ev.SupportsType(t) {
//			ok = true
//			break
//		}
//	}
//	if !ok {
//		return 0, false
//	}
//	// reply accept with action
//	action := dragndrop.DndAtoms.XdndActionCopy
//	return action, true
//}

//func (h *DndHandler) onError(ev0 interface{}) {
//	err := ev0.(error)
//	h.ed.Error(err)
//}
//func (h *DndHandler) onDrop(ev0 interface{}) {
//	// the drop event needs to send and then receive an event - to receive that event, the main eventloop can't be blocking with this procedure
//	go func() {
//		ev := ev0.(*dragndrop.DropEvent)
//		// dnd drop must receive a reply
//		ok := h.onDrop2(ev)
//		if !ok {
//			ev.ReplyDeny()
//		} else {
//			ev.ReplyAccepted()
//			// running on goroutine, must request paint
//			h.ed.UI().RequestPaint()
//		}
//	}()
//}
//func (h *DndHandler) onDrop2(ev *dragndrop.DropEvent) bool {
//	// get event point
//	p, err := ev.WindowPoint()
//	if err != nil {
//		h.ed.Error(err)
//		return false
//	}
//	// find column that matches
//	col, ok := h.columnAtPoint(p)
//	if !ok {
//		return false
//	}
//	// get data in required format
//	data, err := ev.RequestData(dragndrop.DropTypeAtoms.TextURLList)
//	if err != nil {
//		h.ed.Error(err)
//		return false
//	}
//	// parse data
//	urls, err := parseAsTextURLList(data)
//	if err != nil {
//		h.ed.Error(err)
//		return false
//	}

//	h.handleDroppedURLs(col, p, urls)
//	return true
//}

func (h *DndHandler) columnAtPoint(p *image.Point) (*ui.Column, bool) {
	for _, col := range h.ed.UI().Layout.Cols.Columns() {
		if p.In(col.Bounds) {
			return col, true
		}
	}
	return nil, false
}

func (h *DndHandler) handleDroppedURLs(col *ui.Column, p *image.Point, urls []*url.URL) {
	for _, u := range urls {
		if u.Scheme == "file" {
			h.handleDroppedURL(col, p, u)
		}
	}
}
func (h *DndHandler) handleDroppedURL(col *ui.Column, p *image.Point, u *url.URL) {
	var nextRow *ui.Row
	if row, ok := col.PointRow(p); ok {
		// next row for insertion
		r, ok := row.NextRow()
		if ok {
			nextRow = r
		}
	}
	erow := h.ed.NewERowerBeforeRow(u.Path, col, nextRow)
	err := erow.LoadContentClear()
	if err != nil {
		h.ed.Error(err)
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
