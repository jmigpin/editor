package cmdutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/jmigpin/editor/edit/toolbardata"
)

type Sessions struct {
	Sessions []*Session
}
type Session struct {
	Name              string
	LayoutToolbarText string
	Columns           []*Column
}
type Column struct {
	End  float64
	Rows []*Row
}
type Row struct {
	ToolbarText   string
	TaCursorIndex int
	TaOffsetIndex int // kept instead of the offsetY to preserve the top string index if the area has a different size
}

func sessionFilename() string {
	home := os.Getenv("HOME")
	return path.Join(home, ".editor_sessions.json")
}

func SaveSession(ed Editorer, part *toolbardata.Part) {
	if len(part.Args) != 2 {
		ed.Error(fmt.Errorf("savesession: missing session name"))
		return
	}
	sessionName := part.Args[1].Trim()

	s1 := buildSession(ed)
	s1.Name = sessionName

	ss, err := readSessionsFromDisk()
	if err != nil {
		ed.Error(err)
		return
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
	err = saveSessionsToDisk(ss)
	if err != nil {
		ed.Error(err)
		return
	}
}
func OpenSession(ed Editorer, part *toolbardata.Part) {
	if len(part.Args) != 2 {
		ed.Error(fmt.Errorf("opensession: missing session name"))
		return
	}
	sessionName := part.Args[1].Trim()
	OpenSessionFromString(ed, sessionName)
}
func OpenSessionFromString(ed Editorer, sessionName string) {
	ss, err := readSessionsFromDisk()
	if err != nil {
		ed.Error(err)
		return
	}
	for _, s := range ss.Sessions {
		if s.Name == sessionName {
			restoreSession(ed, s)
			return
		}
	}
	ed.Error(fmt.Errorf("opensession: session not found: %v", sessionName))
}
func DeleteSession(ed Editorer, part *toolbardata.Part) {
	if len(part.Args) != 2 {
		ed.Error(fmt.Errorf("deletesession: missing session name"))
		return
	}
	sessionName := part.Args[1].Trim()
	ss, err := readSessionsFromDisk()
	if err != nil {
		ed.Error(err)
		return
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
		ed.Error(fmt.Errorf("deletesession: session not found: %v", sessionName))
	}
	err = saveSessionsToDisk(ss)
	if err != nil {
		ed.Error(err)
		return
	}
}
func ListSessions(ed Editorer) {
	row := ed.FindRowOrCreate("+Sessions")
	s := ""
	ss, err := readSessionsFromDisk()
	if err != nil {
		ed.Error(err)
		return
	}
	for _, session := range ss.Sessions {
		s += fmt.Sprintf("OpenSession %v\n", session.Name)
	}
	row.TextArea.SetStrClear(s, false, false)
}

func saveSessionsToDisk(ss *Sessions) error {
	f, err := os.OpenFile(sessionFilename(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	return enc.Encode(&ss)
}
func readSessionsFromDisk() (*Sessions, error) {
	// read file
	f, err := os.Open(sessionFilename())
	if err != nil {
		if os.IsNotExist(err) {
			// empty sessions if it doesn't exist
			return &Sessions{}, nil
		}
		return nil, err
	}
	// decode
	dec := json.NewDecoder(f)
	var ss Sessions
	err = dec.Decode(&ss)
	if err != nil {
		return nil, err
	}
	return &ss, err
}

func buildSession(ed Editorer) *Session {
	s := Session{LayoutToolbarText: ed.UI().Layout.Toolbar.Str()}
	for _, c := range ed.UI().Layout.Cols.Cols {
		// truncate for a shorter string
		endp := 1.0
		if c.C.Style.EndPercent != nil {
			endp = *c.C.Style.EndPercent
		}
		cend := float64(int(endp*10000)) / 10000

		col := &Column{
			End: cend,
		}
		for _, r := range c.Rows {
			row := &Row{
				ToolbarText:   r.Toolbar.Str(),
				TaCursorIndex: r.TextArea.CursorIndex(),
				TaOffsetIndex: r.TextArea.OffsetIndex(),
			}
			col.Rows = append(col.Rows, row)
		}
		s.Columns = append(s.Columns, col)
	}
	return &s
}
func restoreSession(ed Editorer, s *Session) {
	cols := ed.UI().Layout.Cols

	// layout toolbar
	ed.UI().Layout.Toolbar.SetStrClear(s.LayoutToolbarText, true, true)

	// close all current columns and open n new
	cols.CloseAllAndOpenN(len(s.Columns))
	// setup columns sizes (end percents)
	for i, c := range s.Columns {
		endp := c.End
		cols.Cols[i].C.Style.EndPercent = &endp
	}
	// calc areas since the columns ends have been set
	cols.C.CalcChildsBounds()

	// create the rows
	for i, c := range s.Columns {
		col := cols.Cols[i]
		for _, r := range c.Rows {
			row := ed.NewRow(col)
			row.Toolbar.SetStrClear(r.ToolbarText, true, true)

			// content
			tsd := ed.RowToolbarStringData(row)
			p := tsd.FirstPartFilepath()
			content, err := ed.FilepathContent(p)
			if err != nil {
				ed.Error(err)
				continue
			}

			row.TextArea.SetStrClear(content, true, true)
			row.Square.SetDirty(false)
			row.TextArea.SetCursorIndex(r.TaCursorIndex)
			row.TextArea.SetOffsetIndex(r.TaOffsetIndex)
		}
	}
}
