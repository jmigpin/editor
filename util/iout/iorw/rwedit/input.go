package rwedit

import (
	"errors"
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/uiutil/event"
)

//godebug:annotatefile

func HandleInput(ctx *Ctx, ev any) (event.Handled, error) {
	in := &Input{ctx, ev}
	return in.handle()
}

//----------

type Input struct {
	ctx *Ctx
	ev  any
}

func (in *Input) handle() (event.Handled, error) {
	switch ev := in.ev.(type) {
	case *event.KeyDown:
		return in.onKeyDown(ev)
	case *event.MouseDown:
		return in.onMouseDown(ev)
	case *event.MouseDragMove:
		return in.onMouseDragMove(ev)
	case *event.MouseDragEnd:
		return in.onMouseDragEnd(ev)
	case *event.MouseClick:
		return in.onMouseClick(ev)
	case *event.MouseDoubleClick:
		return in.onMouseDoubleClick(ev)
	case *event.MouseTripleClick:
		return in.onMouseTripleClick(ev)
	}
	return false, nil
}

//----------

func (in *Input) onMouseDown(ev *event.MouseDown) (event.Handled, error) {
	switch ev.Button {
	case event.ButtonLeft:
		if ev.Mods.ClearLocks().Is(event.ModShift) {
			MoveCursorToPoint(in.ctx, ev.Point, true)
		} else {
			MoveCursorToPoint(in.ctx, ev.Point, false)
		}
		return true, nil
	case event.ButtonWheelUp:
		ScrollUp(in.ctx, true)
	case event.ButtonWheelDown:
		ScrollUp(in.ctx, false)
	}
	return false, nil
}

//----------

func (in *Input) onMouseDragMove(ev *event.MouseDragMove) (event.Handled, error) {
	if ev.Buttons.Has(event.ButtonLeft) {
		MoveCursorToPoint(in.ctx, ev.Point, true)
		return true, nil
	}
	return false, nil
}
func (in *Input) onMouseDragEnd(ev *event.MouseDragEnd) (event.Handled, error) {
	switch ev.Button {
	case event.ButtonLeft:
		MoveCursorToPoint(in.ctx, ev.Point, true)
		return true, nil
	}
	return false, nil
}

//----------

func (in *Input) onMouseClick(ev *event.MouseClick) (event.Handled, error) {
	switch ev.Button {
	case event.ButtonMiddle:
		MoveCursorToPoint(in.ctx, ev.Point, false)
		Paste(in.ctx, event.CIPrimary)
		return true, nil
	}
	return false, nil
}

func (in *Input) onMouseDoubleClick(ev *event.MouseDoubleClick) (event.Handled, error) {
	switch ev.Button {
	case event.ButtonLeft:
		MoveCursorToPoint(in.ctx, ev.Point, false)
		err := SelectWord(in.ctx)
		// can select at EOF but avoid error msg
		if errors.Is(err, io.EOF) {
			err = nil
		}

		return true, err
	}
	return false, nil
}

func (in *Input) onMouseTripleClick(ev *event.MouseTripleClick) (event.Handled, error) {
	switch ev.Button {
	case event.ButtonLeft:
		MoveCursorToPoint(in.ctx, ev.Point, false)
		err := SelectLine(in.ctx)
		return true, err
	}
	return false, nil
}

//----------

func (in *Input) onKeyDown(ev *event.KeyDown) (_ event.Handled, err error) {
	mcl := ev.Mods.ClearLocks()

	makeCursorVisible := func() {
		if err == nil {
			in.ctx.Fns.MakeIndexVisible(in.ctx.C.Index())
		}
	}

	switch ev.KeySym {
	case event.KSymAltL,
		event.KSymAltGr,
		event.KSymShiftL,
		event.KSymShiftR,
		event.KSymControlL,
		event.KSymControlR,
		event.KSymCapsLock,
		event.KSymNumLock,
		event.KSymInsert,
		event.KSymEscape,
		event.KSymSuperL: // windows key
		// ignore these
	case event.KSymRight:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			err = MoveCursorJumpRight(in.ctx, true)
		case mcl.Is(event.ModCtrl):
			err = MoveCursorJumpRight(in.ctx, false)
		case mcl.Is(event.ModShift):
			err = MoveCursorRight(in.ctx, true)
		default:
			err = MoveCursorRight(in.ctx, false)
		}
		makeCursorVisible()
		return true, err
	case event.KSymLeft:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			err = MoveCursorJumpLeft(in.ctx, true)
		case mcl.Is(event.ModCtrl):
			err = MoveCursorJumpLeft(in.ctx, false)
		case mcl.Is(event.ModShift):
			err = MoveCursorLeft(in.ctx, true)
		default:
			err = MoveCursorLeft(in.ctx, false)
		}
		makeCursorVisible()
		return true, err
	case event.KSymUp:
		switch {
		case mcl.Is(event.ModCtrl | event.ModAlt):
			err = MoveLineUp(in.ctx)
		//case mcl.Is(event.ModCtrl | event.ModShift):
		//err = MoveCursorJumpUp(in.ctx, true)
		//case mcl.Is(event.ModCtrl):
		//err = MoveCursorJumpUp(in.ctx, false)
		case mcl.HasAny(event.ModShift):
			MoveCursorUp(in.ctx, true)
		default:
			MoveCursorUp(in.ctx, false)
		}
		makeCursorVisible()
		return true, err
	case event.KSymDown:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift | event.ModAlt):
			err = DuplicateLines(in.ctx)
		case mcl.Is(event.ModCtrl | event.ModAlt):
			err = MoveLineDown(in.ctx)
		//case mcl.Is(event.ModCtrl | event.ModShift):
		//err = MoveCursorJumpDown(in.ctx, true)
		//case mcl.Is(event.ModCtrl):
		//err = MoveCursorJumpDown(in.ctx, false)
		case mcl.HasAny(event.ModShift):
			MoveCursorDown(in.ctx, true)
		default:
			MoveCursorDown(in.ctx, false)
		}
		makeCursorVisible()
		return true, err
	case event.KSymHome:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			StartOfString(in.ctx, true)
		case mcl.Is(event.ModCtrl):
			StartOfString(in.ctx, false)
		case mcl.Is(event.ModShift):
			err = StartOfLine(in.ctx, true)
		default:
			err = StartOfLine(in.ctx, false)
		}
		makeCursorVisible()
		return true, err
	case event.KSymEnd:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			EndOfString(in.ctx, true)
		case mcl.Is(event.ModCtrl):
			EndOfString(in.ctx, false)
		case mcl.Is(event.ModShift):
			err = EndOfLine(in.ctx, true)
		default:
			err = EndOfLine(in.ctx, false)
		}
		makeCursorVisible()
		return true, err
	case event.KSymBackspace:
		err = Backspace(in.ctx)
		makeCursorVisible()
		return true, err
	case event.KSymDelete, event.KSymKeypadDelete:
		err = Delete(in.ctx)
		makeCursorVisible() // TODO: on delete?
		return true, err
	case event.KSymReturn, event.KSymKeypadEnter:
		err = AutoIndent(in.ctx)
		makeCursorVisible()
		return true, err
	case event.KSymTabLeft:
		err = TabLeft(in.ctx)
		makeCursorVisible()
		return true, err
	case event.KSymTab:
		switch {
		case mcl.Is(event.ModShift):
			// TODO: using KSymTabLeft case, this still needed?
			err = TabLeft(in.ctx)
		default:
			err = TabRight(in.ctx)
		}
		makeCursorVisible()
		return true, err
	case event.KSymSpace:
		// ensure space even if modifiers are present
		err = InsertString(in.ctx, " ")
		makeCursorVisible()
		return true, err
	case event.KSymPageUp:
		PageUp(in.ctx, true)
		return true, nil
	case event.KSymPageDown:
		PageUp(in.ctx, false)
		return true, nil
	default:
		switch {
		case mcl.Is(event.ModCtrl):
			switch ev.KeySym {
			case event.KSymD:
				err = Comment(in.ctx)
				return true, err
			case event.KSymC:
				err = Copy(in.ctx)
				return true, err
			case event.KSymX:
				err = Cut(in.ctx)
				return true, err
			case event.KSymV:
				Paste(in.ctx, event.CIClipboard)
				return true, nil
			case event.KSymK:
				err = RemoveLines(in.ctx)
				return true, nil
			case event.KSymA:
				err = SelectAll(in.ctx)
				return true, nil
			case event.KSymZ:
				err = Undo(in.ctx)
				return true, nil
			}
		case mcl.Is(event.ModCtrl | event.ModShift):
			switch ev.KeySym {
			case event.KSymD:
				err = Uncomment(in.ctx)
				return true, err
			case event.KSymZ:
				err = Redo(in.ctx)
				return true, nil
			}
		case ev.KeySym >= event.KSymF1 && ev.KeySym <= event.KSymF12:
			// do nothing
		case !unicode.IsPrint(ev.Rune):
			// do nothing
		default:
			err = InsertString(in.ctx, string(ev.Rune))
			makeCursorVisible()
			return true, err
		}
	}
	return false, nil
}
