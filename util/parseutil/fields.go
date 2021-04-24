package parseutil

import (
	"strconv"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/scanutil"
)

func ParseFields(s string, fieldSep rune) ([]string, error) {
	rd := iorw.NewStringReaderAt(s)
	sc := scanutil.NewScanner(rd)

	fields := []string{}
	for i := 0; ; i++ {
		if sc.Match.End() {
			break
		}

		// field separator
		if i > 0 && !sc.Match.Rune(fieldSep) {
			return nil, sc.Errorf("field separator")
		}
		sc.Advance()

		// field (can be empty)
		for {
			if sc.Match.Quoted("\"'", '\\', true, 5000) {
				continue
			}
			if sc.Match.Except(string(fieldSep)) {
				continue
			}
			break
		}
		f := sc.Value()

		// unquote field
		f2, err := strconv.Unquote(f)
		if err == nil {
			f = f2
		}

		// add field
		fields = append(fields, f)
		sc.Advance()
	}
	return fields, nil
}
