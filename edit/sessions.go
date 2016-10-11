package edit

import (
	"encoding/json"
	"fmt"
	"jmigpin/editor/edit/toolbar"
	"jmigpin/editor/ui"
	"os"
	"path"
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
	TaOffsetIndex int
}

func sessionFilename() string {
	home := os.Getenv("HOME")
	return path.Join(home, ".editor_sessions.json")
}

func saveSession(ed *Editor, part *toolbar.Part) {
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
func openSession(ed *Editor, part *toolbar.Part) {
	if len(part.Args) != 2 {
		ed.Error(fmt.Errorf("opensession: missing session name"))
		return
	}
	sessionName := part.Args[1].Trim()
	openSessionFromString(ed, sessionName)
}
func openSessionFromString(ed *Editor, sessionName string) {
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
func deleteSession(ed *Editor, part *toolbar.Part) {
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
func listSessions(ed *Editor) {
	row := ed.getSpecialTagRow("Sessions")
	t := ""
	ss, err := readSessionsFromDisk()
	if err != nil {
		ed.Error(err)
		return
	}
	for _, s := range ss.Sessions {
		t += fmt.Sprintf("OpenSession %v\n", s.Name)
	}
	row.TextArea.SetText(t)
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

func buildSession(ed *Editor) *Session {
	s := Session{LayoutToolbarText: ed.ui.Layout.Toolbar.Text()}
	for _, c := range ed.ui.Layout.Cols.Cols {
		// truncate for a shorter string
		cend := float64(int(c.End*1000)) / 1000

		col := &Column{
			End: cend,
		}
		for _, r := range c.Rows {
			row := &Row{
				ToolbarText:   r.Toolbar.Text(),
				TaCursorIndex: r.TextArea.CursorIndex(),
				TaOffsetIndex: r.TextArea.OffsetIndex(),
			}
			col.Rows = append(col.Rows, row)
		}
		s.Columns = append(s.Columns, col)
	}
	return &s
}
func restoreSession(ed *Editor, s *Session) {
	// close current session
	cols := ed.ui.Layout.Cols
	for len(cols.Cols) > 0 {
		cols.RemoveColumn(cols.Cols[0])
	}
	// restore session
	ed.ui.Layout.Toolbar.SetText(s.LayoutToolbarText)
	// create columns first
	for i, _ := range s.Columns {
		_ = cols.NewColumn()
		if i > 0 {
			cols.Cols[i-1].End = s.Columns[i-1].End
		}
	}
	// calc areas since the columns ends had to be set
	cols.CalcOwnArea()
	// introduce the rows
	for i, c := range s.Columns {
		col := cols.Cols[i]
		for _, r := range c.Rows {
			tsd := toolbar.NewStringData(r.ToolbarText)
			var row *ui.Row
			filename, ok := tsd.FilenameTag()
			if !ok {
				row = col.NewRow()
			} else {
				row0, err := openPathAtCol(ed, filename, col)
				if err != nil {
					ed.Error(fmt.Errorf("%s: %s", err, filename))
					continue
				}
				row = row0
			}
			row.Toolbar.SetText(r.ToolbarText)
			row.TextArea.SetCursorIndex(r.TaCursorIndex)
			row.TextArea.SetOffsetIndex(r.TaOffsetIndex)
		}
	}
}
