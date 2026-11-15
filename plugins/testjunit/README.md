# 📄 JUnit XML Plugin (`testjunit`)

## 📑 Table of Contents

- [Overview](#overview)
- [Documentation](#documentation)

## Overview

`testjunit` exports Axiom case attempts as JUnit XML.

- Automatic incremental file output or explicit export through a reporter.
- Parallel cases, retries, policy skips, and late cleanup status corrections.
- Suite names, counts, timing, and available failure messages.
- Separate Go module; production dependencies are Axiom core and the standard library.

## Documentation

- [Usage and migration](docs/USAGE.md) — installation, options, automatic output, manual export, and CI.
- [Lifecycle and concurrency](docs/LIFECYCLE.md) — publication, shared reporters, and output ownership.
- [XML format](docs/FORMAT.md) — fields, retry semantics, and limitations.
- [Runnable example](example_test.go) — explicit reporter export.
