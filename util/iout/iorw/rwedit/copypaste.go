package rwedit

import (
	"fmt"

	"github.com/jmigpin/editor/util/uiutil/event"
)

func Copy(ctx *Ctx) error {
	if b, ok := ctx.Selection(); ok {
		ctx.Fns.SetClipboardData(event.CIClipboard, string(b))
	}
	return nil
}

func Paste(ctx *Ctx, ci event.ClipboardIndex) {
	ctx.Fns.GetClipboardData(ci, func(s string, err error) {
		if err != nil {
			ctx.Fns.Error(fmt.Errorf("rwedit.paste: %w", err))
			return
		}
		if err := InsertString(ctx, s); err != nil {
			ctx.Fns.Error(fmt.Errorf("rwedit.paste: insertstring: %w", err))
		}
	})
}
