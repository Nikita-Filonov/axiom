package testassert

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

// HandleAssert evaluates a structured assertion against t.
func HandleAssert(t *testing.T, a axiom.Assert) {
	switch a.Type {

	case axiom.AssertEqual:
		assert.Equal(t, a.Expected, a.Actual, a.Message)

	case axiom.AssertTrue, axiom.AssertFalse:
		actual, ok := a.Actual.(bool)
		if !ok {
			assert.Fail(t, fmt.Sprintf("%s assertion requires bool Actual, got %T", a.Type, a.Actual), a.Message)
			return
		}
		if a.Type == axiom.AssertTrue {
			assert.True(t, actual, a.Message)
		} else {
			assert.False(t, actual, a.Message)
		}

	case axiom.AssertError:
		assert.Error(t, a.Error, a.Message)

	case axiom.AssertNoError:
		assert.NoError(t, a.Error, a.Message)

	case axiom.AssertNil:
		assert.Nil(t, a.Actual, a.Message)

	case axiom.AssertNotNil:
		assert.NotNil(t, a.Actual, a.Message)
	}
}
