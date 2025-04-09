package testutil

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
	"golang.org/x/tools/txtar"
)

// based on txtar (txt archive)
type Script struct {
	ScriptsDir     string
	Args           []string
	Cmds           []*ScriptCmd // user cmds (provided)
	Work           bool         // don't remove work dir at end
	NoFilepathsFix bool         // don't rewrite filepaths for current dir

	ScriptStart func(*testing.T) error // each script init
	ScriptStop  func(*testing.T) error // each script close
}

func NewScript(args []string) *Script {
	return &Script{Args: args}
}

//----------

func (scr *Script) Run(t *testing.T) {
	t.Helper()
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
func (scr *Script) runFile(t *testing.T, filename string) bool {
	t.Helper()
	name := filepath.Base(filename)
	return t.Run(name, func(t *testing.T) {
		t.Helper()

		//t.Parallel()

		st := newScriptTest(scr, filename)
		if err := st.runFile(t); err != nil {
			t.Fatal(err)
		}
	})
}

//----------
//----------
//----------

type ScriptTest struct {
	scr *Script
	Env *Env

	ucmds map[string]*ScriptCmd // user cmds (mapped)
	icmds map[string]*ScriptCmd // internal cmds

	filename string

	workDir string // test root dir
	CurDir  string // current dir

	lastCmdStd [2][]byte // stdin, stdout
	lastCmd    struct {
		stdout []byte
		stderr []byte
		err    []byte
	}
}

func newScriptTest(scr *Script, filename string) *ScriptTest {
	st := &ScriptTest{scr: scr, Env: NewEnvMap(), filename: filename}

	// internal cmds
	icmds := []*ScriptCmd{
		{"fail", icFail},
		{"cd", icChangeDir},
		{"setenv", icSetEnv},
		{"contains", icContains},
		{"containsre", icContainsRegexp},

		// TODO: remove, no need for ucmd/exec anymore
		{"ucmd", icUCmd}, // user cmd
		{"exec", icExec}, // internal cmd
	}
	st.icmds = mapScriptCmds(icmds)
	st.ucmds = mapScriptCmds(scr.Cmds) // script user cmds

	return st
}

//----------

func (st *ScriptTest) runFile(t *testing.T) error {
	if err := st.runFile2(t); err != nil {
		return st.error(err) // setup error with location info
	}
	return nil
}
func (st *ScriptTest) runFile2(t *testing.T) error {
	t.Helper()

	st.Logf(t, "SCRIPT_FILENAME: %v", st.filename)

	ar, err := txtar.ParseFile(st.filename)
	if err != nil {
		return err
	}
	if st.scr.ScriptStart != nil {
		if err := st.scr.ScriptStart(t); err != nil {
			return err
		}
	}
	if st.scr.ScriptStop != nil {
		defer func() {
			if err := st.scr.ScriptStop(t); err != nil {
				t.Error(err)
			}
		}()
	}

	// create working dir
	dir, err := os.MkdirTemp("", "editor_testutilscript*")
	if err != nil {
		return err
	}

	st.workDir = dir
	st.CurDir = st.workDir

	// work directory env and cleanup
	st.Env.Set("WORK", st.workDir)
	st.Logf(t, "script_workdir: %v", st.workDir)
	defer func() {
		t.Helper()
		if !st.scr.Work {
			u := st.Env.Get("script_keepwork")
			if v, err := strconv.ParseBool(u); err == nil {
				st.scr.Work = v
			}
		}
		if st.scr.Work {
			//scr.logf(t, "workDir not cleaned")
		} else {
			_ = os.RemoveAll(st.workDir)
		}
	}()

	//// keep/restore current dir
	//keepDir, err := os.Getwd()
	//if err != nil {
	//	return err
	//}
	//defer os.Chdir(keepDir)

	//// switch to working dir
	//if err := os.Chdir(st.workDir); err != nil {
	//	return err
	//}

	// setup tmp dir in workdir for program to create its own tmp files
	scriptTmpDir := filepath.Join(st.workDir, "tmp")
	st.Env.Set("TMPDIR", scriptTmpDir)
	if err := iout.MkdirAll(scriptTmpDir); err != nil {
		return err
	}

	for _, f := range ar.Files {
		u := filepath.Join(st.workDir, f.Name)
		if err := st.writeFile(u, f.Data); err != nil {
			return err
		}
	}

	// run script
	rd := bytes.NewReader(ar.Comment)
	scanner := bufio.NewScanner(rd)
	line := 0
	st.Env.Set("script_filename", st.filename) // update for logs/errors
	for scanner.Scan() {
		line++
		st.Env.Set("script_line", fmt.Sprintf("%d", line)) // update for logs/errors

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
		args := st.splitArgs(txt)
		st.Logf(t, "SCRIPT: %v", args)

		if err := runCmd(t, st, args); err != nil {
			return err
		}
	}
	return scanner.Err()
}

//----------

func (st *ScriptTest) lastCmdContent(name string) ([]byte, bool) {
	switch name {
	case "stdout":
		return st.lastCmd.stdout, true
	case "stderr":
		return st.lastCmd.stderr, true
	case "error":
		return st.lastCmd.err, true
	}
	return nil, false
}

//----------

func (st *ScriptTest) collectOutput(t *testing.T, fn func() error) error {
	logf := func(f string, args ...any) {
		if st.scr.NoFilepathsFix {
			t.Logf(f, args...)
		} else {
			u := fmt.Sprintf(f, args...)
			u = string(fixFilepathsForCurDir([]byte(u), st.CurDir))
			t.Log(u)
		}
	}
	//stdout, stderr, err := CollectLog(t, fn)
	stdout, stderr, err := CollectLog2(t, logf, fn)

	st.lastCmd.stdout = stdout
	st.lastCmd.stderr = stderr
	st.lastCmd.err = nil
	if err != nil {
		st.lastCmd.err = []byte(err.Error())
	}

	return err
}

func (st *ScriptTest) writeFile(filename string, data []byte) error {
	return iout.MkdirAllWriteFile(filename, data, 0o644)
}

//----------

func (st *ScriptTest) Logf(t *testing.T, f string, args ...any) {
	t.Helper()
	st.Log(t, fmt.Sprintf(f, args...))
}
func (st *ScriptTest) Log(t *testing.T, s string) {
	t.Helper()
	if u := st.locationInfo(); u != "" {
		s = u + s
	}
	s = strings.TrimRight(s, "\n") // remove newlines
	t.Log(s)                       // adds one newline
}

func (st *ScriptTest) error(err error) error {
	if s := st.locationInfo(); s != "" {
		return fmt.Errorf("%v%w", s, err)
	}
	return err
}
func (st *ScriptTest) locationInfo() string {
	// add filename line info
	u := ""
	if filename := st.Env.Get("script_filename"); filename != "" {
		u = filename
		if line := st.Env.Get("script_line"); line != "" {
			u += ":" + line
		}
		u += ": "
	}
	return u
}

//----------

func (st *ScriptTest) splitArgs(s string) []string {
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
//----------
//----------

type ScriptCmd struct {
	Name string
	Fn   ScriptCmdFn
}

type ScriptCmdFn func(t *testing.T, st *ScriptTest, args []string) error

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

func runCmd(t *testing.T, st *ScriptTest, args []string) error {
	// user cmds
	if cmd, ok := st.ucmds[args[0]]; ok {
		return st.collectOutput(t, func() error {
			return cmd.Fn(t, st, args)
		})
	}
	// internal cmds
	if cmd, ok := st.icmds[args[0]]; ok {
		//return st.collectOutput(t, func() error {}) // commented: don't collect output from all internal cmds, otherwise it will be getting empty output from cmds like "contains" that follow the pretented cmd that has the desired output

		return cmd.Fn(t, st, args)
	}
	// default cmd
	return icExec(t, st, args)
}

//----------

func icExec(t *testing.T, st *ScriptTest, args []string) error {
	// drop "exec"
	if args[0] == "exec" {
		args = args[1:]
	}

	if len(args) <= 0 {
		return fmt.Errorf("expecting args, got %v", len(args))
	}
	ctx := context.Background()

	ec := exec.CommandContext(ctx, args[0], args[1:]...)

	ec.Dir = st.CurDir
	ec.Env = st.Env.Environ()
	//fmt.Println(ec.Env)

	return st.collectOutput(t, func() error {
		// setup cmd stdout inside collectoutput
		// TODO: stdin?
		ec.Stdout = os.Stdout
		ec.Stderr = os.Stderr

		ci := osutil.NewCmdI(ec)
		ci = osutil.NewCtxCmd(ctx, ci)
		ci = osutil.NewShellCmd(ci, true)
		return osutil.RunCmdI(ci)
	})
}

//----------

// TODO: eventually, remove
func icUCmd(t *testing.T, st *ScriptTest, args []string) error {
	// drop "ucmd"
	if args[0] == "ucmd" {
		args = args[1:]
	}

	cmd, ok := st.ucmds[args[0]]
	if !ok {
		return fmt.Errorf("cmd not found: %v", args[0])
	}
	return st.collectOutput(t, func() error {
		return cmd.Fn(t, st, args)
	})
}

//----------

func icContains(t *testing.T, st *ScriptTest, args []string) error {
	args = args[1:] // drop "contains"
	if len(args) != 2 {
		return fmt.Errorf("expecting 2 args, got %v", args)
	}

	// data source (stdout,stderr,err)
	data, ok := st.lastCmdContent(args[0])
	if !ok {
		return fmt.Errorf("unknown content: %v", args[0])
	}

	// pattern
	pattern, err := strconv.Unquote(args[1])
	if err != nil {
		return err
	}

	if !bytes.Contains(data, []byte(pattern)) {
		//return fmt.Errorf("contains:%s: no match:\ndata=%q\npattern=%q", args[0], string(data), pattern)
		return fmt.Errorf("contains:%s: no match: data=%q", args[0], string(data))
	}
	return nil
}

func icContainsRegexp(t *testing.T, st *ScriptTest, args []string) error {
	args = args[1:] // drop "containsre"
	if len(args) != 2 {
		return fmt.Errorf("expecting 2 args, got %v", args)
	}

	data, ok := st.lastCmdContent(args[0])
	if !ok {
		return fmt.Errorf("unknown content: %v", args[0])
	}

	// pattern
	u, err := strconv.Unquote(args[1])
	if err != nil {
		return err
	}
	pattern := u

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	if re.Find(data) == nil {
		//return fmt.Errorf("contains: no match:\npattern=[%v]\ndata=[%v]", pattern, string(data))
		return fmt.Errorf("containsre: no match")
	}
	return nil
}

//----------

func icSetEnv(t *testing.T, st *ScriptTest, args []string) error {
	args = args[1:] // drop "setenv"
	if len(args) != 1 && len(args) != 2 {
		return fmt.Errorf("expecting 1 or 2 args, got %v", args)
	}
	v := "" // allow setting to empty
	if len(args) == 2 {
		v = args[1]

		// allow env expansion when setting env vars
		//v = os.Expand(v, os.Getenv)
		v = os.Expand(v, st.Env.Get)

		// allow expansion of lastcmd (setenv x stdout)
		data, ok := st.lastCmdContent(v)
		if ok {
			v = string(data)
		}
	}
	st.Env.Set(args[0], v)
	return nil
}

//----------

func icFail(t *testing.T, st *ScriptTest, args []string) error {
	t.Helper()
	args = args[1:] // drop "fail"
	if len(args) < 1 {
		return fmt.Errorf("expecting at least 1 arg, got %v", args)
	}

	err := runCmd(t, st, args)
	if err == nil {
		return fmt.Errorf("expected failure but got no error")
	}

	st.Logf(t, "fail ok: %v", err)
	return nil
}

//----------

func icChangeDir(t *testing.T, st *ScriptTest, args []string) error {
	args = args[1:] // drop "cd"
	if len(args) != 1 {
		return fmt.Errorf("expecting 1 arg, got %v", args)
	}
	dir := args[0]
	if filepath.IsAbs(dir) {
		st.CurDir = dir
	} else {
		st.CurDir = filepath.Join(st.CurDir, dir)
	}
	return nil
}

//----------
//----------
//----------

func fixFilepathsForCurDir(b []byte, curDir string) []byte {
	// NOTE: when there is a compilation problem on the annotated files, the filepaths error are relative to the tmp running dir, which is not the script call dir

	scanPath := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// consume spaces (optional)
		u := 0
		for i, ru := range string(data) {
			if unicode.IsSpace(ru) {
				u = i
				continue
			}
			break
		}

		// consume path
		k := 0
		accept := false
		for i, ru := range string(data) {
			k = i
			if unicode.IsLetter(ru) ||
				unicode.IsDigit(ru) ||
				strings.ContainsRune("._", ru) {
				continue
			}
			if strings.ContainsRune("/", ru) {
				accept = true
				continue
			}
			break
		}
		if accept {
			tok := data[u:k]
			return len(data), tok, nil // done, only deal with first match
		}

		return len(data), nil, nil // done
	}

	// replace map
	m := map[string]string{}

	rd := bytes.NewReader(b)
	sc := bufio.NewScanner(rd)
	for sc.Scan() { // scanlines
		sc2 := bufio.NewScanner(bytes.NewBuffer(sc.Bytes()))
		sc2.Split(scanPath)
		for i := 0; sc2.Scan(); i++ {
			fp := sc2.Text()
			if !filepath.IsAbs(fp) {
				fp2 := filepath.Join(curDir, fp)

				// must exist
				_, err := os.Stat(fp2)
				if err == nil {
					m[fp] = fp2
				}

				//fmt.Printf("**%s\n", fp2)
			}
			//if s, err := filepath.Rel(originDir, fp); err == nil {
			//	fp = s
			//}
		}
	}
	//fmt.Printf("***%s", b)

	// make replacements
	for k, v := range m {
		_, _ = k, v

		////for{ // failing: inf loop
		//if i := bytes.Index(b, []byte(k)); i >= 0 {
		//	h := fmt.Sprintf("[FULLDIR: %s]", v)
		//	b = slices.Insert(b, i, []byte(h)...)
		//	continue
		//}
		////break
		////}

		v = fmt.Sprintf("SCRIPT_FIXPATH: %s", v)
		b = bytes.Replace(b, []byte(k), []byte(v), 1)
	}

	return b
}

//----------
//----------
//----------

type Env struct {
	data []string
}

func NewEnvMap() *Env {
	return &Env{data: os.Environ()}
}
func (e *Env) Get(key string) string {
	return osutil.GetEnv(e.data, key)
}
func (e *Env) Set(key, val string) {
	osutil.SetEnv2(&e.data, key, val)
}
func (e *Env) Environ() []string {
	return e.data
}
