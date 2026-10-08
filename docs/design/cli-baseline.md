# CLI baselines

**Status: implemented, not yet released.**

## Decision

The CLI reports only findings absent from a baseline generated from the base revision on each run:

```sh
aexlint check --write-baseline <file>   # in a checkout of the base
aexlint check --baseline <file>         # in the branch
```

This replaces the committed `.aexlintsilent` file, `--silence` and `AEXLINT_SILENT_FILE`, which were never pushed or published. The matching engine is unchanged: entries are file, rule and exact message without typed help text, counted per file. Paths are relative to the directory aexlint runs from.

Generating the baseline per check removes the committed file's problems: stale entries after fixes, path-scoped partial updates, and several worktrees overwriting one shared file. `--write-baseline` therefore always overwrites the whole file.

## Rejected alternative: diff filter

`aexlint check --diff <patch>` would report only diagnostics whose span overlaps changed lines. It needs no second lint run or base checkout, but misses findings whose cause lies outside their span. Comparing whole results against the base catches those, because a finding is identified by its file, rule and message rather than where its cause was edited.

### Findings a diff filter would miss

A diff filter assumes a finding's cause lies within its span. That holds for some rules and not others.

**Cause within the span.** A change that creates or worsens the finding edits the reported region.

| Rule                        | Span                        |
| --------------------------- | --------------------------- |
| `max-expression-depth`      | outermost over-limit call   |
| `max-expression-complexity` | outermost over-limit region |
| `max-decision-depth`        | outermost over-limit group  |
| `no-else-if`                | `else if`                   |
| `no-multiline-condition`    | condition                   |
| `max-call-assembly`         | argument list               |

`max-call-assembly` has two small gaps: the hook exclusion depends on the callee, which can sit on another line, and moving a call from module level into a function makes it eligible without necessarily editing it.

**Cause elsewhere in the same file.**

| Rule                            | Span                | Unchanged-line causes                                                                                                                                       |
| ------------------------------- | ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `max-held-context`              | decision head       | Wrapping code in a branch, loop or callback deepens the decisions inside. Deleting a reported decision moves the report to a previously suppressed sibling. |
| `prefer-domain-predicate`       | classification      | Adding a statement to a named predicate or validation-only function removes the name that exempted its classifications.                                     |
| `no-forgeable-validated-result` | operation name      | Adding a rejection to the body, or changing the result type, reports on the untouched name.                                                                 |
| `no-caller-enforced-invariant`  | handed-off argument | Adding a check before an existing hand-off reports on the untouched argument.                                                                               |

Wrapping only escapes `max-held-context` when lines are not reindented: without a formatter, or with a whitespace-ignoring diff (`-w`, `-b`).

**Cause in another file or in the program.**

| Rule                                | Unchanged-line causes                                                                                                                                           |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `prefer-truthy-presence-check`      | A type changed elsewhere makes an existing comparison eligible. Disabling `strictNullChecks` reports `strictNullChecksRequired` on untouched first lines.       |
| `require-interface-implementations` | Removing an implementing class or `implements` clause reports on the untouched interface name.                                                                  |
| `no-caller-enforced-invariant`      | Removing a check from the callee reports on every untouched caller argument.                                                                                    |
| `no-forgeable-validated-result`     | A project helper gaining a rejection, or a result class becoming a plain type, reports on untouched operation names.                                            |
| `max-interpretation-spread`         | A file adding a check is reported, but the other files crossing the limit are not. Type changes that merge or qualify vocabularies report only untouched files. |
| `max-projection-spread`             | As above: the new copy is reported, existing copies are not. Ownership type changes report only untouched files.                                                |

The spread rules are partly covered because the new participant is reported once. The other typed rules can miss the event entirely, including the case they exist for: weakening an owner's validation.

## Remaining limitations

- Two lint runs; typed rules need the base's dependencies installed and run a second program check.
- Renamed or moved files report all their findings again.
- A message change, for example a larger measured depth, reports the finding again. This is intended for findings that get worse, but also happens when an aexlint upgrade rewords messages, so both runs should use the same version.
- Messages must therefore describe the flagged code, not the rest of the program. Baselines ignore the `help:` text typed rules append, and the spread rules keep their project-wide file counts there: a file joining an over-limit spread is reported once, in that file, instead of resurfacing every untouched participant. Crossing the limit still reports every participant, since each one becomes a new finding.
- Identity is per file: the same message twice in one file is counted, not located, so swapping which of two identical findings exists goes unnoticed.
