package testallure

import (
	"github.com/Nikita-Filonov/axiom"
	"github.com/dailymotion/allure-go"
)

func Plugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.TestWraps = append(cfg.TestWraps, func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				options := BuildAllureOptions(c)
				allure.Test(c.SubT, append(options, allure.Action(func() { next(c) }))...)
			}
		})

		cfg.StepWraps = append(cfg.StepWraps, func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				allure.Step(
					allure.Description(name),
					allure.Action(next),
				)
			}
		})
	}
}
