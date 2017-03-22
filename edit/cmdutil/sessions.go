package cmdutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type Sessions struct {
	Sessions []*Session
}

func NewSessions(filename string) (*Sessions, error) {
	// read file
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// empty sessions if it doesn't exist
			return &Sessions{}, nil
		}
		return nil, err
	}
	ss := Sessions{}
	// decode
	dec := json.NewDecoder(f)
	err = dec.Decode(&ss)
	if err != nil {
		return nil, err
	}
	return &ss, err
}
func (ss *Sessions) save(filename string) error {
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	return enc.Encode(&ss)
}

func sessionsFilename() string {
	home := os.Getenv("HOME")
	return path.Join(home, ".editor_sessions.json")
}

type Session struct {
	Name        string
	LayoutTbStr string
	Columns     []*ColumnState
}

func NewSessionFromEditor(ed Editorer) *Session {
	s := &Session{
		LayoutTbStr: ed.UI().Layout.Toolbar.Str(),
	}
	for _, c := range ed.UI().Layout.Cols.Cols {
		cstate := NewColumnState(c)
		s.Columns = append(s.Columns, cstate)
	}
	return s
}
func (s *Session) restore(ed Editorer) {
	cols := ed.UI().Layout.Cols

	// layout toolbar
	ed.UI().Layout.Toolbar.SetStrClear(s.LayoutTbStr, true, true)

	// close all current columns and open n new
	cols.CloseAllAndOpenN(len(s.Columns))

	// setup columns sizes (end percents)
	for i, c := range s.Columns {
		endp := c.EndPercent
		cols.Cols[i].C.Style.EndPercent = &endp
	}
	// calc areas since the columns ends have been set
	cols.C.CalcChildsBounds()

	// create the rows
	for i, c := range s.Columns {
		col := cols.Cols[i]
		for j, r := range c.Rows {
			_ = NewERowFromRowState(ed, r, col, j)
		}
	}
}

type ColumnState struct {
	EndPercent float64
	Rows       []*RowState
}

func NewColumnState(col *ui.Column) *ColumnState {
	endp := 1.0
	if col.C.Style.EndPercent != nil {
		endp = *col.C.Style.EndPercent
	}

	// truncate float for a shorter string
	endp = float64(int(endp*10000)) / 10000

	cstate := &ColumnState{
		EndPercent: endp,
	}
	for _, row := range col.Rows {
		rstate := NewRowState(row)
		cstate.Rows = append(cstate.Rows, rstate)
	}
	return cstate
}

func SaveSession(ed Editorer, part *toolbardata.Part) {
	err := saveSession(ed, part, sessionsFilename())
	if err != nil {
		ed.Error(err)
	}
}
func saveSession(ed Editorer, part *toolbardata.Part, filename string) error {
	if len(part.Args) != 2 {
		return fmt.Errorf("savesession: missing session name")
	}
	sessionName := part.Args[1].Trim()

	s1 := NewSessionFromEditor(ed)
	s1.Name = sessionName

	ss, err := NewSessions(filename)
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
	err = ss.save(filename)
	if err != nil {
		return err
	}
	return nil
}

func ListSessions(ed Editorer) {
	ss, err := NewSessions(sessionsFilename())
	if err != nil {
		ed.Error(err)
		return
	}
	str := ""
	for _, session := range ss.Sessions {
		str += fmt.Sprintf("OpenSession %v\n", session.Name)
	}
	s := "+Sessions"
	erow, ok := ed.FindERow(s)
	if !ok {
		col, rowIndex := ed.GoodColRowPlace()
		erow = ed.NewERow(s, col, rowIndex)
	}
	erow.Row().TextArea.SetStrClear(str, false, false)
}

func OpenSession(ed Editorer, part *toolbardata.Part) {
	if len(part.Args) != 2 {
		ed.Errorf("missing session name")
		return
	}
	sessionName := part.Args[1].Trim()
	OpenSessionFromString(ed, sessionName)
}

func OpenSessionFromString(ed Editorer, sessionName string) {
	ss, err := NewSessions(sessionsFilename())
	if err != nil {
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

func DeleteSession(ed Editorer, part *toolbardata.Part) {
	err := deleteSession(ed, part)
	if err != nil {
		ed.Error(err)
	}
}
func deleteSession(ed Editorer, part *toolbardata.Part) error {
	if len(part.Args) != 2 {
		return fmt.Errorf("deletesession: missing session name")
	}
	sessionName := part.Args[1].Trim()
	ss, err := NewSessions(sessionsFilename())
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
	return ss.save(sessionsFilename())
}
