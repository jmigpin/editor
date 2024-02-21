package internalcmds

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
	"github.com/jmigpin/editor/util/parseutil"
)

func Find(args *core.InternalCmdArgs) error {
	// setup flagset
	fs := flag.NewFlagSet("Find", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // don't output to stderr
	reverseFlag := fs.Bool("rev", false, "reverse find")
	iopt := &iorw.IndexOpt{}
	fs.BoolVar(&iopt.IgnoreCase, "icase", true, "ignore case: 'a' will also match 'A'")
	fs.BoolVar(&iopt.IgnoreCaseDiacritics, "icasediac", false, "ignore case diacritics: 'á' will also match 'Á'. Because ignore case is usually on by default, this is a separate option to explicitly lower the case of diacritics due to being more expensive (~8x slower)'")
	fs.BoolVar(&iopt.IgnoreDiacritics, "idiac", false, "ignore diacritics: 'a' will also match 'á'")
	if err := parseFlagSetHandleUsage(args, fs); err != nil {
		return err
	}

	//----------

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	args2 := fs.Args()

	// unquote args
	w := []string{}
	for _, arg := range args2 {
		if u, err := parseutil.UnquoteStringBs(arg); err == nil {
			arg = u
		}
		w = append(w, arg)
	}

	str := strings.Join(w, " ")

	found, err := rwedit.Find(args.Ctx, erow.Row.TextArea.EditCtx(), str, *reverseFlag, iopt)
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
