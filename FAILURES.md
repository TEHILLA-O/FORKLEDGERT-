# Failure modes, fixes, and results

Honest engineering notes for this project. Nothing here is invented for polish.

## What can go wrong

- **Wrong upstream.** A repository with several remotes, or a vendored copy with no remote at all, can point detection at the wrong project. Mitigation: every upstream candidate carries evidence and a confidence score, and `forkledger.yaml` can pin the upstream explicitly.
- **False "already upstream" matches.** Patch equivalence uses Git patch-ids first, then fuzzier signals (MinHash and SimHash fingerprints, Go AST comparison). Fuzzy matches can be wrong. Mitigation: likely and possible equivalents are reported separately and never auto-deleted.
- **Running untrusted code.** Analysing someone else's repository must not execute their hooks or scripts. Mitigation: build and test commands are never inferred, `forkledger upgrade` replays patches in an isolated worktree, and build or test steps only run when `FORKLEDGER_ALLOW_BUILD=1` is set.
- **Leaking private code.** Mitigation: analysis is local by default, `--offline` disables GitHub and OSV lookups, and only owner/repo names, commit hashes and package coordinates are ever sent (`docs/privacy.md`).

## What went wrong

1. **CI went red on the second commit.** `1176ddf` (isolated upgrade replay, richer equivalence, local scan API) changed patch inventory and broke `TestAnalyzeForkWorld` in `internal/scanner`: it expects 3 local patches in the fixture fork and now finds 2. Every other test package passes. **Not fixed yet.**
2. **Flags after the path are ignored.** `forkledger analyze . --json` prints the terminal report, while `forkledger analyze --json .` prints JSON. This is standard Go `flag` parsing stopping at the first positional argument. **Documented in the README, not changed in code yet.**
3. **Install instructions pointed at a repository that does not exist.** The Go module path is `github.com/forkledger/forkledger`, but the code lives at `TEHILLA-O/FORKLEDGERT-`. The README now clones the real repository. `go install` by module path will not work until the module path and the repository match.

## How it was resolved

- README install and flag order fixed (docs only).
- The failing integration test is left visible on purpose so the equivalence change gets a proper look instead of a loosened assertion.

## Results

- `go build ./cmd/forkledger` succeeds on Go 1.24 (module targets Go 1.22).
- `go test ./...`: 11 of 12 test packages pass; `internal/scanner` fails as described above.
- Real smoke run (2026-10-09) on a clone of `mattn/go-isatty` five commits behind, with one local patch, offline: upstream detected correctly, fork point found, 1 patch and 1 potential conflict reported, suggested action RETAIN at LOW (35) confidence, fork health 89. Output is in the README.
