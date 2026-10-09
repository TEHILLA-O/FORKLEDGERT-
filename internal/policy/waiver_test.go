package policy

import (
	"testing"
	"time"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/report"
	"github.com/TEHILLA-O/forkledger/internal/security"
)

func TestWaiverSuppressesExitPath(t *testing.T) {
	cfg := config.Default()
	cfg.Policy.Rules = []config.PolicyRule{{
		ID:            "critical_security_drift",
		Severity:      "critical",
		When:          map[string]string{"fork_fix": "absent", "upstream_fix": "present"},
		WaiverReason:  "fix scheduled next sprint",
		WaiverOwner:   "platform",
		WaiverExpires: time.Now().UTC().Add(48 * time.Hour).Format("2006-01-02"),
	}}
	rep := &report.Report{}
	rep.Security.MissingUpstreamFixes = 1
	rep.Security.Findings = []security.Finding{{
		UpstreamFixAvailable: true,
		FixPresentInFork:     false,
	}}
	got := Check(cfg, rep)
	if len(got) != 1 || !got[0].Waived {
		t.Fatalf("%+v", got)
	}
}
