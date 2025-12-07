package axiom

type Normalize interface {
	Normalize()
}

type Join[T any] interface {
	Join(other T) T
}
