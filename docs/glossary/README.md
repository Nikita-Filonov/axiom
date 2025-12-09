# 📘 Glossary

Core concepts used throughout Axiom. Each term is intentionally defined briefly. Detailed explanations live in dedicated
documents ([/docs/case](./../../docs/case), [/docs/runner](./../../docs/runner), etc.).

## Key Entities

| Concept     | Description                                                                                                                                                              |
|-------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Case**    | Declarative definition of a single test: name, metadata, fixtures, retry policy, parameters, plugins, and execution behavior.                                            |
| **Runner**  | Global test environment. Merges its configuration with a Case and executes it. Applies plugins, hooks, fixtures, retry logic, and parallelization settings.              |
| **Config**  | Runtime object produced per test execution. Represents merged Runner + Case configuration and provides access to fixtures, metadata, steps, hooks, plugins, and context. |
| **Plugin**  | Function that modifies the runtime `Config`. Used for test filtering, reporting, metrics, instrumentation, wrappers, and custom behavior.                                |
| **Fixture** | Lazily evaluated resource (e.g., DB connection). Created on first request, cached for the test duration, and cleaned up automatically.                                   |
| **Meta**    | Test metadata: tags, epic, feature, severity, labels, stories, layers. Used for filtering, reporting, organization, and CI integration.                                  |
| **Retry**   | Configuration controlling how many times a test may re-run and the delay between attempts.                                                                               |
| **Skip**    | Declarative mechanism to mark tests as skipped (static or dynamic).                                                                                                      |
| **Hooks**   | Lifecycle callbacks: before/after test, subtest, or step. Used for logging, reporting, tracing, metrics, and instrumentation.                                            |
| **Step**    | A named operation inside a test. Steps participate in reporting and trigger step-level hooks and wrappers.                                                               |
| **Wraps**   | Middleware around tests or steps (TestWrap / StepWrap). Used by plugins such as Allure or logging tools.                                                                 |
| **Context** | Structured contextual values scoped to Runner or Case. Useful for passing environment data, request IDs, or framework-level information.                                 |
