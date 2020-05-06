package windriver

//go:generate stringer -tags=windows -type=_wm -output zwm_windows.go
//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zwinapi_windows.go winapi.go
