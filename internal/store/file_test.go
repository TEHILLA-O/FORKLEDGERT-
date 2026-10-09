package store

import (
	"testing"
	"time"

	"github.com/TEHILLA-O/fork_ledger/internal/report"
)

func TestPutGetAttention(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rep := &report.Report{
		GeneratedAt: time.Now().UTC(),
		Repository:  report.RepositoryInfo{Path: "acme/widget", Head: "abc"},
		Debt:        report.Debt{Level: "HIGH"},
		Health:      report.Health{Score: 61},
		Lifecycle:   report.Lifecycle{Recommended: report.ActionRebase},
	}
	rep.Security.MissingUpstreamFixes = 2
	id, err := st.Put(rep)
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Repository.Path != "acme/widget" {
		t.Fatalf("%+v", got.Repository)
	}
	att, err := st.Attention()
	if err != nil {
		t.Fatal(err)
	}
	if len(att) != 1 || att[0].Missing != 2 {
		t.Fatalf("%+v", att)
	}
}
