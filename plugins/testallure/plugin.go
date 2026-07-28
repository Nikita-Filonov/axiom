package testallure

import (
	"sync/atomic"

	"github.com/Nikita-Filonov/axiom"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

type allureContextState struct {
	current atomic.Pointer[allure.Context]
}

func Plugin(options ...allure.Option) axiom.Plugin {
	baseOptions := append([]allure.Option(nil), options...)

	return func(cfg *axiom.Config) {
		state := &allureContextState{}

		cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				testOptions := append(BuildAllureOptions(c), baseOptions...)
				allure.Wrap(c.SubT, func(ctx *allure.Context) {
					previous := state.current.Swap(ctx)
					defer state.current.Store(previous)

					next(c)
				}, testOptions...)
			}
		})

		cfg.Runtime.EmitStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				runAllureStep(state, name, next)
			}
		})

		cfg.Runtime.EmitSetupWrap(func(name string, next axiom.SetupAction) axiom.SetupAction {
			return func() {
				runAllureStep(state, name, next)
			}
		})

		cfg.Runtime.EmitTeardownWrap(func(name string, next axiom.TeardownAction) axiom.TeardownAction {
			return func() {
				runAllureStep(state, name, next)
			}
		})

		cfg.Runtime.EmitArtefactSink(func(a axiom.Artefact) {
			handleArtefact(state.current.Load(), cfg, a)
		})
	}
}

func runAllureStep(state *allureContextState, name string, next func()) {
	ctx := state.current.Load()
	if ctx == nil {
		next()
		return
	}

	ctx.Step(name, func(stepContext *allure.Context) {
		previous := state.current.Swap(stepContext)
		defer state.current.Store(previous)

		next()
	})
}
