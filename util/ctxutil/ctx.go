package ctxutil

// This is a reference implementation. In many cases it works best to just copy this code and keep adding methods instead of embedding.

type Ctx struct {
	Parent *Ctx
	// name/value (short names to avoid usage, still exporting it)
	N string
	V any
}

func (ctx *Ctx) WithValue(name string, value any) *Ctx {
	return &Ctx{ctx, name, value}
}

func (ctx *Ctx) Value(name string) (any, *Ctx) {
	for c := ctx; c != nil; c = c.Parent {
		if c.N == name {
			return c.V, c
		}
	}
	return nil, nil
}

//----------

func (ctx *Ctx) ValueBool(name string) bool {
	v, _ := ctx.Value(name)
	if v == nil {
		return false
	}
	return v.(bool)
}

func (ctx *Ctx) ValueInt(name string) int {
	v, _ := ctx.Value(name)
	if v == nil {
		return 0
	}
	return v.(int)
}
