package cmdutil

import (
	"image"
	"net/url"
	"strings"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/uiutil/event"
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
	next, ok := col.PointNextRow(p)
	if !ok {
		next = nil
	}
	erow := h.ed.NewERowerBeforeRow(u.Path, col, next)
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
