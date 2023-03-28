package reflectutil

import (
	"fmt"
	"reflect"
	"strings"
)

// only works with exported names (start with uppercase)
func InvokeByName(v interface{}, name string, args ...interface{}) ([]reflect.Value, error) {
	method := reflect.ValueOf(v).MethodByName(name)
	if method == (reflect.Value{}) {
		return nil, fmt.Errorf("method not found: %v", name)
	}
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	return method.Call(inputs), nil
}

//----------

// Useful to then call functions based on their type names.
func TypeNameBase(v interface{}) (string, error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv == (reflect.Value{}) {
		return "", fmt.Errorf("typenamebase: zero value: %T", v)
	}
	// build name: get base string after last "."
	ts := rv.Type().String()
	if k := strings.LastIndex(ts, "."); k >= 0 {
		ts = ts[k+1:]
	}
	return ts, nil
}
