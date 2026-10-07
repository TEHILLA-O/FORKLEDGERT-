package scanner

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/forkledger/forkledger/internal/config"
	"github.com/forkledger/forkledger/internal/evidence"
	"github.com/forkledger/forkledger/internal/fingerprint"
	"github.com/forkledger/forkledger/internal/git"
	"github.com/forkledger/forkledger/internal/logging"
	"github.com/forkledger/forkledger/internal/patches"
	"github.com/forkledger/forkledger/internal/report"
	"github.com/forkledger/forkledger/internal/security"
	"github.com/forkledger/forkledger/internal/upstream"
)

type Options struct {
	Path     string
	Config   config.Config
	Offline  bool
	Fetch    bool
	GitHubToken string
	Log      *logging.Logger
}

type Engine struct {
	Git     *git.Executor
	GitHub  *upstream.GitHubClient
	OSV     *security.OSV
	Log     *logging.Logger
}

func New(log *logging.Logger) *Engine {
	return &Engine{
		Git:    git.NewExecutor(log),
		GitHub: upstream.NewGitHubClient(os.Getenv("FORKLEDGER_GITHUB_TOKEN")),
		OSV:    security.NewOSV(),
		Log:    log,
	}
}

func (e *Engine) Analyze(ctx context.Context, opt Options) (*report.Report, error) {
	started := time.Now()
	log := opt.Log
	if log == nil {
		log = e.Log
	}
	if opt.GitHubToken != "" && e.GitHub != nil {
		e.GitHub.Token = opt.GitHubToken
	}

	repo, err := git.Open(ctx, opt.Path, e.Git)
	if err != nil {
		return nil, err
	}
	log.Info("inspecting %s", repo.WorkTree)

	head, err := repo.Head(ctx)
	if err != nil {
		return nil, fmt.Errorf("HEAD: %w", err)
	}
	branch, _ := repo.SymbolicHead(ctx)
	remotes, err := repo.Remotes(ctx)
	if err != nil {
		return nil, err
	}

	rep := &report.Report{
		SchemaVersion: report.SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		Repository: report.RepositoryInfo{
			Path:   repo.WorkTree,
			Head:   head,
			Branch: branch,
		},
	}
	for _, r := range remotes {
		rep.Repository.Remotes = append(rep.Repository.Remotes, report.Remote{
			Name: r.Name,
			URL:  r.URL,
		})
	}

	det, err := (&upstream.Detector{GitHub: e.GitHub, Log: log, Offline: opt.Offline}).Detect(ctx, repo, remotes, opt.Config)
	if err != nil {
		return nil, err
	}
	rep.Upstream = report.UpstreamSection{
		Uncertain:   det.Uncertain,
		Explanation: det.Explanation,
	}
	for _, c := range det.Candidates {
		rep.Upstream.Candidates = append(rep.Upstream.Candidates, report.UpstreamCandidate{
			Identity:   c.Identity,
			URL:        c.URL,
			Remote:     c.Remote,
			Confidence: c.Confidence,
			Reasoning:  c.Reasoning,
		})
	}
	if det.Selected != nil {
		rep.Upstream.Selected = det.Selected.Identity
		rep.Upstream.Confidence = det.Selected.Confidence
	}

	if det.Selected != nil {
		ref, refEv, err := upstream.ResolveRef(ctx, repo, remotes, *det.Selected)
		if err != nil {
			rep.Warnings = append(rep.Warnings, err.Error())
			log.Info("%v", err)
		} else {
			rep.Upstream.Ref = ref
			rep.Upstream.Confidence.Evidence = append(rep.Upstream.Confidence.Evidence, refEv...)
			if err := e.analyzeLineage(ctx, repo, head, ref, opt, rep); err != nil {
				return nil, err
			}
		}
	}

	if !opt.Offline && e.OSV != nil && (rep.Lineage.ForkPoint != "" || head != "") {
		advisories := e.queryAdvisories(ctx, repo, remotes, det, rep)
		rep.Security.Findings = security.Interpret(advisories, toPatches(rep), firstNonEmpty(opt.Config.ExpectedBaseline, rep.Lineage.Baseline), opt.Config)
		rep.Security.MissingUpstreamFixes = len(security.MissingFixes(rep.Security.Findings))
		rep.Security.NeedsInvestigation = len(security.NeedsInvestigation(rep.Security.Findings))
		rep.Security.VEX = security.FromFindings(rep.Repository.Path, rep.Security.Findings)
	} else if opt.Offline {
		rep.Warnings = append(rep.Warnings, "offline mode skipped OSV queries")
	}

	sigs, err := fingerprint.DefaultRegistry().Compute(ctx, repo.WorkTree)
	if err != nil {
		log.Verbose("fingerprints skipped: %v", err)
	} else {
		for _, s := range sigs {
			if s.Ready {
				rep.Fingerprints = append(rep.Fingerprints, report.Fingerprint{
					Kind:    string(s.Kind),
					Value:   s.Value,
					Note:    s.Note,
					Details: s.Details,
				})
			}
		}
	}

	rep.Health = ScoreHealth(rep, opt.Config)
	rep.Debt = ScoreDebt(rep, opt.Config)
	rep.Lifecycle = Recommend(rep)
	rep.Summary = BuildSummary(rep)
	rep.Duration = time.Since(started).String()
	return rep, nil
}

func (e *Engine) analyzeLineage(ctx context.Context, repo *git.Repository, head, upstreamRef string, opt Options, rep *report.Report) error {
	log := e.Log
	fp, err := upstream.EstimateForkPoint(ctx, repo, head, upstreamRef)
	if err != nil {
		rep.Warnings = append(rep.Warnings, err.Error())
		rep.Lineage.Confidence = fp.Confidence
		return nil
	}
	rep.Lineage.ForkPoint = fp.Commit
	rep.Lineage.MergeBases = fp.MergeBases
	rep.Lineage.Baseline = fp.Baseline
	if opt.Config.ExpectedBaseline != "" {
		rep.Lineage.ExpectedBaseline = opt.Config.ExpectedBaseline
		if fp.Baseline != "" && stripV(fp.Baseline) != stripV(opt.Config.ExpectedBaseline) {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("configured baseline %s does not match nearest tag %s", opt.Config.ExpectedBaseline, fp.Baseline))
		}
		if rep.Lineage.Baseline == "" {
			rep.Lineage.Baseline = opt.Config.ExpectedBaseline
		}
	}
	rep.Lineage.Confidence = fp.Confidence

	if tag, err := repo.DescribeNearestTag(ctx, upstreamRef); err == nil {
		rep.Lineage.CurrentUpstream = tag
	}

	count, err := repo.CountLeftRight(ctx, upstreamRef, "HEAD")
	if err == nil {
		rep.Lineage.Ahead = count.Ahead
		rep.Lineage.Behind = count.Behind
	}

	localRange := fp.Commit + "..HEAD"
	localCommits, err := repo.Log(ctx, localRange)
	if err != nil {
		return fmt.Errorf("log %s: %w", localRange, err)
	}
	inv, err := patches.Build(ctx, repo, localCommits, opt.Config, log)
	if err != nil {
		return err
	}

	upRange := fp.Commit + ".." + upstreamRef
	upCommits, err := repo.Log(ctx, upRange)
	if err != nil {
		return fmt.Errorf("log %s: %w", upRange, err)
	}
	rep.UpstreamActivity = summariseActivity(upCommits)

	upForEq := make([]patches.UpstreamCommit, 0, len(upCommits))
	upHashes := make([]string, 0, len(upCommits))
	for _, c := range upCommits {
		if len(c.Parents) > 1 {
			continue
		}
		upHashes = append(upHashes, c.Hash)
		files, _ := repo.CommitNumStat(ctx, c.Hash)
		patch, _ := repo.CommitPatch(ctx, c.Hash)
		upForEq = append(upForEq, patches.UpstreamCommit{
			Commit:    c,
			Files:     fileNames(files),
			Functions: patches.UniqueFunctions(git.ParseHunks(patch)),
			Subject:   c.Subject,
		})
	}
	if ids, err := repo.PatchIDs(ctx, upHashes); err == nil {
		by := map[string]string{}
		for i, id := range ids {
			c := id.Commit
			if c == "" && i < len(upHashes) {
				c = upHashes[i]
			}
			by[c] = id.ID
		}
		for i := range upForEq {
			upForEq[i].PatchID = by[upForEq[i].Commit.Hash]
		}
	}

	cherry, err := repo.Cherry(ctx, upstreamRef, "HEAD")
	if err != nil {
		log.Verbose("git cherry skipped: %v", err)
	}
	inv = patches.DetectEquivalence(inv, upForEq, cherry, opt.Config)

	localFiles, _ := repo.NameOnlyDiff(ctx, fp.Commit, "HEAD")
	upFiles, _ := repo.NameOnlyDiff(ctx, fp.Commit, upstreamRef)
	both := intersect(localFiles, upFiles)
	inv = patches.MarkConflicts(inv, both)

	if mt, err := repo.MergeTree(ctx, "HEAD", upstreamRef); err == nil {
		rep.Upgrade.MergeTreeClean = mt.Clean
		rep.Upgrade.ConflictedFiles = mt.Conflicted
		if !mt.Clean {
			rep.Upgrade.Notes = append(rep.Upgrade.Notes, "git merge-tree reports conflicts between HEAD and "+upstreamRef)
		}
	}

	stats, _ := repo.NumStat(ctx, fp.Commit, "HEAD")
	added, deleted := 0, 0
	for _, s := range stats {
		added += s.Added
		deleted += s.Deleted
	}

	rep.Patches.Items = toReportPatches(inv)
	rep.Patches.Count = len(inv)
	rep.Patches.Added = added
	rep.Patches.Deleted = deleted
	for _, p := range inv {
		if p.Equivalence != nil && p.Equivalence.Confidence.Actionable() {
			rep.Patches.LikelyEquivalents++
		} else if p.Equivalence != nil {
			rep.Patches.PossibleEquivalents++
		}
		if p.Status == patches.StatusRedundant || (p.Equivalence != nil && p.Equivalence.Confidence.Level == evidence.LevelVeryHigh) {
			rep.Patches.PotentiallyObsolete++
		}
		if p.ConflictLikely {
			rep.Patches.PotentialConflicts++
		}
	}
	return nil
}

func (e *Engine) queryAdvisories(ctx context.Context, repo *git.Repository, remotes []git.Remote, det upstream.Detection, rep *report.Report) []security.Advisory {
	var all []security.Advisory
	seen := map[string]struct{}{}
	add := func(list []security.Advisory, err error) {
		if err != nil {
			e.Log.Verbose("osv query: %v", err)
			rep.Warnings = append(rep.Warnings, "OSV query failed: "+err.Error())
			return
		}
		for _, a := range list {
			if _, ok := seen[a.ID]; ok {
				continue
			}
			seen[a.ID] = struct{}{}
			all = append(all, a)
		}
	}
	if rep.Lineage.ForkPoint != "" {
		add(e.OSV.Query(ctx, security.Query{Commit: rep.Lineage.ForkPoint}))
	}
	if det.Selected != nil && rep.Lineage.Baseline != "" {
		url := det.Selected.URL
		if url == "" {
			url = "https://" + det.Selected.Identity
		}
		add(e.OSV.Query(ctx, security.Query{RepoURL: url, Version: rep.Lineage.Baseline}))
	}
	_ = repo
	_ = remotes
	return all
}

func toPatches(rep *report.Report) []patches.Patch {
	out := make([]patches.Patch, 0, len(rep.Patches.Items))
	for _, p := range rep.Patches.Items {
		item := patches.Patch{
			ID:               p.ID,
			Commit:           p.Commit,
			Subject:          p.Subject,
			Category:         patches.Category(p.Category),
			Status:           patches.Status(p.Status),
			Added:            p.Added,
			Deleted:          p.Deleted,
			Functions:        p.Functions,
			ConflictLikely:   p.ConflictLikely,
			SecurityRelevant: p.SecurityRelevant,
			Evidence:         p.Evidence,
		}
		if p.Equivalence != nil {
			item.Equivalence = &patches.Equivalence{
				UpstreamCommit:  p.Equivalence.UpstreamCommit,
				UpstreamSubject: p.Equivalence.UpstreamSubject,
				Confidence:      p.Equivalence.Confidence,
				Layers:          p.Equivalence.Layers,
			}
		}
		out = append(out, item)
	}
	return out
}

func toReportPatches(inv []patches.Patch) []report.Patch {
	out := make([]report.Patch, 0, len(inv))
	for _, p := range inv {
		item := report.Patch{
			ID:               p.ID,
			Commit:           p.Commit,
			Author:           p.Author,
			Date:             p.Date,
			Subject:          p.Subject,
			PatchID:          p.PatchID,
			Category:         string(p.Category),
			CategoryReason:   p.CategoryReason,
			Status:           string(p.Status),
			Added:            p.Added,
			Deleted:          p.Deleted,
			Functions:        p.Functions,
			ConflictLikely:   p.ConflictLikely,
			ConflictReason:   p.ConflictReason,
			SecurityRelevant: p.SecurityRelevant,
			Evidence:         p.Evidence,
		}
		for _, f := range p.Files {
			item.Files = append(item.Files, f.Path)
		}
		if p.Equivalence != nil {
			item.Equivalence = &report.PatchEquivalence{
				UpstreamCommit:  p.Equivalence.UpstreamCommit,
				UpstreamSubject: p.Equivalence.UpstreamSubject,
				Confidence:      p.Equivalence.Confidence,
				Layers:          p.Equivalence.Layers,
			}
		}
		out = append(out, item)
	}
	return out
}

func summariseActivity(commits []git.Commit) report.Activity {
	a := report.Activity{Commits: len(commits)}
	if len(commits) == 0 {
		return a
	}
	a.Newest = commits[len(commits)-1].Date
	a.Oldest = commits[0].Date
	securityish := 0
	for _, c := range commits {
		blob := strings.ToLower(c.Subject + " " + c.Body)
		if strings.Contains(blob, "cve-") || strings.Contains(blob, "secur") || strings.Contains(blob, "vulnerab") {
			securityish++
		}
	}
	a.SecurityMentioning = securityish
	return a
}

func fileNames(files []git.FileChange) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

func intersect(a, b []string) []string {
	sb := map[string]struct{}{}
	for _, x := range b {
		sb[x] = struct{}{}
	}
	var out []string
	for _, x := range a {
		if _, ok := sb[x]; ok {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

func stripV(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
