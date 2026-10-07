# Evidence and confidence

ForkLedger prefers a boring explanation over a clever score.

## Sources of truth

- Git is the source of truth for ancestry, diffs, patch-ids, and merge-tree results.
- OSV and similar databases are the source of truth for advisory metadata.
- Build and test commands, when configured, are the source of truth for whether the tree still works.
- Humans are the source of truth for overrides recorded in `forkledger.yaml`.

A language model may later help summarise or classify. It is not allowed to declare equivalence, applicability, or a lifecycle action without supporting evidence.

## Confidence categories

Numeric scores exist internally. The public categories are:

- NONE
- LOW
- MODERATE
- HIGH
- VERY_HIGH

HIGH and VERY_HIGH are treated as actionable. MODERATE is shown, not selected silently.

The mapping is coarse on purpose. If 84 percent and 85 percent would not change an engineer's next step, ForkLedger will not pretend they are different facts.

## Heuristic contract

When a heuristic is added it should record:

- the signal it observes
- what that signal might imply
- common false positives
- common false negatives
- the weight it contributes
- tests for both a hit and a miss

Commit message classification is a good example of a noisy signal. The word "fix" is not a bug taxonomy. Operators can override it.

## Detached copies

Repositories created by copying a tree and starting a new Git history have no merge-base. The fingerprint registry is the seam for later MinHash, SimHash, TLSH, ssdeep, and AST fingerprints. v0.1 ships normalised file hashing and directory signatures so those later signals have somewhere to live.
