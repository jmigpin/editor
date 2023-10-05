package xutil

import (
	"fmt"
	"reflect"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Tags can be used with: `loadAtoms:"atomname"`.
// "st" should be a pointer to a struct with xproto.Atom fields.
// "onlyIfExists" asks the x server to assign a value only if the atom exists.
func LoadAtoms(conn *xgb.Conn, st any, onlyIfExists bool) error {
	// request atoms
	// use reflection to get atoms names
	typ := reflect.Indirect(reflect.ValueOf(st)).Type()
	var cookies []xproto.InternAtomCookie
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)

		name := sf.Name
		tagStr := sf.Tag.Get("loadAtoms")
		if tagStr != "" {
			name = tagStr
		}
		// request value
		cookie := xproto.InternAtom(conn, onlyIfExists, uint16(len(name)), name)
		cookies = append(cookies, cookie)
	}
	// get atoms
	val := reflect.Indirect(reflect.ValueOf(st))
	for i := 0; i < val.NumField(); i++ {
		reply, err := cookies[i].Reply() // get value
		if err != nil {
			return err
		}
		v := val.Field(i)
		v.Set(reflect.ValueOf(reply.Atom))
	}
	return nil
}

//----------

func GetAtomName(conn *xgb.Conn, atom xproto.Atom) (string, error) {
	cookie := xproto.GetAtomName(conn, atom)
	r, err := cookie.Reply()
	if err != nil {
		return "", err
	}
	return r.Name, nil
}

func PrintAtomsNames(conn *xgb.Conn, atoms ...xproto.Atom) {
	for _, a := range atoms {
		name, err := GetAtomName(conn, a)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%d: %s\n", a, name)
	}
}
