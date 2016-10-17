# Editor
Source code editor in pure Go.

This is a know-what-you're-doing source code editor that was inspired by the projects presented at these videos: <br>
Oberon OS: https://www.youtube.com/watch?v=UTIJaKO0iqU <br>
Acme editor: https://www.youtube.com/watch?v=dP1xVpMPn8M <br>

![screenshot](./screenshot.png)

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
ctrl+mod1+down+shift: duplicate lines<br>
tab: (if selection is on): insert tab at beginning of line<br>
shift+tab: remove tab from beginning of line<br>
ctrl+z: undo<br>
ctrl+shift+z: undo<br>
ctrl+d: comment<br>
ctrl+shift+d: uncomment<br>

### Row
ctrl+s: save file<br>
ctrl+f: quick shortcut to "Find"<br>

## Button shortcuts
### Textarea
button1: move cursor to point<br>
button2: move cursor to point + text area cmd<br>
button4: scroll up<br>
button5: scroll down<br>
shift+button1: move cursor to point adding to selection<br>

### Row top right square
button1/drag: move row to point<br>
button2: close row<br>
button3/drag: resize column<br>
crl+button1/drag: move column (where the row belongs) to point<br>

### Column top right square
button2: close column<br>

### Row
Any button press: make row active to layout toolbar commands.<br>

## Commands
### Layout toolbar commands (top toolbar)
ListSessions: lists saved sessions<br>
SaveSession <name>: save session to ~/.editor_sessions.json<br>
DeleteSession: deletes the session from the sessions file.<br>
NewColumn: opens new column<br>
NewRow: opens new row<br>
SaveAll: saves all files<br>
ReloadAll: reloads all filepaths<br>
Exit: exists the program<br>

Note: Some row commands work on the layout toolbar because the act on the current active row (ex: Find, Replace).

### Row toolbar commands
Save: save file<br>
Close: close row<br>
Reload: reload content<br>
Find: find string (ignores case)<br>
GotoLine <num>: goes to line number<br>
Replace<old><new>: replaces old string with new<br>
Stop: stops current processing running in the row.<br>

### Textarea commands
OpenSession <name>: opens previously saved session<br>
\<url\>: opens url in x-www-browser<br>
\<filepath\>: opens filepath<br>
\<filename:number\>: opens filename at line (usual format from compilers)<br>
\<quoted string\>: opens filepath if existent on goroot/gopath<br>

## Other features
Drag and drop files/directories to the editor.
