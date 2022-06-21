package core

import (
	"fmt"
	"time"
)

func Version() string {
	// NOTE: equivalent "go get" version in format x.y.z is 1.x.y (z not used) (ex: 3.3.0 -> 1.3.3). This is done because go doesn't seem to allow having versions bigger then 1 without altering the import paths.
	v := "3.4.0"
	//typ := "release"
	//typ:="rc" // release candidate
	typ := "alpha" // development
	return timedVersion(v, typ)
}
func timedVersion(v string, typ string) string {
	// TODO: build-metadata (possibly from OS, git hash, ...)

	t := versionTime()
	tagt := t.Format("200601021504")
	switch typ {
	case "alpha":
		return fmt.Sprintf("%s-alpha.%s", v, tagt)
	case "rc":
		return fmt.Sprintf("%s-rc.%s", v, tagt)
	case "release":
		// "+" is build-metadata; ignored to determine version precedence
		return fmt.Sprintf("%s+%s", v, tagt)
	default:
		panic("!")
	}
}
func versionTime() time.Time {
	// auto-updated with "go generate" from main directory
	date := "#___202206211951___#"
	tag := date[4 : len(date)-4]

	layout := "200601021504"
	t, err := time.Parse(layout, tag)
	if err != nil {
		panic(err)
	}
	return t
}
