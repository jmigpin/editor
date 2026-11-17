package osutil

import "testing"

func TestIsKnownURI(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  bool
	}{
		{"https://example.org/a?b=c", true},
		{"HTTP://example.org", true},
		{"http://a.com/b/c\n/d/e", true},
		{"ftp://example.org/file", true},
		{"file:///tmp/file.txt", true},
		{"mailto:user@example.org", true},
		{"src/main.go", false},
		{"/tmp/file.txt", false},
		{"file.txt:12:3", false},
		{"abcde:valor", false},
		{"", false},
	} {
		if got := IsKnownURI(tt.value); got != tt.want {
			t.Errorf("IsKnownURI(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}
