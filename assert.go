package axiom

type AssertType string

const (
	AssertEqual AssertType = "equal"
	AssertTrue  AssertType = "true"
	AssertFalse AssertType = "false"

	AssertError   AssertType = "error"
	AssertNoError AssertType = "no-error"

	AssertNil    AssertType = "nil"
	AssertNotNil AssertType = "not-nil"
)

type Assert struct {
	Type AssertType

	Message string

	Expected any
	Actual   any

	Error error
}

type AssertOption func(*Assert)

func NewAssert(options ...AssertOption) Assert {
	a := Assert{}
	for _, option := range options {
		option(&a)
	}

	return a
}

func WithAssertType(t AssertType) AssertOption {
	return func(a *Assert) { a.Type = t }
}

func WithAssertMessage(msg string) AssertOption {
	return func(a *Assert) { a.Message = msg }
}

func WithAssertExpected(v any) AssertOption {
	return func(a *Assert) { a.Expected = v }
}

func WithAssertActual(v any) AssertOption {
	return func(a *Assert) { a.Actual = v }
}

func WithAssertError(err error) AssertOption {
	return func(a *Assert) { a.Error = err }
}

func NewEqualAssert(expected, actual any, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertEqual),
		WithAssertExpected(expected),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

func NewTrueAssert(actual bool, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertTrue),
		WithAssertExpected(true),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

func NewFalseAssert(actual bool, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertFalse),
		WithAssertExpected(false),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

func NewErrorAssert(err error, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertError),
		WithAssertError(err),
		WithAssertMessage(msg),
	)
}

func NewNoErrorAssert(err error, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertNoError),
		WithAssertError(err),
		WithAssertMessage(msg),
	)
}

func NewNilAssert(actual any, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertNil),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

func NewNotNilAssert(actual any, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertNotNil),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}
