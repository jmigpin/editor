# Editor

Source code editor in pure Go.

![screenshot](./screenshot2.png)
Screenshot taken using DejaVuSans.ttf font.

- This is a know-what-you're-doing source code editor
- As the editor is being developed, the rules of how the UI interacts will become more well defined.

### Features

- Auto indentation of wrapped lines.
- No code coloring.
- Many TextArea utilities: undo/redo, replace, comment, ...
- Start external processes from the toolbar with a click, capturing the output to a row. 
- Drag and drop files/directories to the editor.
- Detects if files opened are changed outside of the editor.
- Calls goimports if available when saving a .go file.

### Installation and usage

```
go get -u github.com/jmigpin/editor
cd $GOPATH/src/github.com/jmigpin/editor
go build 
./editor
```

```
./editor --help
Usage of ./editor:
  -acmecolors
    	acme editor color theme
  -cpuprofile string
    	profile cpu filename
  -dpi float
    	monitor dots per inch (default 72)
  -font string
    	ttf font filename
  -fontsize float
    	 (default 12)
  -scrollbarleft
    	set scrollbars on the left side
  -scrollbarwidth int
    	textarea scrollbar width (default 12)
  -tabwidth int
    	 (default 8)
  -wraplinerune int
    	code for wrap line rune (default 8594)
```

### key/button shortcuts

#### Column key/button shortcuts

(top right square):

- `button2`: close column

#### Row key/button shortcuts

- `ctrl`+`s`: save file
- `ctrl`+`f`: warp pointer to "Find" cmd in row toolbar
- Any button press: make row active to layout toolbar commands


(top right square):

- `button1` drag: move row to point
- `button2`: close row
- `button3` drag: resize column
- `ctrl`+`button1` drag: move row column to point

#### Textarea key/button shortcuts

- `up`: move cursor up
- `down`: move cursor down
- `left`: move cursor left
- `right`: move cursor right
- `shift`+`up`: move cursor up adding to selection
- `shift`+`down`: move cursor down adding to selection
- `shift`+`left`: move cursor left adding to selection
- `shift`+`right`: move cursor right adding to selection
- `delete`: delete current rune
- `backspace`: delete previous rune
- `end`: end of line
- `home`: start of line
- `shift`+`end`: end of string
- `shift`+`home`: start of string
- `ctrl`+`a`: select all
- `ctrl`+`c`: copy to clipboard
- `ctrl`+`k`: remove lines
- `ctrl`+`v`: paste from clipboard
- `ctrl`+`x`: cut
- `ctrl`+`mod1`+`down`: move line down
- `ctrl`+`mod1`+`shift`+`down`: duplicate lines
- `tab` (if selection is on): insert tab at beginning of lines
- `shift`+`tab`: remove tab from beginning of line
- `ctrl`+`z`: undo
- `ctrl`+`shift`+`z`: redo
- `ctrl`+`d`: comment lines
- `ctrl`+`shift`+`d`: uncomment lines

- `button1`: move cursor to point
- `button2`: paste from primary
- `button3`: move cursor to point + text area cmd
- `button4`: scroll up
- `button5`: scroll down
- `shift`+`button1`: move cursor to point adding to selection

Note: selecting text with `button1` works as copy and makes it available for paste (primary selection).

### Commands

#### Layout toolbar commands (top toolbar)

- `ListSessions`: lists saved sessions
- `SaveSession <name>`: save session to ~/.editor_sessions.json
- `DeleteSession <name>`: deletes the session from the sessions file
- `NewColumn`: opens new column
- `NewRow`: opens new row
- `ReopenRow`: reopen a previously closed row
- `SaveAllFiles`: saves all files
- `ReloadAll`: reloads all filepaths
- `ReloadAllFiles`: reloads all filepaths that are files
- `XdgOpenDir`: calls xdg-open to open the active row directory with the preferred external application (ex: a filemanager)
- `RowDirectory`: open row with the active row directory: useful when editing a file and want to access the file directory contents
- `DuplicateRow`: make active row share the edit history with a new row that updates when changes are made.
- `Exit`: exits the program

Note: Some row commands work from the layout toolbar because they act on the current active row (ex: Find, Replace).

#### Row toolbar commands

- `Save`: save file
- `Reload`: reload content
- `Close`: close row
- `CloseColumn`: closes row column
- `Find`: find string (ignores case)
- `GotoLine <num>`: goes to line number
- `Replace <old> <new>`: replaces old string with new, respects selections
- `Stop`: stops current processing (external cmd) running in the row
- `ListDir`: lists directory
- `ListDirSub`: lists directory and sub directories
- `ListDirHidden`: lists directory including hidden

#### Textarea commands

- `OpenSession <name>`: opens previously saved session
- `<url>`: opens url in x-www-browser
- `<filepath>`: opens filepath
- `<filename:number>`: opens filename at line (usual format from compilers)
- `<quoted string>`: opens filepath if existent on goroot/gopath

### Notes

- Uses X shared memory extension (MIT-SHM). 
- MacOS might need to have XQuartz installed.
- Notable projects that inspired many features:
- Oberon OS: https://www.youtube.com/watch?v=UTIJaKO0iqU 
- Acme editor: https://www.youtube.com/watch?v=dP1xVpMPn8M 

