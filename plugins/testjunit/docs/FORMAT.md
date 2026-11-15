# 📄 JUnit XML Format

## 📑 Table of Contents

- [Report contents](#report-contents)
- [Limitations](#limitations)

## Report contents

### XML structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="2" failures="1" skipped="0" time="0.300000">
  <testsuite name="auth" tests="2" failures="1" skipped="0" time="0.300000">
    <testcase classname="auth" name="TestLogin/succeeds" time="0.100000"></testcase>
    <testcase classname="auth" name="TestLogin/rejected" time="0.200000">
      <failure message="test failed">test failed</failure>
    </testcase>
  </testsuite>
</testsuites>
```

| Element or attribute           | Value                                                                   |
|--------------------------------|-------------------------------------------------------------------------|
| `<testsuites>`                 | Report root with totals across all suites.                              |
| `<testsuite>`                  | One group per suite name.                                               |
| `<testcase>`                   | One executed or policy-skipped attempt.                                 |
| Testcase `name`                | Full Go test name, distinguishing retry attempts.                       |
| Testcase `classname`           | Configured suite name.                                                  |
| `tests`, `failures`, `skipped` | Counts on the root and each suite.                                      |
| `time`                         | Attempt duration in seconds; root and suite values sum those durations. |

- Suites and their cases follow attempt start order.
- XML control characters in names and messages are replaced. Attempts with the
  same resulting suite name are grouped together.
- An empty reporter produces a `<testsuites>` document with zero totals and no
  child suites.

### Outcomes

| Attempt outcome   | XML result                                                                                           |
|-------------------|------------------------------------------------------------------------------------------------------|
| Passed            | A testcase without `<failure>` or `<skipped>`.                                                       |
| Failed            | `<failure>` with an available Axiom lifecycle message, or `test failed` for direct Go test failures. |
| Skipped           | `<skipped>` with the policy skip reason, or `skipped` when no reason is available.                   |

A lifecycle failure takes precedence over a skip, even when its message is empty.
A failed attempt followed by a successful retry produces both results; the
previous failure remains in the report.

## Limitations

- **Failure text:** Go does not expose messages passed directly to `t.Error` or
  `t.Fatal` to plugins. Those failures use the generic `test failed` message.
- **Logs:** stdout, stderr, and arbitrary test logs are not captured.
- **Source locations:** reports have no `file` attribute because Axiom does not
  record a case's source file. CI features requiring that attribute are unavailable.
- **Test selection:** tests that never enter Axiom and cases excluded by
  `go test -run` are absent from the report.
- **Process crashes:** automatic output retains the last successful publication; an interrupted attempt may be absent
  and late status corrections may not have run. See [partial runs](LIFECYCLE.md#partial-runs).

