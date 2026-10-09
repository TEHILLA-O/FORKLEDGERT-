package upstream

import (
	"context"
	"fmt"
	"strings"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/evidence"
	"github.com/TEHILLA-O/forkledger/internal/git"
	"github.com/TEHILLA-O/forkledger/internal/logging"
)

type Candidate struct {
	Identity   string               `json:"identity"`
	URL        string               `json:"url,omitempty"`
	Remote     string               `json:"remote,omitempty"`
	Ref        string               `json:"ref,omitempty"`
	Confidence evidence.Assessment  `json:"confidence"`
	Reasoning  string               `json:"reasoning"`
}

type Detection struct {
	Selected    *Candidate  `json:"selected,omitempty"`
	Candidates  []Candidate `json:"candidates"`
	Uncertain   bool        `json:"uncertain"`
	Explanation string      `json:"explanation"`
}

type Detector struct {
	GitHub  *GitHubClient
	Log     *logging.Logger
	Offline bool
}

func (d *Detector) Detect(ctx context.Context, repo *git.Repository, remotes []git.Remote, cfg config.Config) (Detection, error) {
	type acc struct {
		identity string
		url      string
		remote   string
		items    []evidence.Item
	}
	byID := map[string]*acc{}
	order := []string{}

	add := func(identity, url, remote string, item evidence.Item) {
		identity = NormalizeRemote(identity)
		if identity == "" {
			return
		}
		a, ok := byID[identity]
		if !ok {
			a = &acc{identity: identity, url: url, remote: remote}
			byID[identity] = a
			order = append(order, identity)
		}
		if a.url == "" {
			a.url = url
		}
		if a.remote == "" {
			a.remote = remote
		}
		a.items = append(a.items, item)
	}

	if strings.TrimSpace(cfg.Upstream) != "" {
		add(cfg.Upstream, cfg.Upstream, "",
			evidence.New(evidence.KindConfiguredUpstream, "forkledger.yaml names this upstream").
				WithTarget(cfg.Upstream).WithWeight(80))
	}

	for _, rem := range remotes {
		name := strings.ToLower(rem.Name)
		identity := NormalizeRemote(rem.URL)
		switch name {
		case "upstream":
			add(identity, rem.URL, rem.Name,
				evidence.New(evidence.KindRemoteName, "git remote named upstream").
					WithSource(rem.Name).WithTarget(identity).WithWeight(70))
		case "origin":
			add(identity, rem.URL, rem.Name,
				evidence.New(evidence.KindRemoteName, "git remote named origin").
					WithSource(rem.Name).WithTarget(identity).WithWeight(8))
		default:
			add(identity, rem.URL, rem.Name,
				evidence.New(evidence.KindRemoteName, "git remote "+rem.Name).
					WithSource(rem.Name).WithTarget(identity).WithWeight(4))
		}
	}

	if meta, err := readPackageSignals(ctx, repo); err == nil {
		for _, s := range meta {
			add(s.Identity, s.URL, "",
				evidence.New(evidence.KindPackageMetadata, s.Summary).
					WithTarget(s.Identity).WithWeight(s.Weight))
		}
	}

	if !d.Offline && d.GitHub != nil {
		for _, rem := range remotes {
			gh, ok := ParseGitHub(rem.URL)
			if !ok {
				continue
			}
			info, err := d.GitHub.Repo(ctx, gh.Owner, gh.Name)
			if err != nil {
				if d.Log != nil {
					d.Log.Verbose("github metadata for %s skipped: %v", gh.String(), err)
				}
				continue
			}
			if info.Fork && info.Parent != nil {
				parent := NormalizeRemote(info.Parent.FullName)
				add(parent, info.Parent.CloneURL, rem.Name,
					evidence.New(evidence.KindGitHubForkMetadata, "GitHub reports this repository as a fork of "+parent).
						WithSource(gh.String()).WithTarget(parent).WithWeight(75))
			}
		}
	}

	var candidates []Candidate
	for _, id := range order {
		a := byID[id]
		conf := evidence.FromWeights("Weighted evidence from remotes, configuration, package metadata, and GitHub fork records.", a.items)
		candidates = append(candidates, Candidate{
			Identity:   a.identity,
			URL:        a.url,
			Remote:     a.remote,
			Confidence: conf,
			Reasoning:  summarise(a.items),
		})
	}
	sortCandidates(candidates)

	det := Detection{Candidates: candidates}
	if len(candidates) == 0 {
		det.Uncertain = true
		det.Explanation = "No upstream candidate was found. Name one in forkledger.yaml or add a git remote."
		return det, nil
	}
	top := candidates[0]
	if top.Confidence.Actionable() {
		selected := top
		det.Selected = &selected
		det.Explanation = fmt.Sprintf("Selected %s with %s confidence.", selected.Identity, selected.Confidence.Level)
		if len(candidates) > 1 && candidates[1].Confidence.Score >= top.Confidence.Score-10 {
			det.Uncertain = true
			det.Explanation = fmt.Sprintf("Top candidates %s and %s are too close to select silently.", top.Identity, candidates[1].Identity)
			det.Selected = nil
		}
		return det, nil
	}
	det.Uncertain = true
	det.Explanation = fmt.Sprintf("Best candidate is %s at %s confidence. ForkLedger will not pretend that is settled.", top.Identity, top.Confidence.Level)
	return det, nil
}

type packageSignal struct {
	Identity string
	URL      string
	Summary  string
	Weight   int
}

func readPackageSignals(ctx context.Context, repo *git.Repository) ([]packageSignal, error) {
	var out []packageSignal
	if repo.FileExists(ctx, "HEAD", "go.mod") {
		text, err := repo.FileContents(ctx, "HEAD", "go.mod")
		if err == nil {
			if mod, ok := parseGoModule(text); ok {
				if gh, ok := ParseGitHub("https://" + mod); ok {
					out = append(out, packageSignal{
						Identity: gh.String(),
						URL:      gh.HTTPS(),
						Summary:  "go.mod module path " + mod,
						Weight:   12,
					})
				}
			}
		}
	}
	if repo.FileExists(ctx, "HEAD", "package.json") {
		text, err := repo.FileContents(ctx, "HEAD", "package.json")
		if err == nil {
			if id, summary, ok := parsePackageJSONRepo(text); ok {
				out = append(out, packageSignal{Identity: id, URL: "https://" + id + ".git", Summary: summary, Weight: 12})
			}
		}
	}
	return out, nil
}

func parseGoModule(text string) (string, bool) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), true
		}
	}
	return "", false
}

func parsePackageJSONRepo(text string) (string, string, bool) {
	// Keep this intentionally dumb. A full JSON parse can wait. The repository
	// field is usually a github URL sitting in plain sight.
	for _, key := range []string{"\"url\"", "\"repository\""} {
		idx := strings.Index(text, key)
		if idx < 0 {
			continue
		}
		slice := text[idx:]
		if gh, ok := extractGitHub(slice); ok {
			return gh.String(), "package.json repository field", true
		}
	}
	return "", "", false
}

func extractGitHub(s string) (GitHubRepo, bool) {
	idx := strings.Index(strings.ToLower(s), "github.com")
	if idx < 0 {
		return GitHubRepo{}, false
	}
	rest := s[idx:]
	end := strings.IndexAny(rest, "\"'\n ")
	if end > 0 {
		rest = rest[:end]
	}
	return ParseGitHub("https://" + rest)
}

func summarise(items []evidence.Item) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, it.Summary)
	}
	return strings.Join(parts, "; ")
}

func sortCandidates(cs []Candidate) {
	for i := 0; i < len(cs); i++ {
		for j := i + 1; j < len(cs); j++ {
			if cs[j].Confidence.Score > cs[i].Confidence.Score {
				cs[i], cs[j] = cs[j], cs[i]
			}
		}
	}
}

func ResolveRef(ctx context.Context, repo *git.Repository, remotes []git.Remote, candidate Candidate) (string, []evidence.Item, error) {
	var items []evidence.Item
	if candidate.Remote != "" {
		for _, name := range []string{"main", "master", "trunk", "develop"} {
			rev := candidate.Remote + "/" + name
			if hash, err := repo.RevParse(ctx, rev); err == nil {
				items = append(items, evidence.New(evidence.KindGitAncestry, "resolved "+rev).WithTarget(hash).WithWeight(5))
				return rev, items, nil
			}
		}
	}
	for _, rem := range remotes {
		if NormalizeRemote(rem.URL) != candidate.Identity && rem.Name != candidate.Remote {
			continue
		}
		for _, name := range []string{"main", "master", "trunk"} {
			rev := rem.Name + "/" + name
			if hash, err := repo.RevParse(ctx, rev); err == nil {
				items = append(items, evidence.New(evidence.KindGitAncestry, "resolved "+rev).WithTarget(hash).WithWeight(5))
				return rev, items, nil
			}
		}
	}
	return "", items, fmt.Errorf("no local remote-tracking branch for %s. Fetch the upstream remote first", candidate.Identity)
}
