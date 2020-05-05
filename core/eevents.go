package core

import (
	"github.com/jmigpin/editor/v2/ui"
	"github.com/jmigpin/editor/v2/util/evreg"
)

// Editor events.
type EEvents struct {
	reg evreg.Register
}

func NewEEvents() *EEvents {
	return &EEvents{}
}

func (eevs *EEvents) emit(eid EEventId, ev interface{}) int {
	return eevs.reg.RunCallbacks(int(eid), ev)
}

func (eevs *EEvents) Register(eid EEventId, fn func(interface{})) *evreg.Regist {
	return eevs.reg.Add(int(eid), fn)
}

//----------

type EEventId int

const (
	PostNewERowEEventId EEventId = iota
	PostFileSaveEEventId
	PreRowCloseEEventId
	RowStateChangeEEventId
)

type PostNewERowEEvent struct {
	ERow *ERow
}

type PostFileSaveEEvent struct {
	Info *ERowInfo
}

type PreRowCloseEEvent struct {
	ERow *ERow
}

type RowStateChangeEEvent struct {
	ERow  *ERow // duplicate rows also emit state change events.
	State ui.RowState
	Value bool // the new value
}
