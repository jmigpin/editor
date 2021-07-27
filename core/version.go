package core

func Version() string {
	// equivalent semantic "go get" version: 1.x.y (z not used)
	v := "3.2.0"
	//return versionReleaseCandidate(v)
	return v
}

func versionReleaseCandidate(v string) string {
	// auto-updated with "go generate" from main directory
	date := "#___202107271423___#"
	// release candidate
	extra := date[4 : len(date)-4]
	return v + "-rc." + extra
}
