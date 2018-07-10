package core

import (
	"os"

	"github.com/jmigpin/editor/core/toolbarparser"
)

type HomeVars struct {
	m toolbarparser.VarMap
}

func NewHomeVars() *HomeVars {
	return &HomeVars{}
}

func (hv *HomeVars) ParseToolbarVars(strs ...string) {
	hv.m = toolbarparser.VarMap{}
	for _, str := range strs {
		data := toolbarparser.Parse(str)
		m := toolbarparser.ParseVars(data)
		// merge
		for k, v := range m {
			hv.m[k] = v
		}
	}

	// add env home var
	h := os.Getenv("HOME")
	if h != "" {
		hv.m["~"] = h
	}
}

//----------

func (hv *HomeVars) Encode(filename string) string {
	return toolbarparser.EncodeVars(filename, hv.m)
}

func (hv *HomeVars) Decode(filename string) string {
	return toolbarparser.DecodeVars(filename, hv.m)
}
