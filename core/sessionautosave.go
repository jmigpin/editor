package core

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/iout"
)

const (
	sessionAutoSaveDelay = 30 * time.Second
)

type SessionAutoSaver struct {
	ed *Editor

	mu       sync.Mutex
	timer    *time.Timer
	deadline time.Time
	disabled int
	stopped  bool
}

func NewSessionAutoSaver(ed *Editor) *SessionAutoSaver {
	return &SessionAutoSaver{ed: ed}
}

func (sas *SessionAutoSaver) Trigger(reason string) {
	if !sas.enabled() {
		return
	}
	if !sas.hasAutoTargets() {
		return
	}
	sas.schedule(sessionAutoSaveDelay)
}

func (sas *SessionAutoSaver) RunDisabled(fn func()) {
	sas.mu.Lock()
	sas.disabled++
	sas.mu.Unlock()
	defer func() {
		sas.mu.Lock()
		sas.disabled--
		sas.mu.Unlock()
	}()

	fn()
}

func (sas *SessionAutoSaver) FlushAndStop() error {
	sas.mu.Lock()
	sas.stopped = true
	if sas.timer != nil {
		sas.timer.Stop()
		sas.timer = nil
	}
	sas.deadline = time.Time{}
	sas.mu.Unlock()

	return sas.saveAutoTargetsDirect(false)
}

//----------

func (sas *SessionAutoSaver) enabled() bool {
	sas.mu.Lock()
	defer sas.mu.Unlock()
	return !sas.stopped && sas.disabled == 0
}

func (sas *SessionAutoSaver) schedule(delay time.Duration) {
	sas.mu.Lock()
	defer sas.mu.Unlock()
	if sas.stopped || sas.disabled > 0 {
		return
	}

	deadline := time.Now().Add(delay)
	if sas.timer != nil && !deadline.Before(sas.deadline) {
		return
	}

	sas.deadline = deadline
	d := time.Until(deadline)
	if d < 0 {
		d = 0
	}
	if sas.timer == nil {
		sas.timer = time.AfterFunc(d, sas.fire)
	} else {
		sas.timer.Reset(d)
	}
}

func (sas *SessionAutoSaver) hasAutoTargets() bool {
	targets, err := sas.sessionSaveTargetsFromRootToolbar(true)
	if err != nil {
		return false
	}
	return len(targets) > 0
}

func (sas *SessionAutoSaver) fire() {
	sas.mu.Lock()
	if sas.stopped {
		sas.mu.Unlock()
		return
	}
	sas.timer = nil
	sas.deadline = time.Time{}
	sas.mu.Unlock()

	if err := sas.saveAutoTargetsOnUI(); err != nil {
		sas.ed.Error(err)
	}
}

//----------

func (sas *SessionAutoSaver) saveAutoTargetsOnUI() error {
	var targets []*sessionSaveTarget
	var err error
	sas.ed.UI.WaitRunOnUIGoRoutine(func() {
		targets, err = sas.sessionSaveTargetsFromRootToolbar(true)
	})
	if err != nil {
		return err
	}
	return sas.saveTargets(targets, "session auto saved")
}

func (sas *SessionAutoSaver) saveAutoTargetsDirect(report bool) error {
	targets, err := sas.sessionSaveTargetsFromRootToolbar(true)
	if err != nil {
		return err
	}
	msg := ""
	if report {
		msg = "session auto saved"
	}
	return sas.saveTargets(targets, msg)
}

func (sas *SessionAutoSaver) sessionSaveTargetsFromRootToolbar(autosOnly bool) ([]*sessionSaveTarget, error) {
	data := toolbarparser.Parse(sas.ed.UI.Root.Toolbar.Str())
	targets := []*sessionSaveTarget{}
	var me iout.MultiError
	for _, part := range data.Parts {
		target, ok, err := parseSessionSavePart(part)
		if err != nil {
			if autosOnly && !partHasAutoFlag(part) {
				continue
			}
			me.Add(err)
			continue
		}
		if autosOnly && (!ok || !target.Auto) {
			continue
		}
		if ok {
			targets = append(targets, target)
		}
	}
	return targets, me.Result()
}

func (sas *SessionAutoSaver) saveTargets(targets []*sessionSaveTarget, msg string) error {
	var me iout.MultiError
	for _, target := range targets {
		me.Add(saveSessionTarget(sas.ed, target))
	}
	if err := me.Result(); err != nil {
		return err
	}
	if msg != "" && len(targets) > 0 {
		sas.ed.Message(sessionSaveMessage(msg, targets, time.Now()))
	}
	return nil
}

//----------

func (ed *Editor) triggerSessionAutoSave(reason string) {
	if ed.sessionAutoSaver != nil {
		ed.sessionAutoSaver.Trigger(reason)
	}
}

func (ed *Editor) triggerSessionAutoSaveText(reason string) {
	ed.triggerSessionAutoSave(reason)
}

func (ed *Editor) triggerSessionAutoSaveLayout(reason string) {
	ed.triggerSessionAutoSave(reason)
}

func (ed *Editor) runSessionAutoSaveDisabled(fn func()) {
	if ed.sessionAutoSaver == nil {
		fn()
		return
	}
	ed.sessionAutoSaver.RunDisabled(fn)
}

func (ed *Editor) flushAndStopSessionAutoSave() {
	if ed.sessionAutoSaver == nil {
		return
	}
	if err := ed.sessionAutoSaver.FlushAndStop(); err != nil {
		ed.Error(err)
	}
}

//----------

type sessionSaveTarget struct {
	Cmd  string
	Dest string
	Auto bool
}

func parseSessionSavePart(part *toolbarparser.Part) (*sessionSaveTarget, bool, error) {
	if len(part.Args) == 0 {
		return nil, false, nil
	}

	cmd := part.Args[0].String()
	if cmd != "SaveSession" && cmd != "SaveSessionFile" {
		return nil, false, nil
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	target := &sessionSaveTarget{Cmd: cmd}
	fs.BoolVar(&target.Auto, "auto", false, sessionAutoSaveAutoUsage())
	if err := fs.Parse(part.ArgsStrings()[1:]); err != nil {
		return nil, true, sessionSaveFlagErr(fs, err)
	}

	args := fs.Args()
	if len(args) > 1 {
		return nil, true, fmt.Errorf("%s: too many arguments", strings.ToLower(cmd))
	}
	if len(args) == 1 {
		target.Dest = args[0]
		if cmd == "SaveSessionFile" {
			if s, err := strconv.Unquote(target.Dest); err == nil {
				target.Dest = s
			}
		}
	}
	if target.Dest == "" {
		return nil, true, fmt.Errorf("%s: missing destination", strings.ToLower(cmd))
	}
	return target, true, nil
}

func sessionSaveFlagErr(fs *flag.FlagSet, err error) error {
	if err == flag.ErrHelp {
		buf := &bytes.Buffer{}
		fs.SetOutput(buf)
		fs.Usage()
		return fmt.Errorf("%w\n%v", err, buf.String())
	}
	return err
}

func sessionAutoSaveAutoUsage() string {
	return fmt.Sprintf("autosave session changes after %v while this command is present in the root toolbar", sessionAutoSaveDelay)
}

func partHasAutoFlag(part *toolbarparser.Part) bool {
	for _, arg := range part.Args[1:] {
		if arg.String() == "-auto" {
			return true
		}
	}
	return false
}

func sessionSaveTargetsString(targets []*sessionSaveTarget) string {
	w := []string{}
	for _, target := range targets {
		w = append(w, target.Dest)
	}
	return strings.Join(w, ", ")
}

func sessionSaveMessage(msg string, targets []*sessionSaveTarget, t time.Time) string {
	return fmt.Sprintf("%s %s: %s", msg, t.Format("15:04:05"), sessionSaveTargetsString(targets))
}

func saveSessionTarget(ed *Editor, target *sessionSaveTarget) error {
	switch target.Cmd {
	case "SaveSession":
		return saveSessionName(ed, target.Dest)
	case "SaveSessionFile":
		return SaveSessionToFile(ed, target.Dest)
	default:
		return errors.New("unexpected session save target")
	}
}
