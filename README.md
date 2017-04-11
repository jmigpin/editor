# Editor

Source code editor in pure Go.

![screenshot](./screenshot.png)

This is a know-what-you're-doing source code editor<br>
As the editor is being developed, the rules of how the UI interacts will become more well defined.<br>

### Features
Auto indentation of wrapped lines.<br>
No code coloring.<br>
Many TextArea utilities: undo/redo, replace, comment, ...<br>
Start external processes from the toolbar with a click, capturing the output to a row. <br>
Drag and drop files/directories to the editor.<br>
Watches files changes on disk.<br>

### Installation and usage

```
go get -u github.com/jmigpin/editor
cd $GOPATH/src/github.com/jmigpin/editor
go build 
./editor
```

### key/button shortcuts

#### Column key/button shortcuts
(top right square):<br>
button2: close column<br>

#### Row key/button shortcuts
ctrl+s: save file<br>
ctrl+f: warp pointer to "Find" cmd in row toolbar<br>
Any button press: make row active to layout toolbar commands<br>
<br>
(top right square):<br>
button1/drag: move row to point<br>
button2: close row<br>
button3/drag: resize column<br>
ctrl+button1/drag: move row column to point<br>

#### Textarea key/button shortcuts
up: move cursor up<br>
down: move cursor down<br>
left: move cursor left<br>
right: move cursor right<br>
shift+up: move cursor up adding to selection<br>
shift+down: move cursor down adding to selection<br>
shift+left: move cursor left adding to selection<br>
shift+right: move cursor right adding to selection<br>
delete: delete current rune<br>
backspace: delete previous rune<br>
end: end of line<br>
home: start of line<br>
shift+end: end of string<br>
shift+home: start of string<br>
ctrl+a: select all<br>
ctrl+c: copy to clipboard<br>
ctrl+k: remove lines<br>
ctrl+v: paste from clipboard<br>
ctrl+x: cut<br>
ctrl+mod1+down: move line down<br>
ctrl+mod1+shift+down: duplicate lines<br>
tab (if selection is on): insert tab at beginning of lines<br>
shift+tab: remove tab from beginning of line<br>
ctrl+z: undo<br>
ctrl+shift+z: undo<br>
ctrl+d: comment<br>
ctrl+shift+d: uncomment<br>
<br>
button1: move cursor to point<br>
button3: move cursor to point + text area cmd<br>
button4: scroll up<br>
button5: scroll down<br>
shift+button1: move cursor to point adding to selection<br>

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
FileManager: open a filemanager (external) at the active row directory<br>
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
Replace \<old\> \<new\>: replaces old string with new<br>
Stop: stops current processing running in the row<br>
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
Uses X shared memory extension (MIT-SHM). Currently linux amd64.<br>
Notable projects that inspired many features:<br>
Oberon OS: https://www.youtube.com/watch?v=UTIJaKO0iqU <br>
Acme editor: https://www.youtube.com/watch?v=dP1xVpMPn8M <br>

