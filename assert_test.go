package axiom_test

import (
	"errors"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewAssert_WithOptions(t *testing.T) {
	err := errors.New("boom")

	a := axiom.NewAssert(
		axiom.WithAssertType(axiom.AssertEqual),
		axiom.WithAssertMessage("values should match"),
		axiom.WithAssertExpected(1),
		axiom.WithAssertActual(2),
		axiom.WithAssertError(err),
	)

	assert.Equal(t, axiom.AssertEqual, a.Type)
	assert.Equal(t, "values should match", a.Message)
	assert.Equal(t, 1, a.Expected)
	assert.Equal(t, 2, a.Actual)
	assert.Equal(t, err, a.Error)
}

func TestNewAssert_Empty(t *testing.T) {
	a := axiom.NewAssert()

	assert.Empty(t, a.Type)
	assert.Empty(t, a.Message)
	assert.Nil(t, a.Expected)
	assert.Nil(t, a.Actual)
	assert.Nil(t, a.Error)
}

func TestNewEqualAssert(t *testing.T) {
	a := axiom.NewEqualAssert(10, 20, "ids must match")

	assert.Equal(t, axiom.AssertEqual, a.Type)
	assert.Equal(t, "ids must match", a.Message)
	assert.Equal(t, 10, a.Expected)
	assert.Equal(t, 20, a.Actual)
	assert.Nil(t, a.Error)
}

func TestNewTrueAssert(t *testing.T) {
	a := axiom.NewTrueAssert(true, "condition should be true")

	assert.Equal(t, axiom.AssertTrue, a.Type)
	assert.Equal(t, "condition should be true", a.Message)
	assert.Equal(t, true, a.Expected)
	assert.Equal(t, true, a.Actual)
	assert.Nil(t, a.Error)
}

func TestNewFalseAssert(t *testing.T) {
	a := axiom.NewFalseAssert(false, "condition should be false")

	assert.Equal(t, axiom.AssertFalse, a.Type)
	assert.Equal(t, "condition should be false", a.Message)
	assert.Equal(t, false, a.Expected)
	assert.Equal(t, false, a.Actual)
	assert.Nil(t, a.Error)
}

func TestNewErrorAssert(t *testing.T) {
	err := errors.New("failed")

	a := axiom.NewErrorAssert(err, "operation should fail")

	assert.Equal(t, axiom.AssertError, a.Type)
	assert.Equal(t, "operation should fail", a.Message)
	assert.Equal(t, err, a.Error)
	assert.Nil(t, a.Expected)
	assert.Nil(t, a.Actual)
}

func TestNewNoErrorAssert(t *testing.T) {
	err := errors.New("ignored")

	a := axiom.NewNoErrorAssert(err, "operation should not fail")

	assert.Equal(t, axiom.AssertNoError, a.Type)
	assert.Equal(t, "operation should not fail", a.Message)
	assert.Equal(t, err, a.Error)
	assert.Nil(t, a.Expected)
	assert.Nil(t, a.Actual)
}

func TestNewNilAssert(t *testing.T) {
	var v any = nil

	a := axiom.NewNilAssert(v, "value must be nil")

	assert.Equal(t, axiom.AssertNil, a.Type)
	assert.Equal(t, "value must be nil", a.Message)
	assert.Equal(t, v, a.Actual)
	assert.Nil(t, a.Expected)
	assert.Nil(t, a.Error)
}

func TestNewNotNilAssert(t *testing.T) {
	v := 123

	a := axiom.NewNotNilAssert(v, "value must exist")

	assert.Equal(t, axiom.AssertNotNil, a.Type)
	assert.Equal(t, "value must exist", a.Message)
	assert.Equal(t, v, a.Actual)
	assert.Nil(t, a.Expected)
	assert.Nil(t, a.Error)
}
