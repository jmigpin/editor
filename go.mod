module github.com/jmigpin/editor

go 1.23.0

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/fsnotify/fsnotify v1.7.0
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/jezek/xgb v1.1.1
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8
	golang.org/x/image v0.18.0
	golang.org/x/mod v0.18.0
	golang.org/x/net v0.38.0
	golang.org/x/sys v0.31.0
	golang.org/x/text v0.23.0
	golang.org/x/tools v0.22.0
)

require golang.org/x/sync v0.12.0 // indirect

retract (
	v2.0.7+incompatible
	v2.0.7-alpha.1+incompatible
	v2.0.6-alpha.2+incompatible
	v2.0.2+incompatible
	v2.0.1+incompatible
	v1.6.1
	v1.6.0
)
