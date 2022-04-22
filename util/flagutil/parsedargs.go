package flagutil

import "strings"

// NOTE: alternative to flag.flagset (more premitive, lacks documentation utility)

func ParseParsedArgs(args []string, isBool map[string]bool) ParsedArgs {
	pa := ParsedArgs{}
	for i := 0; i < len(args); i++ {
		name, value, k := parseArg(args, i, isBool)
		arg := &Arg{Name: name, Value: value}
		i = k
		pa = append(pa, arg)
	}
	return pa
}

//----------

type ParsedArgs []*Arg

func (pa ParsedArgs) Get(name string) (*Arg, bool) {
	for _, a := range pa {
		if a.Name == name {
			return a, true
		}
	}
	return nil, false
}

func (pa *ParsedArgs) Remove(arg *Arg) {
	for i, a := range *pa {
		if a == arg {
			*pa = append((*pa)[:i], (*pa)[i+1:]...)
			break
		}
	}
}

func (pa ParsedArgs) Join() []string {
	u := []string{}
	for _, a := range pa {
		u = append(u, a.String())
	}
	return u
}

//----------

func (pa ParsedArgs) CommonSplit() (ParsedArgs, ParsedArgs, ParsedArgs) {
	pa, binaryPa := pa.SplitAtDoubleDashExclude()
	pa, unnamedPa := pa.SplitAtFirstUnnamed()
	unnamedPa, namedPa := unnamedPa.SplitAtFirstNamed()
	binaryPa = append(namedPa, binaryPa...)
	return pa, unnamedPa, binaryPa
}

func (pa ParsedArgs) SplitAtFirstUnnamed() (ParsedArgs, ParsedArgs) {
	for i, a := range pa {
		if a.Name == "" { // name is empty if no dash was used
			return pa[:i], pa[i:]
		}
	}
	return pa, nil
}

func (pa ParsedArgs) SplitAtFirstNamed() (ParsedArgs, ParsedArgs) {
	for i, a := range pa {
		if a.Name != "" { // name is not empty if a dash was used
			return pa[:i], pa[i:]
		}
	}
	return pa, nil
}

func (pa ParsedArgs) SplitAtDoubleDashExclude() (ParsedArgs, ParsedArgs) {
	return pa.SplitAtNameExclude("--")
}

func (pa ParsedArgs) SplitAtNameExclude(name string) (ParsedArgs, ParsedArgs) {
	for i, a := range pa {
		if a.Name == name {
			return pa[:i], pa[i+1:]
		}
	}
	return pa, nil
}

//----------

type Arg struct {
	Name  string // can be empty if just a value
	Value string // can be empty if just a name
}

func (a *Arg) String() string {
	s := ""
	if a.Name != "" {
		s += "-" + a.Name
	}
	if a.Value != "" {
		if a.Name != "" {
			s += "="
		}
		s += a.Value
	}
	return s
}

//----------

func parseArg(args []string, i int, isBool map[string]bool) (name string, value string, curI int) {
	curI = i
	arg := args[curI]

	dash := "-"
	dashed := strings.HasPrefix(arg, dash)
	if !dashed {
		value = arg
		return
	}

	// TODO: cut only 2?
	noDashArg := strings.TrimLeft(arg, dash)

	// an arg full of dashes (ex: "--")
	if noDashArg == "" {
		name = arg
		return
	}

	name = noDashArg
	if k := strings.Index(noDashArg, "="); k >= 0 {
		name = noDashArg[:k]
		value = noDashArg[k+1:]

		// simplify bool values
		if isBool != nil && isBool[name] {
			if s, ok := simplifyBoolValue(value); ok {
				value = s
			}
		}

		return
	}

	// value after a space from here

	if isBool != nil && isBool[name] {
		return // no spaced value
	}

	if i+1 >= len(args) {
		return // missing spaced value
	}

	// take next arg as value
	curI = i + 1
	arg = args[curI]
	value = arg
	return
}
