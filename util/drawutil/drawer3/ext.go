package drawer3

type Ext interface {
	On() bool
	SetOn(v bool)
	Start(*ExtRunner) // only runs when ext is on
	Iterate(*ExtRunner)
	End(*ExtRunner)
}

//----------

type EExt struct {
	off bool // on by default
}

func (e *EExt) On() bool           { return !e.off }
func (e *EExt) SetOn(v bool)       { e.off = !v }
func (*EExt) Start(r *ExtRunner)   {}
func (*EExt) Iterate(r *ExtRunner) { r.NextExt() }
func (*EExt) End(r *ExtRunner)     {}

//----------

type ExtRunner struct {
	D  Drawer
	RR *RuneReader

	exts []Ext
	curi int         // current index
	curf func(e Ext) // current function
	stop bool        // stop iterate phase
}

func RunExts(d Drawer, rr *RuneReader, exts []Ext, postStart func()) {
	// run only extensions that are on
	extsOn := []Ext{}
	for _, e := range exts {
		if e.On() {
			extsOn = append(extsOn, e)
		}
	}
	if len(extsOn) == 0 {
		return
	}

	r := ExtRunner{D: d, RR: rr, exts: extsOn}
	r.run(postStart)
}

func (r *ExtRunner) run(postStart func()) {
	// start phase
	r.curf = nil
	for _, e := range r.exts {
		e.Start(r)
	}

	if postStart != nil {
		postStart()
	}

	// iterate phase
	r.curf = func(e Ext) {
		e.Iterate(r)
	}
	for r.stop = false; !r.stop; {
		r.curi = 0
		r.NextExt()
	}

	// end phase
	r.curf = nil
	for _, e := range r.exts {
		e.End(r)
	}
}

// Called during iterate phase only.
func (r *ExtRunner) NextExt() bool {
	if r.curf == nil {
		panic("this func should only run inside iteration phase")
	}
	if r.curi < len(r.exts) {
		e := r.exts[r.curi]
		r.curi++
		r.curf(e)
		r.curi--
	}
	return !r.stop
}

// Called during iterate phase only.
func (r *ExtRunner) Stop() {
	r.stop = true
}
