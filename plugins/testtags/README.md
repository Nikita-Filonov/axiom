# 🏷 Tags Plugin (`testtags`)

Filters test execution based on metadata tags using include and exclude rules.

The plugin evaluates test metadata at runtime and decides whether a test should be executed or skipped. It does not
affect test logic and does not modify execution flow directly.

---

## What the plugin does

At runtime, the plugin:

- reads tags from test metadata (`cfg.Meta.Tags`)
- normalizes tags (trimmed, lowercased)
- applies include and exclude rules
- marks tests as skipped when rules do not match

If a rule fails, the plugin sets:

```go
cfg.Skip = axiom.Skip{Enabled: true, Reason:  "..."}

```

Skipped tests are still visible to other plugins (stats, reporting, etc.).

---

## Configuration

The plugin can be configured via code, environment variables, or both.

### Include / exclude rules

- `Include` — only tests with at least one matching tag are executed
- `Exclude` — tests with any matching tag are skipped

Rules are evaluated in the following order:

- exclude rules
- include rules

---

## Environment variables

The plugin supports environment-driven configuration:

- `AXIOM_TEST_TAGS_INCLUDE`
- `AXIOM_TEST_TAGS_EXCLUDE`

Values are comma-separated lists of tags.

Example:

```shell
export AXIOM_TEST_TAGS_INCLUDE=smoke,critical
export AXIOM_TEST_TAGS_EXCLUDE=slow

```

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtags"
)

func TestTagsExample(t *testing.T) {

	// Enable tag-based filtering.
	// Only tests tagged with "smoke" will be executed.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testtags.Plugin(
				testtags.WithConfigInclude("smoke"),
				testtags.ConfigFromEnv(), // optional: merge env-based rules
			),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("tagged test"),

		// Tags are defined via test metadata
		axiom.WithCaseMeta(
			axiom.WithMetaTag("smoke"),
			axiom.WithMetaTag("auth"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// If the tag filter does not match,
		// this test will be skipped before execution.
		cfg.Step("run", func() {
			// Test logic goes here
		})
	})
}

```
