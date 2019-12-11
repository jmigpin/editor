package plugins

// build plugins (usage: "editor --plugins=<p1.so>,...")
//go:generate go build -buildmode=plugin ./autocomplete_gocode/autocomplete_gocode.go
//go:generate go build -buildmode=plugin ./gotodefinition_godef/gotodefinition_godef.go
//go:generate go build -buildmode=plugin ./rownames/rownames.go
//go:generate go build -buildmode=plugin ./eevents/eevents.go
