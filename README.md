# ForkLedger

ForkLedger is the control plane for modified upstream software.

It discovers repositories that were copied, forked, vendored, patched, or otherwise diverged from an upstream project. It tracks the private patches that remain, finds upstream changes that never reached the fork, and helps a team decide whether to retain, upstream, rebase, replace, or retire the work.

It is not another Git dashboard. It is not another vulnerability scanner. It is not an AI wrapper. Git remains the source of truth for history. Security databases remain the source of truth for advisory metadata. ForkLedger interprets those facts in the context of a modified tree.

## First working experience

```
forkledger analyze .
```

The command inspects the current repository and prints a report similar to this:

```
ForkLedger Analysis

Detected upstream                 redis/redis
Baseline                          7.2.4
Current upstream                  8.2.1
Local modifications               43 patches
Changed lines                     7241
Likely upstream equivalents       9 patches
Potentially obsolete              6 patches
Potential conflicts               11 patches

Security
  Relevant upstream fixes missing 2
  Findings needing investigation  1

Suggested lifecycle action        REBASE
Confidence                        HIGH (89). This is an estimate from evidence, not a guarantee.
```

Exact numbers depend on evidence. ForkLedger will not invent confidence.

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

The hosted product, GitHub App, and organisation-wide fork graph are designed. They are not required for the CLI to be useful.

## Install

Go 1.22 or newer and Git 2.38 or newer are required.

```
git clone https://github.com/forkledger/forkledger
cd forkledger
go build -o bin/forkledger ./cmd/forkledger
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
forkledger version
```

Useful flags:

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

This is v0.1. The intelligence engine is the product. Later releases add upgrade worktree simulation, organisation discovery, the web control plane, and a GitHub App with minimal permissions.

## License

Apache License 2.0
