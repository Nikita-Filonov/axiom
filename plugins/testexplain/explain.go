package testexplain

import "github.com/Nikita-Filonov/axiom"

func ExplainRunner(r *axiom.Runner) Explanation {
	if r == nil {
		panic("explain: nil *axiom.Runner")
	}

	meta := r.Meta.Copy()
	meta.Normalize()

	retry := r.Retry.Copy()
	retry.Normalize()

	context := r.Context.Copy()
	context.Normalize()

	fixtures := r.Fixtures.Copy()
	fixtures.Normalize()

	resources := r.Resources.Copy()
	resources.Normalize()

	plugins := explainCallables(r.Plugins)

	return Explanation{
		Kind: ExplanationKindRunner,
		Runner: &RunnerExplanation{
			Fixtures:  sortedMapKeys(fixtures.Registry),
			Resources: sortedMapKeys(resources.Registry),
			Plugins:   plugins,
		},
		Meta:      meta,
		Skip:      explainSkip(r.Skip),
		Retry:     explainRetry(retry),
		Parallel:  explainParallel(r.Parallel),
		Context:   explainContext(context),
		Fixtures:  sortedMapKeys(fixtures.Registry),
		Resources: sortedMapKeys(resources.Registry),
		Hooks:     explainHooks(r.Hooks),
		Plugins: PluginsExplanation{
			Runner: plugins,
			Total:  plugins.Count,
		},
		Runtime: explainRuntime(r.Runtime),
	}
}

func ExplainConfig(c *axiom.Config) Explanation {
	if c == nil {
		panic("explain: nil *axiom.Config")
	}

	meta := c.Meta.Copy()
	meta.Normalize()

	retry := c.Retry.Copy()
	retry.Normalize()

	context := c.Context.Copy()
	context.Normalize()

	fixtures := c.Fixtures.Copy()
	fixtures.Normalize()

	resources := axiom.Resources{}
	if c.Runner != nil {
		resources = c.Runner.Resources.Copy()
		resources.Normalize()
	}

	runnerPlugins := CallableExplanation{}
	var runnerFixtures []string
	var runnerResources []string
	if c.Runner != nil {
		runnerPlugins = explainCallables(c.Runner.Plugins)
		runnerFixtures = sortedMapKeys(c.Runner.Fixtures.Registry)
		runnerResources = sortedMapKeys(c.Runner.Resources.Registry)
	}

	casePlugins := CallableExplanation{}
	var caseExplanation *CaseExplanation
	if c.Case != nil {
		casePlugins = explainCallables(c.Case.Plugins)
		caseExplanation = &CaseExplanation{
			ID:          c.Case.ID,
			Name:        c.Case.Name,
			Description: c.Case.Description,
			ParamsType:  paramsType(c.Case.Params),
			Fixtures:    sortedMapKeys(c.Case.Fixtures.Registry),
			Plugins:     casePlugins,
		}
	}

	return Explanation{
		Kind: ExplanationKindConfig,
		Runner: &RunnerExplanation{
			Fixtures:  runnerFixtures,
			Resources: runnerResources,
			Plugins:   runnerPlugins,
		},
		Case:      caseExplanation,
		Meta:      meta,
		Skip:      explainSkip(c.Skip),
		Retry:     explainRetry(retry),
		Parallel:  explainParallel(c.Parallel),
		Context:   explainContext(context),
		Fixtures:  sortedMapKeys(fixtures.Registry),
		Resources: sortedMapKeys(resources.Registry),
		Hooks:     explainHooks(c.Hooks),
		Plugins: PluginsExplanation{
			Runner: runnerPlugins,
			Case:   casePlugins,
			Total:  runnerPlugins.Count + casePlugins.Count,
		},
		Runtime: explainRuntime(c.Runtime),
	}
}
