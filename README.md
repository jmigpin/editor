# Editor
Source code editor in pure Go.

This is a know-what-you're-doing source code editor that was inspired by the projects presented at these videos: <br>
Oberon OS: https://www.youtube.com/watch?v=UTIJaKO0iqU <br>
Acme editor: https://www.youtube.com/watch?v=dP1xVpMPn8M <br>

![screenshot](./screenshot.png)

## Features
Auto indentation of wrapped lines.<br>
Drag and drop files/directories to the editor.<br>
No code coloring other then comments.<br>
Watches files change on disk to show the file has changed while editing. <br>
Allows to start external processes from the toolbar with a click, capturing the output. Starting another process in the same row cancels the previous process.<br>

## Notes
Uses X shared memory extension (MIT-SHM).<br>

## Keyboard shortcuts
### Textarea
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

### Row
ctrl+s: save file<br>
ctrl+f: warp pointer to "Find" cmd in row toolbar<br>
ctrl+shift+f: open a filemanager in the directory of the current row<br>
ctrl+shift+d: if the row has a file, open a row with file directory list<br>

## Button shortcuts
### Textarea
button1: move cursor to point<br>
button3: move cursor to point + text area cmd<br>
button4: scroll up<br>
button5: scroll down<br>
shift+button1: move cursor to point adding to selection<br>

### Row top right square
button1/drag: move row to point<br>
button2: close row<br>
button3/drag: resize column<br>
ctrl+button1/drag: move row column to point<br>

### Column top right square
button2: close column<br>

### Row
Any button press: make row active to layout toolbar commands<br>

## Commands
### Layout toolbar commands (top toolbar)
ListSessions: lists saved sessions<br>
SaveSession \<name\>: save session to ~/.editor_sessions.json<br>
DeleteSession: deletes the session from the sessions file<br>
NewColumn: opens new column<br>
NewRow: opens new row<br>
ReopenRow: reopen a previously closed row<br>
SaveAllFiles: saves all files<br>
ReloadAll: reloads all filepaths<br>
ReloadAllFiles: reloads all filepaths that are files<br>
Exit: exits the program<br>

Note: Some row commands work from the layout toolbar because they act on the current active row (ex: Find, Replace).

### Row toolbar commands
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

### Textarea commands
OpenSession \<name\>: opens previously saved session<br>
\<url\>: opens url in x-www-browser<br>
\<filepath\>: opens filepath<br>
\<filename:number\>: opens filename at line (usual format from compilers)<br>
\<quoted string\>: opens filepath if existent on goroot/gopath<br>

