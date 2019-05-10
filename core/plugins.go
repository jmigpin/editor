package core

import (
	"context"
	"fmt"
	"plugin"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/iout"
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

// Runs all plugins until it finds one that returns handled=true and has no errors.
func (p *Plugins) RunAutoComplete(ctx context.Context, cfb *ui.ContextFloatBox) (_ error, handled bool) {
	me := iout.MultiError{}
	for _, plug := range p.plugs {
		err, handled := p.runAutoCompletePlug(ctx, plug, cfb)
		if handled {
			return err, true
		}
		me.Add(err)
	}
	return me.Result(), false
}

func (p *Plugins) runAutoCompletePlug(ctx context.Context, plug *Plug, cfb *ui.ContextFloatBox) (_ error, handled bool) {
	// plugin should have this symbol
	fn1, err := plug.Plugin.Lookup("AutoComplete")
	if err != nil {
		return nil, false // ok if plugin doesn't implement this symbol
	}
	// the symbol must implement this signature
	fn2, ok := fn1.(func(context.Context, *Editor, *ui.ContextFloatBox) (_ error, handled bool))
	if !ok {
		err := fmt.Errorf("plugin: %v: bad func signature", plug.Path)
		return err, false
	}
	// run symbol
	return fn2(ctx, p.ed, cfb)
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
