package godebug

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/jmigpin/editor/util/flagutil"
)

var defaultBuildConnectAddr = ":8078"

//----------

type flags struct {
	stderr io.Writer

	mode struct {
		run     bool
		test    bool
		build   bool
		connect bool
	}

	network string
	address string // build/connect

	editorIsServer      bool
	env                 []string
	noInitMsg           bool
	outFilename         string   // build, ex: -o filename
	paths               []string // dirs/files to annotate (args from cmd line)
	srcLines            bool
	startExec           bool
	stringifyBytesRunes bool
	syncSend            bool
	toolExec            string // ex: "wine" will run "wine args..."
	usePkgLinks         bool
	verbose             bool
	work                bool

	unknownArgs []string // unknown args to pass down to tooling
	unnamedArgs []string // args without name (ex: filenames)
	binaryArgs  []string // to be passed to the binary when running
}

func (fl *flags) parseArgs(args []string) error {
	if len(args) == 0 {
		return fl.usageErr()
	}
	name := "GoDebug " + args[0]
	switch args[0] {
	case "run":
		fl.mode.run = true
		return fl.parseRunArgs(name, args[1:])
	case "test":
		fl.mode.test = true
		return fl.parseTestArgs(name, args[1:])
	case "build":
		fl.mode.build = true
		return fl.parseBuildArgs(name, args[1:])
	case "connect":
		fl.mode.connect = true
		return fl.parseConnectArgs(name, args[1:])
	default:
		return fl.usageErr()
	}
}

func (fl *flags) usageErr() error {
	fl.printCmdUsage()
	return flag.ErrHelp
}

func (fl *flags) printCmdUsage() {
	fmt.Fprint(fl.stderr, cmdUsage())
}

//----------

func (fl *flags) parseRunArgs(name string, args []string) error {
	fs := fl.newFlagSet(name)

	fl.addAddrFlag(fs, "")
	fl.addEditorIsServerFlag(fs)
	fl.addEnvFlag(fs)
	fl.addNetworkFlag(fs)
	fl.addNoInitMsgFlag(fs)
	fl.addOutFilenameFlag(fs)
	fl.addPathsFlag(fs)
	fl.addSrcLinesFlag(fs)
	fl.addStartExecFlag(fs)
	fl.addStringifyBytesRunesFlag(fs)
	fl.addSyncSendFlag(fs)
	fl.addToolExecFlag(fs)
	fl.addUsePkgLinksFlag(fs)
	fl.addVerboseFlag(fs)
	fl.addWorkFlag(fs)

	m := goBuildBooleanFlags()
	return fl.parse(name, fs, args, m)
}

func (fl *flags) parseTestArgs(name string, args []string) error {
	// support test "-args" special flag
	for i, a := range args {
		if a == "-args" || a == "--args" {
			args, fl.binaryArgs = args[:i], args[i+1:] // exclude
			break
		}
	}

	fs := fl.newFlagSet(name)

	fl.addAddrFlag(fs, "")
	fl.addEditorIsServerFlag(fs)
	fl.addEnvFlag(fs)
	fl.addNetworkFlag(fs)
	fl.addNoInitMsgFlag(fs)
	fl.addPathsFlag(fs)
	fl.addSrcLinesFlag(fs)
	fl.addStartExecFlag(fs)
	fl.addStringifyBytesRunesFlag(fs)
	fl.addSyncSendFlag(fs)
	fl.addTestRunFlag(fs)
	fl.addTestVFlag(fs)
	fl.addToolExecFlag(fs)
	fl.addUsePkgLinksFlag(fs)
	fl.addVerboseFlag(fs)
	fl.addWorkFlag(fs)

	m := joinMaps(goBuildBooleanFlags(), goTestBooleanFlags())
	return fl.parse(name, fs, args, m)
}

func (fl *flags) parseBuildArgs(name string, args []string) error {
	fs := fl.newFlagSet(name)

	fl.addAddrFlag(fs, defaultBuildConnectAddr)
	fl.addEditorIsServerFlag(fs)
	fl.addEnvFlag(fs)
	fl.addNetworkFlag(fs)
	fl.addNoInitMsgFlag(fs)
	fl.addOutFilenameFlag(fs)
	fl.addPathsFlag(fs)
	fl.addSrcLinesFlag(fs)
	fl.addStringifyBytesRunesFlag(fs)
	fl.addSyncSendFlag(fs)
	fl.addUsePkgLinksFlag(fs)
	fl.addVerboseFlag(fs)
	fl.addWorkFlag(fs)

	m := goBuildBooleanFlags()
	return fl.parse(name, fs, args, m)
}

func (fl *flags) parseConnectArgs(name string, args []string) error {
	fs := fl.newFlagSet(name)

	fl.addAddrFlag(fs, defaultBuildConnectAddr)
	fl.addEditorIsServerFlag(fs)
	fl.addNetworkFlag(fs)
	fl.addToolExecFlag(fs)

	// commented: strict parsing, no unknown flags allowed
	//return fl.parse(name, fs, args)
	return fs.Parse(args)
}

//----------

func (fl *flags) addWorkFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.work, "work", false, "print workdir and don't cleanup on exit")
}
func (fl *flags) addVerboseFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.verbose, "verbose", false, "print extra godebug build info")
}

func (fl *flags) addEditorIsServerFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.editorIsServer, "editorisserver", true, "run editor side as server instead of client")
}
func (fl *flags) addSyncSendFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.syncSend, "syncsend", false, "Don't send msgs in chunks (slow). Useful to get msgs before a crash.")
}
func (fl *flags) addStringifyBytesRunesFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.stringifyBytesRunes, "sbr", true, "Stringify bytes/runes as string (ex: [97 98 99] outputs as \"abc\")")
}
func (fl *flags) addSrcLinesFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.srcLines, "srclines", true, "add src reference lines to the compilation such that in case of panics, the stack refers to the original src file")
}
func (fl *flags) addNoInitMsgFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.noInitMsg, "noinitmsg", false, "omit initial warning message from the compiled binary")
}
func (fl *flags) addToolExecFlag(fs *flag.FlagSet) {
	fs.StringVar(&fl.toolExec, "toolexec", "", "a program to invoke before the program argument. Useful to run a tool with the output file (ex: wine)")
}
func (fl *flags) addNetworkFlag(fs *flag.FlagSet) {
	fs.StringVar(&fl.network, "network", "tcp", "protocol to use to transmit debug data: [tcp, unix, ws]")
}
func (fl *flags) addAddrFlag(fs *flag.FlagSet, def string) {
	fs.StringVar(&fl.address, "addr", def, "address to transmit debug data")
}
func (fl *flags) addOutFilenameFlag(fs *flag.FlagSet) {
	fs.StringVar(&fl.outFilename, "o", "", "output filename")
}
func (fl *flags) addStartExecFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.startExec, "startexec", true, "start executable; useful to set to false in the case of compiling for js/wasm where the browser is the one starting the compiled file")
}

func (fl *flags) addTestRunFlag(fs *flag.FlagSet) {
	ff := flagutil.StringFuncFlag(func(s string) error {
		u := "-test.run=" + s
		fl.binaryArgs = append([]string{u}, fl.binaryArgs...)
		return nil
	})
	fs.Var(ff, "run", "`string` with test name pattern to run")
}
func (fl *flags) addTestVFlag(fs *flag.FlagSet) {
	ff := flagutil.BoolFuncFlag(func(s string) error {
		u := "-test.v"
		fl.binaryArgs = append([]string{u}, fl.binaryArgs...)
		return nil
	})
	fs.Var(ff, "v", "`bool` verbose")
}

func (fl *flags) addEnvFlag(fs *flag.FlagSet) {
	ff := flagutil.StringFuncFlag(func(s string) error {
		fl.env = filepath.SplitList(s)
		return nil
	})
	// The type in usage doc is the backquoted "string" (detected by flagset)
	usage := fmt.Sprintf("`string` with env variables (ex: \"a=1%cb=2%c...\"'", filepath.ListSeparator, filepath.ListSeparator)
	fs.Var(ff, "env", usage)
}

func (fl *flags) addPathsFlag(fs *flag.FlagSet) {
	ff := flagutil.StringFuncFlag(func(s string) error {
		fl.paths = splitCommaList(s)
		return nil
	})
	fs.Var(ff, "paths", "comma-separated `string` of dirs/files to annotate (also see the \"//godebug:annotate*\" source code directives to set files to be annotated)")
}

func (fl *flags) addUsePkgLinksFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.usePkgLinks, "usepkglinks", true, "Use symbolic links to some pkgs directories to build the godebug binary. Helps solving new behaviour introduced by go1.19.x. that fails when an overlaid file depends on a new external module.")
}

//----------

func (fl *flags) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	return fs
}

func (fl *flags) parse(name string, fs *flag.FlagSet, args []string, isBool map[string]bool) error {

	// don't show err "flag provided but not defined"
	fs.SetOutput(io.Discard)

	unknown, unnamed, binary, err := flagutil.ParseFlagSetSets(fs, args, isBool)
	if err != nil {
		if err == flag.ErrHelp {
			fmt.Fprintf(fl.stderr, "Usage of %v:\n", name)
			fs.SetOutput(fl.stderr)
			fs.PrintDefaults()
		}
		return err
	}
	fl.unknownArgs = unknown
	fl.unnamedArgs = unnamed
	fl.binaryArgs = append(fl.binaryArgs, binary...)

	//spew.Dump(fl.flags)

	return nil
}

//----------
//----------
//----------

func cmdUsage() string {
	return `Usage:
	GoDebug <command> [arguments]
The commands are:
	run		build and run program with godebug data
	test		test packages compiled with godebug data
	build 	build binary with godebug data (allows remote debug)
	connect	connect to a binary built with godebug data (allows remote debug)
Env variables:
	GODEBUG_BUILD_FLAGS	comma separated flags for build
Examples:
	GoDebug run
	GoDebug run -help
	GoDebug run main.go -arg1 -arg2
	GoDebug run -paths=dir1,file2.go,dir3 main.go -arg1 -arg2
	GoDebug run -tags=xproto main.go
	GoDebug run -env=GODEBUG_BUILD_FLAGS=-tags=xproto main.go
	GoDebug test
	GoDebug test -help
	GoDebug test -run=mytest -v
	GoDebug test a_test.go -test.run=mytest -test.v
	GoDebug test a_test.go -test.count 5
	GoDebug build -help
	GoDebug build -addr=:8078 main.go
	GoDebug connect -help
	GoDebug connect -addr=:8078
`
}

//----------
//----------
//----------

func joinMaps(ms ...map[string]bool) map[string]bool {
	m := map[string]bool{}
	for _, m2 := range ms {
		for k, v := range m2 {
			m[k] = v
		}
	}
	return m
}

func goBuildBooleanFlags() map[string]bool {
	return map[string]bool{
		"a":          true,
		"asan":       true,
		"buildvcs":   true,
		"i":          true,
		"linkshared": true,
		"modcacherw": true,
		"msan":       true,
		"n":          true,
		"race":       true,
		"trimpath":   true,
		"v":          true,
		"work":       true,
		"x":          true,
	}
}
func goTestBooleanFlags() map[string]bool {
	return map[string]bool{
		"benchmem": true,
		"c":        true,
		"cover":    true,
		"failfast": true,
		"json":     true,
		"short":    true,
	}
}
