# Contributing

Contributions to the master branch are welcome.

The stable branch is an attempt of having something that is not totally broken.

## Github workflow

Golang's import paths makes the usual github workflow a bit different:

1. get the upstream repo via `go get -u github.com/jmigpin/editor`
2. fork upstream to `git@github.com:FOO/editor.git`
3. add your fork as a new remote: `cd $GOPATH/src/github.com/jmigpin/editor && git remote add FOO git@github.com:FOO/editor.git`
4. create a local branch for your feature: `git co -b your-feature-branch`
5. commit your changes and push them to your remote branch: `git push FOO your-feature-branch`
6. make a pull request from your fork
