package scanner

import (
	"testing"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/report"
	"github.com/TEHILLA-O/forkledger/internal/security"
)

func TestScoreHealthExplainsFactors(t *testing.T) {
	rep := &report.Report{}
	rep.Security.MissingUpstreamFixes = 2
	rep.Lineage.Behind = 40
	rep.Patches.PotentialConflicts = 3
	h := ScoreHealth(rep, config.Default())
	if h.Score >= 100 {
		t.Fatalf("score %d", h.Score)
	}
	if len(h.Factors) < 3 {
		t.Fatalf("factors %+v", h.Factors)
	}
}

func TestRecommendRebaseWhenFixesMissing(t *testing.T) {
	rep := &report.Report{}
	rep.Security.MissingUpstreamFixes = 2
	rep.Lineage.Behind = 20
	rep.Lineage.Confidence.Score = 90
	lc := Recommend(rep)
	if lc.Recommended != report.ActionRebase {
		t.Fatalf("got %s", lc.Recommended)
	}
	if len(lc.Why) == 0 {
		t.Fatal("recommendation must say why")
	}
}

func TestPolicyCriticalSecurityDrift(t *testing.T) {
	// Keep this next to health tests because policy is a consumer of the same report shape.
}

func TestDebtLevels(t *testing.T) {
	rep := &report.Report{}
	rep.Patches.Count = 80
	rep.Lineage.Behind = 400
	rep.Security.Findings = []security.Finding{{}}
	d := ScoreDebt(rep, config.Default())
	if d.Level == "LOW" {
		t.Fatalf("expected higher debt, got %s", d.Level)
	}
	if d.Summary == "" {
		t.Fatal("debt must stay descriptive")
	}
}
