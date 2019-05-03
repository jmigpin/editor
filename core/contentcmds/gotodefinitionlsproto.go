package contentcmds

import (
	"io/ioutil"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/lsproto"
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

func GoToDefinitionLSProto(erow *core.ERow, index int) (bool, error) {
	if erow.Info.IsDir() {
		return false, nil
	}

	// TODO: contexts, a new click could cancel this

	ed := erow.Ed
	tc := erow.Row.TextArea.TextCursor
	rw := tc.RW()

	// must have a registration that handles the filename
	_, err := ed.LSProtoMan.FileRegistration(erow.Info.Name())
	if err != nil {
		return false, nil
	}

	filename, rang, err := ed.LSProtoMan.TextDocumentDefinition(erow.Info.Name(), rw, index)
	if err != nil {
		return true, err
	}

	// content reader
	var rd iorw.Reader
	info, ok := ed.ERowInfos[filename]
	if ok && len(info.ERows) > 0 {
		// file is in memory already
		erow2 := info.ERows[0]
		rd = erow2.Row.TextArea.TextCursor.RW()
	} else {
		// read file
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return true, err
		}
		rd = iorw.NewBytesReadWriter(b)
	}

	// translate range
	offset, length, err := lsproto.RangeToOffsetLen(rd, rang)
	if err != nil {
		return true, err
	}

	// build filepos
	filePos := &parseutil.FilePos{
		Filename: filename,
		Offset:   offset,
		Len:      length,
	}

	// place the file under the calling row
	rowPos := erow.Row.PosBelow()

	conf := &core.OpenFileERowConfig{
		FilePos:               filePos,
		RowPos:                rowPos,
		FlashVisibleOffsets:   true,
		NewIfNotExistent:      true,
		NewIfOffsetNotVisible: true,
	}
	core.OpenFileERow(erow.Ed, conf)

	return true, nil
}
