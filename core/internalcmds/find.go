package internalcmds

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
)

func Find(args0 *core.InternalCmdArgs) error {
	// setup flagset
	fs := flag.NewFlagSet("Find", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // don't output to stderr
	reverseFlag := fs.Bool("rev", false, "reverse find")
	iopt := &iorw.IndexOpt{}
	fs.BoolVar(&iopt.IgnoreCase, "icase", true, "ignore case: 'a' will also match 'A'")
	fs.BoolVar(&iopt.IgnoreCaseDiacritics, "icasediac", false, "ignore case diacritics: 'á' will also match 'Á'. Because ignore case is usually on by default, this is a separate option to explicitly lower the case of diacritics due to being more expensive (~8x slower)'")
	fs.BoolVar(&iopt.IgnoreDiacritics, "idiac", false, "ignore diacritics: 'a' will also match 'á'")

	// parse flags
	part := args0.Part
	args := part.ArgsStrs()[1:]
	err := fs.Parse(args)
	if err != nil {
		if err == flag.ErrHelp {
			buf := &bytes.Buffer{}
			fs.SetOutput(buf)
			fs.Usage()
			args0.Ed.Message(buf.String())
			return nil
		}
		return err
	}

	// this cmd is allowed to get here without a row in order to run the help cmd from the toolbar easily
	erow := args0.ERow
	if erow == nil {
		arg0 := part.Args[0].UnquotedStr()
		return fmt.Errorf("%s: no active row", arg0)
	}

	args2 := fs.Args()

	// unquote args
	w := []string{}
	for _, arg := range args2 {
		if u, err := strconv.Unquote(arg); err == nil {
			arg = u
		}
		w = append(w, arg)
	}

	str := strings.Join(w, " ")

	found, err := rwedit.Find(args0.Ctx, erow.Row.TextArea.EditCtx(), str, *reverseFlag, iopt)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("string not found: %q", str)
	}

	// flash
	ta := erow.Row.TextArea
	if a, b, ok := ta.Cursor().SelectionIndexes(); ok {
		erow.MakeRangeVisibleAndFlash(a, b-a)
	}

	return nil
}
