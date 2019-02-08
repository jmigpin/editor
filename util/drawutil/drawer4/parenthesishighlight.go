package drawer4

func updateParenthesisHighlight(d *Drawer, maxDist int) {
	opt := &d.Opt.ParenthesisHighlight
	opt.Group.Ops = ParenthesisHighlightOps(d, maxDist)
}

//----------

func ParenthesisHighlightOps(d *Drawer, maxDist int) []*ColorizeOp {
	if !d.Opt.ParenthesisHighlight.On {
		return nil
	}
	if !d.Opt.Cursor.On {
		return nil
	}

	// read current rune
	ci := d.Opt.Cursor.index
	cru, _, err := d.reader.ReadRuneAt(ci)
	if err != nil {
		return nil
	}

	// find parenthesis type
	pairs := []rune{'{', '}', '(', ')', '[', ']'}
	var pi int
	for ; pi < len(pairs); pi++ {
		if pairs[pi] == cru {
			break
		}
	}
	if pi >= len(pairs) {
		return nil
	}

	// assign open/close parenthesis
	var open, close rune
	isOpen := pi%2 == 0
	var nextRune func() (rune, int, error)
	if isOpen {
		open, close = pairs[pi], pairs[pi+1]
		ri := ci + len(string(open))
		nextRune = func() (rune, int, error) {
			ru, size, err := d.reader.ReadRuneAt(ri)
			if err != nil {
				return 0, 0, err
			}
			ri2 := ri
			ri += size
			return ru, ri2, nil
		}
	} else {
		open, close = pairs[pi], pairs[pi-1]
		ri := ci
		nextRune = func() (rune, int, error) {
			ru, size, err := d.reader.ReadLastRuneAt(ri)
			if err != nil {
				return 0, 0, err
			}
			ri -= size
			return ru, ri, nil
		}
	}

	// colorize open
	op1 := &ColorizeOp{
		Offset: ci,
		Fg:     d.Opt.ParenthesisHighlight.Fg,
		Bg:     d.Opt.ParenthesisHighlight.Bg,
	}
	op2 := &ColorizeOp{Offset: ci + len(string(open))}
	var ops []*ColorizeOp
	ops = append(ops, op1, op2)

	// find parenthesis
	match := 0
	for i := 0; i < maxDist; i++ {
		ru, ri, err := nextRune()
		if err != nil {
			break
		}
		if ru == open {
			match++
		} else if ru == close {
			if match > 0 {
				match--
			} else {
				// colorize close
				op1 := &ColorizeOp{
					Offset: ri,
					Fg:     d.Opt.ParenthesisHighlight.Fg,
					Bg:     d.Opt.ParenthesisHighlight.Bg,
				}
				op2 := &ColorizeOp{Offset: ri + len(string(close))}
				ops = append(ops, op1, op2)
				if !isOpen {
					// invert order
					l := len(ops)
					ops[l-4], ops[l-2] = ops[l-2], ops[l-4]
					ops[l-3], ops[l-1] = ops[l-1], ops[l-3]
				}
				break
			}
		}
	}

	return ops
}
