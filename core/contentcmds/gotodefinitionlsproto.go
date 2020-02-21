package contentcmds

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/lsproto"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil"
)

func GoToDefinitionLSProto(ctx context.Context, erow *core.ERow, index int) (error, bool) {
	if erow.Info.IsDir() {
		return nil, false
	}

	// timeout for the cmd to run
	timeout := 8 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ed := erow.Ed
	tc := erow.Row.TextArea.TextCursor
	rw := tc.RW()

	// must have a registration that handles the filename
	_, err := ed.LSProtoMan.LangManager(erow.Info.Name())
	if err != nil {
		return nil, false
	}

	filename, rang, err := ed.LSProtoMan.TextDocumentDefinition(ctx, erow.Info.Name(), rw, index)
	if err != nil {
		return err, true
	}

	// content reader
	var rd iorw.Reader
	info, ok := ed.ERowInfo(filename)
	if ok {
		// file is in memory already
		if erow0, ok := info.FirstERow(); ok {
			rd = erow0.Row.TextArea.TextCursor.RW()
		}
	}
	if rd == nil {
		// read file
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return err, true
		}
		rd = iorw.NewBytesReadWriter(b)
	}

	// translate range
	offset, length, err := lsproto.RangeToOffsetLen(rd, rang)
	if err != nil {
		return err, true
	}

	// build filepos
	filePos := &parseutil.FilePos{
		Filename: filename,
		Offset:   offset,
		Len:      length,
	}

	erow.Ed.UI.RunOnUIGoRoutine(func() {
		// place the file under the calling row
		rowPos := erow.Row.PosBelow() // needs ui goroutine

		conf := &core.OpenFileERowConfig{
			FilePos:               filePos,
			RowPos:                rowPos,
			FlashVisibleOffsets:   true,
			NewIfNotExistent:      true,
			NewIfOffsetNotVisible: true,
		}
		core.OpenFileERow(erow.Ed, conf) // needs ui goroutine
	})

	return nil, true
}
