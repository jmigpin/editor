module github.com/jmigpin/editor

go 1.25.0

require (
	github.com/creack/pty v1.1.24
	github.com/davecgh/go-spew v1.1.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/jezek/xgb v1.1.1
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8
	golang.org/x/exp/jsonrpc2 v0.0.0-20250911091902-df9299821621
	golang.org/x/image v0.41.0
	golang.org/x/mod v0.35.0
	golang.org/x/net v0.55.0
	golang.org/x/sys v0.45.0
	golang.org/x/term v0.43.0
	golang.org/x/text v0.37.0
	golang.org/x/tools v0.44.0
)

require (
	golang.org/x/exp/event v0.0.0-20250819193227-8b4c13bb791b // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
)

retract (
	v2.0.7+incompatible
	v2.0.7-alpha.1+incompatible
	v2.0.6-alpha.2+incompatible
	v2.0.2+incompatible
	v2.0.1+incompatible
	v1.6.1
	v1.6.0
)
