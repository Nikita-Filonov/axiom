package testallure

import (
	"github.com/Nikita-Filonov/axiom"
	"github.com/dailymotion/allure-go"
	"github.com/dailymotion/allure-go/severity"
)

func BuildAllureOptions(cfg *axiom.Config) []allure.Option {
	var options []allure.Option

	if cfg.ID != "" {
		options = append(options, allure.ID(cfg.ID))
	}
	if cfg.Name != "" {
		options = append(options, allure.Name(cfg.Name))
	}
	if len(cfg.Meta.Tags) > 0 {
		options = append(options, allure.Tags(cfg.Meta.Tags...))
	}
	if cfg.Meta.Epic != "" {
		options = append(options, allure.Epic(cfg.Meta.Epic))
	}
	if cfg.Meta.Story != "" {
		options = append(options, allure.Story(cfg.Meta.Story))
	}
	if cfg.Meta.Layer != "" {
		options = append(options, allure.Layer(cfg.Meta.Layer))
	}
	if cfg.Meta.Feature != "" {
		options = append(options, allure.Feature(cfg.Meta.Feature))
	}
	if cfg.Meta.Severity != "" {
		options = append(options, allure.Severity(severity.Severity(cfg.Meta.Severity)))
	}

	for k, v := range cfg.Meta.Labels {
		options = append(options, allure.Label(k, v))
	}

	return options
}
