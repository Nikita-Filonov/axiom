package testleaks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithIgnoreFunction(t *testing.T) {
	c := newConfig(WithIgnoreFunction("worker"), WithGracePeriod(0), WithoutGoroutines())
	assert.Equal(t, []string{"worker"}, c.IgnoreFunctions)
	assert.Zero(t, c.GracePeriod)
	assert.False(t, c.Goroutines)
	assert.Panics(t, func() { WithIgnoreFunction("") })
}

func TestDefaultConfig(t *testing.T) {
	c := newConfig()
	assert.Equal(t, 200*time.Millisecond, c.GracePeriod)
	assert.True(t, c.Goroutines)
	assert.Empty(t, c.IgnoreFunctions)
}
