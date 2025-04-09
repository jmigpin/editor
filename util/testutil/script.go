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
	Parallel       bool

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

	//dir, err := filepath.Abs(dir)
	//if err != nil {
	//	return err
	//}

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
	name := filepath.Base(filename)
	return t.Run(name, func(t *testing.T) {
		//t.Helper()
		if scr.Parallel {
			t.Parallel()
		}
		st := newST(t, scr, filename)
		if err := st.runFile(t); err != nil {
			t.Fatal(err)
		}
	})
}

//----------
//----------
//----------

type ST struct { // script test
	T   *testing.T
	Env osutil.Envm

	workDir string // test root dir
	Dir     string // current dir

	scr *Script

	filename string

	ucmds map[string]*ScriptCmd // user cmds (mapped)
	icmds map[string]*ScriptCmd // internal cmds

	// collects output for lastcmd
	// error is returned by the func
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer

	lastCmd struct {
		stdout []byte
		stderr []byte
		err    []byte
	}
}

func newST(t *testing.T, scr *Script, filename string) *ST {
	st := &ST{T: t, scr: scr, Env: osutil.NewEnvm(nil), filename: filename}

	// internal cmds
	icmds := []*ScriptCmd{
		{"fail", icFail},
		{"cd", icChangeDir},
		{"setenv", icSetEnv},
		{"contains", icContains},
		{"containsre", icContainsRegexp},
	}
	st.icmds = mapScriptCmds(icmds)
	st.ucmds = mapScriptCmds(scr.Cmds) // script user cmds

	return st
}

//----------

func (st *ST) runFile(t *testing.T) error {
	if err := st.runFile2(t); err != nil {
		return st.error(err) // setup error with location info
	}
	return nil
}
func (st *ST) runFile2(t *testing.T) error {
	t.Helper()

	st.Logf("SCRIPT_FILENAME: %v", st.filename)

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
	st.Dir = st.workDir

	// work directory env and cleanup
	st.Env.Set("WORK", st.workDir)
	st.Logf("script_workdir: %v", st.workDir)
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
		st.Logf("SCRIPT: %v", args)

		if err := runCmd(st, args); err != nil {
			return err
		}
	}
	return scanner.Err()
}

//----------

func (st *ST) lastCmdErrBytes(err error) []byte {
	if err == nil {
		return nil
	}
	return []byte(err.Error())
}

func (st *ST) lastCmdContent(name string) ([]byte, bool) {
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

func (st *ST) collectOutput(t *testing.T, fn func() error) error {
	logf := func(f string, args ...any) {
		if st.scr.NoFilepathsFix {
			t.Logf(f, args...)
		} else {
			u := fmt.Sprintf(f, args...)
			u = string(fixFilepathsForCurDir([]byte(u), st.Dir))
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

func (st *ST) writeFile(filename string, data []byte) error {
	return iout.MkdirAllWriteFile(filename, data, 0o644)
}

//----------

func (st *ST) DirJoin(fp string) string {
	return filepath.Join(st.Dir, fp)
}

//----------

// to be used in user cmds, as st.stdout might not be defined

func (st *ST) Printf(f string, args ...any) {
	fmt.Fprintf(st.Stdout, f, args...)
}
func (st *ST) Println(args ...any) {
	fmt.Fprintln(st.Stdout, args...)
}

//----------

func (st *ST) Logf(f string, args ...any) {
	st.T.Helper()
	st.T.Log(fmt.Sprintf(f, args...))
}
func (st *ST) Log(s string) {
	st.T.Helper()
	if u := st.locationInfo(); u != "" {
		s = u + s
	}
	s = strings.TrimRight(s, "\n") // remove newlines
	st.T.Log(s)                    // adds one newline
}

//----------

func (st *ST) error(err error) error {
	if s := st.locationInfo(); s != "" {
		return fmt.Errorf("%v%w", s, err)
	}
	return err
}
func (st *ST) locationInfo() string {
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

func (st *ST) splitArgs(s string) []string {
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

type ScriptCmdFn func(st *ST, args []string) error

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

func runCmd(st *ST, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expecting args")
	}

	// user cmds
	if cmd, ok := st.ucmds[args[0]]; ok {
		//return st.collectOutput(t, func() error {
		//return cmd.Fn(t, st, args)
		//})

		st.Stdout = &bytes.Buffer{}
		st.Stderr = &bytes.Buffer{}
		err := cmd.Fn(st, args)
		st.lastCmd.stdout = st.Stdout.Bytes()
		st.lastCmd.stderr = st.Stderr.Bytes()
		st.lastCmd.err = st.lastCmdErrBytes(err)
		return err

	}

	// internal cmds
	if cmd, ok := st.icmds[args[0]]; ok {
		//return st.collectOutput(t, func() error {}) // commented: don't collect output from all internal cmds, otherwise it will be getting empty output from cmds like "contains" that follow the pretented cmd that has the desired output

		return cmd.Fn(st, args)
	}

	// default cmd
	//return icExec(t, st, args)

	ctx := context.Background()
	ec := exec.CommandContext(ctx, args[0], args[1:]...)
	ec.Dir = st.Dir
	ec.Env = st.Env.Environ()
	ci := osutil.NewCmdI(ec)
	ci = osutil.NewCtxCmd(ctx, ci)
	ci = osutil.NewShellCmd(ci, true)
	sout, serr, err := osutil.RunCmdIOutputs(ci)
	st.lastCmd.stdout = sout
	st.lastCmd.stderr = serr
	st.lastCmd.err = st.lastCmdErrBytes(err)
	return err
}

//----------

func icContains(st *ST, args []string) error {
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
		return fmt.Errorf("contains: no match\ncontent=%q\ndata=%q", args[0], string(data))
	}
	return nil
}

func icContainsRegexp(st *ST, args []string) error {
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
		return fmt.Errorf("containsre: no match\ncontent=%q\ndata=%q", args[0], string(data))
	}
	return nil
}

//----------

func icSetEnv(st *ST, args []string) error {
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

		// allow expansion of lastcmd (setenv <name> stdout)
		data, ok := st.lastCmdContent(v)
		if ok {
			v = string(data)
		}
	}
	st.Env.Set(args[0], v)
	return nil
}

//----------

func icFail(st *ST, args []string) error {
	st.T.Helper()
	args = args[1:] // drop "fail"
	if len(args) < 1 {
		return fmt.Errorf("expecting at least 1 arg, got %v", args)
	}

	err := runCmd(st, args)
	if err == nil {
		return fmt.Errorf("expected failure but got no error")
	}

	st.Logf("fail ok: %v", err)
	return nil
}

//----------

func icChangeDir(st *ST, args []string) error {
	args = args[1:] // drop "cd"
	if len(args) != 1 {
		return fmt.Errorf("expecting 1 arg, got %v", args)
	}
	dir := args[0]
	if filepath.IsAbs(dir) {
		st.Dir = dir
	} else {
		st.Dir = filepath.Join(st.Dir, dir)
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
