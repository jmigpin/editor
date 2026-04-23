package toolbarparser

import (
	"fmt"

	"github.com/jmigpin/editor/util/parseutil/btparser"
)

// scan for cmd position, update with arg, or insert new cmd
func UpdateOrInsertPartCmd(data *Data, cmd, arg string) uoipcResult {
	part, ok := findLastCmdPart(data, cmd)
	if ok {
		return updateCmdPartWithArg(part, arg)
	}
	return insertCmdPartAtEnd(data, cmd, arg)
}

//----------

func findLastCmdPart(data *Data, cmd string) (*Part, bool) {
	found := (*Part)(nil)
	for _, p := range data.Parts {
		if len(p.Args) > 0 && p.Args[0].String() == cmd {
			found = p
		}
	}
	if found != nil {
		return found, true
	}
	return nil, false
}

//----------

func updateCmdPartWithArg(part *Part, arg string) uoipcResult {
	// asssume at least one arg (cmd arg)
	start, end := part.Args[0].End(), part.End()

	res := uoipcResult{Pos: start, End: end}
	res.S = part.Data.Str

	args := part.Args[1:]
	if len(args) == 0 {
		// need to insert starting space
		res.S, res.Pos = insDelStr(res.S, res.Pos, res.Pos, " ")
		res.End = res.Pos
	} else {
		// use args positions, don't disturb original disposition
		res.Pos = args[0].Pos()
		res.End = args[len(args)-1].End()
	}

	if arg != "" {
		res.S, res.End = insDelStr(res.S, res.Pos, res.End, arg)
	}

	return res
}

//----------

func insertCmdPartAtEnd(data *Data, cmd, arg string) uoipcResult {
	g := btparser.NewRules()
	prefix := " | "
	assignPrefix := func(fn btparser.MFn, v string) btparser.MFn {
		return btparser.AssignLocal(&prefix, btparser.VConst(fn, v))
	}
	fn := g.ReverseSource(g.And(
		g.Optional(g.Loop1(g.RuneAnyOf(' ', '\t'))),
		g.Or(
			assignPrefix(g.Peek(g.Rune('\n')), ""),
			assignPrefix(g.Peek(g.Rune('|')), " "),
			g.NoOp(),
		),
	))

	//----------

	ps := btparser.NewParserStateFromString(data.Str)
	p2, _ := g.ParseAt(ps, btparser.Pos(len(data.Str)), fn)

	replaceAt := func(pos int, prefix string) uoipcResult {
		res := uoipcResult{}
		s1 := fmt.Sprintf("%s%s ", prefix, cmd)
		res.S, res.Pos = insDelStr(data.Str, pos, len(data.Str), s1)
		res.S, res.End = insDelStr(res.S, res.Pos, len(res.S), arg)
		return res
	}

	return replaceAt(int(p2), prefix)
}

//----------

func insDelStr(s string, i1, i2 int, a string) (string, int) {
	u := s[:i1] + a + s[i2:]
	return u, i1 + len(a)
}

//----------
//----------
//----------

type uoipcResult struct {
	S        string
	Pos, End int
}
