package textutil

import (
	"log"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Paste(te *widget.TextEdit, i event.CopyPasteIndex) {
	te.GetCPPaste(i, func(str string, ok bool) {
		if ok {
			te.RunOnUIGoRoutine(func() {
				if err := InsertString(te, str); err != nil {
					log.Println("textutil.paste: %v", err)
				}
			})
		}
	})
}
