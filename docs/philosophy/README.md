# 📘 Philosophy

---

## 📑 Table of Contents

- [Overview](#overview)
- [What Axiom is not](#what-axiom-is-_not_)
- [Where Axiom fits](#where-axiom-fits)
- [Axiom and other testing tools](#axiom-and-other-testing-tools)
- [Core idea](#core-idea)
- [Design principles](#design-principles)
- [Summary](#summary)

---

## Overview

Axiom is **not a replacement** for Go’s `testing` package. It is also not an **assertion or mocking framework**.

Axiom is a **test execution engine** that extends Go’s native testing model with structured lifecycle, composition, and
extensibility — without modifying or hiding the underlying `testing` package.

---

## What Axiom is _not_

Axiom does not:

- replace `testing.T`
- introduce a DSL (`describe`, `it`, `expect`, etc.)
- provide assertions or mocks
- modify or wrap Go’s testing runtime
- change how tests are discovered or executed by `go test`

If you are looking for a DSL-driven test framework or a replacement for Go’s testing philosophy, Axiom is **not** that.

> **If you like Go’s testing philosophy — Axiom is for you.**
> **If you want a DSL — it’s not.**

---

## Where Axiom fits

Axiom operates at the **execution layer** of a test stack:

```text
┌──────────────────────────────┐
│ Assertions & Mocks           │  ← testify, require, gomock
├──────────────────────────────┤
│ Test Logic & Steps           │  ← your test code
├──────────────────────────────┤
│ Execution Model              │  ← Axiom
│  - fixtures                  │
│  - retries                   │
│  - hooks                     │
│  - metadata                  │
│  - plugins                   │
├──────────────────────────────┤
│ Go testing package           │  ← testing.T
└──────────────────────────────┘
```

- Go’s `testing` package remains the foundation
- Axiom builds **on top of it**, not around it
- Assertion and mocking libraries work **unchanged**

---

## Axiom and other testing tools

Axiom is designed to work **alongside** existing Go testing tools:

- `testify` / `require` — assertions
- `gomock`, `testify/mock` — mocks
- standard `testing` helpers

Example:

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runner = axiom.NewRunner(
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
)

type LoginResponse struct {
	OK     bool
	Status int
}

func login() LoginResponse {
	// Simulated login call
	return LoginResponse{
		OK:     true,
		Status: 200,
	}
}

func TestUserLogin(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseName("user can login"),
	)

	// Runner is assumed to be defined elsewhere (platform / domain runner)
	runner.RunCase(t, c, func(cfg *axiom.Config) {

		cfg.Step("validate response", func() {
			resp := login()

			// Assertions are handled by testify
			require.True(cfg.SubT, resp.OK)
			assert.Equal(cfg.SubT, 200, resp.Status)
		})
	})
}

```

Axiom does not interfere with these tools and does not attempt to replace them.

---

## Core idea

Axiom focuses on problems that Go’s testing package intentionally leaves open:

- test lifecycle and execution model
- resource management (fixtures)
- retries and flaky tests
- metadata and test classification
- hooks and extensibility
- predictable composition of global and local configuration

Instead of reinventing these concepts per project, Axiom provides a **composable runtime engine** that teams can build
upon.

---

## Design principles

- explicit over implicit
- composition over inheritance
- no hidden magic
- no DSL
- no reflection-driven execution
- full compatibility with `go test`

Axiom preserves the spirit of Go testing: simple, explicit, and predictable — while providing the missing building
blocks needed for large, structured test suites.

---

## Summary

- Axiom extends Go testing — it does not replace it
- Axiom complements assertion and mocking libraries
- Axiom provides execution structure, not test logic
- Axiom is designed for teams building test platforms, not just tests
