# Axiom

<p align="center">
  <img src="./docs/assets/logo.png" alt="Axiom logo" width="220" />
</p>

🚀 Modern, extensible, and composable Go test framework.

[![CI](https://github.com/Nikita-Filonov/axiom/actions/workflows/workflow-test.yml/badge.svg)](https://github.com/Nikita-Filonov/axiom/actions/workflows/workflow-test.yml)
[![codecov](https://codecov.io/gh/Nikita-Filonov/axiom/branch/main/graph/badge.svg)](https://codecov.io/gh/Nikita-Filonov/axiom)
[![License](https://img.shields.io/github/license/Nikita-Filonov/axiom)](./LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/Nikita-Filonov/axiom?style=social)](https://github.com/Nikita-Filonov/axiom/stargazers)

_Made with ❤️ by [@NikitaFilonov](https://t.me/sound_right)_

---

## 📑 Table of Contents

- ✨ [About](#-about)
- 📦 [Installation](#-installation)
- 🚀 [Quick Start](#-quick-start)
- ❓ [Why Axiom?](#-why-axiom)
- 📘 [Documentation](#-documentation)

---

## ✨ About

**Axiom** is a modern testing framework for Go, built around **extensibility**, **composition**, and a clean, powerful
**runtime execution model**. It enhances Go’s standard `testing` package with capabilities normally found in mature
ecosystems like **pytest**, **JUnit5**, and **allure frameworks** — without hiding or replacing Go’s native tooling.

Axiom provides:

- **Composable test configuration** — merge global & local config seamlessly (`Runner` ↔ `Case`).
- **Powerful runtime engine** — deterministic lifecycle, step execution, subtests, and retries.
- **Hooks system** — before/after test, step, and subtest execution.
- **Plugins API** — extend framework behavior without touching the core.
- **Fixtures** — lazy-evaluated resources with automatic cleanup.
- **Metadata system** — tags, severity, labels, epics, features, stories.
- **Parallelization control** — opt-in at both runner & case granularity.

Axiom is not a DSL replacement — it’s **a composable execution engine** that sits on top of Go’s native testing stack
and supercharges it.

---

## 📦 Installation

```bash
go get github.com/Nikita-Filonov/axiom
```

For version pinning:

```bash
go get github.com/Nikita-Filonov/axiom@v0.3.0
```

---

## 🚀 Quick Start

This example demonstrates the core power of **Axiom**: fixtures, metadata, hooks, plugins, steps, and retryable
subtests — all working together seamlessly.

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
	"github.com/Nikita-Filonov/axiom/plugins/testtags"
)

// -----------------------------------------------------------------------------
// Fixtures
// -----------------------------------------------------------------------------

// DBFixture simulates a database connection with automatic teardown.
func DBFixture(cfg *axiom.Config) (any, func(), error) {
	// setup
	db := fmt.Sprintf("db-connection-%s", cfg.Case.ID)

	// teardown
	cleanup := func() {
		fmt.Printf("Closing %s\n", db)
	}

	return db, cleanup, nil
}

// UserFixture depends on the DB fixture and derives a user from it.
func UserFixture(cfg *axiom.Config) (any, func(), error) {
	db := axiom.GetFixture[string](cfg, "db")
	user := fmt.Sprintf("user-from-%s", db)
	return user, nil, nil
}

// -----------------------------------------------------------------------------
// Global Runner (shared test environment)
// -----------------------------------------------------------------------------

var runner = axiom.NewRunner(
	axiom.WithRunnerMeta(
		axiom.WithMetaEpic("authentication"),
		axiom.WithMetaFeature("login"),
		axiom.WithMetaSeverity(axiom.SeverityCritical),
	),

	// Plugins extend the runtime behavior:
	axiom.WithRunnerPlugins(
		testtags.Plugin(testtags.WithConfigInclude("smoke")), // filter by tag
		teststats.Plugin(teststats.NewStats()),               // metrics
		testallure.Plugin(),                                  // reporting
	),

	// Global retry configuration (per test case):
	axiom.WithRunnerRetry(
		axiom.WithRetryTimes(3),
		axiom.WithRetryDelay(15),
	),

	// Global fixtures:
	axiom.WithRunnerFixture("db", DBFixture),

	// Enable parallel execution across test cases:
	axiom.WithRunnerParallel(),
)

func TestUserLogin(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseName("user can login with valid credentials"),
		axiom.WithCaseMeta(
			axiom.WithMetaTag("smoke"),
			axiom.WithMetaStory("valid login"),
			axiom.WithMetaLabel("component", "auth-service"),
		),

		// Local fixtures:
		axiom.WithCaseFixture("user", UserFixture),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("prepare user", func() {
			user := axiom.GetFixture[string](cfg, "user")
			fmt.Println("Using:", user)
		})

		cfg.Step("validate response", func() {
			fmt.Println("Login OK")
		})
	})
}

```

---

## ❓ Why Axiom?

Go’s built-in testing package is intentionally minimal. This philosophy makes tests **simple**, **fast**, **and
approachable** — but it also leaves developers on their own when building **large**, **structured**, and **maintainable
test suites**.

When a project grows, you quickly run into hard limitations:

- no fixtures or resource lifecycle management
- no before/after hooks
- no retry mechanism for flaky operations
- no metadata (tags, severity, labels, layers, epics)
- no step model for readable reporting
- no plugin or extension architecture
- no way to compose configuration (global ↔ per-test)
- limited reporting capabilities
- and no clear path to build these features on top

Most teams end up reinventing these tools internally — often in incompatible ways.

### Axiom solves these real-world problems.

Instead of replacing Go’s testing ecosystem, Axiom extends it with a powerful execution engine:

- **Fixtures** — lazy-evaluated, cached resources with automatic cleanup
- **Hooks** — before/after test, step, and subtest
- **Retries** — deterministic flaky-test handling
- **Metadata** — tags, severity, labels, epics, features, stories
- **Step model** — structured execution with reporting support
- **Plugins** — clean extension mechanism for integrating tooling (Allure, metrics, filtering, etc.)
- **Composable configuration** — merge global runner settings with per-test overrides
- **Parallelization control** — opt-in, explicit, predictable

Axiom preserves the core spirit of Go — clarity, composability, explicit behavior —
while adding the missing building blocks needed for **serious test engineering**.

It’s not a DSL replacement.
It’s not a “magic” wrapper.

**Axiom is a test runtime engine** that unlocks capabilities traditionally found in frameworks like _pytest_, _JUnit5_,
and _Allure_, but implemented the Go way: simple, explicit, and pragmatic.

---

## 📘 Documentation

Axiom includes structured, minimal, and maintainable documentation for every core concept of the framework. See the
following folders:

- [./docs/usage](./docs/usage) — realistic end-to-end example of building a test framework with Axiom
- [./docs/philosophy](./docs/philosophy) — design principles and how Axiom fits into the Go testing ecosystem
- [./docs/runner](./docs/runner) — global execution environment, plugins, hooks, shared fixtures, retries
- [./docs/case](./docs/case) — declarative test definitions, metadata, parameters, per-test configuration
- [./docs/config](./docs/config) — merged runtime state for each test attempt (steps, wraps, hooks, fixtures, metadata)
- [./docs/runtime](./docs/runtime) — execution runtime: wraps, logs, artefacts, sinks
- [./docs/fixture](./docs/fixture) — lazy resource lifecycle, fixture dependencies, automatic cleanup
- [./docs/resource](./docs/resource) — runner-scoped shared resources, lifecycle, concurrency, deterministic teardown
- [./docs/meta](./docs/meta) — metadata: tags, labels, severity, epics, features, stories, layers
- [./docs/log](./docs/log) — structured logging via Runtime log sinks
- [./docs/assert](./docs/assert) — structured assertion events and runtime assert sinks
- [./docs/artefacts](./docs/artefact) — binary and structured test outputs
- [./docs/parallel](./docs/parallel) — parallel execution flags and merging behavior
- [./docs/retry](./docs/retry) — retry policies, isolated attempts, override rules
- [./docs/skip](./docs/skip) — static & dynamic skip rules with reasons
- [./docs/hooks](./docs/hooks) — lifecycle hooks for tests, steps, and subtests
- [./docs/params](./docs/params) — typed parameter injection for test cases
- [./docs/context](./docs/context) — structured global and per-test context values
- [./docs/plugins](./docs/plugins) — plugin system, built-in plugins, and guidelines for writing custom plugins
- [./docs/glossary](./docs/glossary) — definitions of all core Axiom concepts
