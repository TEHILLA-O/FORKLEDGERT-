package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/evidence"
	"github.com/TEHILLA-O/forkledger/internal/git"
	"github.com/TEHILLA-O/forkledger/internal/isolate"
	"github.com/TEHILLA-O/forkledger/internal/logging"
	"github.com/TEHILLA-O/forkledger/internal/patches"
	"github.com/TEHILLA-O/forkledger/internal/report"
	"github.com/TEHILLA-O/forkledger/internal/scanner"
	"github.com/TEHILLA-O/forkledger/internal/upstream"
)

type PatchOutcome struct {
	Commit  string   `json:"commit"`
	Subject string   `json:"subject"`
	Result  string   `json:"result"`
	Files   []string `json:"files,omitempty"`
	Detail  string   `json:"detail,omitempty"`
}

type Result struct {
	Target            string              `json:"target"`
	IsolatedWorktree  string              `json:"isolated_worktree,omitempty"`
	Evaluated         int                 `json:"evaluated"`
	Clean             int                 `json:"clean"`
	Equivalent        int                 `json:"equivalent"`
	LimitedConflicts  int                 `json:"limited_conflicts"`
	MajorConflicts    int                 `json:"major_conflicts"`
	Obsolete          int                 `json:"obsolete"`
	ConflictedFiles   []string            `json:"conflicted_files,omitempty"`
	Outcomes          []PatchOutcome      `json:"outcomes,omitempty"`
	MergeTreeClean    bool                `json:"merge_tree_clean"`
	BuildRan          bool                `json:"build_ran"`
	BuildOK           bool                `json:"build_ok"`
	TestsRan          bool                `json:"tests_ran"`
	TestsOK           bool                `json:"tests_ok"`
	ReviewSurface     int                 `json:"review_surface"`
	Confidence        evidence.Assessment `json:"confidence"`
	Recommendation    string              `json:"recommendation"`
	Notes             []string            `json:"notes,omitempty"`
	UserTreeUntouched bool                `json:"user_tree_untouched"`
}

func Simulate(ctx context.Context, path, target string, cfg config.Config, log *logging.Logger, runBuild bool) (*Result, error) {
	exec := git.NewExecutor(log)
	repo, err := git.Open(ctx, path, exec)
	if err != nil {
		return nil, err
	}
	if _, err := repo.RevParse(ctx, target); err != nil {
		return nil, fmt.Errorf("unknown upgrade target %q: %w", target, err)
	}

	eng := scanner.New(log)
	rep, err := eng.Analyze(ctx, scanner.Options{Path: path, Config: cfg, Offline: true, Log: log})
	if err != nil {
		return nil, err
	}

	res := &Result{
		Target:            target,
		UserTreeUntouched: true,
		Evaluated:         len(rep.Patches.Items),
	}

	mt, err := repo.MergeTree(ctx, "HEAD", target)
	if err != nil {
		return nil, err
	}
	res.MergeTreeClean = mt.Clean
	res.ConflictedFiles = mt.Conflicted

	wt, err := isolatedWorktree(ctx, repo, target)
	if err != nil {
		res.Notes = append(res.Notes, "isolated worktree failed: "+err.Error()+". Using merge-tree overlap instead.")
		fallbackOverlap(res, mt, rep.Patches.Items)
	} else {
		res.IsolatedWorktree = wt
		defer func() {
			_ = repo.WorktreeRemove(ctx, wt)
			_ = os.RemoveAll(filepath.Dir(wt))
		}()
		work, err := git.Open(ctx, wt, exec)
		if err != nil {
			res.Notes = append(res.Notes, "could not open isolated worktree: "+err.Error())
			fallbackOverlap(res, mt, rep.Patches.Items)
		} else {
			replay(ctx, work, rep.Patches.Items, res)
		}
	}

	if runBuild && os.Getenv("FORKLEDGER_ALLOW_BUILD") == "1" && res.IsolatedWorktree != "" {
		if len(cfg.BuildCommand) > 0 {
			res.BuildRan = true
			br, err := isolate.Run(ctx, res.IsolatedWorktree, cfg.BuildCommand)
			res.BuildOK = err == nil && br.OK
			if err != nil {
				res.Notes = append(res.Notes, "build: "+err.Error())
			}
		}
		if len(cfg.TestCommand) > 0 {
			res.TestsRan = true
			tr, err := isolate.Run(ctx, res.IsolatedWorktree, cfg.TestCommand)
			res.TestsOK = err == nil && tr.OK
			if err != nil {
				res.Notes = append(res.Notes, "test: "+err.Error())
			}
		}
	} else if len(cfg.BuildCommand) > 0 || len(cfg.TestCommand) > 0 {
		res.Notes = append(res.Notes, "build and test commands are configured. Set FORKLEDGER_ALLOW_BUILD=1 to run them inside the isolated worktree.")
	}

	res.ReviewSurface = res.LimitedConflicts + res.MajorConflicts*3
	items := []evidence.Item{
		evidence.New(evidence.KindMergeTree, fmt.Sprintf("merge-tree clean=%t, conflicted files=%d", mt.Clean, len(mt.Conflicted))).WithWeight(20),
	}
	if res.IsolatedWorktree != "" {
		items = append(items, evidence.New(evidence.KindMergeTree, "cherry-pick replay ran in an isolated worktree").WithWeight(25))
	}
	if res.Equivalent > 0 {
		items = append(items, evidence.New(evidence.KindMatchingPatchID, fmt.Sprintf("%d equivalent patches", res.Equivalent)).WithWeight(10))
	}
	if res.MajorConflicts > 0 {
		items = append(items, evidence.New(evidence.KindMergeTree, fmt.Sprintf("%d patches produced substantial conflicts", res.MajorConflicts)).WithWeight(-10))
	}
	if res.BuildRan {
		w := 8
		if !res.BuildOK {
			w = -12
		}
		items = append(items, evidence.New(evidence.KindSuccessfulBuild, fmt.Sprintf("isolated build ok=%t", res.BuildOK)).WithWeight(w))
	}
	res.Confidence = evidence.FromWeights("Upgrade confidence is an estimate from merge-tree plus isolated cherry-pick replay. It is not a promise.", items)
	res.Recommendation = recommend(res)
	res.Notes = append(res.Notes, "The operator working tree was not modified.")
	return res, nil
}

func replay(ctx context.Context, work *git.Repository, items []report.Patch, res *Result) {
	for _, p := range items {
		out := PatchOutcome{Commit: p.Commit, Subject: p.Subject}
		if p.Equivalence != nil && p.Equivalence.Confidence.Actionable() {
			out.Result = "equivalent"
			res.Equivalent++
			res.Outcomes = append(res.Outcomes, out)
			continue
		}
		if p.Status == string(patches.StatusRedundant) {
			out.Result = "obsolete"
			res.Obsolete++
			res.Outcomes = append(res.Outcomes, out)
			continue
		}
		cp := work.CherryPick(ctx, p.Commit)
		switch {
		case cp.Empty:
			out.Result = "obsolete"
			out.Detail = cp.Message
			res.Obsolete++
		case cp.Clean:
			out.Result = "clean"
			res.Clean++
		case len(cp.Files) <= 1:
			out.Result = "limited_conflict"
			out.Files = cp.Files
			out.Detail = cp.Message
			res.LimitedConflicts++
			res.ConflictedFiles = appendUnique(res.ConflictedFiles, cp.Files...)
		default:
			out.Result = "major_conflict"
			out.Files = cp.Files
			out.Detail = cp.Message
			res.MajorConflicts++
			res.ConflictedFiles = appendUnique(res.ConflictedFiles, cp.Files...)
		}
		res.Outcomes = append(res.Outcomes, out)
	}
}

func fallbackOverlap(res *Result, mt git.MergeTreeResult, items []report.Patch) {
	set := map[string]struct{}{}
	for _, f := range mt.Conflicted {
		set[f] = struct{}{}
	}
	for _, p := range items {
		if p.Equivalence != nil && p.Equivalence.Confidence.Actionable() {
			res.Equivalent++
			continue
		}
		hits := 0
		for _, f := range p.Files {
			if _, ok := set[f]; ok {
				hits++
			}
		}
		if hits == 0 && mt.Clean {
			res.Clean++
		} else if hits <= 1 {
			res.LimitedConflicts++
		} else {
			res.MajorConflicts++
		}
	}
}

func recommend(res *Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d internal patches evaluated. %d appear clean. %d appear equivalent upstream. %d produce limited conflicts. %d produce substantial conflicts. %d appear obsolete. ",
		res.Evaluated, res.Clean, res.Equivalent, res.LimitedConflicts, res.MajorConflicts, res.Obsolete)
	if res.MergeTreeClean && res.MajorConflicts == 0 {
		b.WriteString("A rebase looks feasible. Confirm equivalents, then upgrade in a throwaway clone.")
	} else {
		b.WriteString("Do not treat this as a green light. Inspect the conflicted files before attempting a rebase.")
	}
	if res.BuildRan {
		fmt.Fprintf(&b, " Isolated build ok=%t.", res.BuildOK)
	}
	if res.TestsRan {
		fmt.Fprintf(&b, " Isolated tests ok=%t.", res.TestsOK)
	}
	fmt.Fprintf(&b, " Upgrade confidence %s (%d). That number is an estimate.", res.Confidence.Level, res.Confidence.Score)
	return b.String()
}

func isolatedWorktree(ctx context.Context, repo *git.Repository, commit string) (string, error) {
	dir, err := os.MkdirTemp("", "forkledger-upgrade-*")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(dir, "work")
	if err := repo.WorktreeAddDetached(ctx, dest, commit); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dest, nil
}

func appendUnique(dst []string, extra ...string) []string {
	seen := map[string]struct{}{}
	for _, d := range dst {
		seen[d] = struct{}{}
	}
	for _, e := range extra {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		dst = append(dst, e)
	}
	return dst
}

func ResolveUpstreamCandidate(ctx context.Context, repo *git.Repository, remotes []git.Remote, identity string) (string, error) {
	c := upstream.Candidate{Identity: identity}
	ref, _, err := upstream.ResolveRef(ctx, repo, remotes, c)
	return ref, err
}
