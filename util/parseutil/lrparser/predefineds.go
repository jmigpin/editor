package lrparser

import (
	"fmt"
)

func setupPredefineds(ri *RuleIndex) error {
	if err := ri.setSingletonRule(nilRule); err != nil {
		return err
	}
	if err := ri.setSingletonRule(endRule); err != nil {
		return nil
	}

	if err := setupAnyRuneFn(ri); err != nil {
		return err
	}
	if err := setupQuotedStringFn(ri); err != nil {
		return err
	}
	if err := setupDropRunesFn(ri); err != nil {
		return err
	}
	if err := setupEscapeAnyFn(ri); err != nil {
		return err
	}

	// also add these rules to the ruleindex
	gram := `
		// § = don't print child rules when printing the ruleindex (helpful for tests)
		§digit = ("09")-;
		§digits = (digit)+;
		§letter = ("az")- | ("AZ")-;
		§anyRuneFirst = @anyRune(-10000);
		§anyRuneLast = @anyRune(10000);
	`
	gp0 := newGrammarParser(ri)
	fset0 := NewFileSetFromBytes([]byte(gram))
	if err := gp0.parse(fset0); err != nil {
		return err
	}

	return nil
}

//----------

func setupAnyRuneFn(ri *RuleIndex) error {
	name := "anyRune"
	fn := func(args ProcRuleArgs) (Rule, error) {
		parseOrder, err := args.Int(0)
		if err != nil {
			return nil, err
		}
		fr := &FuncRule{name: name, parseOrder: parseOrder}
		fr.fn = func(ps *PState) error {
			if _, p2, err := ps.Sc.ReadRune(ps.Pos); err != nil {
				return err // fails at eof
			} else {
				ps.Pos = p2
				return nil
			}
		}
		return fr, nil
	}
	return ri.setProcRuleFn(name, fn) // grammar call name
}

func setupQuotedStringFn(ri *RuleIndex) error {
	fn := func(args ProcRuleArgs) (Rule, error) {
		// arg: parse order
		parseOrder, err := args.Int(0)
		if err != nil {
			return nil, err
		}
		// arg: escape rune
		sr, err := args.MergedStringRule(1)
		if err != nil {
			return nil, err
		}
		if len(sr.runes) != 1 {
			return nil, fmt.Errorf("expecting escape to be 1 rune only")
		}
		esc := sr.runes[0]
		// arg
		maxLen1, err := args.Int(2)
		if err != nil {
			return nil, err
		}
		// arg
		maxLen2, err := args.Int(3)
		if err != nil {
			return nil, err
		}

		name := fmt.Sprintf("quotedString(%q,%v,%v)", esc, maxLen1, maxLen2)
		fr := &FuncRule{name: name, parseOrder: parseOrder}
		fr.fn = func(ps *PState) error {
			if p2, err := ps.Sc.M.QuotedString2(ps.Pos, esc, int(maxLen1), int(maxLen2)); err != nil {
				return err
			} else {
				ps.Pos = p2
				return nil
			}
		}
		return fr, nil
	}
	return ri.setProcRuleFn("quotedString", fn) // grammar call name
}

func setupDropRunesFn(ri *RuleIndex) error {
	fn := func(args ProcRuleArgs) (Rule, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("expecting at least 2 args")
		}
		srs := []*StringRule{}
		for i := range args {
			sr, err := args.MergedStringRule(i)
			if err != nil {
				return nil, err
			}
			if sr.typ != stringRTOr {
				return nil, fmt.Errorf("expecting type %q: %v", stringRTOr, sr)
			}
			if len(sr.rranges) != 0 {
				return nil, fmt.Errorf("not expecting ranges: %v", sr)
			}
			srs = append(srs, sr)
		}
		// join rules to remove
		m2 := map[rune]bool{}
		for i := 1; i < len(srs); i++ {
			for _, ru := range srs[i].runes {
				m2[ru] = true
			}
		}
		// remove from first rule
		rs := []rune{}
		for _, ru := range srs[0].runes {
			if m2[ru] {
				continue
			}
			rs = append(rs, ru)
		}
		sr3 := *srs[0] // copy
		sr3.runes = rs
		return &sr3, nil
	}
	return ri.setProcRuleFn("dropRunes", fn) // grammar call name
}

func setupEscapeAnyFn(ri *RuleIndex) error {
	// allows to rewind in case of failure
	fn := func(args ProcRuleArgs) (Rule, error) {
		// arg
		parseOrder, err := args.Int(0)
		if err != nil {
			return nil, err
		}
		// arg
		sr, err := args.MergedStringRule(1)
		if err != nil {
			return nil, err
		}
		if len(sr.runes) != 1 {
			return nil, fmt.Errorf("expecting rule with one rune")
		}
		esc := sr.runes[0]

		name := fmt.Sprintf("escapeAny(%q)", esc)
		fr := &FuncRule{name: name, parseOrder: parseOrder}
		fr.fn = func(ps *PState) error {
			if p2, err := ps.Sc.M.EscapeAny(ps.Pos, esc); err != nil {
				return err
			} else {
				ps.Pos = p2
				return nil
			}
		}
		return fr, nil
	}
	return ri.setProcRuleFn("escapeAny", fn) // grammar call name
}

//----------
//----------
//----------

//// commented: using grammar definition
//func parseLetter(ps *PState) error {
//	ps2 := ps.Copy()
//	ru, err := ps2.ReadRune()
//	if err != nil {
//		return err
//	}
//	if !unicode.IsLetter(ru) {
//		return errors.New("not a letter")
//	}
//	ps.Set(ps2)
//	return nil
//}

//// commented: using grammar definition
//func parseDigit(ps *PState) error {
//	ps2 := ps.Copy()
//	ru, err := ps2.ReadRune()
//	if err != nil {
//		return err
//	}
//	if !unicode.IsDigit(ru) {
//		return errors.New("not a digit")
//	}
//	ps.Set(ps2)
//	return nil
//}

// commented: using this won't recognize "digit" in "digits", which won't allow to parse correctly in some cases
//func parseDigits(ps *PState) error {
//	for i := 0; ; i++ {
//		ps2 := ps.copy()
//		ru, err := ps2.readRune()
//		if err != nil {
//			if i > 0 {
//				return nil
//			}
//			return err
//		}
//		if !unicode.IsDigit(ru) {
//			if i == 0 {
//				return errors.New("not a digit")
//			}
//			return nil
//		}
//		ps.set(ps2)
//	}
//}
