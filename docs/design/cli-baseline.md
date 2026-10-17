# CLI baselines

The CLI reports only findings missing from a baseline recorded on the base revision for each check (`--write-baseline` on the base, `--baseline` on the branch). A finding is its file, rule and message, without help text, counted per file. The plugin records and filters findings as rules report them, before Oxlint applies disable comments, so both runs count the same findings.

The baseline is regenerated for every check rather than committed, so it never holds stale entries, partial updates, or conflicts between worktrees.

A diff filter (report only findings on changed lines) was rejected. Several rules report a place whose cause is elsewhere: a spread rule reports every file when one more joins, and a type change in one file creates findings in its callers. Comparing whole results with the base catches those.
