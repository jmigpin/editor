package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type rowWheelGesture struct {
	state *rowWheelState
}

func (g *rowWheelGesture) reset() {
	g.state = nil
}

func (g *rowWheelGesture) endRow(row *Row) {
	if g.state != nil && g.state.row == row {
		g.reset()
	}
}

func (g *rowWheelGesture) begin(row *Row, source widget.Node, p image.Point) {
	if g.state != nil {
		return
	}
	g.state = &rowWheelState{
		row:          row,
		source:       source,
		sourceBounds: source.Embed().Bounds,
		zone:         source.Embed().Bounds,
		warpPoints:   []image.Point{p},
	}
}

func (g *rowWheelGesture) noteWarp(row *Row, dy int, p image.Point) {
	s := g.state
	if s == nil || s.row != row {
		return
	}
	s.sourceBounds = s.sourceBounds.Add(image.Pt(0, dy))
	s.zone = s.zone.Union(s.sourceBounds)
	for i, q := range s.warpPoints {
		if q == p {
			s.warpPoints = append(s.warpPoints[:i], s.warpPoints[i+1:]...)
			break
		}
	}
	s.warpPoints = append(s.warpPoints, p)
}

func (g *rowWheelGesture) filter(wi *event.WindowInput) {
	s := g.state
	if s == nil {
		return
	}
	if s.row.Col == nil {
		g.reset()
		return
	}

	switch ev := wi.Event.(type) {
	case *event.MouseDown:
		if !rowWheelCanBegin(ev) {
			g.reset()
			return
		}
		if s.source.Embed().Bounds != s.sourceBounds || !s.contains(wi.Point) {
			g.reset()
			return
		}
		p := image.Pt(wi.Point.X, s.row.Bounds.Min.Y)
		if _, ok := s.source.(*RowSquare); ok {
			p.Y++
		}
		wi.Point = p
		ev.Point = p
	case *event.MouseMove:
		if !s.isWarpMove(wi.Point) {
			g.reset()
		}
	case *event.KeyDown:
		g.reset()
	}
}

//----------

type rowWheelState struct {
	row          *Row
	source       widget.Node
	sourceBounds image.Rectangle
	zone         image.Rectangle // includes old positions while wheel events are queued
	warpPoints   []image.Point   // movement events can be coalesced by MoveFilter
}

func (s *rowWheelState) contains(p image.Point) bool {
	return p.In(s.zone)
}

func (s *rowWheelState) isWarpMove(p image.Point) bool {
	for i, q := range s.warpPoints {
		if rowWheelAbs(p.X-q.X) <= 4 && rowWheelAbs(p.Y-q.Y) <= 4 {
			s.warpPoints = s.warpPoints[i:]
			return true
		}
	}
	return false
}

//----------

func rowWheelCanBegin(ev *event.MouseDown) bool {
	return (ev.Button == event.ButtonWheelUp || ev.Button == event.ButtonWheelDown) && ev.Mods.ClearLocks().IsEmpty()
}

func rowWheelAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
