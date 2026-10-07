# Security model

ForkLedger is not a general vulnerability scanner. It interprets known advisories against a modified tree.

## Pipeline

1. Identify the upstream project and the fork baseline.
2. Query OSV by commit and, when possible, by GIT package coordinates.
3. Read whether an advisory lists a fix event.
4. Search the local patch ledger for an equivalent fix.
5. Record applicability as AFFECTED, FIXED, or UNDER_INVESTIGATION.
6. Emit a VEX complement that cites evidence.

NOT_AFFECTED is allowed only when a human recorded a reason, or when later reachability evidence is actually implemented. A model guess is not a reason.

## Adapters

v0.1 includes OSV. GitHub Security Advisories, Trivy, CycloneDX, SPDX, and VEX consumers are the next adapters. The finding type is the stable boundary.

## Untrusted repositories

Treat repository content as hostile:

- argv only, never a shell
- no hooks
- no `file:` or `ext::` transports
- no inferred build commands
- timeouts on Git
- credential helpers disabled during analysis
- secrets must not appear in logs

Upgrade simulation happens in an isolated worktree.
