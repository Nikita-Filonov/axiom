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
	e := &caseExecution{
		rootT:        rootT,
		runner:       runner,
		action:       action,
		caseTemplate: testCase,
	}
	e.baseConfig = e.newConfig(newExecution())

	return e
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
	execution := e.baseConfig.Execution
	for execution.Attempt < e.baseConfig.Retry.Times {
		execution = execution.nextAttempt()
		e.waitBeforeAttempt(execution.Attempt)

		attemptConfig := e.newConfig(execution)
		ok := parentT.Run(attemptConfig.Case.Name, func(attemptT *testing.T) {
			attemptConfig.SubT = attemptT
			for _, policy := range policies {
				policy(attemptConfig)
			}
			attemptConfig.test(e.action)
		})

		if ok {
			return
		}
	}
}

func (e *caseExecution) newConfig(execution Execution) *Config {
	attemptCase := e.caseTemplate.Copy()
	cfg := e.runner.buildConfig(e.rootT, &attemptCase, execution)
	cfg.applyPlugins()

	return cfg
}

func (e *caseExecution) waitBeforeAttempt(attempt int) {
	if attempt > 1 && e.baseConfig.Retry.Delay > 0 {
		time.Sleep(e.baseConfig.Retry.Delay)
	}
}
