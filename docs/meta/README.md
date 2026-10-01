# 📘 Meta

`Meta` provides optional descriptive information for a test. Metadata is used for filtering, reporting,
classification, grouping, and CI integration. It is purely declarative and can be applied at both `Runner` and `Case`
level. `Case` metadata overrides `Runner` metadata when merged.

Metadata includes:

- epic, feature, story
- owner (person or team responsible for the test)
- tags
- severity
- labels (key/value)
- layer

A `Meta` object does not affect test execution logic. It is consumed by plugins (filters, reporters, tracers).

---

## Example

The following example demonstrates all metadata options and how Runner-level values are merged with Case-level
overrides.

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestMetaExample(t *testing.T) {

	// -------------------------------------------------------------------------
	// Runner-level metadata
	// -------------------------------------------------------------------------

	runner := axiom.NewRunner(
		axiom.WithRunnerMeta(
			axiom.WithMetaEpic("authentication"),
			axiom.WithMetaOwner("backend-team"),
			axiom.WithMetaFeature("login"),
			axiom.WithMetaPlatform("backend"),
			axiom.WithMetaSeverity(axiom.SeverityCritical),
			axiom.WithMetaTag("regression"),
			axiom.WithMetaLabel("team", "backend"),
		),
	)

	// -------------------------------------------------------------------------
	// Case metadata (overrides/extends Runner metadata)
	// -------------------------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("user can authenticate"),
		axiom.WithCaseMeta(
			axiom.WithMetaOwner("auth-team"), // overrides Runner owner
			axiom.WithMetaStory("valid login flow"),
			axiom.WithMetaTag("smoke"),
			axiom.WithMetaLayer("api"),
			axiom.WithMetaLabel("component", "auth-service"),
			axiom.WithMetaSeverity(axiom.SeverityBlocker), // overrides Runner severity
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		meta := cfg.Meta

		fmt.Println("Epic:", meta.Epic)
		fmt.Println("Owner:", meta.Owner)
		fmt.Println("Feature:", meta.Feature)
		fmt.Println("Platform:", meta.Platform)
		fmt.Println("Story:", meta.Story)
		fmt.Println("Layer:", meta.Layer)
		fmt.Println("Severity:", meta.Severity)
		fmt.Println("Tags:", meta.Tags)
		fmt.Println("Labels:", meta.Labels)

		cfg.Step("validate", func() {
			fmt.Println("validating...")
		})
	})
}

```
