# GitHub App

The hosted product should authenticate as a GitHub App. Personal access tokens are the wrong default for an organisation.

## Permissions

Request the minimum:

- Contents: read
- Metadata: read
- Pull requests: write, only if comment updates are enabled
- Security events: read, if advisory correlation is enabled later

Do not request organisation administration, workflows, or secrets.

## Events

Subscribe only to events ForkLedger uses:

- `installation`
- `installation_repositories`
- `push`
- `pull_request`
- `repository`

Webhook handling must verify signatures, store delivery ids for idempotency, know which installation owns the payload, and back off on rate limits.

## Comments

The GitHub Action updates a single comment marked `<!-- forkledger-comment -->`. It does not create a new comment on every push.

## v0.1 status

The Action in `action.yml` runs the CLI. The App itself is not shipped yet. This document is the constraint the App must keep.
