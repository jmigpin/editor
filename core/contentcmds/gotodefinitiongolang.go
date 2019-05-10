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
	"github.com/jmigpin/editor/util/osutil"
)

func GoToDefinitionGolang(ctx context.Context, erow *core.ERow, index int) (error, bool) {
	if erow.Info.IsDir() {
		return nil, false
	}
	if path.Ext(erow.Info.Name()) != ".go" {
		return nil, false
	}

	// timeout for the cmd to run
	timeout := 8 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// it's a go file, return true from here

	// go guru args
	pos := fmt.Sprintf("%v:#%v", erow.Info.Name(), index)
	args := []string{osutil.ExecName("guru"), "-modified", "definition", pos}

	// use stdin
	in := &bytes.Buffer{}

	// guru format: filename
	s := fmt.Sprintf("%v\n", erow.Info.Name())
	_, err := in.Write([]byte(s))
	if err != nil {
		return err, true
	}
	// guru format: filesize
	s = fmt.Sprintf("%v\n", erow.Row.TextArea.Len())
	_, err = in.Write([]byte(s))
	if err != nil {
		return err, true
	}
	// guru format: content
	bin, err := erow.Row.TextArea.Bytes()
	if err != nil {
		return err, true
	}
	_, err = in.Write(bin)
	if err != nil {
		return err, true
	}

	// execute external cmd
	dir := filepath.Dir(erow.Info.Name())
	out, err := core.ExecCmdStdin(ctx, dir, in, args...)
	if err != nil {
		return err, true
	}

	filePos, err := parseutil.ParseFilePos(string(out))
	if err != nil {
		return err, true
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

	return nil, true
}
