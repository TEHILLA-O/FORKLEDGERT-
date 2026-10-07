# CLI

The CLI is the product surface for v0.1.

## Output modes

Default output is a terminal report. `--json` writes the same facts as JSON. `--quiet` keeps stdout for the report and suppresses diagnostics. `--verbose` and `--debug` go to stderr.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Completed without blocking findings |
| 1 | Findings present |
| 2 | Usage error |
| 3 | Not a Git repository |
| 4 | Analysis failed |
| 5 | Invalid configuration |
| 20 | Security findings requiring attention |
| 30 | Policy violations |

CI should treat 0 as green, 20 and 30 as fail, and 1 as a team choice.

## `forkledger upgrade`

The working tree is not modified. Conflict prediction uses `git merge-tree` and the patch ledger. Build and test commands listed in `forkledger.yaml` are recorded, not executed, unless a later isolated runner is enabled with an explicit environment flag.

Confidence is an estimate. The report says so.
