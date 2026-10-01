package testassert_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAssert_Equal_Pass(t *testing.T) {
	ok := t.Run("equal pass", func(st *testing.T) {
		testassert.HandleAssert(
			st,
			axiom.NewEqualAssert(1, 1, "values must be equal"),
		)
	})

	require.True(t, ok)
}

func TestHandleAssert_True_Pass(t *testing.T) {
	ok := t.Run("true pass", func(st *testing.T) {
		testassert.HandleAssert(
			st,
			axiom.NewTrueAssert(true, "must be true"),
		)
	})

	require.True(t, ok)
}

func TestHandleAssert_False_Pass(t *testing.T) {
	ok := t.Run("false pass", func(st *testing.T) {
		testassert.HandleAssert(
			st,
			axiom.NewFalseAssert(false, "must be false"),
		)
	})

	require.True(t, ok)
}

func TestHandleAssert_Error_Pass(t *testing.T) {
	ok := t.Run("error pass", func(st *testing.T) {
		testassert.HandleAssert(
			st,
			axiom.NewErrorAssert(assert.AnError, "must error"),
		)
	})

	require.True(t, ok)
}

func TestHandleAssert_NoError_Pass(t *testing.T) {
	ok := t.Run("no error pass", func(st *testing.T) {
		testassert.HandleAssert(
			st,
			axiom.NewNoErrorAssert(nil, "must not error"),
		)
	})

	require.True(t, ok)
}

func TestHandleAssert_Nil_Pass(t *testing.T) {
	ok := t.Run("nil pass", func(st *testing.T) {
		testassert.HandleAssert(
			st,
			axiom.NewNilAssert(nil, "must be nil"),
		)
	})

	require.True(t, ok)
}

func TestHandleAssert_NotNil_Pass(t *testing.T) {
	ok := t.Run("not nil pass", func(st *testing.T) {
		testassert.HandleAssert(
			st,
			axiom.NewNotNilAssert(123, "must not be nil"),
		)
	})

	require.True(t, ok)
}
