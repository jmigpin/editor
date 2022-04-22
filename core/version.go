package core

func Version() string {
	// NOTE: equivalent "go get" version in format x.y.z is 1.x.y (z not used) (ex: 3.3.0 -> 1.3.3). This is done because go doesn't seem to allow having versions bigger then 1 without altering the import paths.

	v := "3.3.0"
	return taggedVersion(v)
}

func taggedVersion(v string) string {
	// auto-updated with "go generate" from main directory
	date := "#___202204221709___#"
	tag := date[4 : len(date)-4]

	return v + "-rc." + tag // release candidate format
	//return v + "+" + tag // build metadata format (ignored to determine version precedence)
}
