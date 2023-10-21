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

type Flags struct {
	stderr io.Writer

	mode struct {
		run     bool
		test    bool
		build   bool
		connect bool
	}

	address             string // build/connect
	continueServing     bool
	editorIsServer      bool
	env                 []string
	network             string
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
	execArgs    []string // to be passed to the executable when running
}

func (fl *Flags) parseArgs(args []string) error {
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

func (fl *Flags) usageErr() error {
	fl.printCmdUsage()
	return flag.ErrHelp
}

func (fl *Flags) printCmdUsage() {
	fmt.Fprint(fl.stderr, cmdUsage())
}

//----------

func (fl *Flags) parseRunArgs(name string, args []string) error {
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

func (fl *Flags) parseTestArgs(name string, args []string) error {
	// support test "-args" special flag
	for i, a := range args {
		if a == "-args" || a == "--args" {
			args, fl.execArgs = args[:i], args[i+1:] // exclude
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

func (fl *Flags) parseBuildArgs(name string, args []string) error {
	fs := fl.newFlagSet(name)

	fl.addAddrFlag(fs, defaultBuildConnectAddr)
	fl.addContinueServingFlag(fs)
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

func (fl *Flags) parseConnectArgs(name string, args []string) error {
	fs := fl.newFlagSet(name)

	fl.addAddrFlag(fs, defaultBuildConnectAddr)
	fl.addContinueServingFlag(fs)
	fl.addEditorIsServerFlag(fs)
	fl.addNetworkFlag(fs)
	fl.addToolExecFlag(fs)

	// commented: doesn't fail on not defined
	//return fl.parse(name, fs, args, nil)

	// strict: fail on flags not defined
	fs.SetOutput(fl.stderr)
	return fs.Parse(args)
}

//----------

func (fl *Flags) addAddrFlag(fs *flag.FlagSet, def string) {
	fs.StringVar(&fl.address, "addr", def, "address to transmit debug data")
}

func (fl *Flags) addContinueServingFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.continueServing, "continueserving", false, "Keep serving after the client connection is closed. Ex: useful when listening from a web page (js/wasm) that is being reloaded. Use with caution. In the case of the editor side it can be canceled with Stop.")
}

func (fl *Flags) addEditorIsServerFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.editorIsServer, "editorisserver", true, "run editor side as server instead of client")
}

func (fl *Flags) addEnvFlag(fs *flag.FlagSet) {
	ff := flagutil.StringFuncFlag(func(s string) error {
		fl.env = filepath.SplitList(s)
		return nil
	})
	// The type in usage doc is the backquoted "string" (detected by flagset)
	usage := fmt.Sprintf("`string` with env variables (ex: \"a=1%cb=2%c...\"'", filepath.ListSeparator, filepath.ListSeparator)
	fs.Var(ff, "env", usage)
}

func (fl *Flags) addNetworkFlag(fs *flag.FlagSet) {
	fs.StringVar(&fl.network, "network", "tcp", "protocol to use to transmit debug data: [tcp, unix, ws]")
}

func (fl *Flags) addNoInitMsgFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.noInitMsg, "noinitmsg", false, "omit initial warning message from the compiled binary")
}

func (fl *Flags) addOutFilenameFlag(fs *flag.FlagSet) {
	fs.StringVar(&fl.outFilename, "o", "", "output filename")
}

func (fl *Flags) addPathsFlag(fs *flag.FlagSet) {
	ff := flagutil.StringFuncFlag(func(s string) error {
		fl.paths = splitCommaList(s)
		return nil
	})
	fs.Var(ff, "paths", "comma-separated `string` of dirs/files to annotate (also see the \"//godebug:annotate*\" source code directives to set files to be annotated)")
}

func (fl *Flags) addSrcLinesFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.srcLines, "srclines", true, "add src reference lines to the compilation such that in case of panics, the stack refers to the original src file")
}

func (fl *Flags) addStartExecFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.startExec, "startexec", true, "start executable; useful to set to false in the case of compiling for js/wasm where the browser is the one starting the compiled file")
}

func (fl *Flags) addSyncSendFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.syncSend, "syncsend", false, "Don't send msgs in chunks (slow). Useful to get msgs before a crash.")
}

func (fl *Flags) addStringifyBytesRunesFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.stringifyBytesRunes, "sbr", true, "Stringify bytes/runes as string (ex: [97 98 99] outputs as \"abc\")")
}

func (fl *Flags) addToolExecFlag(fs *flag.FlagSet) {
	fs.StringVar(&fl.toolExec, "toolexec", "", "a program to invoke before the program argument. Useful to run a tool with the output file (ex: wine)")
}

func (fl *Flags) addUsePkgLinksFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.usePkgLinks, "usepkglinks", true, "Use symbolic links to some pkgs directories to build the godebug binary. Helps solving new behaviour introduced by go1.19.x. that fails when an overlaid file depends on a new external module.")
}

func (fl *Flags) addVerboseFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.verbose, "verbose", false, "print extra godebug build info")
}

func (fl *Flags) addWorkFlag(fs *flag.FlagSet) {
	fs.BoolVar(&fl.work, "work", false, "print workdir and don't cleanup on exit")
}

//----------

func (fl *Flags) addTestVFlag(fs *flag.FlagSet) {
	ff := flagutil.BoolFuncFlag(func(s string) error {
		u := "-test.v"
		fl.execArgs = append([]string{u}, fl.execArgs...)
		return nil
	})
	fs.Var(ff, "v", "`bool` verbose")
}
func (fl *Flags) addTestRunFlag(fs *flag.FlagSet) {
	ff := flagutil.StringFuncFlag(func(s string) error {
		u := "-test.run=" + s
		fl.execArgs = append([]string{u}, fl.execArgs...)
		return nil
	})
	fs.Var(ff, "run", "`string` with test name pattern to run")
}

//----------

func (fl *Flags) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	return fs
}

func (fl *Flags) parse(name string, fs *flag.FlagSet, args []string, isBool map[string]bool) error {

	// don't show err "flag provided but not defined"
	fs.SetOutput(io.Discard)

	unknown, unnamed, exec, err := flagutil.ParseFlagSetSets(fs, args, isBool)
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
	fl.execArgs = append(fl.execArgs, exec...)

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
	GoDebug run -env=GODEBUG_BUILD_FLAGS=-cover main.go
	GoDebug run -network=ws -startexec=false -env=GOOS=js:GOARCH=wasm -o=static/main.wasm client/main.go
	GoDebug test
	GoDebug test -help
	GoDebug test -run=mytest -v
	GoDebug test a_test.go -test.run=mytest -test.v -test.count 5
	GoDebug build -help
	GoDebug build -addr=:8078 main.go
	GoDebug build -network=ws -addr=:8078 -env=GOOS=js:GOARCH=wasm -o=static/main.wasm client/main.go
	GoDebug connect -help
	GoDebug connect -addr=:8078
	GoDebug connect -network=ws -addr=:8078
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
