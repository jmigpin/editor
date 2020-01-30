package core

import (
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/osutil"
)

type HomeVars struct {
	m toolbarparser.HomeVarMap
}

func NewHomeVars() *HomeVars {
	return &HomeVars{}
}

func (hv *HomeVars) ParseToolbarVars(strs ...string) {
	// merge strings maps
	m := toolbarparser.VarMap{}
	for _, str := range strs {
		data := toolbarparser.Parse(str)
		m2 := toolbarparser.ParseVars(data)
		// merge
		for k, v := range m2 {
			m[k] = v
		}
	}
	// add env home var at the end to enforce value
	h := osutil.HomeEnvVar()
	if h != "" {
		m["~"] = h
	}

	hv.m = toolbarparser.FilterHomeVars(m)
}

//----------

func (hv *HomeVars) Encode(filename string) string {
	return toolbarparser.EncodeHomeVar(filename, hv.m)
}

func (hv *HomeVars) Decode(filename string) string {
	return toolbarparser.DecodeHomeVar(filename, hv.m)
}
