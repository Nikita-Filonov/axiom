package usage

import "github.com/Nikita-Filonov/axiom"

var UsersRunner = BaseRunner.Join(
	axiom.NewRunner(
		axiom.WithRunnerMeta(
			axiom.WithMetaFeature("users"),
			axiom.WithMetaTags("users", "auth"),
		),

		axiom.WithRunnerContext(
			axiom.WithContextData("service", "users-api"),
		),
	),
)
