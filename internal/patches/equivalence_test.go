package patches

import (
	"testing"

	"github.com/TEHILLA-O/fork_ledger/internal/config"
	"github.com/TEHILLA-O/fork_ledger/internal/git"
)

func TestPatchIDEquivalence(t *testing.T) {
	local := []Patch{{
		Commit:  "aaa111",
		Subject: "compat: retain legacy token refresh behaviour",
		PatchID: "same-id",
		Files:   []git.FileChange{{Path: "src/auth.go"}},
		Status:  StatusActive,
	}}
	up := []UpstreamCommit{{
		Commit:  git.Commit{Hash: "bbb222", Subject: "compat: retain legacy token refresh behaviour"},
		PatchID: "same-id",
		Files:   []string{"src/auth.go"},
		Subject: "compat: retain legacy token refresh behaviour",
	}}
	got := DetectEquivalence(local, up, nil, config.Default())
	if got[0].Equivalence == nil {
		t.Fatal("expected equivalence")
	}
	if got[0].Equivalence.UpstreamCommit != "bbb222" {
		t.Fatalf("%+v", got[0].Equivalence)
	}
	if got[0].Status != StatusUpstreamEquivalent {
		t.Fatalf("status %s", got[0].Status)
	}
}

func TestNoEquivalenceWithoutEvidence(t *testing.T) {
	local := []Patch{{
		Commit:  "aaa111",
		Subject: "brand: add logo",
		PatchID: "one",
		Files:   []git.FileChange{{Path: "logo.svg"}},
		Status:  StatusActive,
	}}
	up := []UpstreamCommit{{
		Commit:  git.Commit{Hash: "bbb222", Subject: "docs: rewrite readme"},
		PatchID: "two",
		Files:   []string{"README.md"},
	}}
	got := DetectEquivalence(local, up, nil, config.Default())
	if got[0].Equivalence != nil {
		t.Fatalf("unexpected %+v", got[0].Equivalence)
	}
}
