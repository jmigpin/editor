package parseutil

import "strings"

func VersionLessThan(a, b string) bool {
	return VersionOrdinal(a) < VersionOrdinal(b)
}

// constructs a byte array (returned as a string) with the count of sequential digits to be able to compare "1.9"<"1.10"
func VersionOrdinal(version string) string {
	a := strings.Split(version, ".")
	r := []byte{}
	for _, s := range a {
		r = append(r, byte(len(s)))
		r = append(r, []byte(s)...)
	}
	return string(r)
}
