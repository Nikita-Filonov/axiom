package testallure

import (
	"github.com/Nikita-Filonov/axiom"
	"github.com/dailymotion/allure-go"
)

func Plugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				options := BuildAllureOptions(c)
				allure.Test(c.SubT, append(options, allure.Action(func() { next(c) }))...)
			}
		})

		cfg.Runtime.EmitStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				allure.Step(allure.Description(name), allure.Action(next))
			}
		})

		cfg.Runtime.EmitSetupWrap(func(name string, next axiom.SetupAction) axiom.SetupAction {
			return func() {
				allure.BeforeTest(cfg.SubT, allure.Description(name), allure.Action(next))
			}
		})

		cfg.Runtime.EmitTeardownWrap(func(name string, next axiom.TeardownAction) axiom.TeardownAction {
			return func() {
				allure.AfterTest(cfg.SubT, allure.Description(name), allure.Action(next))
			}
		})

		cfg.Runtime.EmitArtefactSink(func(a axiom.Artefact) { HandleArtefact(cfg, a) })
	}
}
