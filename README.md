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
- 🧪 [IDE Support](#-ide-support)
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

---

## 🚀 Quick Start

This example shows the smallest useful Axiom flow: define a runner, describe a case, and execute steps inside standard 
`go test`.

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestUserLogin(t *testing.T) {
	runner := axiom.NewRunner()

	c := axiom.NewCase(
		axiom.WithCaseName("user can login"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		var token string

		cfg.Step("submit credentials", func() {
			token = "access-token"
		})

		cfg.Step("check login result", func() {
			if token == "" {
				t.Fatal("expected access token")
			}
		})
	})
}

```

For a full project layout with shared runners, fixtures, metadata, and domain-specific test organization, see
[./docs/usage](./docs/usage).

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

## 🧪 IDE Support

**GoLand / IntelliJ IDEA Ultimate** — install the official
[**Axiom Test Runner** plugin](https://plugins.jetbrains.com/plugin/32606-axiom-test-runner)
([source](https://github.com/Nikita-Filonov/axiom-jetbrains)) from JetBrains Marketplace to get:

- 🟢 Green gutter icon on every `TestXxx` method of an Axiom suite.
- ▶️ Full native menu — Run, Debug, Run with Coverage, Profile (CPU/Memory/Blocking).
- 🌳 Results in the standard *Test Results* tool window.
- ⚡ Zero configuration — works with any struct passed to `axiom.NewSuite`
  or `axiom.NewSuiteFactory`, regardless of what it embeds.

```bash
# Or install directly from the IDE:
#   Settings → Plugins → Marketplace → search "Axiom Test Runner"
```

---

## 📘 Documentation

Axiom includes structured, minimal, and maintainable documentation for every core concept of the framework. See the
following folders:

- [./docs/usage](./docs/usage) — realistic end-to-end example of building a test framework with Axiom
- [./docs/philosophy](./docs/philosophy) — design principles and how Axiom fits into the Go testing ecosystem
- [./docs/runner](./docs/runner) — global execution environment, plugins, hooks, shared fixtures, retries
- [./docs/suite](./docs/suite) — optional execution boundary for grouped tests, shared runners, resources, and lifecycle
- [./docs/package](./docs/package) — `TestMain` lifecycle boundary for runners shared across many top-level `TestXxx` functions
- [./docs/case](./docs/case) — declarative test definitions, metadata, parameters, per-test configuration
- [./docs/config](./docs/config) — merged runtime state for each test attempt (steps, wraps, hooks, fixtures, metadata)
- [./docs/local](./docs/local) — per-attempt typed local state stored on Config
- [./docs/toolset](./docs/toolset) — typed helper bundles built into Local and consumed as cfg.Tools
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
