package parseutil

import (
	"reflect"
	"testing"
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		sep     rune
		want    []string
		wantErr bool
	}{
		{"simple", "a,b,c", ',', []string{"a", "b", "c"}, false},
		{"escaped-separator", `a\,b,c`, ',', []string{"a,b", "c"}, false},
		{"quoted-separator", `"a,b",c`, ',', []string{"a,b", "c"}, false},
		{"single-quoted-separator", `'a,b',c`, ',', []string{"a,b", "c"}, false},
		{"trailing-separator", "a,", ',', nil, true},
		{"empty-field", "a,,b", ',', nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFields(tt.src, tt.sep)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got=%#v want=%#v", got, tt.want)
			}
		})
	}
}
