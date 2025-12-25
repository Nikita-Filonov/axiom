package testassert

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func HandleAssert(t *testing.T, a axiom.Assert) {
	switch a.Type {

	case axiom.AssertEqual:
		assert.Equal(t, a.Expected, a.Actual, a.Message)

	case axiom.AssertTrue:
		assert.True(t, a.Actual.(bool), a.Message)

	case axiom.AssertFalse:
		assert.False(t, a.Actual.(bool), a.Message)

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
