# Editor

Source code editor in pure Go.

![screenshot](./screenshot2.png)
Screenshot taken using DejaVuSans.ttf font.

This is a know-what-you're-doing source code editor<br>
As the editor is being developed, the rules of how the UI interacts will become more well defined.<br>

### Features
Auto indentation of wrapped lines.<br>
No code coloring.<br>
Many TextArea utilities: undo/redo, replace, comment, ...<br>
Start external processes from the toolbar with a click, capturing the output to a row. <br>
Drag and drop files/directories to the editor.<br>
Detects if files opened are changed outside of the editor.<br>
Calls goimports if available when saving a .go file.<br>

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
  -scrollbarwidth int
    	textarea scrollbar width (default 12)
  -tabwidth int
    	 (default 8)
  -wraplinerune int
    	code for wrap line rune (default 8594)
```

### key/button shortcuts

#### Column key/button shortcuts
(top right square):<br>
<kbd>button2</kbd>: close column<br>

#### Row key/button shortcuts
<kbd>ctrl</kbd>+<kbd>s</kbd>: save file<br>
<kbd>ctrl</kbd>+<kbd>f</kbd>: warp pointer to "Find" cmd in row toolbar<br>
Any button press: make row active to layout toolbar commands<br>
<br>
(top right square):<br>
<kbd>button1</kbd> drag: move row to point<br>
<kbd>button2</kbd>: close row<br>
<kbd>button3</kbd> drag: resize column<br>
<kbd>ctrl</kbd>+<kbd>button1</kbd> drag: move row column to point<br>

#### Textarea key/button shortcuts
<kbd>up</kbd>: move cursor up<br>
<kbd>down</kbd>: move cursor down<br>
<kbd>left</kbd>: move cursor left<br>
<kbd>right</kbd>: move cursor right<br>
<kbd>shift</kbd>+<kbd>up</kbd>: move cursor up adding to selection<br>
<kbd>shift</kbd>+<kbd>down</kbd>: move cursor down adding to selection<br>
<kbd>shift</kbd>+<kbd>left</kbd>: move cursor left adding to selection<br>
<kbd>shift</kbd>+<kbd>right</kbd>: move cursor right adding to selection<br>
<kbd>delete</kbd>: delete current rune<br>
<kbd>backspace</kbd>: delete previous rune<br>
<kbd>end</kbd>: end of line<br>
<kbd>home</kbd>: start of line<br>
<kbd>shift</kbd>+<kbd>end</kbd>: end of string<br>
<kbd>shift</kbd>+<kbd>home</kbd>: start of string<br>
<kbd>ctrl</kbd>+<kbd>a</kbd>: select all<br>
<kbd>ctrl</kbd>+<kbd>c</kbd>: copy to clipboard<br>
<kbd>ctrl</kbd>+<kbd>k</kbd>: remove lines<br>
<kbd>ctrl</kbd>+<kbd>v</kbd>: paste from clipboard<br>
<kbd>ctrl</kbd>+<kbd>x</kbd>: cut<br>
<kbd>ctrl</kbd>+<kbd>mod1</kbd>+<kbd>down</kbd>: move line down<br>
<kbd>ctrl</kbd>+<kbd>mod1</kbd>+<kbd>shift</kbd>+<kbd>down</kbd>: duplicate lines<br>
<kbd>tab</kbd> (if selection is on): insert tab at beginning of lines<br>
<kbd>shift</kbd>+<kbd>tab</kbd>: remove tab from beginning of line<br>
<kbd>ctrl</kbd>+<kbd>z</kbd>: undo<br>
<kbd>ctrl</kbd>+<kbd>shift</kbd>+<kbd>z</kbd>: redo<br>
<kbd>ctrl</kbd>+<kbd>d</kbd>: comment lines<br>
<kbd>ctrl</kbd>+<kbd>shift</kbd>+<kbd>d</kbd>: uncomment lines<br>
<br>
<kbd>button1</kbd>: move cursor to point<br>
<kbd>button3</kbd>: move cursor to point + text area cmd<br>
<kbd>button4</kbd>: scroll up<br>
<kbd>button5</kbd>: scroll down<br>
<kbd>shift</kbd>+<kbd>button1</kbd>: move cursor to point adding to selection<br>

### Commands

#### Layout toolbar commands (top toolbar)
ListSessions: lists saved sessions<br>
SaveSession \<name\>: save session to ~/.editor_sessions.json<br>
DeleteSession \<name\>: deletes the session from the sessions file<br>
NewColumn: opens new column<br>
NewRow: opens new row<br>
ReopenRow: reopen a previously closed row<br>
SaveAllFiles: saves all files<br>
ReloadAll: reloads all filepaths<br>
ReloadAllFiles: reloads all filepaths that are files<br>
XdgOpenDir: calls xdg-open to open the active row directory with the preferred external application (ex: a filemanager)<br>
RowDirectory: open row with the active row directory: useful when editing a file and want to access the file directory contents<br>
Exit: exits the program<br>

Note: Some row commands work from the layout toolbar because they act on the current active row (ex: Find, Replace).

#### Row toolbar commands
Save: save file<br>
Reload: reload content<br>
Close: close row<br>
CloseColumn: closes row column<br>
Find: find string (ignores case)<br>
GotoLine \<num\>: goes to line number<br>
Replace \<old\> \<new\>: replaces old string with new, respects selections<br>
Stop: stops current processing (external cmd) running in the row<br>
ListDir: lists directory<br>
ListDirSub: lists directory and sub directories<br>
ListDirHidden: lists directory including hidden<br>

#### Textarea commands
OpenSession \<name\>: opens previously saved session<br>
\<url\>: opens url in x-www-browser<br>
\<filepath\>: opens filepath<br>
\<filename:number\>: opens filename at line (usual format from compilers)<br>
\<quoted string\>: opens filepath if existent on goroot/gopath<br>

### Notes
Uses X shared memory extension (MIT-SHM). <br>
MacOS might need to have XQuartz installed.<br>
Notable projects that inspired many features:<br>
Oberon OS: https://www.youtube.com/watch?v=UTIJaKO0iqU <br>
Acme editor: https://www.youtube.com/watch?v=dP1xVpMPn8M <br>

