package flagutil

import (
	"strings"
)

//----------

func ExtractFlagBool(args []string, name string) (bool, []string, bool) {
	v, poss, ok := GetFlagValue(args, name, true)
	if !ok {
		return false, nil, false
	}
	args = removePositions(poss, args)
	return v == "true", args, true
}

func ExtractFlagString(args []string, name string) (string, []string, bool) {
	v, poss, ok := GetFlagValue(args, name, false)
	if !ok {
		return "", nil, false
	}
	args = removePositions(poss, args)
	return v, args, true
}

//----------

func GetFlagBool(args []string, name string) bool {
	v, _, ok := GetFlagValue(args, name, true)
	if ok && (v == "" || v == "true") {
		return true
	}
	return false
}

func GetFlagString(args []string, name string) (string, bool) {
	v, _, ok := GetFlagValue(args, name, false)
	return v, ok
}

//----------

func GetFlagValue(args []string, name string, isBool bool) (string, []int, bool) {
	noSpacedValue := map[string]bool{name: isBool}
	for i := 0; i < len(args); i++ {
		n, val, k := parseArg(args, i, noSpacedValue)
		if n == name {
			if isBool {
				if s, ok := simplifyBoolValue(val); ok {
					val = s
				}
			}

			// build positions indexes (used for removal if wanted)
			poss := []int{i}
			if k != i {
				poss = append(poss, k)
			}

			return val, poss, true
		}
	}
	return "", nil, false
}

//----------

func simplifyBoolValue(v string) (string, bool) {
	a := strings.Split("1,0,t,f,T,F,true,false,TRUE,FALSE,True,False", ",")
	for i, v2 := range a {
		if v2 == v {
			if i%2 == 0 {
				return "true", true
			} else {
				return "false", true
			}
		}
	}
	return "", false
}

func removePositions(poss []int, a []string) []string {
	// remove positions (remove last first)
	for i := len(poss) - 1; i >= 0; i-- {
		p := poss[i]
		a = append(a[:p], a[p+1:]...)
	}
	return a
}

//----------

//func SplitArgsAtDoubleDash(args []string) ([]string, []string) {
//	for i, a := range args {
//		if a == "--" {
//			return args[:i], args[i+1:]
//		}
//	}
//	return args, nil
//}

//func SplitArgsAtFirstNonDash(args []string, map[string]bool) ([]string, []string) {
//	for i, a := range args {
//		if !strings.HasPrefix(a, "-") {
//			return args[:i], args[i+1:]
//		}
//	}
//	return args, nil
//}

//----------
