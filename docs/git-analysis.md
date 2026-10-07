# Git analysis

ForkLedger calls Git. It does not reimplement the commit graph.

## Commands used

- `git rev-parse`
- `git remote -v`
- `git merge-base` and `git merge-base --all`
- `git rev-list --left-right --count`
- `git log`
- `git diff --name-only` and `git diff --numstat`
- `git show`
- `git patch-id --stable`
- `git cherry -v`
- `git merge-tree`
- `git describe --tags`
- `git for-each-ref`
- `git cat-file`
- `git ls-remote` when a URL is explicitly requested

## Safety

Invocations are argv arrays. Timeouts are enforced. Hooks are pointed at an empty path. `protocol.file.allow` is `never`. Credential helpers are cleared. `GIT_TERMINAL_PROMPT` is `0`.

Revisions, paths, and URLs are rejected when they look like option injection.

Upgrade simulation uses an isolated worktree. The operator working tree is not modified.

## Fork point

The fork point is `git merge-base HEAD <upstream-ref>`. If Git reports several merge-bases, the primary is used and the others remain evidence.

No shared ancestor means this is probably a detached copy. Upstream detection should then use fingerprints, not pretend ancestry exists.

## Patch identity

`git patch-id --stable` is layer one of equivalence. Later layers add file overlap, function overlap from hunk headers, cherry marks, and manual overrides.

AST similarity and model assisted comparison are interfaces, not v0.1 behaviour.
