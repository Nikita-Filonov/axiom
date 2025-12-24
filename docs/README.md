# 📘 Axiom Documentation

This directory contains structured, minimal, and maintainable documentation for all core concepts of the Axiom testing
framework. Each subfolder provides focused reference material and examples.

---

## 📂 Documentation Index

- [./usage](./usage) — realistic end-to-end example of building a test framework with Axiom
- [./philosophy](./philosophy) — design principles and how Axiom fits into the Go testing ecosystem
- [./runner](./runner) — global execution environment, hooks, shared fixtures, retries
- [./case](./case) — declarative test definitions, metadata, parameters, per-test configuration
- [./config](./config) — merged runtime state for each test attempt (steps, wraps, hooks, fixtures, metadata)
- [./runtime](./runtime) — execution runtime: wraps, logs, artefacts, sinks
- [./fixture](./fixture) — lazy resource lifecycle, dependency model, cleanup
- [./meta](./meta) — tags, labels, severity, epics, features, stories, layers
- [./log](./log) — structured logging via Runtime log sinks
- [./assert](./assert) — structured assertion events and runtime assert sinks
- [./artefact](./artefact) — binary and structured test outputs
- [./parallel](./parallel) — parallel execution flags (Runner-level & Case-level overrides)
- [./retry](./retry) — retry policies, overrides, and isolated execution attempts
- [./skip](./skip) — static and dynamic skip rules with reasons
- [./hooks](./hooks) — lifecycle hooks for tests, steps, and subtests
- [./params](./params) — typed parameter injection for tests
- [./context](./context) — structured global and per-test context values
- [./plugins](./plugins) — plugin architecture, mutation model, extension guidelines
- [./glossary](./glossary) — concise definitions of all Axiom concepts
