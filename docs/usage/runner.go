package usage

import "github.com/Nikita-Filonov/axiom"

var BaseRunner = axiom.NewRunner(
	axiom.WithRunnerMeta(
		axiom.WithMetaEpic("platform"),
		axiom.WithMetaLayer("e2e"),
		axiom.WithMetaSeverity(axiom.SeverityNormal),
	),

	axiom.WithRunnerContext(
		axiom.WithContextData("env", "staging"),
	),

	axiom.WithRunnerRetry(
		axiom.WithRetryTimes(2),
	),

	axiom.WithRunnerParallel(),

	axiom.WithRunnerFixture("config", ConfigFixture),
)
