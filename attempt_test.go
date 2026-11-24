package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestAttemptExcludeKeepsFirstReason(t *testing.T) {
	attempt := axiom.Attempt{Decision: axiom.Decision{Kind: axiom.DecisionRun}}
	attempt.Exclude("")
	attempt.Exclude("later exclusion")

	assert.Equal(t, axiom.Decision{Kind: axiom.DecisionExclude}, attempt.Decision)
}
