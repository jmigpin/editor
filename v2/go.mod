module github.com/jmigpin/editor/v2

go 1.14

require (
	github.com/BurntSushi/xgb v0.0.0-20200324125942-20f126ea2843
	github.com/BurntSushi/xgbutil v0.0.0-20190907113008-ad855c713046
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	golang.org/x/image v0.0.0-20200430140353-33d19683fad8
	golang.org/x/mod v0.2.0
	golang.org/x/sys v0.0.0-20200501145240-bc7a7d42d5c3
	golang.org/x/tools v0.0.0-20200505023115-26f46d2f7ef8
)

retract [v2.0.1, v2.0.8]
