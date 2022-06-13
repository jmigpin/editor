package core

import (
	"fmt"
	"time"
)

func Version() string {
	// NOTE: equivalent "go get" version in format x.y.z is 1.x.y (z not used) (ex: 3.3.0 -> 1.3.3). This is done because go doesn't seem to allow having versions bigger then 1 without altering the import paths.
	v := "3.3.0"
	return timedVersion(v, true)
}
func timedVersion(v string, release bool) string {
	t := versionTime()
	if !release {
		tag := t.Format("200601021504")
		return fmt.Sprintf("%s-rc.%s", v, tag) // release-candidate format
		//return fmt.Sprintf("%s+%s", tag) // build-metadata format (ignored to determine version precedence)
	}
	return fmt.Sprintf("%s (%s)", v, t.Format("2006/01/02 15:04"))
}
func versionTime() time.Time {
	// auto-updated with "go generate" from main directory
	date := "#___202206131603___#"
	tag := date[4 : len(date)-4]

	layout := "200601021504"
	t, err := time.Parse(layout, tag)
	if err != nil {
		panic(err)
	}
	return t
}
