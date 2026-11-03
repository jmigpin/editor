package termemu

import (
	"encoding/base64"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type clipboardTui struct {
	TuiMock
	copies []string
}

func (m *clipboardTui) SetClipboard(target byte, text string) {
	m.copies = append(m.copies, string(target)+":"+text)
}

func TestOSC52(t *testing.T) {
	text := "hello\n\u00e1"
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	for _, tt := range []struct {
		name, payload, end string
		want               []string
	}{
		{"clipboard", "52;c;" + encoded, "\a", []string{"c:" + text}},
		{"primary", "52;p;" + encoded, "\x1b\\", []string{"p:" + text}},
		{"both", "52;cppc;" + encoded, "\a", []string{"c:" + text, "p:" + text}},
		{"default", "52;;" + encoded, "\a", []string{"p:" + text}},
		{"selection", "52;s;" + encoded, "\a", []string{"p:" + text}},
		{"clear", "52;c;", "\a", []string{"c:"}},
		{"query", "52;c;?", "\a", nil},
		{"invalid", "52;c;!bad", "\a", nil},
		{"unsupported", "52;q;" + encoded, "\a", nil},
		{"title", "0;title", "\a", nil},
		{"cancel", "52;c;" + encoded, "\x18", nil},
		{"oversize", "52;c;" + strings.Repeat("A", maxOSCPayload), "\a", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &clipboardTui{}
			emu := &Emu{tui: m}
			printed := ""
			p := NewVTParser(strings.NewReader("\x1b]"+tt.payload+tt.end+"OK"), func(op *TermOp) {
				if op.kind == "osc" {
					emu.applyEmit(op)
				} else if op.kind == "print" {
					printed += op.s
				}
			})
			if err := p.Run(); !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(m.copies, tt.want) {
				t.Fatalf("copies: got %q, want %q", m.copies, tt.want)
			}
			if printed != "OK" {
				t.Fatalf("print: %q", printed)
			}
		})
	}
}
