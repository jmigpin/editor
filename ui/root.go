package ui

import (
	"image"

	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

// User Interface root (top) node.
type Root struct {
	*widget.MultiLayer
	UI              *UI
	Toolbar         *Toolbar
	MainMenuButton  *MainMenuButton
	ContextFloatBox *ContextFloatBox
	Cols            *Columns
	EvReg           *evreg.Register
}

func NewRoot(ui *UI) *Root {
	return &Root{MultiLayer: widget.NewMultiLayer(), UI: ui, EvReg: evreg.NewRegister()}
}

func (root *Root) Init() {
	//  background layer
	bgLayer := widget.NewBoxLayout()
	bgLayer.YAxis = true
	root.BgLayer.Append(bgLayer)

	// background layer
	{
		// top toolbar
		{
			ttb := widget.NewBoxLayout()
			bgLayer.Append(ttb)

			// toolbar
			root.Toolbar = NewToolbar(root.UI)
			ttb.Append(root.Toolbar)
			ttb.SetChildFlex(root.Toolbar, true, false)

			// main menu button
			mmb := NewMainMenuButton(root)
			mmb.Label.Border.Left = 1
			ttb.Append(mmb)
			ttb.SetChildFill(mmb, false, true)
			root.MainMenuButton = mmb
		}

		// columns
		root.Cols = NewColumns(root)
		bgLayer.Append(root.Cols)
	}

	root.ContextFloatBox = NewContextFloatBox(root)
}

func (l *Root) OnChildMarked(child widget.Node, newMarks widget.Marks) {
	l.MultiLayer.OnChildMarked(child, newMarks)
	// dynamic toolbar
	if l.Toolbar != nil && l.Toolbar.HasAnyMarks(widget.MarkNeedsLayout) {
		l.BgLayer.MarkNeedsLayout()
	}
}

//----------

func (l *Root) OnInputEvent(ev0 interface{}, p image.Point) event.Handled {
	switch ev := ev0.(type) {
	case *event.KeyDown:
		m := ev.Mods.ClearLocks()
		switch {
		case m.Is(event.ModCtrl):
			switch ev.KeySym {
			case event.KSymF4:
				l.selAnnEv(RootSelAnnTypeFirst)
				return event.HTrue
			case event.KSymF5:
				l.selAnnEv(RootSelAnnTypeLast)
				return event.HTrue
			case event.KSymF9:
				l.selAnnEv(RootSelAnnTypeClear)
				return event.HTrue
			}
		}
	case *event.MouseDown:
		m := ev.Mods.ClearLocks()
		switch {
		case m.Is(event.ModCtrl):
			switch ev.Button {
			case event.ButtonWheelUp:
				l.selAnnEv(RootSelAnnTypePrev)
				return event.HTrue
			case event.ButtonWheelDown:
				l.selAnnEv(RootSelAnnTypeNext)
				return event.HTrue
			}
		}
	}
	return event.HFalse
}

//----------

func (l *Root) selAnnEv(typ RootSelectAnnotationType) {
	ev2 := &RootSelectAnnotationEvent{typ}
	l.EvReg.RunCallbacks(RootSelectAnnotationEventId, ev2)
}

//----------

const (
	RootSelectAnnotationEventId = iota
)

//----------

type RootSelectAnnotationEvent struct {
	Type RootSelectAnnotationType
}

type RootSelectAnnotationType int

const (
	RootSelAnnTypeFirst RootSelectAnnotationType = iota
	RootSelAnnTypeLast
	RootSelAnnTypePrev
	RootSelAnnTypeNext
	RootSelAnnTypeClear
)
