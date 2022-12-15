package ctxutil

type keeper[T any] struct {
	ps []*T
	vs []T
}

func Keep[T any](ps ...*T) *keeper[T] {
	k := &keeper[T]{ps: ps}
	for _, p := range ps {
		k.vs = append(k.vs, *p)
	}
	return k
}
func (k *keeper[T]) Restore() {
	for i, v := range k.vs {
		*k.ps[i] = v
	}
}
