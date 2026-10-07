package config

import (
	"fmt"
	"net/mail"
	"strings"
)

var allowedSeverities = map[string]struct{}{
	"critical": {},
	"high":     {},
	"medium":   {},
	"low":      {},
	"info":     {},
}

var allowedVulnStates = map[string]struct{}{
	"AFFECTED":           {},
	"NOT_AFFECTED":       {},
	"FIXED":              {},
	"UNDER_INVESTIGATION": {},
}

func (c Config) Validate() error {
	var errs []string
	if c.Upstream != "" && strings.ContainsAny(c.Upstream, "\n\r;|&$`") {
		errs = append(errs, "upstream contains unsafe characters")
	}
	if c.ExpectedBaseline != "" && strings.HasPrefix(c.ExpectedBaseline, "-") {
		errs = append(errs, "expected_baseline must not start with a dash")
	}
	if c.Owner.Contact != "" {
		if _, err := mail.ParseAddress(c.Owner.Contact); err != nil && !strings.Contains(c.Owner.Contact, "/") {
			// Slack handles and team aliases are allowed. Email is validated when it looks like email.
			if strings.Contains(c.Owner.Contact, "@") {
				errs = append(errs, "owner.contact is not a valid email address")
			}
		}
	}
	if len(c.BuildCommand) > 0 {
		if err := rejectShellCommand(c.BuildCommand); err != nil {
			errs = append(errs, "build_command: "+err.Error())
		}
	}
	if len(c.TestCommand) > 0 {
		if err := rejectShellCommand(c.TestCommand); err != nil {
			errs = append(errs, "test_command: "+err.Error())
		}
	}
	for i, p := range c.PatchClassifications {
		if p.Commit == "" {
			errs = append(errs, fmt.Sprintf("patch_classifications[%d] needs a commit", i))
		}
		if p.Category != "" && !validCategory(p.Category) {
			errs = append(errs, fmt.Sprintf("patch_classifications[%d] has unknown category %q", i, p.Category))
		}
		if p.Status != "" && !validStatus(p.Status) {
			errs = append(errs, fmt.Sprintf("patch_classifications[%d] has unknown status %q", i, p.Status))
		}
	}
	for i, eq := range c.KnownEquivalents {
		if eq.Local == "" || eq.Upstream == "" {
			errs = append(errs, fmt.Sprintf("known_equivalent_patches[%d] needs local and upstream commits", i))
		}
	}
	for i, v := range c.ManualVulns {
		if v.ID == "" {
			errs = append(errs, fmt.Sprintf("manual_vulnerability_decisions[%d] needs an id", i))
		}
		if _, ok := allowedVulnStates[v.State]; !ok {
			errs = append(errs, fmt.Sprintf("manual_vulnerability_decisions[%d] has unknown state %q", i, v.State))
		}
		if strings.TrimSpace(v.Reason) == "" {
			errs = append(errs, fmt.Sprintf("manual_vulnerability_decisions[%d] needs a reason. ForkLedger will not accept a guess without evidence", i))
		}
	}
	for i, rule := range c.Policy.Rules {
		if strings.TrimSpace(rule.ID) == "" {
			errs = append(errs, fmt.Sprintf("policy.rules[%d] needs an id", i))
		}
		if _, ok := allowedSeverities[strings.ToLower(rule.Severity)]; !ok {
			errs = append(errs, fmt.Sprintf("policy.rules[%d] has unknown severity %q", i, rule.Severity))
		}
		if len(rule.When) == 0 {
			errs = append(errs, fmt.Sprintf("policy.rules[%d] needs at least one condition", i))
		}
		if rule.WaiverExpires != "" && strings.TrimSpace(rule.WaiverReason) == "" {
			errs = append(errs, fmt.Sprintf("policy.rules[%d] waiver needs a reason", i))
		}
	}
	if c.GitHub.CommentMode != "" {
		switch c.GitHub.CommentMode {
		case "update", "create", "off":
		default:
			errs = append(errs, "github.comment_mode must be update, create, or off")
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(errs, "\n  - "))
}

func validCategory(s string) bool {
	switch s {
	case "SECURITY", "BUG_FIX", "COMPATIBILITY", "FEATURE", "PERFORMANCE", "CONFIGURATION", "BRANDING", "COMPLIANCE", "TEMPORARY_WORKAROUND", "EXPERIMENTAL", "UNKNOWN":
		return true
	default:
		return false
	}
}

func validStatus(s string) bool {
	switch s {
	case "ACTIVE", "UPSTREAM_SUBMITTED", "UPSTREAM_ACCEPTED", "UPSTREAM_EQUIVALENT_EXISTS", "REDUNDANT", "BLOCKS_UPGRADE", "MUST_RETAIN", "DEPRECATED", "UNKNOWN":
		return true
	default:
		return false
	}
}

func rejectShellCommand(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty command")
	}
	joined := strings.Join(argv, " ")
	if strings.ContainsAny(joined, ";|&`$") {
		return fmt.Errorf("refusing shell metacharacters. Provide a program and its arguments as a list")
	}
	if strings.Contains(argv[0], " ") {
		return fmt.Errorf("program name must be a single executable path")
	}
	return nil
}
