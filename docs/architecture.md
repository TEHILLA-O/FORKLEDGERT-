# Architecture

ForkLedger is a monorepo. The first useful product is a local CLI. Later pieces reuse the same analysis types rather than inventing a second model.

## Layout

```
cmd/forkledger          portable command line entry point
internal/git            Git execution, parsing, and safety
internal/upstream       candidate detection and fork point estimation
internal/patches        patch ledger, classification, equivalence
internal/scanner        analysis orchestration, health, debt, lifecycle
internal/security       OSV adapter and applicability interpretation
internal/policy         constrained rule matching
internal/upgrade        isolated upgrade simulation
internal/report         JSON and terminal reports
internal/fingerprint    pluggable signals for detached copies
internal/storage        relational schema for the future control plane
packages/contracts      shared JSON schemas
apps/web                report viewer for a single analysis
docs                    operator and researcher documentation
```

Python is reserved for later semantic comparison. It is not on the v0.1 path.

PostgreSQL is the planned system of record for organisation-wide graphs. The CLI does not need it. Analysis results are files.

Redis, Kafka, Temporal, Elasticsearch, and graph databases are not part of v0.1. Complexity has to be earned.

## Data flow

1. The operator runs `forkledger analyze`.
2. Git is invoked through a guarded executor. No shell. No repository hooks.
3. Remotes, configuration, package metadata, and optional GitHub fork records become upstream candidates.
4. If a candidate is strong enough, `git merge-base` estimates the fork point.
5. Local commits after that point become patch records. `git patch-id --stable` identifies them.
6. Upstream commits after that point are compared using patch-id, `git cherry`, file overlap, and function names from hunk headers.
7. `git merge-tree` and dual file lists predict upgrade friction.
8. OSV is queried by commit and, when known, by GIT package coordinates.
9. Health, maintenance debt, and a lifecycle recommendation are computed from those facts.
10. A JSON report and a terminal report are written. Exit codes follow the findings.

## Evidence

Every inferred conclusion stores evidence. Heuristics contribute signals. They do not become facts by sitting in a struct.

See `docs/evidence.md`.

## Future hosted path

The GitHub App will use installation tokens and a short event list. Webhook signatures, idempotency, and rate limit handling belong there, not in the CLI.

Organisations should be able to run a local agent that uploads derived evidence instead of source. The cloud is optional.
