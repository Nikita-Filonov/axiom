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
	assert.False(t, r.TimesSet)
	assert.False(t, r.DelaySet)
}

func TestRetryNormalize_Defaults_RetryDisabledByDefault(t *testing.T) {
	r := axiom.Retry{}
	r.Normalize()

	assert.Equal(t, 1, r.Times)
	assert.Equal(t, time.Duration(0), r.Delay)
}

func TestNewRetry_WithOptions_SetsFlags(t *testing.T) {
	r := axiom.NewRetry(
		axiom.WithRetryTimes(5),
		axiom.WithRetryDelay(10*time.Millisecond),
	)

	assert.Equal(t, 5, r.Times)
	assert.Equal(t, 10*time.Millisecond, r.Delay)
	assert.True(t, r.TimesSet)
	assert.True(t, r.DelaySet)
}

func TestRetryNormalize_DoesNotOverride_WhenSetFlagsPresent(t *testing.T) {
	r := axiom.NewRetry(
		axiom.WithRetryTimes(5),
		axiom.WithRetryDelay(1*time.Second),
	)
	r.Normalize()

	// Normalize не затирает явно заданные значения.
	assert.Equal(t, 5, r.Times)
	assert.Equal(t, 1*time.Second, r.Delay)
}

func TestRetryJoin_NoOverride_WhenFlagsNotSet(t *testing.T) {
	base := axiom.Retry{
		Times:    3,
		Delay:    1 * time.Second,
		TimesSet: true,
		DelaySet: true,
	}
	other := axiom.Retry{
		Times: 10,
		Delay: 2 * time.Second,
	}

	result := base.Join(other)

	assert.Equal(t, 3, result.Times)
	assert.Equal(t, 1*time.Second, result.Delay)
	assert.True(t, result.TimesSet)
	assert.True(t, result.DelaySet)
}

func TestRetryJoin_OverridesTimes_WhenTimesSet(t *testing.T) {
	base := axiom.Retry{
		Times:    3,
		Delay:    1 * time.Second,
		TimesSet: true,
		DelaySet: true,
	}
	other := axiom.Retry{
		Times:    10,
		TimesSet: true,
	}

	result := base.Join(other)

	assert.Equal(t, 10, result.Times)
	assert.Equal(t, 1*time.Second, result.Delay)
	assert.True(t, result.TimesSet)
	assert.True(t, result.DelaySet)
}

func TestRetryJoin_OverridesDelay_WhenDelaySet(t *testing.T) {
	base := axiom.Retry{
		Times:    5,
		Delay:    500 * time.Millisecond,
		TimesSet: true,
		DelaySet: true,
	}
	other := axiom.Retry{
		Delay:    2 * time.Second,
		DelaySet: true,
	}

	result := base.Join(other)

	assert.Equal(t, 5, result.Times)
	assert.Equal(t, 2*time.Second, result.Delay)
	assert.True(t, result.TimesSet)
	assert.True(t, result.DelaySet)
}

func TestRetryJoin_OverridesDelay_ToZero_WhenDelaySet(t *testing.T) {
	base := axiom.Retry{
		Times:    5,
		Delay:    2 * time.Second,
		TimesSet: true,
		DelaySet: true,
	}
	other := axiom.Retry{
		Delay:    0,
		DelaySet: true,
	}

	result := base.Join(other)

	assert.Equal(t, 5, result.Times)
	assert.Equal(t, time.Duration(0), result.Delay)
	assert.True(t, result.DelaySet)
}

func TestRetryJoin_OverridesTimes_ToOne_WhenTimesSet(t *testing.T) {
	base := axiom.Retry{
		Times:    5,
		Delay:    1 * time.Second,
		TimesSet: true,
		DelaySet: true,
	}
	other := axiom.Retry{
		Times:    1,
		TimesSet: true,
	}

	result := base.Join(other)

	assert.Equal(t, 1, result.Times)
	assert.Equal(t, 1*time.Second, result.Delay)
	assert.True(t, result.TimesSet)
}

func TestRetryNormalize_DefaultsApply_PerField(t *testing.T) {
	t.Run("Times defaulted when TimesSet=false", func(t *testing.T) {
		r := axiom.Retry{
			Delay:    50 * time.Millisecond,
			DelaySet: true,
		}
		r.Normalize()

		assert.Equal(t, 1, r.Times) // default
		assert.Equal(t, 50*time.Millisecond, r.Delay)
	})

	t.Run("Delay defaulted when DelaySet=false", func(t *testing.T) {
		r := axiom.Retry{
			Times:    3,
			TimesSet: true,
		}
		r.Normalize()

		assert.Equal(t, 3, r.Times)
		assert.Equal(t, time.Duration(0), r.Delay) // default
	})
}

func TestRetryJoin_PreservesBaseFlags_WhenOtherDoesNotOverride(t *testing.T) {
	base := axiom.Retry{
		Times:    3,
		Delay:    1 * time.Second,
		TimesSet: true,
		DelaySet: true,
	}
	other := axiom.Retry{}

	result := base.Join(other)

	assert.True(t, result.TimesSet)
	assert.True(t, result.DelaySet)
}

func TestRetryJoin_SetsFlags_WhenOtherOverrides(t *testing.T) {
	base := axiom.Retry{Times: 3, Delay: 1 * time.Second}
	other := axiom.Retry{Times: 10, TimesSet: true}

	result := base.Join(other)

	assert.Equal(t, 10, result.Times)
	assert.True(t, result.TimesSet)
	assert.Equal(t, 1*time.Second, result.Delay)
	assert.False(t, result.DelaySet)
}

func TestRetryNormalize_FixesInvalidTimes_WhenTimesSet_Zero(t *testing.T) {
	r := axiom.NewRetry(axiom.WithRetryTimes(0)) // TimesSet=true
	r.Normalize()

	assert.Equal(t, 1, r.Times)
	assert.True(t, r.TimesSet)
}

func TestRetryNormalize_FixesInvalidTimes_WhenTimesSet_Negative(t *testing.T) {
	r := axiom.NewRetry(axiom.WithRetryTimes(-10)) // TimesSet=true
	r.Normalize()

	assert.Equal(t, 1, r.Times)
	assert.True(t, r.TimesSet)
}

func TestRetryNormalize_FixesInvalidDelay_WhenDelaySet_Negative(t *testing.T) {
	r := axiom.NewRetry(axiom.WithRetryDelay(-5 * time.Second)) // DelaySet=true
	r.Normalize()

	assert.Equal(t, time.Duration(0), r.Delay)
	assert.True(t, r.DelaySet)
}

func TestRetryNormalize_DoesNotOverrideDelay_WhenDelaySet_ZeroIsValid(t *testing.T) {
	r := axiom.NewRetry(axiom.WithRetryDelay(0)) // DelaySet=true
	r.Normalize()

	assert.Equal(t, time.Duration(0), r.Delay)
	assert.True(t, r.DelaySet)
}

func TestRetryJoin_OverridesBoth_WhenBothSet(t *testing.T) {
	base := axiom.Retry{
		Times:    3,
		Delay:    1 * time.Second,
		TimesSet: true,
		DelaySet: true,
	}
	other := axiom.Retry{
		Times:    10,
		Delay:    0,
		TimesSet: true,
		DelaySet: true,
	}

	result := base.Join(other)

	assert.Equal(t, 10, result.Times)
	assert.Equal(t, time.Duration(0), result.Delay)
	assert.True(t, result.TimesSet)
	assert.True(t, result.DelaySet)
}

func TestRetryNormalize_DoesNotOverrideTimes_WhenTimesSet_OneIsValid(t *testing.T) {
	r := axiom.NewRetry(axiom.WithRetryTimes(1))
	r.Normalize()

	assert.Equal(t, 1, r.Times)
	assert.True(t, r.TimesSet)
}
