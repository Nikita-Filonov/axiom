package axiom

import (
	"testing"
	"time"
)

type executionPolicy func(*Config)

type caseExecution struct {
	rootT        *testing.T
	action       TestAction
	runner       *Runner
	baseConfig   *Config
	caseTemplate Case
}

func newCaseExecution(runner *Runner, rootT *testing.T, testCase Case, action TestAction) *caseExecution {
	baseCase := testCase.Copy()
	baseConfig := runner.BuildConfig(rootT, &baseCase)
	baseConfig.ApplyPlugins()

	return &caseExecution{
		rootT:        rootT,
		runner:       runner,
		action:       action,
		baseConfig:   baseConfig,
		caseTemplate: testCase,
	}
}

func (e *caseExecution) run() {
	if e.baseConfig.Parallel.Enabled && e.baseConfig.Retry.Times > 1 {
		e.runParallelRetry()
		return
	}

	e.runAttempts(
		e.rootT,
		(*Config).applySkipPolicy,
		(*Config).applyParallelPolicy,
	)
}

func (e *caseExecution) runParallelRetry() {
	e.rootT.Run(e.baseConfig.Case.Name, func(caseT *testing.T) {
		e.baseConfig.SubT = caseT
		e.baseConfig.applySkipPolicy()
		e.baseConfig.applyParallelPolicy()

		e.runAttempts(caseT, (*Config).applySkipPolicy)
	})
}

func (e *caseExecution) runAttempts(parentT *testing.T, policies ...executionPolicy) {
	for attempt := 1; attempt <= e.baseConfig.Retry.Times; attempt++ {
		e.waitBeforeAttempt(attempt)

		attemptConfig := e.newAttemptConfig()
		ok := parentT.Run(attemptConfig.Case.Name, func(attemptT *testing.T) {
			attemptConfig.SubT = attemptT
			for _, policy := range policies {
				policy(attemptConfig)
			}
			attemptConfig.Test(e.action)
		})

		if ok {
			return
		}
	}
}

func (e *caseExecution) newAttemptConfig() *Config {
	attemptCase := e.caseTemplate.Copy()
	attemptConfig := e.runner.BuildConfig(e.rootT, &attemptCase)
	attemptConfig.ApplyPlugins()

	return attemptConfig
}

func (e *caseExecution) waitBeforeAttempt(attempt int) {
	if attempt > 1 && e.baseConfig.Retry.Delay > 0 {
		time.Sleep(e.baseConfig.Retry.Delay)
	}
}
