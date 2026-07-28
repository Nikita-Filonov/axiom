package testallure

import (
	"sort"

	"github.com/Nikita-Filonov/axiom"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/allure-framework/allure-go/commons/model"
)

func BuildAllureOptions(cfg *axiom.Config) []allure.Option {
	var options []allure.Option

	if cfg.Case != nil && cfg.Case.ID != "" {
		options = append(
			options,
			allure.WithAllureID(cfg.Case.ID),
			allure.WithTestCaseID(cfg.Case.ID),
		)
	}
	if cfg.Case != nil && cfg.Case.Name != "" {
		options = append(options, allure.WithDisplayName(cfg.Case.Name))
	}
	if cfg.Case != nil && cfg.Case.Description != "" {
		options = append(options, allure.WithDescription(cfg.Case.Description))
	}
	for _, tag := range cfg.Meta.Tags {
		options = append(options, allure.WithTag(tag))
	}
	if cfg.Meta.Epic != "" {
		options = append(options, allure.WithEpic(cfg.Meta.Epic))
	}
	if cfg.Meta.Suite != "" {
		options = append(options, allure.WithSuite(cfg.Meta.Suite))
	}
	if cfg.Meta.Story != "" {
		options = append(options, allure.WithStory(cfg.Meta.Story))
	}
	if cfg.Meta.Layer != "" {
		options = append(options, allure.WithLabel("layer", cfg.Meta.Layer))
	}
	if cfg.Meta.Platform != "" {
		options = append(options, allure.WithLabel("platform", cfg.Meta.Platform))
	}
	if cfg.Meta.Feature != "" {
		options = append(options, allure.WithFeature(cfg.Meta.Feature))
	}
	if cfg.Meta.Severity != "" {
		options = append(options, allure.WithSeverity(string(cfg.Meta.Severity)))
	}
	if cfg.Meta.SubSuite != "" {
		options = append(options, allure.WithSubSuite(cfg.Meta.SubSuite))
	}
	if cfg.Meta.ParentSuite != "" {
		options = append(options, allure.WithParentSuite(cfg.Meta.ParentSuite))
	}
	for _, issue := range cfg.Meta.Issues {
		if issue == "" {
			continue
		}
		options = append(
			options,
			allure.WithLink(issue, issue, string(model.LinkTypeIssue)),
		)
	}
	for _, testCase := range cfg.Meta.TestCases {
		if testCase == "" {
			continue
		}
		options = append(
			options,
			allure.WithLink(testCase, testCase, string(model.LinkTypeTMS)),
		)
	}

	labelNames := make([]string, 0, len(cfg.Meta.Labels))
	for name := range cfg.Meta.Labels {
		labelNames = append(labelNames, name)
	}
	sort.Strings(labelNames)

	for _, name := range labelNames {
		options = append(options, allure.WithLabel(name, cfg.Meta.Labels[name]))
	}

	return options
}
