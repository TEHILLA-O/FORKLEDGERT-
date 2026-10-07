package scanner

import (
	"fmt"
	"strings"

	"github.com/forkledger/forkledger/internal/config"
	"github.com/forkledger/forkledger/internal/evidence"
	"github.com/forkledger/forkledger/internal/report"
)

func ScoreHealth(rep *report.Report, cfg config.Config) report.Health {
	score := 100
	var factors []report.Factor

	add := func(name string, delta int, reason string) {
		if delta == 0 {
			return
		}
		score += delta
		factors = append(factors, report.Factor{Name: name, Delta: delta, Reason: reason})
	}

	missing := rep.Security.MissingUpstreamFixes
	if missing > 0 {
		delta := -12 * missing
		if delta < -40 {
			delta = -40
		}
		add("security", delta, fmt.Sprintf("%d relevant upstream fixes appear missing", missing))
	}

	if rep.Lineage.Behind > 0 {
		delta := -rep.Lineage.Behind / 10
		if delta < -20 {
			delta = -20
		}
		if delta == 0 {
			delta = -1
		}
		add("upstream drift", delta, fmt.Sprintf("%d commits behind upstream", rep.Lineage.Behind))
	}

	if rep.Patches.PotentialConflicts > 0 {
		delta := -2 * rep.Patches.PotentialConflicts
		if delta < -15 {
			delta = -15
		}
		add("conflicting patches", delta, fmt.Sprintf("%d patches touch files also changed upstream", rep.Patches.PotentialConflicts))
	}

	if !cfg.HasOwner() {
		add("ownership", -8, "no owner is recorded in forkledger.yaml")
	} else {
		add("ownership", 2, "owner is recorded")
	}

	if cfg.LastReviewed != "" {
		add("review", 2, "a last reviewed date is recorded")
	} else if cfg.ReviewInterval.Duration > 0 {
		add("review", -6, "review interval is configured but no last reviewed date is present")
	}

	if rep.Upgrade.MergeTreeClean && rep.Lineage.ForkPoint != "" {
		add("merge surface", 3, "git merge-tree currently reports a clean tree")
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return report.Health{Score: score, Factors: factors}
}

func ScoreDebt(rep *report.Report, cfg config.Config) report.Debt {
	_ = cfg
	points := 0
	var signals []report.Factor
	add := func(name string, delta int, reason string) {
		points += delta
		signals = append(signals, report.Factor{Name: name, Delta: delta, Reason: reason})
	}

	add("internal patches", rep.Patches.Count, fmt.Sprintf("%d internal patches", rep.Patches.Count))
	add("changed lines", (rep.Patches.Added+rep.Patches.Deleted)/200, fmt.Sprintf("%d changed lines versus the fork point", rep.Patches.Added+rep.Patches.Deleted))
	add("upstream distance", rep.Lineage.Behind/20, fmt.Sprintf("%d commits behind", rep.Lineage.Behind))
	add("conflicts", rep.Patches.PotentialConflicts*3, fmt.Sprintf("%d likely conflict patches", rep.Patches.PotentialConflicts))
	add("missing fixes", rep.Security.MissingUpstreamFixes*8, fmt.Sprintf("%d missing upstream fixes", rep.Security.MissingUpstreamFixes))
	if rep.Patches.LikelyEquivalents > 0 {
		add("duplicated work", rep.Patches.LikelyEquivalents, fmt.Sprintf("%d patches already look equivalent upstream", rep.Patches.LikelyEquivalents))
	}

	level := "LOW"
	switch {
	case points >= 80:
		level = "SEVERE"
	case points >= 40:
		level = "HIGH"
	case points >= 15:
		level = "MODERATE"
	}
	summary := fmt.Sprintf("Maintenance debt is %s. This is a descriptive category, not a currency estimate.", level)
	if rep.Patches.LikelyEquivalents > 0 {
		summary += fmt.Sprintf(" %d internal patches now appear to exist upstream.", rep.Patches.LikelyEquivalents)
	}
	return report.Debt{Level: level, Signals: signals, Summary: strings.TrimSpace(summary)}
}

func Recommend(rep *report.Report) report.Lifecycle {
	lc := report.Lifecycle{
		Recommended: report.ActionRetain,
		Why:         []string{},
		Risks:       []string{},
	}

	missing := rep.Security.MissingUpstreamFixes
	behind := rep.Lineage.Behind
	eq := rep.Patches.LikelyEquivalents
	conflicts := rep.Patches.PotentialConflicts
	patchesN := rep.Patches.Count

	switch {
	case missing > 0 && behind >= 10:
		lc.Recommended = report.ActionRebase
		lc.Why = append(lc.Why,
			fmt.Sprintf("The fork is %d commits behind upstream.", behind),
			fmt.Sprintf("%d relevant upstream fixes appear missing.", missing),
		)
		lc.Benefit = "A rebase is the most direct way to inherit missing fixes while keeping the internal patch ledger."
		lc.RequiredWork = fmt.Sprintf("Review %d likely conflict patches, drop equivalents, then rebase onto current upstream.", conflicts)
		lc.Alternatives = []report.LifecycleAction{report.ActionRetain, report.ActionReplace}
	case eq > 0 && eq*2 >= patchesN && patchesN > 0:
		lc.Recommended = report.ActionRebase
		lc.Why = append(lc.Why, fmt.Sprintf("%d of %d internal patches already look equivalent upstream.", eq, patchesN))
		lc.Benefit = "Dropping equivalents before rebasing shrinks the conflict surface."
		lc.RequiredWork = "Confirm equivalents, remove them from the fork, then rebase."
		lc.Alternatives = []report.LifecycleAction{report.ActionUpstream, report.ActionRetain}
	case patchesN <= 3 && behind > 50 && conflicts == 0:
		lc.Recommended = report.ActionReplace
		lc.Why = append(lc.Why, "The fork is far behind and carries almost no unique work.")
		lc.Benefit = "Replacing with upstream removes ongoing drift cost."
		lc.RequiredWork = "Confirm the remaining patches are unused, then switch dependents to upstream."
		lc.Alternatives = []report.LifecycleAction{report.ActionRetire, report.ActionRebase}
	case patchesN > 0 && eq == 0 && conflicts >= patchesN/2 && behind < 20:
		lc.Recommended = report.ActionRetain
		lc.Why = append(lc.Why, "Unique internal patches overlap heavily with live upstream files.")
		lc.Benefit = "Retaining avoids a high conflict rebase until those patches are upstreamed or isolated."
		lc.RequiredWork = "Assign an owner and consider upstreaming the patches that still matter."
		lc.Alternatives = []report.LifecycleAction{report.ActionUpstream, report.ActionRebase}
	default:
		if behind > 30 {
			lc.Recommended = report.ActionRebase
			lc.Why = append(lc.Why, fmt.Sprintf("The fork is %d commits behind upstream.", behind))
			lc.Benefit = "Catching up now is cheaper than waiting for another security cycle."
			lc.RequiredWork = "Run forkledger upgrade against the current upstream ref in an isolated worktree."
			lc.Alternatives = []report.LifecycleAction{report.ActionRetain}
		} else {
			lc.Recommended = report.ActionRetain
			lc.Why = append(lc.Why, "Divergence is still small enough that an emergency rebase is not required.")
			lc.Benefit = "Keep the fork, keep the ledger, review on the configured interval."
			lc.RequiredWork = "Record an owner if none exists and keep watching missing fixes."
			lc.Alternatives = []report.LifecycleAction{report.ActionRebase, report.ActionUpstream}
		}
	}

	if conflicts > 0 {
		lc.Risks = append(lc.Risks, fmt.Sprintf("%d patches modify files that also moved upstream.", conflicts))
	}
	if missing > 0 {
		lc.Risks = append(lc.Risks, "Missing upstream security fixes remain applicable until proven otherwise.")
	}
	if rep.Upstream.Uncertain {
		lc.Risks = append(lc.Risks, "Upstream identity is not settled. Do not rebase until it is.")
		if lc.Recommended == report.ActionRebase {
			lc.Recommended = report.ActionRetain
			lc.Why = append(lc.Why, "Recommendation downgraded because upstream is uncertain.")
		}
	}

	lc.Confidence = lifecycleConfidence(lc, rep)
	return lc
}

func lifecycleConfidence(lc report.Lifecycle, rep *report.Report) evidence.Assessment {
	items := []evidence.Item{
		evidence.New(evidence.KindGitAncestry, fmt.Sprintf("ahead %d, behind %d", rep.Lineage.Ahead, rep.Lineage.Behind)).WithWeight(15),
	}
	if rep.Lineage.Confidence.Actionable() {
		items = append(items, evidence.New(evidence.KindGitAncestry, "fork point is known from merge-base").WithWeight(20))
	}
	if rep.Security.MissingUpstreamFixes > 0 {
		items = append(items, evidence.New(evidence.KindSecurityAdvisory, "missing upstream fixes influenced the recommendation").WithWeight(20))
	}
	if rep.Patches.LikelyEquivalents > 0 {
		items = append(items, evidence.New(evidence.KindMatchingPatchID, "equivalent patches influenced the recommendation").WithWeight(15))
	}
	if lc.Recommended == report.ActionRebase && !rep.Upstream.Uncertain {
		items = append(items, evidence.New(evidence.KindUpstreamActivity, "upstream identity is settled enough to rebase").WithWeight(10))
	}
	if rep.Upstream.Uncertain {
		items = append(items, evidence.New(evidence.KindConfiguredUpstream, "upstream remains uncertain").WithWeight(-20))
	}
	return evidence.FromWeights("Lifecycle advice is a deterministic rule over collected evidence, not a model guess.", items)
}
