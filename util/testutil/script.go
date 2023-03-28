package testutil

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
	"golang.org/x/tools/txtar"
)

// based on txtar (txt archive)
type Script struct {
	ScriptsDir string
	Args       []string
	Cmds       []*ScriptCmd // user cmds (provided)
	Work       bool         // don't remove work dir at end

	ScriptStart func(*testing.T) error // each script init
	ScriptStop  func(*testing.T) error // each script close

	ucmds map[string]*ScriptCmd // user cmds (mapped)
	icmds map[string]*ScriptCmd // internal cmds

	workDir    string
	lastCmdStd [2][]byte // stdin, stdout
	lastCmd    struct {
		stdout []byte
		stderr []byte
		err    []byte
	}
}

func NewScript(args []string) *Script {
	return &Script{Args: args}
}

//----------

func (scr *Script) log(t *testing.T, s string) {
	t.Helper()
	if u := scr.locationInfo(); u != "" {
		s = u + s
	}
	s = strings.TrimRight(s, "\n") // remove newlines
	t.Log(s)                       // adds one newline
}
func (scr *Script) logf(t *testing.T, f string, args ...any) {
	t.Helper()
	scr.log(t, fmt.Sprintf(f, args...))
}

//----------

func (scr *Script) error(err error) error {
	if s := scr.locationInfo(); s != "" {
		return fmt.Errorf("%v%w", s, err)
	}
	return err
}

//----------

func (scr *Script) locationInfo() string {
	// add filename line info
	u := ""
	if filename := os.Getenv("script_filename"); filename != "" {
		u = filename
		if line := os.Getenv("script_line"); line != "" {
			u += ":" + line
		}
		u += ": "
	}
	return u
}

//----------

func (scr *Script) Run(t *testing.T) {
	t.Helper()
	// internal cmds
	icmds := []*ScriptCmd{
		{"ucmd", scr.icUCmd}, // run user cmd
		{"exec", scr.icExec},
		{"contains", scr.icContains},
		{"setenv", scr.icSetEnv},
		{"fail", scr.icFail},
		{"cd", scr.icChangeDir},
	}
	scr.icmds = mapScriptCmds(icmds)
	// user cmds
	scr.ucmds = mapScriptCmds(scr.Cmds)

	if err := scr.runDir(t, scr.ScriptsDir); err != nil {
		t.Fatal(err)
	}
}
func (scr *Script) runDir(t *testing.T, dir string) error {
	t.Helper()
	des, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, de := range des {
		if de.IsDir() {
			continue
		}
		filename := filepath.Join(dir, de.Name())
		if !scr.runFile(t, filename) {
			break
		}
	}
	return nil
}
func (scr *Script) runFile(t1 *testing.T, filename string) bool {
	t1.Helper()
	name := filepath.Base(filename)
	return t1.Run(name, func(t2 *testing.T) {
		t1.Helper()
		t2.Helper()
		if err := scr.runSubTest(t2, filename); err != nil {
			err2 := scr.error(err) // setup error with location info
			t2.Fatal(err2)
		}
	})
}
func (scr *Script) runSubTest(t *testing.T, filename string) error {
	t.Helper()

	scr.logf(t, "subtest: %v", filename)

	ar, err := txtar.ParseFile(filename)
	if err != nil {
		return err
	}
	if scr.ScriptStart != nil {
		if err := scr.ScriptStart(t); err != nil {
			return err
		}
	}
	if scr.ScriptStop != nil {
		defer func() {
			if err := scr.ScriptStop(t); err != nil {
				t.Error(err)
			}
		}()
	}
	return scr.runScript(t, filename, ar)
}
func (scr *Script) runScript(t *testing.T, filename string, ar *txtar.Archive) error {
	t.Helper()

	// create working dir
	// TODO: review, not working properly
	//dir, err := os.MkdirTemp(t.TempDir(), "editor_testutil_work.*")
	dir, err := os.MkdirTemp("", "editor_testutilscript*")
	if err != nil {
		return err
	}
	scr.workDir = dir
	t.Setenv("WORK", scr.workDir)
	scr.logf(t, "script_workdir: %v", scr.workDir)
	defer func() {
		t.Helper()
		if scr.Work {
			scr.logf(t, "workDir not cleaned")
		} else {
			_ = os.RemoveAll(scr.workDir)
		}
	}()

	// keep/restore current dir
	keepDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(keepDir)

	// switch to working dir
	if err := os.Chdir(scr.workDir); err != nil {
		return err
	}

	// setup tmp dir in workdir for program to create its own tmp files
	scriptTmpDir := filepath.Join(scr.workDir, "tmp")
	t.Setenv("TMPDIR", scriptTmpDir)
	if err := iout.MkdirAll(scriptTmpDir); err != nil {
		return err
	}

	for _, f := range ar.Files {
		if err := scr.writeToTmp(f.Name, f.Data); err != nil {
			return err
		}
	}

	// run script
	rd := bytes.NewReader(ar.Comment)
	scanner := bufio.NewScanner(rd)
	line := 0
	t.Setenv("script_filename", filename) // update for logs/errors
	for scanner.Scan() {
		line++
		t.Setenv("script_line", fmt.Sprintf("%d", line)) // update for logs/errors

		txt := strings.TrimSpace(scanner.Text())
		// comments
		if strings.HasPrefix(txt, "#") {
			continue
		}
		// empty lines
		if txt == "" {
			continue
		}
		// as least an arg after empty lines check
		args := scr.splitArgs(txt)

		cmd, ok := scr.icmds[args[0]]
		if !ok {
			return fmt.Errorf("cmd not found: %v", args[0])
		}
		scr.logf(t, "$%v", args)
		if err := cmd.Fn(t, args); err != nil {
			return err
		}
	}
	return scanner.Err()
}

//----------

func (scr *Script) splitArgs(s string) []string {
	quoted := false
	escape := false
	a := strings.FieldsFunc(s, func(r rune) bool {
		if r == '\\' {
			escape = true
			return false
		}
		if escape {
			escape = false
			return false
		}
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})
	return a
}

//----------

func (scr *Script) collectOutput(t *testing.T, fn func() error) error {
	stdout, stderr, err := CollectLog(t, fn)

	scr.lastCmd.stdout = stdout
	scr.lastCmd.stderr = stderr
	scr.lastCmd.err = nil
	if err != nil {
		scr.lastCmd.err = []byte(err.Error())
	}

	return err
}

//----------

func (scr *Script) writeToTmp(filename string, data []byte) error {
	filename2 := filepath.Join(scr.workDir, filename)
	return iout.MkdirAllWriteFile(filename2, data, 0o644)
}

//----------

func (scr *Script) icExec(t *testing.T, args []string) error {
	args = args[1:] // drop "exec"
	if len(args) <= 0 {
		return fmt.Errorf("expecting args, got %v", len(args))
	}
	ctx := context.Background()
	ec := exec.CommandContext(ctx, args[0], args[1:]...)

	//ec.Dir = // commented: dir set with os.Chdir previously

	return scr.collectOutput(t, func() error {
		// setup cmd stdout inside collectoutput
		// TODO: stdin?
		ec.Stdout = os.Stdout
		ec.Stderr = os.Stderr

		ci := osutil.NewCmdI(ec)
		ci = osutil.NewSetSidCmd(ctx, ci)
		ci = osutil.NewShellCmd(ci)
		return osutil.RunCmdI(ci)
	})
}

//----------

func (scr *Script) icUCmd(t *testing.T, args []string) error {
	args = args[1:] // drop "cmd"
	cmd, ok := scr.ucmds[args[0]]
	if !ok {
		return fmt.Errorf("cmd not found: %v", args[0])
	}
	return scr.collectOutput(t, func() error {
		return cmd.Fn(t, args)
	})
}

//----------

func (scr *Script) icContains(t *testing.T, args []string) error {
	args = args[1:] // drop "contains"
	if len(args) != 2 {
		return fmt.Errorf("expecting 2 args, got %v", args)
	}

	data, ok := scr.lastCmdContent(args[0])
	if !ok {
		return fmt.Errorf("unknown content: %v", args[0])
	}

	// pattern
	u, err := strconv.Unquote(args[1])
	if err != nil {
		return err
	}
	pattern := u

	if !bytes.Contains(data, []byte(pattern)) {
		//return fmt.Errorf("contains: no match:\npattern=[%v]\ndata=[%v]", pattern, string(data))
		return fmt.Errorf("contains: no match")
	}
	return nil
}

//----------

func (scr *Script) icSetEnv(t *testing.T, args []string) error {
	args = args[1:] // drop "setenv"
	if len(args) != 1 && len(args) != 2 {
		return fmt.Errorf("expecting 1 or 2 args, got %v", args)
	}
	v := "" // allow setting to empty
	if len(args) == 2 {
		v = args[1]

		// allow env expansion when setting env vars
		v = os.Expand(v, os.Getenv)

		// allow expansion of lastcmd
		data, ok := scr.lastCmdContent(v)
		if ok {
			v = string(data)
		}
	}
	t.Setenv(args[0], v)
	return nil
}

//----------

func (scr *Script) icFail(t *testing.T, args []string) error {
	t.Helper()
	args = args[1:] // drop "fail"
	if len(args) < 1 {
		return fmt.Errorf("expecting at least 1 arg, got %v", args)
	}
	cmd, ok := scr.icmds[args[0]]
	if !ok {
		return fmt.Errorf("cmd not found: %v", args[0])
	}
	err := cmd.Fn(t, args)
	if err == nil {
		return fmt.Errorf("expected failure but got no error")
	}
	scr.logf(t, "fail ok: %v", err)
	return nil
}

//----------

func (scr *Script) icChangeDir(t *testing.T, args []string) error {
	args = args[1:] // drop "cd"
	if len(args) != 1 {
		return fmt.Errorf("expecting 1 arg, got %v", args)
	}
	dir := args[0]
	return os.Chdir(dir)
}

//----------

func (scr *Script) lastCmdContent(name string) ([]byte, bool) {
	switch name {
	case "stdout":
		return scr.lastCmd.stdout, true
	case "stderr":
		return scr.lastCmd.stderr, true
	case "error":
		return scr.lastCmd.err, true
	}
	return nil, false
}

//----------
//----------
//----------

type ScriptCmd struct {
	Name string
	Fn   func(t *testing.T, args []string) error
}

func mapScriptCmds(w []*ScriptCmd) map[string]*ScriptCmd {
	m := map[string]*ScriptCmd{}
	for _, cmd := range w {
		m[cmd.Name] = cmd
	}
	return m
}

//----------
//----------
//----------
