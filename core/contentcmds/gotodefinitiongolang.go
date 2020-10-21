package contentcmds

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

func GoToDefinitionGolang(ctx context.Context, erow *core.ERow, index int) (error, bool) {
	if erow.Info.IsDir() {
		return nil, false
	}

	// "guru" doesn't work well with modules and  "gopls query" needs the file to be saved
	return guruOrGoplsQuery(ctx, erow, index)
}

//----------

func guruOrGoplsQuery(ctx context.Context, erow *core.ERow, index int) (error, bool) {
	if path.Ext(erow.Info.Name()) != ".go" {
		return nil, false
	}

	// it's a go file, return true from here

	// timeout for the cmd to run
	timeout := 8 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// TODO: remove later since guru won't be updated to support modules.
	err1 := goGuru(ctx, erow, index)
	if err1 == nil {
		return nil, true
	}

	// Try gopls. Slower then goguru, but works better with modules
	err2 := goplsQuery(ctx, erow, index)
	if err2 == nil {
		return nil, true
	}

	return iout.MultiErrors(err1, err2), true
}

//----------

func goplsQuery(ctx context.Context, erow *core.ERow, index int) error {
	// TODO: no way to send current buffer, needs file to be saved

	// gopls query args
	pos := fmt.Sprintf("%v:#%v", erow.Info.Name(), index)
	//args := []string{osutil.ExecName("gopls"), "query", "-emulate", "guru", "definition", pos}
	args := []string{osutil.ExecName("gopls"), "definition", pos}

	// execute external cmd
	dir := filepath.Dir(erow.Info.Name())
	out, err := core.ExecCmd(ctx, dir, args...)
	if err != nil {
		return err
	}

	filePos, err := parseutil.ParseFilePos(string(out))
	if err != nil {
		return err
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

	return nil
}

//----------

func goGuru(ctx context.Context, erow *core.ERow, index int) error {
	// go guru args
	pos := fmt.Sprintf("%v:#%v", erow.Info.Name(), index)
	args := []string{osutil.ExecName("guru"), "-modified", "definition", pos}

	// use stdin
	in := &bytes.Buffer{}

	// guru format: filename
	s := fmt.Sprintf("%v\n", erow.Info.Name())
	_, err := in.Write([]byte(s))
	if err != nil {
		return err
	}
	// guru format: filesize
	s = fmt.Sprintf("%v\n", erow.Row.TextArea.Len())
	_, err = in.Write([]byte(s))
	if err != nil {
		return err
	}
	// guru format: content
	bin, err := erow.Row.TextArea.Bytes()
	if err != nil {
		return err
	}
	_, err = in.Write(bin)
	if err != nil {
		return err
	}

	// execute external cmd
	dir := filepath.Dir(erow.Info.Name())
	out, err := core.ExecCmdStdin(ctx, dir, in, args...)
	if err != nil {
		return err
	}

	filePos, err := parseutil.ParseFilePos(string(out))
	if err != nil {
		return err
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

	return nil
}
