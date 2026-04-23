package contentcmds

import (
	"context"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil/btparser"
)

func OpenSession(ctx context.Context, erow *core.ERow, index int) (error, bool) {
	ta := erow.Row.TextArea

	// limit reading
	rd := iorw.NewLimitedReaderAtPad(ta.RW(), index, index, 1000)

	sname, err := sessionName(rd, index)
	if err != nil {
		return nil, false
	}

	erow.Ed.UI.RunOnUIGoRoutine(func() {
		core.OpenSessionFromString(erow.Ed, sname)
	})

	return nil, true
}

//----------

func sessionName(rd iorw.ReaderAt, index int) (string, error) {
	src, err := iorw.ReadFastFull(rd)
	if err != nil {
		return "", err
	}

	g := btparser.NewRules()
	ps := btparser.NewParserStateFromBytes(src)
	pos := btparser.Pos(index - rd.Min())

	cmdStr := "OpenSession"
	revCmdStr := btparser.ReverseString(cmdStr)
	name := ""

	parseName := g.Loop1(g.RuneFn(sessionNameRune))
	fn := g.And(
		// consume backwards
		// example positions "Open●Session ●ses●sionname●"
		g.WithLineBounds(0, 0, g.ReverseSource(g.Or(
			g.SeqOrMid(revCmdStr), // try before parsename in reverse
			g.And(
				g.Optional(parseName),
				g.Optional(g.SpacesExceptNewline()),
				g.Seq(revCmdStr),
			),
		))),
		// parse
		g.And(
			g.Seq(cmdStr),
			g.SpacesExceptNewline(),
			btparser.AssignLocal(&name, g.VString(parseName)),
		),
	)

	if _, err := g.ParseAt(ps, pos, fn); err != nil {
		return "", err
	}
	return name, nil
}

func sessionNameRune(ru rune) bool {
	return unicode.IsLetter(ru) ||
		unicode.IsDigit(ru) ||
		strings.ContainsRune("_-.", ru)
}
