# CLI baselines

The CLI reports only findings missing from a baseline (`--write-baseline` to record, `--baseline` to check). A finding is its file, rule and message, without help text, counted per file. The plugin records and filters findings as rules report them, before Oxlint applies disable comments, so both runs count the same findings.

A baseline can be committed, so a project adopts the rules gradually, or recorded on the base revision for every check. A committed baseline holds stale entries until they are recorded again; the base-revision recipe never does, but runs twice per check.

`--write-baseline` with paths replaces only the entries of files under those paths. Accepting findings in one area then touches only that area's lines, a branch behind its target keeps the target's entries for other files, and entries sorted by file keep merge conflicts to the files both branches recorded. Without paths it replaces the whole file, which also drops entries of files no longer linted.

A diff filter (report only findings on changed lines) was rejected. Several rules report a place whose cause is elsewhere: a spread rule reports every file when one more joins, and a type change in one file creates findings in its callers. Comparing whole results catches those, as long as the whole project is checked.
