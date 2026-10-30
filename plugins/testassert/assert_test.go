package testassert_test

import (
	"flag"
	"os"
	"os/exec"
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

func TestHandleAssert_InvalidBoolActual(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{"true-string", "true assertion requires bool Actual, got string"},
		{"true-nil", "true assertion requires bool Actual, got <nil>"},
		{"false-string", "false assertion requires bool Actual, got string"},
		{"false-nil", "false assertion requires bool Actual, got <nil>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"-test.run=^TestHandleAssert_InvalidBoolActual_HelperProcess$", "-test.v"}
			if coverDir := flag.Lookup("test.gocoverdir"); coverDir != nil && coverDir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+coverDir.Value.String())
			}
			cmd := exec.Command(os.Args[0], args...)
			cmd.Env = append(os.Environ(), "AXIOM_TESTASSERT_INVALID_BOOL="+tc.name)
			output, err := cmd.CombinedOutput()

			require.Error(t, err, "invalid assertion must fail the test")
			assert.Contains(t, string(output), tc.want)
			assert.Contains(t, string(output), "invalid bool value")
			assert.Contains(t, string(output), "HandleAssert returned")
			assert.NotContains(t, string(output), "panic:")
		})
	}
}

func TestHandleAssert_InvalidBoolActual_HelperProcess(t *testing.T) {
	kind := os.Getenv("AXIOM_TESTASSERT_INVALID_BOOL")
	if kind == "" {
		t.Skip("helper process only")
	}

	a := axiom.Assert{Message: "invalid bool value"}
	switch kind {
	case "true-string":
		a.Type, a.Actual = axiom.AssertTrue, "yes"
	case "true-nil":
		a.Type = axiom.AssertTrue
	case "false-string":
		a.Type, a.Actual = axiom.AssertFalse, "no"
	case "false-nil":
		a.Type = axiom.AssertFalse
	default:
		t.Fatalf("unknown helper case %q", kind)
	}

	testassert.HandleAssert(t, a)
	t.Log("HandleAssert returned")
}
