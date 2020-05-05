package rwedit

import "github.com/jmigpin/editor/v2/util/iout/iorw"

func MoveLineUp(ctx *Ctx) error {
	a, b, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}
	// already at the first line
	if a <= ctx.RW.Min() {
		return nil
	}

	s0, err := ctx.RW.ReadFastAt(a, b-a)
	if err != nil {
		return err
	}
	s := iorw.MakeBytesCopy(s0)

	if err := ctx.RW.OverwriteAt(a, b-a, nil); err != nil {
		return err
	}

	rd := ctx.LocalReader(a - 1)
	a2, err := iorw.LineStartIndex(rd, a-1) // start of previous line, -1 is size of '\n'
	if err != nil {
		return err
	}

	// remove newline to honor the moving line
	if !newline {
		if err := ctx.RW.OverwriteAt(a-1, 1, nil); err != nil {
			return err
		}
		s = append(s, '\n')
	}

	if err := ctx.RW.OverwriteAt(a2, 0, s); err != nil {
		return err
	}

	if ctx.C.HaveSelection() {
		b2 := a2 + len(s)
		_, size, err := iorw.ReadLastRuneAt(ctx.RW, b2)
		if err != nil {
			return nil
		}
		ctx.C.SetSelection(a2, b2-size)
	} else {
		// position cursor at same position
		ctx.C.SetIndex(ctx.C.Index() - (a - a2))
	}
	return nil
}

func MoveLineDown(ctx *Ctx) error {
	a, b, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}
	// already at the last line
	if !newline && b >= ctx.RW.Max() {
		return nil
	}

	// keep copy of the moving line
	s0, err := ctx.RW.ReadFastAt(a, b-a)
	if err != nil {
		return err
	}
	s := iorw.MakeBytesCopy(s0)

	// delete moving line
	if err := ctx.RW.OverwriteAt(a, b-a, nil); err != nil {
		return err
	}

	// line end of the line below
	rd2 := ctx.LocalReader(a)
	a2, newline, err := iorw.LineEndIndex(rd2, a)
	if err != nil {
		return err
	}

	// remove newline
	if !newline {
		// remove newline
		s = s[:len(s)-1]
		// insert newline
		if err := ctx.RW.OverwriteAt(a2, 0, []byte{'\n'}); err != nil {
			return err
		}
		a2 += 1 // 1 is '\n' added to s before insertion
	}

	if err := ctx.RW.OverwriteAt(a2, 0, s); err != nil {
		return err
	}

	if ctx.C.HaveSelection() {
		b2 := a2 + len(s)
		// don't select newline
		if newline {
			_, size, err := iorw.ReadLastRuneAt(ctx.RW, b2)
			if err != nil {
				return nil
			}
			b2 -= size
		}
		ctx.C.SetSelection(a2, b2)
	} else {
		// position cursor at same position
		ctx.C.SetIndex(ctx.C.Index() + (a2 - a))
	}
	return nil
}
