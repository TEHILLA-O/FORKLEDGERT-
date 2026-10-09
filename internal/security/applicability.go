package security

import (
	"fmt"
	"strings"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/evidence"
	"github.com/TEHILLA-O/forkledger/internal/patches"
)

func Interpret(advisories []Advisory, local []patches.Patch, baseline string, cfg config.Config) []Finding {
	manual := map[string]config.ManualVuln{}
	for _, m := range cfg.ManualVulns {
		manual[strings.ToUpper(m.ID)] = m
	}

	var out []Finding
	for _, adv := range advisories {
		items := []evidence.Item{
			evidence.New(evidence.KindSecurityAdvisory, "OSV advisory "+adv.ID).WithSource(adv.ID).WithWeight(20),
		}
		f := Finding{
			ID:                   adv.ID,
			Aliases:              adv.Aliases,
			Summary:              adv.Summary,
			Severity:             NormalizeSeverity(adv.Severity),
			UpstreamAffected:     true,
			ForkBaselineAffected: true,
			Fixed:                adv.Fixed,
			Introduced:           adv.Introduced,
		}
		if len(adv.Fixed) > 0 {
			f.UpstreamFixAvailable = true
			items = append(items, evidence.New(evidence.KindSecurityAdvisory, "advisory lists a fix event").
				WithDetail(strings.Join(adv.Fixed, ", ")).WithWeight(15))
		}

		if eq := equivalentFix(local, adv); eq != nil {
			f.FixPresentInFork = true
			items = append(items, evidence.New(evidence.KindMatchingPatchID, "local patch appears equivalent to an upstream fix").
				WithSource(eq.Commit).WithWeight(25))
		} else if mentionsFix(local, adv) {
			f.FixPresentInFork = true
			items = append(items, evidence.New(evidence.KindCommitMessage, "a local commit message references this advisory").WithWeight(15))
		}

		if overlaps(local, adv) {
			f.InternalOverlap = true
			items = append(items, evidence.New(evidence.KindFileOverlap, "internal patches touch paths commonly associated with this class of fix").WithWeight(5))
		}

		if m, ok := matchManual(manual, adv); ok {
			f.Applicability = m.State
			items = append(items, evidence.New(evidence.KindManualOverride, m.Reason).WithWeight(40))
			f.RecommendedAction = "Honor the recorded manual decision and keep the evidence attached."
			f.Confidence = evidence.FromWeights("Manual decision from forkledger.yaml.", items)
			f.Evidence = items
			out = append(out, f)
			continue
		}

		switch {
		case f.FixPresentInFork:
			f.Applicability = ApplicabilityFixed
			f.RecommendedAction = "Treat as fixed in the fork. Keep the patch until upstream is rebased past the official fix."
		case f.UpstreamFixAvailable && !f.FixPresentInFork:
			f.Applicability = ApplicabilityAffected
			f.RecommendedAction = "Bring the upstream fix into the fork, or show evidence that the vulnerable code is absent."
			items = append(items, evidence.New(evidence.KindSecurityAdvisory, "upstream fix exists and no equivalent local patch was found").WithWeight(20))
		default:
			f.Applicability = ApplicabilityUnderInvestigation
			f.RecommendedAction = "Investigate whether the vulnerable code exists at the fork baseline. ForkLedger will not claim NOT_AFFECTED from a guess."
		}
		f.Confidence = evidence.FromWeights("Applicability follows advisory metadata plus local patch evidence. Reachability is not proven.", items)
		f.Evidence = items
		if baseline != "" {
			f.Evidence = append(f.Evidence, evidence.New(evidence.KindReleaseTag, "evaluated against fork baseline "+baseline).WithTarget(baseline))
		}
		out = append(out, f)
	}
	return out
}

func equivalentFix(local []patches.Patch, adv Advisory) *patches.Patch {
	for i := range local {
		p := &local[i]
		if p.Equivalence == nil {
			continue
		}
		for _, fixed := range adv.Fixed {
			if strings.HasPrefix(p.Equivalence.UpstreamCommit, fixed) || strings.HasPrefix(fixed, p.Equivalence.UpstreamCommit) {
				return p
			}
		}
		if p.SecurityRelevant && p.Equivalence.Confidence.Actionable() {
			return p
		}
	}
	return nil
}

func mentionsFix(local []patches.Patch, adv Advisory) bool {
	ids := append([]string{adv.ID}, adv.Aliases...)
	for _, p := range local {
		blob := strings.ToLower(p.Subject + " " + p.Body)
		for _, id := range ids {
			if id != "" && strings.Contains(blob, strings.ToLower(id)) {
				return true
			}
		}
	}
	return false
}

func overlaps(local []patches.Patch, adv Advisory) bool {
	_ = adv
	for _, p := range local {
		if p.SecurityRelevant {
			return true
		}
	}
	return false
}

func matchManual(manual map[string]config.ManualVuln, adv Advisory) (config.ManualVuln, bool) {
	keys := append([]string{adv.ID}, adv.Aliases...)
	for _, k := range keys {
		if m, ok := manual[strings.ToUpper(k)]; ok {
			return m, true
		}
	}
	return config.ManualVuln{}, false
}

func MissingFixes(findings []Finding) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.Applicability == ApplicabilityAffected && f.UpstreamFixAvailable && !f.FixPresentInFork {
			out = append(out, f)
		}
	}
	return out
}

func NeedsInvestigation(findings []Finding) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.Applicability == ApplicabilityUnderInvestigation {
			out = append(out, f)
		}
	}
	return out
}

func FindingAction(f Finding) string {
	return fmt.Sprintf("%s (%s)", f.ID, f.Applicability)
}
