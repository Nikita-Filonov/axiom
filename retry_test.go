package axiom_test

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewRetry_Defaults(t *testing.T) {
	r := axiom.NewRetry()

	assert.Equal(t, 0, r.Times)
	assert.Equal(t, time.Duration(0), r.Delay)
}

func TestRetryNormalize_Defaults(t *testing.T) {
	r := axiom.Retry{}
	r.Normalize()

	assert.Equal(t, 3, r.Times)
	assert.Equal(t, 2*time.Second, r.Delay)
}

func TestNewRetry_WithOptions(t *testing.T) {
	r := axiom.NewRetry(
		axiom.WithRetryTimes(5),
		axiom.WithRetryDelay(10*time.Millisecond),
	)

	assert.Equal(t, 5, r.Times)
	assert.Equal(t, 10*time.Millisecond, r.Delay)
}

func TestRetryJoin_OverridesTimes(t *testing.T) {
	base := axiom.Retry{
		Times: 3,
		Delay: 1 * time.Second,
	}
	other := axiom.Retry{
		Times: 10,
	}

	result := base.Join(other)

	assert.Equal(t, 10, result.Times)
	assert.Equal(t, 1*time.Second, result.Delay) // unchanged
}

func TestRetryJoin_OverridesDelay(t *testing.T) {
	base := axiom.Retry{
		Times: 5,
		Delay: 500 * time.Millisecond,
	}
	other := axiom.Retry{
		Delay: 2 * time.Second,
	}

	result := base.Join(other)

	assert.Equal(t, 5, result.Times)             // unchanged
	assert.Equal(t, 2*time.Second, result.Delay) // replaced
}

func TestRetryJoin_NoOverride(t *testing.T) {
	base := axiom.Retry{
		Times: 3,
		Delay: 1 * time.Second,
	}
	other := axiom.Retry{} // zero struct

	result := base.Join(other)

	assert.Equal(t, 3, result.Times)
	assert.Equal(t, 1*time.Second, result.Delay)
}

func TestRetryNormalize_DefaultTimes(t *testing.T) {
	r := axiom.Retry{
		Times: 0,
	}
	r.Normalize()

	assert.Equal(t, 3, r.Times)
}

func TestRetryNormalize_DefaultDelay(t *testing.T) {
	r := axiom.Retry{
		Delay: -5 * time.Second, // invalid delay
	}
	r.Normalize()

	assert.Equal(t, 2*time.Second, r.Delay)
}

func TestRetryNormalize_NoOverrideWhenValid(t *testing.T) {
	r := axiom.Retry{
		Times: 5,
		Delay: 1 * time.Second,
	}
	r.Normalize()

	assert.Equal(t, 5, r.Times)
	assert.Equal(t, 1*time.Second, r.Delay)
}
