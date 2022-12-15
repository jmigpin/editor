package core

import (
	"fmt"
	"runtime/debug"
	"time"
)

func Version() string {
	// NOTE: equivalent "go get" version in format x.y.z is 1.x.y (z not used) (ex: 3.3.0 -> 1.3.3). This is done because go doesn't seem to allow having versions bigger then 1 without altering the import paths.

	v := "3.7.0"
	//typ := "release"
	//typ := "rc" // release candidate
	typ := "alpha" // development
	return taggedVersion(v, typ)
}
func taggedVersion(v string, typ string) string {
	tag := "nobuildinfo"
	bi, ok := getBuildInfo()
	if ok {
		tag = bi
	}

	switch typ {
	case "alpha":
		return fmt.Sprintf("%s-alpha.%s", v, tag)
	case "rc":
		return fmt.Sprintf("%s-rc.%s", v, tag)
	case "release":
		// "+" is build-metadata; ignored to determine version precedence
		return fmt.Sprintf("%s+%s", v, tag)
	default:
		panic("!")
	}
}

//----------

func getBuildInfo() (string, bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	//spew.Dump(bi)

	get := func(name string) (string, bool) {
		for _, bs := range bi.Settings {
			if bs.Key == name {
				return bs.Value, true
			}
		}
		return "", false
	}

	str := ""
	addToStr := func(s string) {
		if len(str) > 0 {
			str += "-"
		}
		str += s
	}

	// vcs revision commit time
	tstr, ok := get("vcs.time")
	if ok {
		t, err := time.Parse(time.RFC3339, tstr)
		if err != nil {
			return "", false
		}
		str := t.Format("20060102150405")
		addToStr(str)
	}

	// vcs revision string
	rev, ok := get("vcs.revision")
	if ok {
		if len(rev) > 12 {
			rev = rev[:12]
		}
		addToStr(rev)
	}

	// modified since last commit
	modified, ok := get("vcs.modified")
	if ok && modified == "true" {
		addToStr("modified")
	}

	return str, str != ""
}
