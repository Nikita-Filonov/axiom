package testexplain

import "github.com/Nikita-Filonov/axiom"

// ExplainRunner snapshots a runner's current configuration. It panics for nil r.
func ExplainRunner(r *axiom.Runner) Explanation {
	if r == nil {
		panic("explain: nil *axiom.Runner")
	}
	return explainRunner(r, map[*axiom.Runner]bool{})
}

func explainRunner(r *axiom.Runner, path map[*axiom.Runner]bool) Explanation {
	if path[r] {
		return Explanation{Kind: ExplanationKindRunner, Runner: &RunnerExplanation{Cycle: true}}
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

	shape := &RunnerExplanation{
		Fixtures:  sortedMapKeys(fixtures.Registry),
		Resources: sortedMapKeys(resources.Registry),
		Plugins:   plugins,
		Meta:      meta.Copy(),
		Skip:      explainSkip(r.Skip),
		Retry:     explainRetry(retry),
		Parallel:  explainParallel(r.Parallel),
		Context:   explainContext(context),
		Hooks:     explainHooks(r.Hooks),
		Runtime:   explainRuntime(r.Runtime),
	}
	explainRunnerSources(r, shape, path)

	return Explanation{
		Kind:      ExplanationKindRunner,
		Runner:    shape,
		Meta:      meta,
		Skip:      shape.Skip,
		Retry:     shape.Retry,
		Parallel:  shape.Parallel,
		Context:   shape.Context,
		Fixtures:  sortedMapKeys(fixtures.Registry),
		Resources: sortedMapKeys(resources.Registry),
		Hooks:     shape.Hooks,
		Plugins: PluginsExplanation{
			Runner: plugins,
			Total:  plugins.Count,
		},
		Runtime: shape.Runtime,
	}
}

func explainRunnerSources(r *axiom.Runner, shape *RunnerExplanation, path map[*axiom.Runner]bool) {
	path[r] = true
	defer delete(path, r)
	if r.Parent != nil {
		parent := explainRunner(r.Parent, path)
		shape.Parent = &parent
	}
	if r.Overlay != nil {
		overlay := explainRunner(r.Overlay, path)
		shape.Overlay = &overlay
	}
}

// ExplainConfig snapshots an attempt's merged configuration. It panics for nil c.
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
	runnerShape := &RunnerExplanation{}
	if c.Runner != nil {
		runnerShape = ExplainRunner(c.Runner).Runner
		runnerPlugins = runnerShape.Plugins
	}

	casePlugins := CallableExplanation{}
	var caseExplanation *CaseExplanation
	if c.Case != nil {
		casePlugins = explainCallables(c.Case.Plugins)
		caseMeta := c.Case.Meta.Copy()
		caseExplanation = &CaseExplanation{
			ID:          c.Case.ID,
			Name:        c.Case.Name,
			Description: c.Case.Description,
			ParamsType:  paramsType(c.Case.Params),
			Fixtures:    sortedMapKeys(c.Case.Fixtures.Registry),
			Plugins:     casePlugins,
			Meta:        caseMeta,
			Skip:        explainSkip(c.Case.Skip),
			Retry:       explainRetry(c.Case.Retry),
			Parallel:    explainParallel(c.Case.Parallel),
			Context:     explainContext(c.Case.Context),
			Hooks:       explainHooks(c.Case.Hooks),
			Runtime:     explainRuntime(c.Case.Runtime),
		}
	}

	return Explanation{
		Kind:      ExplanationKindConfig,
		Runner:    runnerShape,
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
