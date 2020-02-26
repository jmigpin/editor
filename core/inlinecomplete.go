package core

import (
	"context"
	"strings"
	"sync"
	"unicode"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/scanutil"
)

type InlineComplete struct {
	ed *Editor

	mu struct {
		sync.Mutex
		cancel context.CancelFunc
		ta     *ui.TextArea // if not nil, inlinecomplete is on
		index  int          // cursor index
	}
}

func NewInlineComplete(ed *Editor) *InlineComplete {
	ic := &InlineComplete{ed: ed}
	ic.mu.cancel = func() {} // avoid testing for nil
	return ic
}

//----------

func (ic *InlineComplete) Complete(erow *ERow, ev *ui.TextAreaInlineCompleteEvent) bool {

	// early pre-check if filename is supported
	_, err := ic.ed.LSProtoMan.LangManager(erow.Info.Name())
	if err != nil {
		return false // not handled
	}

	ta := ev.TextArea

	ic.mu.Lock()
	ic.mu.cancel() // cancel previous run
	ctx, cancel := context.WithCancel(erow.ctx)
	ic.mu.cancel = cancel
	ic.mu.ta = ta
	ic.mu.index = ta.TextCursor.Index()
	ic.mu.Unlock()

	go func() {
		defer cancel()
		ic.setAnnotationsMsg(ta, "loading...")
		err := ic.complete2(ctx, erow.Info.Name(), ta, ev)
		if err != nil {
			ic.setAnnotations(ta, nil)
			ic.ed.Error(err)
		}
	}()
	return true
}

func (ic *InlineComplete) complete2(ctx context.Context, filename string, ta *ui.TextArea, ev *ui.TextAreaInlineCompleteEvent) error {
	comps, err := ic.completions(ctx, filename, ta)
	if err != nil {
		return err
	}

	// insert complete
	tc := ta.TextCursor
	tc.BeginEdit()
	newIndex, completed, comps2, err := insertComplete(comps, tc.RW(), tc.Index())
	if err != nil {
		tc.EndEdit()
		return err
	}
	if newIndex != 0 {
		tc.SetIndex(newIndex)
		// update index for CancelOnCursorChange
		ic.mu.Lock()
		ic.mu.index = newIndex
		ic.mu.Unlock()
	}
	tc.EndEdit()
	comps = comps2

	switch len(comps) {
	case 0:
		ic.setAnnotationsMsg(ta, "0 results")
	case 1:
		if completed {
			ic.setAnnotations(ta, nil)
		} else {
			ic.setAnnotationsMsg(ta, "already complete")
		}
	default:
		// show completions
		entries := []*drawer4.Annotation{}
		for _, v := range comps {
			u := &drawer4.Annotation{Offset: ev.Offset, Bytes: []byte(v)}
			entries = append(entries, u)
		}
		ic.setAnnotations(ta, entries)
	}
	return nil
}

//----------

func (ic *InlineComplete) completions(ctx context.Context, filename string, ta *ui.TextArea) ([]string, error) {
	tc := ta.TextCursor
	compList, err := ic.ed.LSProtoMan.TextDocumentCompletion(ctx, filename, tc.RW(), tc.Index())
	if err != nil {
		return nil, err
	}
	res := []string{}
	for _, ci := range compList.Items {
		// trim labels (clangd: has some entries prefixed with space)
		label := strings.TrimSpace(ci.Label)

		res = append(res, label)
	}
	return res, nil
}

//----------

func (ic *InlineComplete) setAnnotationsMsg(ta *ui.TextArea, s string) {
	tc := ta.TextCursor
	offset := tc.Index()
	entries := []*drawer4.Annotation{{offset, []byte(s), nil}}
	ic.setAnnotations(ta, entries)
}

func (ic *InlineComplete) setAnnotations(ta *ui.TextArea, entries []*drawer4.Annotation) {
	on := entries != nil && len(entries) > 0
	ic.ed.SetAnnotations(EdAnnReqInlineComplete, ta, on, -1, entries)
	if !on {
		ic.setOff(ta)
	}
}

//----------

func (ic *InlineComplete) IsOn(ta *ui.TextArea) bool {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return ic.mu.ta != nil && ic.mu.ta == ta
}

func (ic *InlineComplete) setOff(ta *ui.TextArea) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if ic.mu.ta == ta {
		ic.mu.ta = nil
		// possible early cancel for this textarea
		ic.mu.cancel()
	}
}

//----------

func (ic *InlineComplete) CancelAndClear() {
	ic.mu.Lock()
	ta := ic.mu.ta
	ic.mu.Unlock()
	if ta != nil {
		ic.setAnnotations(ta, nil)
	}
}

func (ic *InlineComplete) CancelOnCursorChange() {
	ic.mu.Lock()
	ta := ic.mu.ta
	index := ic.mu.index
	ic.mu.Unlock()
	if ta != nil {
		if index != ta.TextCursor.Index() {
			ic.setAnnotations(ta, nil)
		}
	}
}

//----------

func insertComplete(comps []string, rw iorw.ReadWriter, index int) (newIndex int, completed bool, _ []string, _ error) {
	// build prefix from start of string
	start, prefix, ok := readLastUntilStart(rw, index)
	if !ok {
		return 0, false, comps, nil
	}

	expand, canComplete, comps2 := filterPrefixedAndExpand(comps, prefix)
	comps = comps2
	if len(comps) == 0 {
		return 0, false, comps, nil
	}

	if canComplete {
		// original string
		origStr := prefix

		// string to insert
		n := len(origStr)
		insStr := comps[0][:n+expand]

		// try to expand the index to the existing text
		for i := 0; i < expand; i++ {
			b, err := rw.ReadNAtFast(index+i, 1)
			if err != nil {
				break
			}
			if b[0] != insStr[n] {
				break
			}
			n++
		}

		// insert completion
		if insStr != origStr {
			err := rw.Overwrite(start, n, []byte(insStr))
			if err != nil {
				return 0, false, nil, err
			}
			newIndex = start + len(insStr)
			return newIndex, true, comps, nil
		}
	}

	return 0, false, comps, nil
}

//----------

func filterPrefixedAndExpand(comps []string, prefix string) (expand int, canComplete bool, _ []string) {
	// find all matches from start to index
	strLow := strings.ToLower(prefix)
	res := []string{}
	for _, v := range comps {
		vLow := strings.ToLower(v)
		if strings.HasPrefix(vLow, strLow) {
			res = append(res, v)
		}
	}
	// find possible expansions if all matches have common extra runes
	if len(res) == 1 {
		// special case to allow overwriting string casing "aaa"->"aAa"
		canComplete = true
		expand = len(res[0]) - len(prefix)
	} else if len(res) >= 1 {
	loop1:
		for j := 0; j < len(res[0]); j++ { // test up to first result length
			// break on any result that fails to expand
			for i := 1; i < len(res); i++ {
				if !(j < len(res[i]) && res[i][j] == res[0][j]) {
					break loop1
				}
			}
			if j >= len(prefix) {
				expand++
				canComplete = true
			}
		}
	}

	return expand, canComplete, res
}

//----------

func readLastUntilStart(rd iorw.Reader, index int) (int, string, bool) {
	sc := scanutil.NewScanner(rd)
	sc.Reverse = true
	sc.SetStartPos(index)
	max := 1000
	ok := sc.Match.FnLoop(func(ru rune) bool {
		max--
		if max <= 0 {
			return false
		}
		return ru == '_' ||
			unicode.IsLetter(ru) ||
			unicode.IsNumber(ru) ||
			unicode.IsDigit(ru)
	})
	if !ok || sc.Empty() {
		return 0, "", false
	}
	return sc.Pos, sc.Value(), true
}
