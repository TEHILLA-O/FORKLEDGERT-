# Tests

Git fixture repositories are created in process by `internal/testutil`. That keeps them portable on Windows and Linux and avoids committing object packs.

`go test ./...` is the full suite.

Integration coverage lives in `internal/scanner/analyze_test.go` and uses a real upstream plus a real fork with:

- a shared tagged baseline
- unique internal patches
- an internal patch later duplicated upstream
- an upstream security commit the fork does not have

Those cases exist to show both a successful equivalence hit and a missing fix, not to flatter the heuristics.
