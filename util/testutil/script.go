package testutil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
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
	s = strings.TrimRight(s, "\n") // remove newlines
	t.Log(s)                       // adds one newline
}
func (scr *Script) logf(t *testing.T, f string, args ...interface{}) {
	scr.log(t, fmt.Sprintf(f, args...))
}

//----------

func (scr *Script) Run(t *testing.T) {
	// internal cmds
	icmds := []*ScriptCmd{
		&ScriptCmd{"ucmd", scr.icUCmd}, // run user cmd
		&ScriptCmd{"exec", scr.icExec},
		&ScriptCmd{"contains", scr.icContains},
		&ScriptCmd{"setenv", scr.icSetEnv},
		&ScriptCmd{"fail", scr.icFail},
		&ScriptCmd{"cd", scr.icChangeDir},
	}
	scr.icmds = mapScriptCmds(icmds)
	// user cmds
	scr.ucmds = mapScriptCmds(scr.Cmds)

	if err := scr.runDir(t, scr.ScriptsDir); err != nil {
		t.Fatal(err)
	}
}
func (scr *Script) runDir(t *testing.T, dir string) error {
	des, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, de := range des {
		if de.IsDir() {
			continue
		}
		filename := filepath.Join(dir, de.Name())
		if err := scr.runFile(t, filename); err != nil {
			return err
		}
	}
	return nil
}
func (scr *Script) runFile(t1 *testing.T, filename string) error {
	err0 := error(nil)
	name := filepath.Base(filename)
	ok := t1.Run(name, func(t2 *testing.T) {
		// running as a sub-test
		scr.logf(t2, "script: %v", filename)

		ar, err := txtar.ParseFile(filename)
		if err != nil {
			err0 = err // stop testing by returning an error
			return
		}

		if err := scr.runScript(t2, filename, ar); err != nil {
			t2.Logf("FAIL: %v", err)
			//t2.Fail()  // continues testing
			t2.Fatal() // also seems to continue, need t1
			t1.Fatal() // stop testing
		}
	})
	_ = ok
	return err0
}
func (scr *Script) runScript(t *testing.T, filename string, ar *txtar.Archive) error {
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
	for scanner.Scan() {
		line++
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
			err := fmt.Errorf("cmd not found: %v", args[0])
			return &lineError{filename, line, err}
		}
		scr.logf(t, "%v: %v", args[0], args[1:])
		if err := cmd.Fn(t, args); err != nil {
			return &lineError{filename, line, err}
		}
	}
	if err := scanner.Err(); err != nil {
		return &lineError{filename, line, err}
	}
	return nil
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

	type datat struct {
		name string
		data []byte
	}
	datats := []*datat{
		&datat{"stdout", scr.lastCmd.stdout},
		&datat{"stderr", scr.lastCmd.stderr},
		&datat{"error", scr.lastCmd.err},
	}

	for _, d := range datats {
		if d.name != args[0] {
			continue
		}
		data := d.data

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
	return fmt.Errorf("unhandled args: %v", args)
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
	}
	t.Setenv(args[0], v)
	return nil
}

//----------

func (scr *Script) icFail(t *testing.T, args []string) error {
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

type lineError struct {
	filename string
	line     int
	err      error
}

func (le *lineError) Error() string {
	return fmt.Sprintf("%v:%v: %v", le.filename, le.line, le.err)
}
func (le *lineError) Is(err error) bool {
	return errors.Is(le.err, err)
}

//----------
//----------
//----------
