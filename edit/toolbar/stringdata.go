package toolbar

type StringData struct {
	Parts []*Part
}

func NewStringData(str string) *StringData {
	parts := parseParts(str)
	return &StringData{Parts: parts}
}

// TODO: should be unique
func (tb *StringData) FilenameTag() (string, bool) {
	part, ok := tb.TagPart("f")
	if !ok {
		return "", false
	}
	s := part.JoinArgs().Trim()
	s = insertHomeVar(s)
	return s, true
}

// TODO: should be unique
func (tb *StringData) DirectoryTag() (string, bool) {
	part, ok := tb.TagPart("d")
	if !ok {
		return "", false
	}
	s := part.JoinArgs().Trim()
	s = insertHomeVar(s)
	return s, true
}

// TODO: should be unique
func (tb *StringData) SpecialTag() (string, bool) {
	part, ok := tb.TagPart("s")
	if !ok {
		return "", false
	}
	return part.JoinArgs().Trim(), true
}
func (tb *StringData) TagPart(tag string) (*Part, bool) {
	for _, p := range tb.Parts {
		if p.Tag == tag {
			return p, true
		}
	}
	return nil, false
}
func (tb *StringData) GetPartAtIndex(index int) (*Part, bool) {
	for _, p := range tb.Parts {
		if index >= p.Start && index < p.End {
			return p, true
		}
	}
	return nil, false
}
