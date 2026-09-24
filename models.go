package axiom

// Normalize fills defaults or initializes an object's internal state.
type Normalize interface {
	Normalize()
}

// Join describes values that can merge with another value of the same type.
type Join[T any] interface {
	Join(other T) T
}

// Copy describes values that can return an independent configuration copy.
type Copy[T any] interface {
	Copy() T
}
