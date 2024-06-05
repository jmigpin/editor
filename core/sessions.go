package core

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/mathutil"
	"github.com/jmigpin/editor/util/osutil"
)

type Sessions struct {
	Sessions []*Session
}

func newSessionsFromPlain(filename string) (*Sessions, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return decodeSessionsFromJson(f)
}
func newSessionsFromZip(zipFilename, filename string) (*Sessions, error) {
	zr, err := zip.OpenReader(zipFilename)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	// find filename inside zip file
	zf := (*zip.File)(nil)
	for _, zf2 := range zr.File {
		if zf2.Name == filename {
			zf = zf2
			break
		}
	}
	if zf == nil {
		return nil, fmt.Errorf("file not found inside zip: %v", filename)
	}

	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return decodeSessionsFromJson(rc)
}

//----------

func (ss *Sessions) saveToPlain(filename string) error {
	f, err := openToWriteSessionsFile(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	jsonBytes, err := ss.encodeToJson()
	if err != nil {
		return err
	}
	_, err = f.Write(jsonBytes)
	return err
}
func (ss *Sessions) saveToZip(zipFilename, filename string) error {
	jsonBytes, err := ss.encodeToJson()
	if err != nil {
		return err
	}

	f, err := openToWriteSessionsFile(zipFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	zipw := zip.NewWriter(f)
	defer zipw.Close()

	// create file to put inside the zip
	h := &zip.FileHeader{}
	h.Name = filename
	h.UncompressedSize64 = uint64(len(jsonBytes))
	h.SetModTime(time.Now())
	h.SetMode(0644)
	h.Method = zip.Deflate

	fzipw, err := zipw.CreateHeader(h)
	if err != nil {
		return err
	}
	if _, err := fzipw.Write(jsonBytes); err != nil {
		return err
	}
	return nil
}

//----------

func (ss *Sessions) encodeToJson() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "\t")
	if err := enc.Encode(&ss); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//----------
//----------
//----------

func openToWriteSessionsFile(filename string) (*os.File, error) {
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	return os.OpenFile(filename, flags, 0644)
}

func decodeSessionsFromJson(r io.Reader) (*Sessions, error) {
	dec := json.NewDecoder(r)
	ss := &Sessions{}
	if err := dec.Decode(ss); err != nil {
		return nil, err
	}
	return ss, nil
}

//----------
//----------
//----------

func sessionsFilename() string {
	return homeFilename(sessionsBasicFilename())
}

func sessionsZipFilenames() (zipFilename string, innerFilename string) {
	fn1 := sessionsBasicFilename()
	ext := path.Ext(fn1)
	fn2 := strings.TrimSuffix(fn1, ext) + ".zip"
	return homeFilename(fn2), fn1
}

func sessionsBasicFilename() string {
	return ".editor_sessions.json"
}
func homeFilename(filename string) string {
	home := osutil.HomeEnvVar()
	return filepath.Join(home, filename)
}

//----------

type Session struct {
	Name      string
	RootTbStr string
	Columns   []*ColumnState
}

func NewSessionFromEditor(ed *Editor) *Session {
	s := &Session{
		RootTbStr: ed.UI.Root.Toolbar.Str(),
	}
	for _, c := range ed.UI.Root.Cols.Columns() {
		cstate := NewColumnState(ed, c)
		s.Columns = append(s.Columns, cstate)
	}
	return s
}
func (s *Session) restore(ed *Editor) {
	uicols := ed.UI.Root.Cols

	// layout toolbar
	tbStr := s.RootTbStr

	ed.UI.Root.Toolbar.SetStrClearHistory(tbStr)

	// close all current columns
	for _, c := range uicols.Columns() {
		c.Close()
	}

	// open n new columns
	// allow other columns to exist already (ex: on close, the editor could be ensuring one column)
	for len(uicols.Columns()) < len(s.Columns) {
		_ = ed.NewColumn()
	}

	// setup columns sizes (end percents)
	uicolumns := uicols.Columns()
	for i, c := range s.Columns {
		sp := c.StartPercent

		uicols.ColsLayout.Spl.SetRawStartPercent(uicolumns[i], sp)
	}

	// create rows
	m := make(map[*RowState]*ERow)
	for i, c := range s.Columns {
		uicol := uicolumns[i]

		for _, rs := range c.Rows {
			rowPos := &ui.RowPos{Column: uicol}
			erow, ok, err := rs.OpenERow(ed, rowPos)
			if err != nil {
				ed.Error(err)
			}
			if ok {
				m[rs] = erow

				// setup row size
				sp := rs.StartPercent

				uicol.RowsLayout.Spl.SetRawStartPercent(erow.Row, sp)
			}
		}
	}

	// restore positions after positioning rows to have correct dimensions
	for rs, erow := range m {
		rs.RestorePos(erow)
	}
}

//----------

type ColumnState struct {
	StartPercent float64
	Rows         []*RowState
}

func NewColumnState(ed *Editor, col *ui.Column) *ColumnState {
	cstate := &ColumnState{
		StartPercent: roundStartPercent(col.Cols.ColsLayout.Spl.RawStartPercent(col)),
	}
	for _, row := range col.Rows() {
		rstate := NewRowState(ed, row)
		cstate.Rows = append(cstate.Rows, rstate)
	}
	return cstate
}

//----------

// Used in sessions and reopenrow.
type RowState struct {
	TbStr         string
	TbCursorIndex int
	TaCursorIndex int
	TaOffsetIndex int
	StartPercent  float64
}

func NewRowState(ed *Editor, row *ui.Row) *RowState {
	tbStr := row.Toolbar.Str()

	rs := &RowState{
		TbStr:         tbStr,
		TbCursorIndex: row.Toolbar.CursorIndex(),
		TaCursorIndex: row.TextArea.CursorIndex(),
		TaOffsetIndex: row.TextArea.RuneOffset(),
	}

	// check row.col in case the row has been removed from columns (reopenrow?)
	if row.Col != nil {
		rs.StartPercent = roundStartPercent(row.Col.RowsLayout.Spl.RawStartPercent(row))
	}

	return rs
}

func (state *RowState) OpenERow(ed *Editor, rowPos *ui.RowPos) (*ERow, bool, error) {
	data := toolbarparser.Parse(state.TbStr)
	arg0, ok := data.Part0Arg0()
	if !ok {
		return nil, false, fmt.Errorf("missing toolbar arg 0: %s", state.TbStr)
	}

	name := ed.HomeVars.Decode(arg0.String())
	info := ed.ReadERowInfo(name)

	// create erow, even if it had have errors
	erow := NewLoadedERowOrNewBasic(info, rowPos)

	// setup toolbar even if erow had errors
	w := data.Str[arg0.End():]
	if strings.TrimSpace(w) != "" {
		erow.ToolbarSetStrAfterNameClearHistory(w)
	}

	return erow, true, nil
}

func (state *RowState) RestorePos(erow *ERow) {
	erow.Row.Toolbar.SetCursorIndex(state.TbCursorIndex)
	erow.Row.TextArea.SetCursorIndex(state.TaCursorIndex)
	erow.Row.TextArea.SetRuneOffset(state.TaOffsetIndex)
}

//----------
//----------
//----------

func SaveSession(ed *Editor, part *toolbarparser.Part) {
	err := saveSession(ed, part)
	if err != nil {
		ed.Error(err)
	}
}
func saveSession(ed *Editor, part *toolbarparser.Part) error {
	if len(part.Args) != 2 {
		return fmt.Errorf("savesession: missing session name")
	}
	sessionName := part.Args[1].String()

	s1 := NewSessionFromEditor(ed)
	s1.Name = sessionName

	ss, err := ed.loadSessions()
	if err != nil {
		return err
	}
	// replace session already stored
	replaced := false
	for i, s := range ss.Sessions {
		if s.Name == sessionName {
			ss.Sessions[i] = s1
			replaced = true
			break
		}
	}
	// append if a new session
	if !replaced {
		ss.Sessions = append(ss.Sessions, s1)
	}
	// save to file
	err = ed.saveSessions(ss)
	if err != nil {
		return err
	}
	return nil
}

//----------

func ListSessions(ed *Editor) {
	ss, err := ed.loadSessions()
	if err != nil {
		ed.Error(err)
		return
	}

	// sort sessions names
	var u []string
	for _, session := range ss.Sessions {
		u = append(u, session.Name)
	}
	sort.Strings(u)

	// concat opensession lines
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "sessions: %d\n", len(u))
	for _, sname := range u {
		fmt.Fprintf(buf, "OpenSession %v\n", sname)
	}

	erow, _ := ExistingERowOrNewBasic(ed, "+Sessions")
	erow.Row.TextArea.SetBytesClearPos(buf.Bytes())
	erow.Flash()
}

//----------

func OpenSession(ed *Editor, part *toolbarparser.Part) {
	if len(part.Args) != 2 {
		ed.Errorf("missing session name")
		return
	}
	sessionName := part.Args[1].String()
	OpenSessionFromString(ed, sessionName)
}

func OpenSessionFromString(ed *Editor, sessionName string) {
	ss, err := ed.loadSessions()
	if err != nil {
		ed.Error(err)
		return
	}
	for _, s := range ss.Sessions {
		if s.Name == sessionName {
			s.restore(ed)
			return
		}
	}
	ed.Errorf("session not found: %v", sessionName)
}

//----------

func DeleteSession(ed *Editor, part *toolbarparser.Part) {
	err := deleteSession(ed, part)
	if err != nil {
		ed.Error(err)
	}
}
func deleteSession(ed *Editor, part *toolbarparser.Part) error {
	if len(part.Args) != 2 {
		return fmt.Errorf("deletesession: missing session name")
	}
	sessionName := part.Args[1].String()
	ss, err := ed.loadSessions()
	if err != nil {
		return err
	}
	found := false
	for i, s := range ss.Sessions {
		if s.Name == sessionName {
			found = true
			u := ss.Sessions
			ss.Sessions = append(u[:i], u[i+1:]...)
			break
		}
	}
	if !found {
		return fmt.Errorf("deletesession: session not found: %v", sessionName)
	}
	return ed.saveSessions(ss)
}

//----------

func roundStartPercent(v float64) float64 {
	return mathutil.RoundFloat64(v, 8)
}

//----------
//----------
//----------
