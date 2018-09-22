package contentcmds

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
)

func goDefinition(erow *core.ERow, index int) (bool, error) {
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

	out, err := goGuruDefinition(ctx, erow, index)
	if err != nil {
		return true, err
	}

	filePos, err := parseutil.ParseFilePos(string(out))
	if err != nil {
		return true, err
	}

	// place the file under the calling row
	rowPos := erow.Row.PosBelow()
	// if calling erow is dir, place the file at a good place
	if erow.Info.IsDir() {
		rowPos = erow.Ed.GoodRowPos()
	}

	core.OpenERowFilePosVisibleOrNew(erow.Ed, filePos, rowPos)

	return true, nil
}

func goGuruDefinition(ctx context.Context, erow *core.ERow, index int) ([]byte, error) {
	dir := filepath.Dir(erow.Info.Name())
	position := fmt.Sprintf("%v:#%v", erow.Info.Name(), index)
	args := []string{"guru", "definition", position}
	return core.ExecCmd(ctx, dir, args...)
}
