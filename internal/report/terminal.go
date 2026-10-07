package report

import (
	"fmt"
	"io"
	"strings"
)

func WriteTerminal(w io.Writer, rep *Report) error {
	p := printer{w: w}
	p.line("ForkLedger Analysis")
	p.blank()

	up := rep.Summary.DetectedUpstream
	if up == "" {
		up = "(not settled)"
	}
	p.kv("Detected upstream", up)
	if rep.Upstream.Uncertain {
		p.kv("Upstream note", rep.Upstream.Explanation)
	}
	p.kv("Baseline", dash(rep.Summary.Baseline))
	p.kv("Current upstream", dash(rep.Summary.CurrentUpstream))
	if rep.Lineage.ForkPoint != "" {
		p.kv("Fork point", short(rep.Lineage.ForkPoint))
	}
	p.kv("Ahead / behind", fmt.Sprintf("%d / %d", rep.Lineage.Ahead, rep.Lineage.Behind))
	p.blank()

	p.kv("Local modifications", countNoun(rep.Summary.LocalModifications, "patch", "patches"))
	p.kv("Changed lines", fmt.Sprintf("%d", rep.Summary.ChangedLines))
	p.kv("Likely upstream equivalents", countNoun(rep.Summary.LikelyEquivalents, "patch", "patches"))
	p.kv("Potentially obsolete", countNoun(rep.Summary.PotentiallyObsolete, "patch", "patches"))
	p.kv("Potential conflicts", countNoun(rep.Summary.PotentialConflicts, "patch", "patches"))
	p.blank()

	p.line("Security")
	p.kv("  Relevant upstream fixes missing", fmt.Sprintf("%d", rep.Summary.MissingSecurityFixes))
	p.kv("  Findings needing investigation", fmt.Sprintf("%d", rep.Summary.SecurityInvestigations))
	for _, f := range rep.Security.Findings {
		p.kv("  "+f.ID, fmt.Sprintf("%s. %s", f.Applicability, f.RecommendedAction))
	}
	p.blank()

	p.kv("Suggested lifecycle action", string(rep.Lifecycle.Recommended))
	p.kv("Confidence", fmt.Sprintf("%s (%d). %s", rep.Lifecycle.Confidence.Level, rep.Lifecycle.Confidence.Score, "This is an estimate from evidence, not a guarantee."))
	if rep.Summary.Headline != "" {
		p.kv("What changed", rep.Summary.Headline)
	}
	if len(rep.Lifecycle.Why) > 0 {
		p.kv("Why it matters", strings.Join(rep.Lifecycle.Why, " "))
	}
	if rep.Lifecycle.RequiredWork != "" {
		p.kv("What to do next", rep.Lifecycle.RequiredWork)
	}
	p.blank()

	p.line(fmt.Sprintf("Fork health %d", rep.Health.Score))
	for _, f := range rep.Health.Factors {
		sign := "+"
		if f.Delta < 0 {
			sign = ""
		}
		p.kv("  "+f.Name, fmt.Sprintf("%s%d. %s", sign, f.Delta, f.Reason))
	}
	p.kv("Maintenance debt", rep.Debt.Level+". "+rep.Debt.Summary)
	if len(rep.Warnings) > 0 {
		p.blank()
		p.line("Warnings")
		for _, w := range rep.Warnings {
			p.kv("  -", w)
		}
	}
	return p.err
}

type printer struct {
	w   io.Writer
	err error
}

func (p *printer) line(s string) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintln(p.w, s)
}

func (p *printer) blank() { p.line("") }

func (p *printer) kv(key, value string) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, "%-32s %s\n", key, value)
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(unknown)"
	}
	return s
}

func short(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

func countNoun(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
