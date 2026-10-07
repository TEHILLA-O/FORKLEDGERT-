package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/forkledger/forkledger/internal/config"
	"github.com/forkledger/forkledger/internal/report"
	"github.com/forkledger/forkledger/internal/security"
)

func Check(cfg config.Config, rep *report.Report) []report.PolicyViolation {
	var out []report.PolicyViolation
	for _, rule := range cfg.Policy.Rules {
		if !match(rule, rep) {
			continue
		}
		v := report.PolicyViolation{
			RuleID:      rule.ID,
			Severity:    strings.ToLower(rule.Severity),
			Description: first(rule.Description, defaultDescription(rule)),
			Remediation: rule.Remediation,
		}
		if waived, note := waiverActive(rule); waived {
			v.Waived = true
			v.WaiverNote = note
		}
		out = append(out, v)
	}
	rep.Policy.Violations = out
	return out
}

func waiverActive(rule config.PolicyRule) (bool, string) {
	if strings.TrimSpace(rule.WaiverReason) == "" {
		return false, ""
	}
	if rule.WaiverExpires != "" {
		exp, err := time.Parse("2006-01-02", rule.WaiverExpires)
		if err != nil {
			return false, ""
		}
		if time.Now().UTC().After(exp.Add(24 * time.Hour)) {
			return false, ""
		}
	}
	note := rule.WaiverReason
	if rule.WaiverOwner != "" {
		note += " (owner " + rule.WaiverOwner + ")"
	}
	if rule.WaiverExpires != "" {
		note += " expires " + rule.WaiverExpires
	}
	return true, note
}

func match(rule config.PolicyRule, rep *report.Report) bool {
	if len(rule.When) == 0 {
		return false
	}
	for key, val := range rule.When {
		switch key {
		case "vulnerability_severity":
			if !hasVulnSeverity(rep, val) {
				return false
			}
		case "upstream_fix":
			present := anyFinding(rep, func(f security.Finding) bool { return f.UpstreamFixAvailable })
			if strings.EqualFold(val, "present") && !present {
				return false
			}
			if strings.EqualFold(val, "absent") && present {
				return false
			}
		case "fork_fix":
			if strings.EqualFold(val, "absent") && !hasMissingFix(rep) {
				return false
			}
			if strings.EqualFold(val, "present") && !anyFinding(rep, func(f security.Finding) bool { return f.FixPresentInFork }) {
				return false
			}
		case "min_behind":
			if rep.Lineage.Behind < atoi(val) {
				return false
			}
		case "no_owner":
			want := strings.EqualFold(val, "true")
			hasOwner := false
			for _, f := range rep.Health.Factors {
				if f.Name == "ownership" && f.Delta > 0 {
					hasOwner = true
				}
			}
			if want == hasOwner {
				return false
			}
		case "debt":
			if !strings.EqualFold(rep.Debt.Level, val) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func hasVulnSeverity(rep *report.Report, sev string) bool {
	want := strings.ToUpper(sev)
	for _, f := range rep.Security.Findings {
		if strings.EqualFold(f.Severity, want) || strings.Contains(strings.ToUpper(f.Severity), want) {
			return true
		}
	}
	return false
}

func hasMissingFix(rep *report.Report) bool {
	return rep.Security.MissingUpstreamFixes > 0
}

func anyFinding(rep *report.Report, fn func(security.Finding) bool) bool {
	for _, f := range rep.Security.Findings {
		if fn(f) {
			return true
		}
	}
	return false
}

func defaultDescription(rule config.PolicyRule) string {
	return fmt.Sprintf("policy rule %s matched", rule.ID)
}

func first(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
