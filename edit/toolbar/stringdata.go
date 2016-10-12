package toolbar

import "os"

type StringData struct {
	Parts []*Part
}

func NewStringData(str string) *StringData {
	parts := parseParts(str)
	return &StringData{Parts: parts}
}
func (sd *StringData) FirstPartFilepath() string {
	if len(sd.Parts) == 0 {
		return ""
	}
	if len(sd.Parts[0].Args) != 1 {
		return ""
	}
	v := sd.Parts[0].Trim()
	v = RemoveHomeTilde(v)
	return v
}

func (sd *StringData) FirstPartFilename() (string, bool) {
	v := sd.FirstPartFilepath()
	fi, err := os.Stat(v)
	if err != nil {
		return "", false
	}
	if !fi.Mode().IsRegular() {
		return "", false
	}
	return v, true
}
func (sd *StringData) FirstPartDirectory() (string, bool) {
	v := sd.FirstPartFilepath()
	fi, err := os.Stat(v)
	if err != nil {
		return "", false
	}
	if !fi.IsDir() {
		return "", false
	}
	return v, true
}

func (sd *StringData) GetPartAtIndex(index int) (*Part, bool) {
	for _, p := range sd.Parts {
		if index >= p.Start && index < p.End {
			return p, true
		}
	}
	return nil, false
}
