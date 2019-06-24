package main

import (
	"path/filepath"

	"github.com/jmigpin/editor/core"
)

var h Handler

func OnLoad(ed *core.Editor) {
	h.ed = ed
	// Register for editor events. Use return value to unregister.
	_ = ed.EEvents.Register(core.PostNewERowEEventId, h.onEvent1)
	_ = ed.EEvents.Register(core.PostFileSaveEEventId, h.onEvent1)
	_ = ed.EEvents.Register(core.PreRowCloseEEventId, h.onEvent1)
	_ = ed.EEvents.Register(core.RowStateChangeEEventId, h.onEvent2)
}

//----------

type Handler struct {
	ed *core.Editor
}

func (h *Handler) onEvent1(ev interface{}) {
	h.ed.Messagef("handler1: %T\n", ev)
}
func (h *Handler) onEvent2(ev interface{}) {
	h.ed.Messagef("handler2: %T\n", ev)

	e := ev.(*core.RowStateChangeEEvent)
	name := filepath.Base(e.ERow.Info.Name())
	h.ed.Messagef("handler2: %T, %p, %v, %v, %v\n", ev, e.ERow, name, e.State, e.Value)
}
