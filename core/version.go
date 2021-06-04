package core

func Version() string {
	// equivalent semantic "go get" version: 1.x.y (z not used)
	v := "3.2.0"
	// auto-updated with "go generate" from main directory
	date := "#___202106050846___#"
	extra := date[4 : len(date)-4]
	// release candidate
	extra = "rc." + extra

	return v + "-" + extra
}
