#!/bin/sh

# stringer is not working (to compile to windows, from linux)
# 1. gives 0 package found because we are in linux (this is a win pkg)
# 2. with -tags=windows, gives a "no values defined for type" err

# under normal conditions, this should work
# //go:generate stringer -type=wm -output zwm.go .
# //go:generate stringer -tags=windows -type=wm -output zwm.go .

# this works here in a shell
GOOS=windows stringer -type=_wm -output zwm.go
