package rwedit

import (
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func SelectWord(ctx *Ctx) error {
	// index rune
	ci := ctx.C.Index()
	ru, _, err := ctx.RW.ReadRuneAt(ci)
	if err != nil {
		return err
	}

	var index int
	var word []byte
	if !iorw.IsWordRune(ru) {
		// select just the index rune
		index = ci
		word = []byte(string(ru))
	} else {
		// select word at index
		rd := ctx.LocalReader(ci)
		w, i, err := iorw.WordAtIndex(rd, ci)
		if err != nil {
			return err
		}

		index = i
		word = w
	}

	ctx.C.SetSelection(index, index+len(word))

	// set primary copy
	if b, ok := ctx.Selection(); ok {
		ctx.Fns.SetClipboardData(event.CIPrimary, string(b))
	}

	return nil
}
