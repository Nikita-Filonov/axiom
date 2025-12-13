package usage

import (
	"github.com/Nikita-Filonov/axiom"
)

var UsersAdminRunner = UsersRunner.Join(
	axiom.NewRunner(
		axiom.WithRunnerMeta(
			axiom.WithMetaFeature("admin"),
			axiom.WithMetaTags("admin"),
		),
	),
)
