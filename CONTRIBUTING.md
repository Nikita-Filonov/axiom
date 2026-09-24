# Contributing to Axiom

Axiom extends Go's `testing` package. Keep it small, explicit, predictable, and
compatible with ordinary `go test`. These rules apply to the core module, every
plugin module, tests, examples, and documentation.

## Before any pull request

Every change, including a bug fix, refactor, plugin, dependency, test, or
documentation update, starts with a written issue or discussion with the
project author or a designated maintainer.

1. Describe the observed problem, evidence or reproduction, impact, and desired
   behavior. Establish the cause before proposing a patch to a symptom.
2. Discuss the scope, approach, alternatives, and compatibility impact. Wait
   for explicit maintainer agreement on the problem and proposed direction
   before implementing or opening a PR.
3. Link that discussion in the PR. A PR without documented prior agreement is
   closed without an implementation review.

## Project principles

- Prefer the simplest design that solves the demonstrated problem. Add a new
  abstraction, option, dependency, or public API only when an existing one
  cannot express the behavior clearly.
- Follow [Effective Go](https://go.dev/doc/effective_go) and
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Use
  `gofmt`, clear names, explicit errors, and small, cohesive packages.
- Match the existing field-order convention: within a group of independent
  struct fields, put shorter names before longer ones. Preserve semantic
  grouping and do not reorder public fields solely for style.
- Keep test discovery and execution in Go's `testing` package. Do not add a DSL,
  implicit registration, hidden global state, or new uses of `reflect`. Existing
  reflection for suite validation and explanation diagnostics is not a precedent
  for extending it.
- Preserve opt-in modularity. Users can build tests with only `Runner`, `Case`,
  and `Config`; fixtures, resources, suites, hooks, retries, metadata,
  parallelism, plugins, and integrations are optional. A new capability must
  not require unrelated setup or change behavior when it is unused.
- Production files in the root `axiom` package may import only the Go standard
  library. Third-party packages are allowed in core tests; integrations that
  need them belong in separately versioned plugin modules.
- Preserve backward compatibility by default. A breaking change requires a
  compelling documented reason, maintainer agreement before implementation,
  a migration path, updated documentation and tests, and the appropriate major
  version.

## Change checklist

### Design and compatibility

- [ ] The change has one clear purpose and contains no unrelated refactor or
      speculative API.
- [ ] The basic `Runner`/`Case`/`Config` flow still works without configuring
      optional features or importing plugins. New features remain opt-in.
- [ ] Existing import paths, exported names and signatures, struct fields,
      defaults, zero values, minimum Go version, and error or panic behavior
      remain compatible.
- [ ] Existing event names and payloads, hook and wrapper order, retry and skip
      behavior, parallel scheduling, and fixture or resource lifetimes remain
      compatible unless a breaking change was explicitly approved.
- [ ] Ownership, cleanup, cancellation, and concurrency behavior are explicit.
      Failure, panic, skip, retry, and partial setup do not leak resources.
- [ ] A change to the core has been checked against every plugin module; a
      plugin change does not introduce a dependency from the core to that plugin.
- [ ] The core has no new third-party production imports. Core test dependencies
      are used only from `_test.go` files. Integration dependencies are confined
      to the plugin module that needs them, with tidy `go.mod` and `go.sum` files.

### Tests

- [ ] Every production package in every module has **100.0% statement coverage**
      from unit and/or integration tests. Coverage is measured per package, not
      averaged across the repository. Existing gaps are not an exemption.
- [ ] Tests verify observable behavior, not just that code executes. Cover
      relevant success, failure, boundary, nil or zero-value, type mismatch,
      panic or Goexit, skip, retry, and cancellation cases. Statement coverage
      alone is insufficient.
- [ ] Lifecycle tests verify hook and wrapper order, fixture isolation per
      attempt, resource reuse per runner, and cleanup exactly once in reverse
      setup order, including on failures and partial setup.
- [ ] A bug fix has a regression test that fails before the fix. A public API
      change has tests showing how a caller uses it.
- [ ] Shared state, parallel execution, and cancellation paths have tests for
      concurrent use; run the affected modules with `go test -race ./...`.
- [ ] Tests run reliably in CI without private secrets or manually managed
      services. Provision and clean up required dependencies; avoid sleeps as
      synchronization and reliance on test order.
- [ ] Tests pass in the core and all plugin modules affected by the change.
      A passing root-module test run does not cover the nested plugin modules.

### Plugins

- [ ] Installation only configures a `Config`; it does not run test actions or
      perform irreversible external work.
- [ ] The plugin is deterministic and safe when applied to a planning Config
      and again to each attempt Config. Retries do not accidentally accumulate
      state, callbacks, or side effects.
- [ ] Runner and Case plugin precedence, wrapper nesting, hook timing, and
      cleanup behavior are tested where relevant.
- [ ] Mutable state shared across attempts or cases is synchronized. A plugin
      documents what it observes, changes, owns, and leaves to the caller.
- [ ] The plugin's own module, tests, versioning, and README remain independent
      of unrelated plugins.

### Documentation

- [ ] Every new or changed exported declaration has accurate GoDoc. Document
      defaults, zero values, errors, panics, ownership, concurrency, ordering,
      and lifecycle rules wherever a caller needs them.
- [ ] User-facing behavior changes are reflected in the relevant `docs/` page,
      plugin README, and example. Remove or correct descriptions that no longer
      match the implementation.
- [ ] Examples compile and demonstrate the actual behavior. Comments explain
      non-obvious decisions and invariants instead of restating the code.

### Submission and review

- [ ] The PR links the prior discussion and the maintainer's agreement on the
      problem and proposed direction.
- [ ] The PR explains the problem, chosen approach, compatibility impact,
      affected modules, tests run, and documentation changed.
- [ ] Formatting, vet, lint, tests, and coverage checks pass for the affected
      modules before requesting review.
- [ ] Blocking review comments identify a violated rule or concrete risk and
      the change needed to resolve it. Style preferences outside these rules
      are marked as suggestions.

If a rule cannot be met, discuss a change to the rule before submitting the
implementation. Do not rely on an undocumented exception.

## Local verification

Each plugin has its own `go.mod`. Run checks from each module directory:

```sh
for module in . plugins/*; do
  test -f "$module/go.mod" || continue
  (cd "$module" && go vet ./... && go test -cover ./...)
done
```

Run `gofmt` and the repository's linter before submitting. For concurrency
changes, run `go test -race ./...` in each affected module. The coverage output
for every production package must be `100.0%`.
