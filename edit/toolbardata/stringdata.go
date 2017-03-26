package toolbardata

import "os"

type StringData struct {
	Str   string
	Parts []*Part
}

func NewStringData(str string) *StringData {
	parts := parseParts(str)
	return &StringData{Str: str, Parts: parts}
}
func (sd *StringData) ReplacePart(i int, str string) string {
	return sd.Parts[i].Replace(sd.Str, str)
}
func (sd *StringData) GetPartAtIndex(index int) (*Part, bool) {
	for _, p := range sd.Parts {
		if index >= p.Start && index < p.End {
			return p, true
		}
	}
	return nil, false
}
func (sd *StringData) EncodePart0Arg0() (string, bool) {
	if len(sd.Parts) == 0 {
		return "", false
	}
	if len(sd.Parts[0].Args) < 1 {
		return "", false
	}
	s1 := sd.Parts[0].Args[0].Str
	s2 := decodeHomeVars(s1)
	s3 := encodeHomeVars(s2)
	return s3, true
}
func (sd *StringData) DecodePart0Arg0() (string, bool) {
	if len(sd.Parts) == 0 {
		return "", false
	}
	if len(sd.Parts[0].Args) == 0 {
		return "", false
	}
	s1 := sd.Parts[0].Args[0].Str
	s2 := decodeHomeVars(s1)
	return s2, true
}
func (sd *StringData) StrWithPart0Arg0Encoded() string {
	s1, ok := sd.EncodePart0Arg0()
	if !ok {
		return sd.Str
	}
	// replace
	s2 := sd.Parts[0].ReplaceArg(0, s1)
	s3 := sd.ReplacePart(0, s2)
	return s3
}
func (sd *StringData) StrWithPart0Arg0Decoded() string {
	s1, ok := sd.DecodePart0Arg0()
	if !ok {
		return sd.Str
	}
	// replace
	s2 := sd.Parts[0].ReplaceArg(0, s1)
	s3 := sd.ReplacePart(0, s2)
	return s3
}

func encodeHomeVars(str string) string {
	s1 := ensureTrailingSlashIfDir(str)
	return InsertHomeVars(s1)
}
func decodeHomeVars(str string) string {
	s1 := RemoveHomeVars(str)
	return removeTrailingSlash(s1)
}

func ensureTrailingSlashIfDir(s string) string {
	if len(s) > 0 && s[len(s)-1] != '/' {
		fi, err := os.Stat(s)
		if err == nil && fi.IsDir() {
			s += "/"
		}
	}
	return s
}
