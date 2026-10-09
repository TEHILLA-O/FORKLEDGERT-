package policy

import (
	"testing"

	"github.com/TEHILLA-O/fork_ledger/internal/config"
	"github.com/TEHILLA-O/fork_ledger/internal/report"
	"github.com/TEHILLA-O/fork_ledger/internal/security"
)

func TestCriticalSecurityDrift(t *testing.T) {
	cfg := config.Default()
	cfg.Policy.Rules = []config.PolicyRule{{
		ID:          "critical_security_drift",
		Severity:    "critical",
		Description: "A critical vulnerability has an upstream fix that is not in the fork.",
		Remediation: "Rebase or cherry-pick the upstream fix.",
		When: map[string]string{
			"vulnerability_severity": "critical",
			"upstream_fix":           "present",
			"fork_fix":               "absent",
		},
	}}
	rep := &report.Report{}
	rep.Security.MissingUpstreamFixes = 1
	rep.Security.Findings = []security.Finding{{
		ID:                   "CVE-2024-0001",
		Severity:             "CRITICAL",
		UpstreamFixAvailable: true,
		FixPresentInFork:     false,
		Applicability:        security.ApplicabilityAffected,
	}}
	got := Check(cfg, rep)
	if len(got) != 1 {
		t.Fatalf("got %d violations", len(got))
	}
}

func TestUnknownConditionDoesNotMatch(t *testing.T) {
	cfg := config.Default()
	cfg.Policy.Rules = []config.PolicyRule{{
		ID:       "weird",
		Severity: "low",
		When:     map[string]string{"not_a_real_field": "yes"},
	}}
	rep := &report.Report{}
	if got := Check(cfg, rep); len(got) != 0 {
		t.Fatalf("unknown conditions must not fire, got %+v", got)
	}
}
