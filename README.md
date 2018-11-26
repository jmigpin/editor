# Editor

Source code editor in pure Go.

![screenshot](./screenshot.png)

![screenshot](./screenshot2.png)

![screenshot](./screenshot3.png)

## About

- This is a know-what-you're-doing source code editor
- As the editor is being developed, the rules of how the UI interacts will become more well defined.
- Primarily developed and tested in Linux. Works in MacOS through XQuartz. Should work in MSWindows using an xserver for windows (just needs a tiny bit of work/testing to allocate memory for the mit-shm extension).

## Features

- Auto-indentation of wrapped lines.
- No code coloring (except comments).
- Many TextArea utilities: undo/redo, replace, comment, ...
- Start external processes from the toolbar with a click, capturing the output to a row. 
- Drag and drop files/directories to the editor.
- Detects if files opened are changed outside of the editor.
- Calls goimports if available when saving a .go file.
- Clicking on `.go` files identifiers will jump to the identifier definition (uses go guru).
- Debug utility for go programs (`GoDebug` cmd). (__experimental__)
  - allows to go back and forth in time to consult code lines.
- Plugin support
  - examples such as `gotodefinition` and `autocomplete` [below](#plugins).

## Installation and usage

```
go get -u github.com/jmigpin/editor
cd $GOPATH/src/github.com/jmigpin/editor
go build 
./editor
```

```
Usage of ./editor:
  -colortheme string
    	available: light, dark, acme (default "light")
  -commentscolor int
    	Colorize comments. Can be set to zero to use a percentage of the font color. Ex: 0=auto, 1=Black, 0xff0000=red.
  -cpuprofile string
    	profile cpu filename
  -dpi float
    	monitor dots per inch (default 72)
  -font string
    	font: regular, medium, mono, or a filename (default "regular")
  -fonthinting string
    	font hinting: none, vertical, full (default "full")
  -fontsize float
    	 (default 12)
  -plugins string
    	comma separated string of plugin filenames
  -scrollbarleft
    	set scrollbars on the left side (default true)
  -scrollbarwidth int
    	Textarea scrollbar width in pixels. A value of 0 takes 3/4 of the font size.
  -sessionname string
    	open existing session
  -shadows
    	shadow effects on some elements (default true)
  -tabwidth int
    	 (default 8)
  -usemultikey
    	use multi-key to compose characters ([multi-key, ~, a]=ã, ...)
  -wraplinerune int
    	code for wrap line rune, can be set to zero (default 8594)
```

## Basic Layout

The editor has a top toolbar and columns. Columns have rows. Rows have a toolbar and a textarea.

These row toolbars are also textareas where clicking on the text will run that text as a command. 

The row toolbar has a square showing the state of the row.

## Toolbar usage examples

Commands in toolbars are separated by "|" (not to be confused with the shell pipe). If a shell pipe is needed it should be escaped with a backslash.

All commands implemented by the editor start with an **Uppercase letter**. Otherwise it tries to run an existent external program. 

Examples:
  - `~/tmp/subdir/file1.txt | ls`
  Clicking at `ls` will run `ls` at `~/tmp/subdir`
  - `~/tmp/subdir/file1.txt | ls -l \| grep fi`
  Notice how "|" is escaped, allowing to run `ls -l | grep fi`
  - `~/tmp/subdir/file1.txt`
  Clicking at `file1.txt` opens a new row to edit the same file.
  Clicking at `~/tmp` opens a new row located at that directory.
  - `gorename -offset $edPosOffset -to abc`
  Usage of external command with active row position as argument.
  [gorename godoc](https://godoc.org/golang.org/x/tools/cmd/gorename), [go tools](https://github.com/golang/tools).
  - `guru -scope fmt callers $edPosOffset`
  Usage of external command with active row position as argument.
  [guru godoc](https://godoc.org/golang.org/x/tools/cmd/guru), [go tools](https://github.com/golang/tools).
  - `grep -niIR someword`
  Grep results with line positions that are clickable.
  - `xdg-open $edDir`
  Open favorite external application with active row directory.
  - `xterm` 
  Open an xterm at the active row directory.

## Commands

*Top toolbar commands*

- `ListSessions`: lists saved sessions
- `SaveSession <name>`: save session to ~/.editor_sessions.json
- `DeleteSession <name>`: deletes the session from the sessions file
- `NewColumn`: opens new column
- `NewRow`: opens new empty row located at the active-row directory, or if there is none, the current directory. Useful to run commands in a directory.
- `ReopenRow`: reopen a previously closed row
- `SaveAllFiles`: saves all files
- `ReloadAll`: reloads all filepaths
- `ReloadAllFiles`: reloads all filepaths that are files
- `ColorTheme`: cycles through available color themes.
- `FontTheme`: cycles through available font themes.
- `Exit`: exits the program

*Row toolbar commands*

These commands run on a row toolbar, or on the top toolbar with the active-row.

- `Save`: save file
- `Reload`: reload content
- `CloseRow`: close row
- `CloseColumn`: closes row column
- `Find`: find string (ignores case)
- `GotoLine <num>`: goes to line number
- `Replace <old> <new>`: replaces old string with new, respects selections
- `Stop`: stops current process (external cmd) running in the row
- `ListDir`: lists directory
  - `-sub`: lists directory and sub directories
  - `-hidden`: lists directory including hidden
- `MaximizeRow`: maximize row. Will push other rows up/down.
- `CopyFilePosition`: copy to clipboard/primary the cursor file position in the format "file:line:col". Useful to paste a clickable text with the file position.
- `ToggleRowHBar`: toggles row textarea horizontal scrollbar.
- `XdgOpenDir`: calls `xdg-open` to open the row directory with the preferred external application (ex: a filemanager).
- `GoRename <new-name>`: calls `gorename` to rename the identifier under the text cursor. Uses the row/active-row filename, and the cursor index as the "offset" argument. Reloads the calling row at the end if there are no errors.
- `GoDebug {run,test} <filename.go>`: debugger utility for go programs.
  - `-h`: help (show usage).
  - `-dirs`: comma separated directories to include in the debug session.
  - `-work`: print out temporary work dir, and don't cleanup (allows to see the generated code).
  - use `esc` key to stop the debug session.
  - Function that allows to control sending debug messages to the editor. Helpful to bypass programs tight loops that would take too long with debug messages being sent.
    - `debug.SetSend(bool)`
- toolbar first part (usually the row filename): clicking on a section of the path of the filename will open a new row with that content. Ex: if a row filename is "/a/b/c.txt" clicking on "/a" will open a new row with that directory listing, while clicking on "/a/b/c.txt" will open another row to edit the same file.

*Textarea commands*

- `OpenSession <name>`: opens previously saved session
- `<url>`: opens url in preferred application.
- `<filename(:number?)(:number?)>`: opens filename, possibly at line/column (usual output from compilers). Check common locations like `$GOROOT` and C include directories.
- `<identifier-in-a-.go-file>`: opens definition of the identifier. Ex: clicking in `Println` on `fmt.Println` will open the file at the line that contains the `Println` function definition.

## Environment variables set available to external commands

- `$edName`: row name. 
- `$edDir`: row directory. 
- `$edPosOffset`: filename with offset position from active row cursor. Ex: "filename:#123".

## Row states

- background colors:
  - `blue`: row file has been edited.
  - `orange`: row file doesn't exist.
- dot colors:
  - `black`: row currently active. There is only one active row.
  - `red`: row file was edited outside (changed on disk) and doesn't match last known save. Use `Reload` cmd to update.
  - `blue`: there are other rows with the same filename (2 or more).
  - `yellow`: there are other rows with the same filename (2 or more). Color will change when the pointer is over one of the rows.

## Plugins

Plugins allow extra functionality to be added to the editor without changing the binary. 

A plugin can be compiled with (will output a `*.so`):
- `go build -buildmode=plugin <filename.go>` 

and used in the editor by using the `--plugins` option:
- `editor --plugins <plugin1.so>,<plugin2.so>`

Functions that can be implemented by a plugin are (subject to changes - __work-in-progress__ ):
- `func OnLoad(ed *core.Editor)`
- `func AutoComplete(ed *core.Editor, cfb *ui.ContextFloatBox)`
- `func ToolbarCmd(ed *core.Editor, erow *core.ERow, part *toolbarparser.Part) bool`

Note that plugins might need to be recompiled everytime there are changes in the libraries provided by the editor.

Plugins located at: `./plugins`.
- `gotodefinition_godef.go`: plugin that shows how to override the textarea click action and use godef.
- `autocomplete_gocode.go`: plugin that shows a context with suggestions for `.go` files (uses gocode).

## Key/button shortcuts

*Global key/button shortcuts*

- `esc`: stop debugging session
- `f1`: toggle context float box
  - triggers call to plugins that implement `AutoComplete`

*Column key/button shortcuts*

- `buttonLeft`:
  - on left border: drag to move/resize
  - on square-button: close

*Row key/button shortcuts*

- `ctrl`+`s`: save file
- `ctrl`+`f`: warp pointer to "Find" cmd in row toolbar
- `buttonLeft` on square-button: close row
- on top border:
  - `buttonLeft`: drag to move/resize row
  - `buttonMiddle`: close row
  - `buttonWheelUp`: adjust row vertical position, pushing other rows up
  - `buttonWheelDown`: adjust row vertical position, pushing other rows down
- Any button/key press: make row active to layout toolbar commands

*Textarea key/button shortcuts*

- `left`: move cursor left
- `right`: move cursor right
- `up`: move cursor up
- `down`: move cursor down
- `home`: start of line
- `end`: end of line
- `delete`: delete current rune
- `backspace`: delete previous rune
- `pageUp`: page up
- `pageDown`: page down
- `tab` (if selection is on): insert tab at beginning of lines
- `shift`+`left`: move cursor left adding to selection
- `shift`+`right`: move cursor right adding to selection
- `shift`+`up`: move cursor up adding to selection
- `shift`+`down`: move cursor down adding to selection
- `shift`+`home`: start of string adding to selection
- `shift`+`end`: end of string adding to selection
- `shift`+`tab`: remove tab from beginning of line
- `ctrl`+`a`: select all
- `ctrl`+`c`: copy to clipboard
- `ctrl`+`d`: comment lines
- `ctrl`+`k`: remove lines
- `ctrl`+`v`: paste from clipboard
- `ctrl`+`x`: cut
- `ctrl`+`z`: undo
- `ctrl`+`alt`+`down`: move line down
- `ctrl`+`alt`+`shift`+`down`: duplicate lines
- `ctrl`+`shift`+`z`: redo
- `ctrl`+`shift`+`d`: uncomment lines
- `buttonLeft`: move cursor to point
  - drag: selects text - works as copy making it available for paste (primary selection).
- `buttonMiddle`: paste from primary
- `buttonRight`: move cursor to point + text area cmd
- `buttonWheelUp`: scroll up
- `buttonWheelDown`: scroll down
- `buttonWheelUp` on scrollbar: page up
- `buttonWheelDown` on scrollbar: page down
- `shift`+`buttonLeft`: move cursor to point adding to selection
- `ctrl`+`buttonLeft`:
  - `godebug`: select annotation 
- `ctrl`+`buttonRight`:
  - `godebug` over an annotation: print the annotation value.
- `ctrl`+`buttonWheelUp`: 
  - `godebug` on textarea: show previous debug step
  - `godebug` over an annotation: show line previous annotation
- `ctrl`+`buttonWheelDown`: 
  - `godebug` on textarea: show next debug step
  - `godebug` over an annotation: show line next annotation

## Row placement algorithm

When a new row is created, it is placed either below the current row (measuring available space), or in a "good position".

The "good position" measures the available space between all rows, and uses the position with the most space.

The measuring of space is done as follows:
1) if the cursor is visible, measure space after visible cursor to the end of the textarea and use it if not smaller than two lines in height, otherwise use 2)
2) about 2/3 of the textarea

## Notes

- Uses X shared memory extension (MIT-SHM).
- Notable projects that inspired many features:
  - Oberon OS: https://www.youtube.com/watch?v=UTIJaKO0iqU 
  - Acme editor: https://www.youtube.com/watch?v=dP1xVpMPn8M 

