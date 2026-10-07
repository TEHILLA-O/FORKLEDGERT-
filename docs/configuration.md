# Configuration

`forkledger.yaml` is optional. When present it is validated before analysis starts.

Lookups walk from the repository path toward the filesystem root.

## Fields

| Field | Meaning |
| --- | --- |
| `upstream` | Expected upstream identity or URL |
| `expected_baseline` | Version you believe the fork started from |
| `owner.team` | Owning team |
| `owner.contact` | Email or team alias |
| `review_interval` | How often the fork should be reviewed, for example `90d` |
| `last_reviewed` | ISO date of the last human review |
| `security_sla` | Target time to absorb upstream security fixes |
| `build_command` | Argv list. Never a shell string |
| `test_command` | Argv list. Never a shell string |
| `ignored_paths` | Paths excluded from the patch ledger |
| `allowed_licences` | SPDX ids that policy may later enforce |
| `patch_classifications` | Manual category or status for a commit |
| `known_equivalent_patches` | Manual local to upstream mapping |
| `manual_vulnerability_decisions` | VEX-like state with a mandatory reason |
| `policy.rules` | Constrained condition map, not a programming language |
| `github.comment_mode` | `update`, `create`, or `off` |

## Policy schema

Each rule has an id, a severity, optional prose, and a `when` map. Supported keys:

- `vulnerability_severity`
- `upstream_fix` (`present` or `absent`)
- `fork_fix` (`present` or `absent`)
- `min_behind`
- `no_owner`
- `debt`

Unknown keys fail closed. The rule does not match.

Manual `NOT_AFFECTED` decisions require a reason. ForkLedger will not accept an empty justification.
