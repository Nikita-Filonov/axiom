package testexplain

func copyPointer[T any](value *T, copyValue func(T) T) *T {
	if value == nil {
		return nil
	}
	result := copyValue(*value)
	return &result
}

func copyStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

func copyExplanation(in Explanation) Explanation {
	out := in
	out.Runner = copyPointer(in.Runner, copyRunnerExplanation)
	out.Case = copyPointer(in.Case, copyCaseExplanation)
	out.Meta = in.Meta.Copy()
	out.Context = copyContextExplanation(in.Context)
	out.Fixtures = copyStrings(in.Fixtures)
	out.Resources = copyStrings(in.Resources)
	out.Hooks = copyHooksExplanation(in.Hooks)
	out.Plugins = copyPluginsExplanation(in.Plugins)
	out.Runtime = copyRuntimeExplanation(in.Runtime)
	return out
}

func copyRunnerExplanation(in RunnerExplanation) RunnerExplanation {
	out := in
	out.Fixtures = copyStrings(in.Fixtures)
	out.Resources = copyStrings(in.Resources)
	out.Plugins = copyCallableExplanation(in.Plugins)
	out.Meta = in.Meta.Copy()
	out.Context = copyContextExplanation(in.Context)
	out.Hooks = copyHooksExplanation(in.Hooks)
	out.Runtime = copyRuntimeExplanation(in.Runtime)
	out.Parent = copyPointer(in.Parent, copyExplanation)
	out.Overlay = copyPointer(in.Overlay, copyExplanation)
	return out
}

func copyCaseExplanation(in CaseExplanation) CaseExplanation {
	out := in
	out.Fixtures = copyStrings(in.Fixtures)
	out.Plugins = copyCallableExplanation(in.Plugins)
	out.Meta = in.Meta.Copy()
	out.Context = copyContextExplanation(in.Context)
	out.Hooks = copyHooksExplanation(in.Hooks)
	out.Runtime = copyRuntimeExplanation(in.Runtime)
	return out
}

func copyContextExplanation(in ContextExplanation) ContextExplanation {
	out := in
	out.DataKeys = copyStrings(in.DataKeys)
	return out
}

func copyHooksExplanation(in HooksExplanation) HooksExplanation {
	out := in
	out.BeforeAll = copyCallableExplanation(in.BeforeAll)
	out.AfterAll = copyCallableExplanation(in.AfterAll)
	out.BeforeTest = copyCallableExplanation(in.BeforeTest)
	out.AfterTest = copyCallableExplanation(in.AfterTest)
	out.BeforeStep = copyCallableExplanation(in.BeforeStep)
	out.AfterStep = copyCallableExplanation(in.AfterStep)
	return out
}

func copyPluginsExplanation(in PluginsExplanation) PluginsExplanation {
	out := in
	out.Runner = copyCallableExplanation(in.Runner)
	out.Case = copyCallableExplanation(in.Case)
	return out
}

func copyRuntimeExplanation(in RuntimeExplanation) RuntimeExplanation {
	out := in
	out.TestWraps = copyCallableExplanation(in.TestWraps)
	out.StepWraps = copyCallableExplanation(in.StepWraps)
	out.SetupWraps = copyCallableExplanation(in.SetupWraps)
	out.TeardownWraps = copyCallableExplanation(in.TeardownWraps)
	out.LogSinks = copyCallableExplanation(in.LogSinks)
	out.AssertSinks = copyCallableExplanation(in.AssertSinks)
	out.ArtefactSinks = copyCallableExplanation(in.ArtefactSinks)
	out.EventSinks = copyCallableExplanation(in.EventSinks)
	return out
}

func copyCallableExplanation(in CallableExplanation) CallableExplanation {
	out := in
	out.Names = copyStrings(in.Names)
	return out
}
