package flagutil

import (
	"flag"
	"strings"
)

// usefull to allow unknown flags to be collected, to possibly pass them to another program. The main issue is knowning which unknown flags are boolean that won't receive a value after space (ex: -mybool main.go, main.go is not an arg to mybool). In this case, the provided map allows to correct this without having to define the flags in the flagset.
func ParseFlagSetSets(fs *flag.FlagSet, args []string, isBool map[string]bool) (unknownArgs, unnamedArgs, execArgs []string, _ error) {
	// fs.parse stops parsing at the first non-flag

	// ex: "-flag1=1 -flag2=2 main.go -arg1 -arg2"
	// 	- flags are parsed until "main.go"
	// 	- unknown flags go to the unknownArgs
	// 	- "main.go" goes to the fl.unnamedArgs
	// 	- "-arg1" "-arg2" goes to the fl.execArgs

	// don't output help on error (will try to recover)
	usageFn := fs.Usage
	fs.Usage = func() {}
	defer func() { fs.Usage = usageFn }()

	for {
		err := fs.Parse(args)
		if err == nil {
			break
		}
		if err == flag.ErrHelp {
			return nil, nil, nil, err
		}

		// get failing arg (consumed by fs.parse, need to get from end)
		k := len(args) - 1 - len(fs.Args())
		arg := args[k]
		unknownArgs = append(unknownArgs, arg)
		args = args[k+1:]

		// if not a boolean arg, it consumes the next arg
		name := strings.TrimLeft(arg, "-")
		if isBool != nil && !isBool[name] {
			haveVal := strings.Index(arg, "=") >= 1
			if !haveVal && len(args) > 0 {
				arg2 := args[0]
				args = args[1:]
				unknownArgs = append(unknownArgs, arg2)
			}
		}
	}

	// split unnamed from named afterwards
	args2 := fs.Args()
	split1, split2 := len(args2), len(args2)
	for i, a := range args2 {
		if a[0] == '-' {
			split1, split2 = i, i

			// special double dash case (exclude arg)
			if a == "--" {
				split2 = i + 1
			}

			break
		}
	}
	unnamedArgs, execArgs = args2[:split1], args2[split2:]

	return
}
