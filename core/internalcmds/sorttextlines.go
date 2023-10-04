package internalcmds

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw"
)

func sortTextLines(args0 *core.InternalCmdArgs) error {
	// setup flagset
	fs := flag.NewFlagSet("SortTextLines", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // don't output to stderr
	identFlag := fs.Bool("firstIndent", false, "sorts the first identation level and leaves inner indented lines untouched; ex: sort switch/case statements while keeping each case body")

	// parse flags
	part := args0.Part
	args := part.ArgsStrings()[1:]
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

	//----------

	erow := args0.ERow
	ta := erow.Row.TextArea
	ctx := ta.EditCtx()
	// get selection text indexes (get start/end of lines)
	a, b, ok := ctx.C.SelectionIndexes()
	if !ok {
		return fmt.Errorf("missing selection")
	}
	a0, err := iorw.LineStartIndex(ctx.RW, a)
	if err != nil {
		return err
	}
	b0, isNL, err := iorw.LineEndIndex(ctx.RW, b)
	if err != nil {
		return err
	}
	if isNL {
		b0--
	}
	// get text itself
	src, err := ctx.RW.ReadFastAt(a0, b0-a0)
	if err != nil {
		return err
	}

	// split and sort
	cutset := " \t"
	s := string(src)
	u := strings.Split(s, "\n")

	// write u slice to have only the wanted indented strings
	if *identFlag {
		nonCutSetStart := func(s string) int {
			for i, ru := range s {
				if !strings.ContainsRune(cutset, ru) {
					return i // start of non-cutset
				}
			}
			return -1
		}

		// find lowest ident
		bestK := -1
		for _, s2 := range u {
			k := nonCutSetStart(s2)
			if k < 0 { // no non-cutset found, empty/all-spaces
				continue
			}
			if bestK < 0 || k < bestK {
				bestK = k
			}
		}

		// make the new strings
		if bestK >= 0 {
			u2 := []string{}
			batch := []string{}
			addBatch := func() {
				if len(batch) > 0 {
					s3 := strings.Join(batch, "\n")
					u2 = append(u2, s3)
					batch = []string{}
				}
			}
			for _, s2 := range u {
				if nonCutSetStart(s2) == bestK {
					addBatch()
				}
				batch = append(batch, s2)
			}
			addBatch()
			if len(u2) > 0 {
				u = u2
			}
		}
	}

	// sort
	slices.SortFunc(u, func(sa, sb string) int {
		sa = strings.TrimLeft(sa, cutset)
		sb = strings.TrimLeft(sb, cutset)
		if sa < sb {
			return -1
		} else if sa > sb {
			return 1
		} else {
			return 0
		}
	})
	s2 := strings.Join(u, "\n")

	if s == s2 {
		return fmt.Errorf("selection already sorted")
	}

	// replace
	if err := ctx.RW.OverwriteAt(a0, len(s), []byte(s2)); err != nil {
		return err
	}

	ctx.C.SetSelection(a0, a0+len(s2))

	return nil
}
