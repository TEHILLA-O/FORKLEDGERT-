# Privacy

Private source is treated as sensitive by default.

## Local only

`forkledger analyze --offline` never leaves the machine except for reading the local Git objects you already have.

The JSON report is written under `.forkledger/` unless `--output` is set. That directory is yours.

## Local analysis with metadata upload

Not implemented in v0.1. The intended payload is:

- repository identity
- remote URLs already present in Git config
- commit hashes
- patch-ids
- file paths
- function names from hunk headers
- advisory identifiers
- evidence records

Not intended:

- full file contents
- credentials
- hook scripts
- object packs beyond hashes already used as evidence

## Network queries in v0.1

If `--offline` is not set, ForkLedger may call:

- `api.github.com` with owner and repository names, to read whether GitHub already recorded a fork parent
- `api.osv.dev` with a commit hash or a GIT package coordinate

A GitHub token is optional. When present it is sent as a bearer token and never logged.

## Self hosted and hosted analysis

Designed, not shipped. The relational schema in `internal/storage/migrations` is the starting point for a self hosted control plane.

## What ForkLedger will not do

- Execute repository scripts unless you listed an exact argv in `forkledger.yaml`
- Follow `file:` or `ext::` Git transports
- Enable credential helpers while talking to Git
- Run hooks from the repository under analysis
