package core

func Version() string {
	// equivalent semantic "go get" version: 1.x.y (z not used)
	v := "3.2.1"
	return taggedVersion(v)
	//return v
}

func taggedVersion(v string) string {
	// auto-updated with "go generate" from main directory
	date := "#___202107292201___#"
	tag := date[4 : len(date)-4]

	return v + "-rc." + tag // release candidate
}
