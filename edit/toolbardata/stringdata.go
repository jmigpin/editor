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
func (sd *StringData) GetPartAtIndex(index int) (*Part, bool) {
	for _, p := range sd.Parts {
		if index >= p.Start && index < p.End {
			return p, true
		}
	}
	return nil, false
}
func (sd *StringData) StrWithFirstPartEncoded() (string, bool) {
	if len(sd.Parts) == 0 {
		return "", false
	}
	if len(sd.Parts[0].Args) != 1 {
		return "", false
	}

	arg := sd.Parts[0].Args[0]

	s1 := arg.Str

	s2 := decodeHomeVars(s1)
	s3 := encodeHomeVars(s2)

	if s3 == s1 {
		return "", false
	}

	// replace
	s := sd.Parts[0].Args[0].Start
	e := sd.Parts[0].Args[0].End
	s4 := sd.Str[:s] + s3 + sd.Str[e:]

	return s4, true
}

func (sd *StringData) EncodeFirstPart() (string, bool) {
	if len(sd.Parts) == 0 {
		return "", false
	}
	if len(sd.Parts[0].Args) != 1 {
		return "", false
	}

	s1 := sd.Parts[0].Args[0].Str

	// ensure trailing slash if dir
	s2 := RemoveHomeVars(s1)
	fi, err := os.Stat(s2)
	if err == nil && fi.IsDir() {
		if len(s2) > 0 && s2[len(s2)-1] != '/' {
			s2 += "/"
		}
	}

	s3 := InsertHomeVars(s2)
	if s3 == s1 {
		return "", false
	}

	// replace
	s := sd.Parts[0].Args[0].Start
	e := sd.Parts[0].Args[0].End
	s4 := sd.Str[:s] + s3 + sd.Str[e:]

	return s4, true
}

func (sd *StringData) DecodeFirstPart() string {
	if len(sd.Parts) == 0 {
		return ""
	}
	if len(sd.Parts[0].Args) != 1 {
		return ""
	}
	s1 := sd.Parts[0].Args[0].Str
	s1 = RemoveHomeVars(s1)

	// remove trailing slash if dir
	if len(s1) > 0 && s1[len(s1)-1] == '/' {
		fi, err := os.Stat(s1)
		if err == nil && fi.IsDir() {
			s1 = s1[:len(s1)-1]
		}
	}
	return s1
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
