package core

import (
	"errors"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/jmigpin/editor/core/toolbarparser"
)

func TestParseSessionSavePart(t *testing.T) {
	tests := []struct {
		src  string
		cmd  string
		dest string
		auto bool
	}{
		{"SaveSession aa", "SaveSession", "aa", false},
		{"SaveSession -auto aa", "SaveSession", "aa", true},
		{"SaveSessionFile -auto project.editor_session", "SaveSessionFile", "project.editor_session", true},
	}

	for _, test := range tests {
		part := toolbarparser.Parse(test.src).Parts[0]
		target, ok, err := parseSessionSavePart(part)
		if err != nil {
			t.Fatalf("%q: %v", test.src, err)
		}
		if !ok {
			t.Fatalf("%q: not handled", test.src)
		}
		if target.Cmd != test.cmd || target.Dest != test.dest || target.Auto != test.auto {
			t.Fatalf("%q: got %#v", test.src, target)
		}
	}
}

func TestParseSessionSavePartErrors(t *testing.T) {
	tests := []string{
		"SaveSession",
		"SaveSession -auto",
		"SaveSession -delay 2s aa",
		"SaveSession -bad aa",
		"SaveSession aa bb",
	}

	for _, test := range tests {
		part := toolbarparser.Parse(test).Parts[0]
		if _, _, err := parseSessionSavePart(part); err == nil {
			t.Fatalf("%q: expected error", test)
		}
	}
}

func TestParseSessionSavePartHelpShowsAutoDelay(t *testing.T) {
	part := toolbarparser.Parse("SaveSession -h").Parts[0]
	_, _, err := parseSessionSavePart(part)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("got %v, want flag.ErrHelp", err)
	}
	errStr := err.Error()
	for _, want := range []string{"-auto", sessionAutoSaveDelay.String()} {
		if !strings.Contains(errStr, want) {
			t.Fatalf("help missing %q:\n%v", want, errStr)
		}
	}
}

func TestPartHasAutoFlag(t *testing.T) {
	tests := []struct {
		src  string
		want bool
	}{
		{"SaveSession aa", false},
		{"SaveSession -auto aa", true},
		{"SaveSessionFile -auto file.editor_session", true},
		{"NewRow", false},
	}

	for _, test := range tests {
		part := toolbarparser.Parse(test.src).Parts[0]
		got := partHasAutoFlag(part)
		if got != test.want {
			t.Fatalf("%q: got %v, want %v", test.src, got, test.want)
		}
	}
}

func TestSessionSaveTargetsString(t *testing.T) {
	targets := []*sessionSaveTarget{
		{Dest: "aa"},
		{Dest: "bb.editor_session"},
	}
	got := sessionSaveTargetsString(targets)
	if got != "aa, bb.editor_session" {
		t.Fatalf("got %q", got)
	}
}

func TestSessionSaveMessage(t *testing.T) {
	targets := []*sessionSaveTarget{
		{Dest: "aa"},
		{Dest: "bb.editor_session"},
	}
	t1 := time.Date(2026, 6, 22, 9, 8, 7, 0, time.Local)
	got := sessionSaveMessage("session auto saved", targets, t1)
	want := "session auto saved 09:08:07: aa, bb.editor_session"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSessionAutoSaverEarliestDeadline(t *testing.T) {
	sas := NewSessionAutoSaver(nil)
	defer stopSessionAutoSaverTimer(sas)

	sas.schedule(30 * time.Second)
	d1 := sas.deadline
	sas.schedule(5 * time.Second)
	d2 := sas.deadline
	if !d2.Before(d1) {
		t.Fatalf("short trigger did not move deadline earlier")
	}

	sas.schedule(30 * time.Second)
	d3 := sas.deadline
	if !d3.Equal(d2) {
		t.Fatalf("long trigger postponed deadline: %v -> %v", d2, d3)
	}
}

func stopSessionAutoSaverTimer(sas *SessionAutoSaver) {
	sas.mu.Lock()
	defer sas.mu.Unlock()
	sas.stopped = true
	if sas.timer != nil {
		sas.timer.Stop()
	}
}
