package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkledger/forkledger/internal/config"
	"github.com/forkledger/forkledger/internal/evidence"
	"github.com/forkledger/forkledger/internal/git"
	"github.com/forkledger/forkledger/internal/logging"
	"github.com/forkledger/forkledger/internal/patches"
	"github.com/forkledger/forkledger/internal/scanner"
	"github.com/forkledger/forkledger/internal/upstream"
)

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

	conflictSet := map[string]struct{}{}
	for _, f := range mt.Conflicted {
		conflictSet[f] = struct{}{}
	}

	for _, p := range rep.Patches.Items {
		switch {
		case p.Equivalence != nil && p.Equivalence.Confidence.Actionable():
			res.Equivalent++
		case p.Status == string(patches.StatusRedundant):
			res.Obsolete++
		default:
			hits := 0
			for _, f := range p.Files {
				if _, ok := conflictSet[f]; ok {
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
	res.ReviewSurface = res.LimitedConflicts + res.MajorConflicts*3

	if runBuild && len(cfg.BuildCommand) > 0 {
		wt, err := isolatedWorktree(ctx, repo, target)
		if err != nil {
			res.Notes = append(res.Notes, "isolated worktree failed: "+err.Error())
		} else {
			res.IsolatedWorktree = wt
			defer func() {
				_ = repo.WorktreeRemove(ctx, wt)
				_ = os.RemoveAll(wt)
			}()
			res.Notes = append(res.Notes, "build and test commands are configured and would run only inside the isolated worktree")
			res.Notes = append(res.Notes, "this simulator does not execute them unless FORKLEDGER_ALLOW_BUILD=1 is set")
			if os.Getenv("FORKLEDGER_ALLOW_BUILD") == "1" {
				res.Notes = append(res.Notes, "build execution is intentionally still opt in and not implemented as a shell. Provide a later isolated runner.")
			}
		}
	}

	items := []evidence.Item{
		evidence.New(evidence.KindMergeTree, fmt.Sprintf("merge-tree clean=%t, conflicted files=%d", mt.Clean, len(mt.Conflicted))).WithWeight(25),
	}
	if res.Equivalent > 0 {
		items = append(items, evidence.New(evidence.KindMatchingPatchID, fmt.Sprintf("%d equivalent patches", res.Equivalent)).WithWeight(15))
	}
	if res.MajorConflicts > 0 {
		items = append(items, evidence.New(evidence.KindMergeTree, fmt.Sprintf("%d patches look like substantial conflicts", res.MajorConflicts)).WithWeight(-10))
	}
	res.Confidence = evidence.FromWeights("Upgrade confidence is an estimate from merge-tree and patch overlap. It is not a promise that the rebase will succeed.", items)
	res.Recommendation = recommend(res)
	res.Notes = append(res.Notes, "The working tree was not modified.")
	return res, nil
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

func recommend(res *Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d internal patches evaluated. %d appear clean. %d appear equivalent upstream. %d produce limited conflicts. %d produce substantial conflicts. %d appear obsolete. ",
		res.Evaluated, res.Clean, res.Equivalent, res.LimitedConflicts, res.MajorConflicts, res.Obsolete)
	if res.MergeTreeClean && res.MajorConflicts == 0 {
		b.WriteString("A rebase looks feasible. Confirm equivalents, then upgrade in a throwaway clone.")
	} else {
		b.WriteString("Do not treat this as a green light. Inspect the conflicted files before attempting a rebase.")
	}
	fmt.Fprintf(&b, " Upgrade confidence %s (%d). That number is an estimate.", res.Confidence.Level, res.Confidence.Score)
	return b.String()
}

func ResolveUpstreamCandidate(ctx context.Context, repo *git.Repository, remotes []git.Remote, identity string) (string, error) {
	c := upstream.Candidate{Identity: identity}
	ref, _, err := upstream.ResolveRef(ctx, repo, remotes, c)
	return ref, err
}
