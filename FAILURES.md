# Failure modes, fixes, and results

Honest engineering notes for this project. Nothing here is invented for polish.

## What can go wrong

- **Wrong upstream.** A repository with several remotes, or a vendored copy with no remote at all, can point detection at the wrong project. Mitigation: every upstream candidate carries evidence and a confidence score, and `forkledger.yaml` can pin the upstream explicitly.
- **False "already upstream" matches.** Patch equivalence uses Git patch-ids first, then fuzzier signals (MinHash and SimHash fingerprints, Go AST comparison). Fuzzy matches can be wrong. Mitigation: likely and possible equivalents are reported separately and never auto-deleted.
- **Running untrusted code.** Analysing someone else's repository must not execute their hooks or scripts. Mitigation: build and test commands are never inferred, `forkledger upgrade` replays patches in an isolated worktree, and build or test steps only run when `FORKLEDGER_ALLOW_BUILD=1` is set.
- **Leaking private code.** Mitigation: analysis is local by default, `--offline` disables GitHub and OSV lookups, and only owner/repo names, commit hashes and package coordinates are ever sent (`docs/privacy.md`).

## What went wrong

1. **CI went red on the second commit.** After `1176ddf`, `TestAnalyzeForkWorld` in `internal/scanner` expected 3 local patches in the fixture fork and found 2. The cause was the test fixture, not the analyser: the fork commit and its upstream copy had the same message, tree, parent and author, and were created in the same second, so Git gave them the same commit hash and the fork patch looked like it was already upstream. The first CI run only passed because the two commits happened to land in different seconds.
2. **Flags after the path are ignored.** `forkledger analyze . --json` prints the terminal report, while `forkledger analyze --json .` prints JSON. This is standard Go `flag` parsing stopping at the first positional argument. **Documented in the README, not changed in code yet.**
3. **Install instructions pointed at a repository that does not exist.** The Go module path was `github.com/forkledger/forkledger`, which did not match where the code actually lives. The repository is now `TEHILLA-O/fork_ledger` and the module path is `github.com/TEHILLA-O/fork_ledger`, so `go install github.com/TEHILLA-O/fork_ledger/cmd/forkledger@latest` works.

## How it was resolved

- The fixture now gives the upstream copy its own commit message, so it is a separate commit with the same patch. The test still expects 3 local patches and still checks that patch-id equivalence finds the upstream copy.
- Module path and README install steps now match the repository. Flag order is documented in the README.

## Results

- `go build ./cmd/forkledger` succeeds on Go 1.24 (module targets Go 1.22).
- `go test ./...`: all 12 test packages pass, and the scanner test passed 5 runs in a row after the fixture fix.
- Real smoke run (2026-10-09) on a clone of `mattn/go-isatty` five commits behind, with one local patch, offline: upstream detected correctly, fork point found, 1 patch and 1 potential conflict reported, suggested action RETAIN at LOW (35) confidence, fork health 89. Output is in the README.
