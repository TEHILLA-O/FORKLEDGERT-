package scanner

import (
	"fmt"
	"strings"

	"github.com/TEHILLA-O/fork_ledger/internal/report"
)

func BuildSummary(rep *report.Report) report.Summary {
	s := report.Summary{
		DetectedUpstream:       rep.Upstream.Selected,
		Baseline:               firstNonEmpty(rep.Lineage.ExpectedBaseline, rep.Lineage.Baseline),
		CurrentUpstream:        rep.Lineage.CurrentUpstream,
		LocalModifications:     rep.Patches.Count,
		ChangedLines:           rep.Patches.Added + rep.Patches.Deleted,
		LikelyEquivalents:      rep.Patches.LikelyEquivalents,
		PotentiallyObsolete:    rep.Patches.PotentiallyObsolete,
		PotentialConflicts:     rep.Patches.PotentialConflicts,
		MissingSecurityFixes:   rep.Security.MissingUpstreamFixes,
		SecurityInvestigations: rep.Security.NeedsInvestigation,
		SuggestedAction:        string(rep.Lifecycle.Recommended),
		Confidence:             string(rep.Lifecycle.Confidence.Level),
	}
	s.Headline = headline(s, rep)
	return s
}

func headline(s report.Summary, rep *report.Report) string {
	parts := []string{}
	if s.DetectedUpstream != "" {
		parts = append(parts, "derived from "+s.DetectedUpstream)
	}
	if s.MissingSecurityFixes > 0 {
		parts = append(parts, fmt.Sprintf("%d missing upstream fixes", s.MissingSecurityFixes))
	}
	if s.LikelyEquivalents > 0 {
		parts = append(parts, fmt.Sprintf("%d patches look equivalent upstream", s.LikelyEquivalents))
	}
	if len(parts) == 0 {
		if rep.Upstream.Uncertain {
			return "Upstream is not settled. Start there before talking about health."
		}
		return "Analysis complete."
	}
	return strings.Join(parts, ". ") + "."
}
