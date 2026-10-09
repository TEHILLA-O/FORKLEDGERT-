# ForkLedger

ForkLedger is the control plane for modified upstream software.

It discovers repositories that were copied, forked, vendored, patched, or otherwise diverged from an upstream project. It tracks the private patches that remain, finds upstream changes that never reached the fork, and helps a team decide whether to retain, upstream, rebase, replace, or retire the work.

It is not another Git dashboard. It is not another vulnerability scanner. It is not an AI wrapper. Git remains the source of truth for history. Security databases remain the source of truth for advisory metadata. ForkLedger interprets those facts in the context of a modified tree.

## First run

```
forkledger analyze .
```

This is a real run against a clone of `mattn/go-isatty`, rolled back five commits, with `origin` renamed to `upstream` and one local patch on top (offline mode):

```
ForkLedger Analysis

Detected upstream                github.com/mattn/go-isatty
Baseline                         0.0.20
Current upstream                 v0.0.24
Fork point                       433c12b4c9fa
Ahead / behind                   1 / 5

Local modifications              1 patch
Changed lines                    1
Likely upstream equivalents      0 patches
Potentially obsolete             0 patches
Potential conflicts              1 patch

Security
  Relevant upstream fixes missing 0
  Findings needing investigation 0

Suggested lifecycle action       RETAIN
Confidence                       LOW (35). This is an estimate from evidence, not a guarantee.

Fork health 89
  upstream drift                 -1. 5 commits behind upstream
  conflicting patches            -2. 1 patches touch files also changed upstream
  ownership                      -8. no owner is recorded in forkledger.yaml
```

Confidence is low here because there is only one tiny patch and no owner in `forkledger.yaml`. Exact numbers depend on evidence. ForkLedger will not invent confidence.

## Why this matters

Most companies run patched copies of open source somewhere: a vendored library, an old fork of an SDK, an internal build of a database. Those patches quietly go stale, upstream security fixes never land, and nobody remembers why the fork exists. ForkLedger turns that into a list of facts a team can act on: what was changed, what upstream already fixed, what will conflict, and who owns it.

## What v0.1 does

- Inspects a Git repository and its remotes
- Detects upstream candidates with evidence and confidence
- Estimates the fork point from `git merge-base`
- Inventories local patches, including Git patch-ids
- Finds likely upstream equivalents
- Flags files changed on both sides of the fork
- Associates OSV advisories with the fork when network access is allowed
- Writes a machine readable report and a terminal report
- Exits with codes that CI can act on

`forkledger serve` exposes a local HTTP API for stored scans. The web app at `apps/web` reads `/v1/attention` when that API is running.

`--fetch` updates remote-tracking refs only. It does not checkout the working tree.

`forkledger upgrade` replays local patches with cherry-pick inside an isolated worktree. Build and test commands run there only if `FORKLEDGER_ALLOW_BUILD=1` is set.

## Install

Go 1.22 or newer and Git 2.38 or newer are required.

```
git clone https://github.com/TEHILLA-O/fork_ledger.git forkledger
cd forkledger
go build -o bin/forkledger ./cmd/forkledger
go test ./...
```

Or install straight from the module path:

```
go install github.com/TEHILLA-O/fork_ledger/cmd/forkledger@latest
```

## Commands

```
forkledger analyze [path]
forkledger upstream
forkledger patches
forkledger security
forkledger upgrade <ref>
forkledger report [file]
forkledger policy check
forkledger explain
forkledger doctor
forkledger serve [:8080]
forkledger ingest [report.json]
forkledger discover <org>
forkledger sbom
forkledger vex
forkledger version
```

Useful flags (put them before the path, for example `forkledger analyze --json --offline .`):

```
--json
--quiet
--verbose
--debug
--config PATH
--output PATH
--offline
--path PATH
```

## Configuration

Place `forkledger.yaml` at the repository root. See `examples/forkledger.yaml`.

Build and test commands are never inferred from repository scripts. They must be listed explicitly. ForkLedger will not execute hooks or helper programs from the repository under analysis.

## Privacy

Default analysis is local. Repository contents stay on the machine unless you opt into a network query.

`--offline` disables GitHub metadata and OSV lookups.

When network is enabled, ForkLedger may send:

- Public GitHub owner and repository names, to read fork parent metadata
- Commit hashes and package coordinates, to query OSV

It does not upload source trees. See `docs/privacy.md`.

## Status

This is v0.1. Local analysis, isolated upgrade replay, the local scan API and the file store work today. The web control plane in `apps/web` reads from the local API and is early. The GitHub App webhook handler and organisation discovery exist in code but are not production tested yet. Known gaps are written down in [FAILURES.md](FAILURES.md).

## Stack

Go 1.22 for the CLI and analysis engine, Git plumbing through `git` subprocesses, a JSON file store for scans (with a relational schema drafted for the future control plane), OSV and GHSA for advisories, CycloneDX or SPDX SBOM and OpenVEX output, and a small Next.js app for the attention queue. CI runs `go test` and `go vet` on every push, and `action.yml` lets other repositories run ForkLedger in a pull request.

## License

Apache License 2.0
