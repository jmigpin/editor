package core

import (
	"fmt"
	"plugin"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/pkg/errors"
)

type Plugins struct {
	ed    *Editor
	plugs []*Plug
	added map[string]bool
}

func NewPlugins(ed *Editor) *Plugins {
	return &Plugins{ed: ed, added: map[string]bool{}}
}

func (p *Plugins) AddPath(path string) error {
	if p.added[path] {
		return nil
	}
	p.added[path] = true

	oplugin, err := plugin.Open(path)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("plugin: %v", path))
	}

	plug := &Plug{Plugin: oplugin, Path: path}
	p.plugs = append(p.plugs, plug)

	return p.runOnLoad(plug)
}

//----------

func (p *Plugins) runOnLoad(plug *Plug) error {
	// plugin should have this symbol
	f, err := plug.Plugin.Lookup("OnLoad")
	if err != nil {
		return nil // ok if plugin doesn't implement this symbol
	}
	// the symbol must implement this signature
	f2, ok := f.(func(*Editor))
	if !ok {
		return fmt.Errorf("plugin: %v: bad func signature", plug.Path)
	}
	// run symbol
	f2(p.ed)
	return nil
}

//----------

func (p *Plugins) RunAutoComplete(cfb *ui.ContextFloatBox) {
	for _, plug := range p.plugs {
		p.runAutoCompletePlug(plug, cfb)
	}
}

func (p *Plugins) runAutoCompletePlug(plug *Plug, cfb *ui.ContextFloatBox) {
	// plugin should have this symbol
	f, err := plug.Plugin.Lookup("AutoComplete")
	if err != nil {
		return // ok if plugin doesn't implement this symbol
	}
	// the symbol must implement this signature
	f2, ok := f.(func(*Editor, *ui.ContextFloatBox))
	if !ok {
		err := fmt.Errorf("plugin: %v: bad func signature", plug.Path)
		p.ed.Error(err)
		return
	}
	// run symbol
	f2(p.ed, cfb)
}

//----------

func (p *Plugins) RunToolbarCmd(erow *ERow, part *toolbarparser.Part) bool {
	for _, plug := range p.plugs {
		handled := p.runToolbarCmdPlug(plug, erow, part)
		if handled {
			return true
		}
	}
	return false
}

func (p *Plugins) runToolbarCmdPlug(plug *Plug, erow *ERow, part *toolbarparser.Part) bool {
	// plugin should have this symbol
	f, err := plug.Plugin.Lookup("ToolbarCmd")
	if err != nil {
		// no error: ok if plugin doesn't implement this symbol
		return false // false: doesn't implemente the required cmd
	}
	// the symbol must implement this signature
	f2, ok := f.(func(*Editor, *ERow, *toolbarparser.Part) bool)
	if !ok {
		err := fmt.Errorf("plugin: %v: bad func signature", plug.Path)
		p.ed.Error(err)
		return false // false: doesn't implement the required cmd
	}
	// run symbol
	return f2(p.ed, erow, part)
}

//----------

type Plug struct {
	Path   string
	Plugin *plugin.Plugin
}
