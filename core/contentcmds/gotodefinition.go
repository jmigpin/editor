package contentcmds

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
)

func GoToDefinition(erow *core.ERow, index int) (bool, error) {
	if erow.Info.IsDir() {
		return false, nil
	}
	if path.Ext(erow.Info.Name()) != ".go" {
		return false, nil
	}

	// timeout for the cmd to run
	timeout := 8000 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// it's a go file, return true from here

	// go guru args
	pos := fmt.Sprintf("%v:#%v", erow.Info.Name(), index)
	args := []string{"guru", "-modified", "definition", pos}

	// use stdin
	in := &bytes.Buffer{}

	// guru format: filename
	s := fmt.Sprintf("%v\n", erow.Info.Name())
	_, err := in.Write([]byte(s))
	if err != nil {
		return true, err
	}
	// guru format: filesize
	s = fmt.Sprintf("%v\n", erow.Row.TextArea.Len())
	_, err = in.Write([]byte(s))
	if err != nil {
		return true, err
	}
	// guru format: content
	bin, err := erow.Row.TextArea.Bytes()
	_, err = in.Write(bin)
	if err != nil {
		return true, err
	}

	// execute external cmd
	dir := filepath.Dir(erow.Info.Name())
	out, err := core.ExecCmdStdin(ctx, dir, in, args...)
	if err != nil {
		return true, err
	}

	filePos, err := parseutil.ParseFilePos(string(out))
	if err != nil {
		return true, err
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
