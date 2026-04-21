package reslocparser

import "github.com/jmigpin/editor/util/parseutil/btparser"

type coverIndexScan struct {
	g  btparser.Rules
	fn btparser.MFn
}

func newCoverIndexScan(g btparser.Rules, fn btparser.MFn) *coverIndexScan {
	return &coverIndexScan{g: g, fn: fn}
}

func (s *coverIndexScan) parse(src []byte, start, index int, rl *ResLoc) (int, btparser.Pos, error) {
	var err0 error
	iMax := min(index, len(src)-1)
	for i := start; i <= iMax; i++ {
		rl2 := *rl
		ps := btparser.NewParserStateFromBytes(src)
		ps.UserData = &rl2

		p2, err := s.g.ParseAt(ps, btparser.Pos(i), s.fn)
		if err != nil {
			if err0 == nil {
				err0 = err
			}
			continue
		}
		if int(p2) < index {
			continue
		}

		*rl = rl2
		return i, p2, nil
	}
	if err0 != nil {
		return 0, 0, err0
	}
	return 0, 0, btparser.NoMatchErr
}
