package core

import (
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/osutil"
)

type HomeVars struct {
	hvm *toolbarparser.HomeVarMap
}

func NewHomeVars() *HomeVars {
	return &HomeVars{}
}

func (hv *HomeVars) ParseToolbarVars(strs []string, caseInsensitive bool) {
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

	hv.hvm = toolbarparser.NewHomeVarMap(m, caseInsensitive)
}

//----------

func (hv *HomeVars) Encode(filename string) string {
	return hv.hvm.Encode(filename)
}

func (hv *HomeVars) Decode(filename string) string {
	return hv.hvm.Decode(filename)
}
