package axiom

// AssertType identifies the kind of structured assertion fact.
type AssertType string

// AssertType values identify the assertion recorded by an Assert.
const (
	AssertEqual AssertType = "equal"
	AssertTrue  AssertType = "true"
	AssertFalse AssertType = "false"

	AssertError   AssertType = "error"
	AssertNoError AssertType = "no-error"

	AssertNil    AssertType = "nil"
	AssertNotNil AssertType = "not-nil"
)

// String returns the assertion type name.
func (t AssertType) String() string {
	return string(t)
}

// Assert describes an assertion for runtime sinks. Constructing or emitting
// one does not evaluate a condition or fail a test by itself.
type Assert struct {
	Type AssertType

	Message string

	Expected any
	Actual   any

	Error error
}

// AssertOption configures an Assert.
type AssertOption func(*Assert)

// NewAssert returns an Assert with the supplied options.
func NewAssert(options ...AssertOption) Assert {
	a := Assert{}
	for _, option := range options {
		option(&a)
	}

	return a
}

// WithAssertType sets the kind of assertion fact.
func WithAssertType(t AssertType) AssertOption {
	return func(a *Assert) { a.Type = t }
}

// WithAssertMessage sets descriptive text for the assertion fact.
func WithAssertMessage(msg string) AssertOption {
	return func(a *Assert) { a.Message = msg }
}

// WithAssertExpected records the expected value without evaluating it.
func WithAssertExpected(v any) AssertOption {
	return func(a *Assert) { a.Expected = v }
}

// WithAssertActual records the observed value without evaluating it.
func WithAssertActual(v any) AssertOption {
	return func(a *Assert) { a.Actual = v }
}

// WithAssertError records an error for an assertion fact.
func WithAssertError(err error) AssertOption {
	return func(a *Assert) { a.Error = err }
}

// NewEqualAssert records expected and actual values for an equality assertion.
func NewEqualAssert(expected, actual any, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertEqual),
		WithAssertExpected(expected),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

// NewTrueAssert records a boolean value expected to be true.
func NewTrueAssert(actual bool, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertTrue),
		WithAssertExpected(true),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

// NewFalseAssert records a boolean value expected to be false.
func NewFalseAssert(actual bool, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertFalse),
		WithAssertExpected(false),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

// NewErrorAssert records an error expected to be non-nil.
func NewErrorAssert(err error, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertError),
		WithAssertError(err),
		WithAssertMessage(msg),
	)
}

// NewNoErrorAssert records an error expected to be nil.
func NewNoErrorAssert(err error, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertNoError),
		WithAssertError(err),
		WithAssertMessage(msg),
	)
}

// NewNilAssert records a value expected to be nil.
func NewNilAssert(actual any, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertNil),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}

// NewNotNilAssert records a value expected to be non-nil.
func NewNotNilAssert(actual any, msg string) Assert {
	return NewAssert(
		WithAssertType(AssertNotNil),
		WithAssertActual(actual),
		WithAssertMessage(msg),
	)
}
