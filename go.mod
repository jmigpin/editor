module github.com/jmigpin/editor

go 1.18

require (
	github.com/BurntSushi/xgb v0.0.0-20200324125942-20f126ea2843
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/fsnotify/fsnotify v1.6.0
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9
	golang.org/x/image v0.12.0
	golang.org/x/mod v0.12.0
	golang.org/x/net v0.15.0
	golang.org/x/sys v0.12.0
	golang.org/x/text v0.13.0
	golang.org/x/tools v0.13.0
)

retract (
	v2.0.7+incompatible
	v2.0.7-alpha.1+incompatible
	v2.0.6-alpha.2+incompatible
	v2.0.2+incompatible
	v2.0.1+incompatible
)
