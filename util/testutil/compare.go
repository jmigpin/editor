package testutil

import "strings"

// Useful to compare src code lines.
func TrimLineSpaces(str string) string {
	return TrimLineSpaces2(str, "")
}

func TrimLineSpaces2(str string, pre string) string {
	a := strings.Split(str, "\n")
	u := []string{}
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" {
			u = append(u, s)
		}
	}
	return pre + strings.Join(u, "\n"+pre)
}
