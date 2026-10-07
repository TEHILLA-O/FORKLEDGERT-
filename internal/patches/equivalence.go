package patches

import (
	"fmt"
	"strings"

	"github.com/forkledger/forkledger/internal/config"
	"github.com/forkledger/forkledger/internal/evidence"
	"github.com/forkledger/forkledger/internal/git"
)

type UpstreamCommit struct {
	Commit    git.Commit
	PatchID   string
	Files     []string
	Functions []string
	Subject   string
	IssueIDs  []string
	Tokens    []string
	ASTFuncs  []string
}

func DetectEquivalence(local []Patch, upstream []UpstreamCommit, cherry []git.Cherry, cfg config.Config) []Patch {
	byPatchID := map[string]UpstreamCommit{}
	byHash := map[string]UpstreamCommit{}
	byFileKey := map[string][]UpstreamCommit{}
	for _, u := range upstream {
		byHash[u.Commit.Hash] = u
		if u.PatchID != "" {
			byPatchID[u.PatchID] = u
		}
		key := fileKey(u.Files)
		byFileKey[key] = append(byFileKey[key], u)
	}
	cherryEq := map[string]struct{}{}
	for _, c := range cherry {
		if c.Mark == git.CherryEquivalent {
			cherryEq[c.Commit] = struct{}{}
		}
	}
	manual := map[string]config.KnownEquivalent{}
	for _, k := range cfg.KnownEquivalents {
		manual[strings.ToLower(k.Local)] = k
	}

	for i, p := range local {
		var items []evidence.Item
		layers := []string{}
		var match UpstreamCommit
		found := false

		if k, ok := matchKnown(manual, p.Commit); ok {
			if u, ok := byHash[k.Upstream]; ok {
				match = u
			} else {
				match = UpstreamCommit{Commit: git.Commit{Hash: k.Upstream}, Subject: k.Note}
			}
			found = true
			layers = append(layers, "manual")
			items = append(items, evidence.New(evidence.KindManualOverride, "configured as equivalent in forkledger.yaml").
				WithSource(p.Commit).WithTarget(k.Upstream).WithWeight(40))
		}
		if u, ok := byPatchID[p.PatchID]; ok && p.PatchID != "" {
			match = u
			found = true
			layers = append(layers, "patch-id")
			items = append(items, evidence.New(evidence.KindMatchingPatchID, "stable git patch-id matches an upstream commit").
				WithSource(p.Commit).WithTarget(u.Commit.Hash).WithWeight(75))
		}
		if _, ok := cherryEq[p.Commit]; ok {
			found = true
			layers = append(layers, "cherry")
			items = append(items, evidence.New(evidence.KindCherryEquivalent, "git cherry reports this patch as already upstream").
				WithSource(p.Commit).WithWeight(25))
			if !found || match.Commit.Hash == "" {
				// cherry does not name the upstream twin. The patch-id layer usually does.
			}
		}

		bestOverlap := 0.0
		var overlapHit UpstreamCommit
		localFiles := fileNames(p.Files)
		for _, u := range upstream {
			score := jaccard(localFiles, u.Files)
			if score > bestOverlap {
				bestOverlap = score
				overlapHit = u
			}
		}
		if bestOverlap >= 0.8 && len(localFiles) > 0 {
			layers = append(layers, "file-overlap")
			items = append(items, evidence.New(evidence.KindFileOverlap, fmt.Sprintf("file overlap %.0f%% with upstream %s", bestOverlap*100, short(overlapHit.Commit.Hash))).
				WithSource(p.Commit).WithTarget(overlapHit.Commit.Hash).WithWeight(8))
			fnScore := jaccard(p.Functions, overlapHit.Functions)
			if fnScore >= 0.5 && len(p.Functions) > 0 {
				layers = append(layers, "function-overlap")
				items = append(items, evidence.New(evidence.KindFunctionOverlap, fmt.Sprintf("function overlap %.0f%%", fnScore*100)).
					WithSource(p.Commit).WithTarget(overlapHit.Commit.Hash).WithWeight(10))
			}
			subj := similarSubject(p.Subject, overlapHit.Commit.Subject)
			if subj {
				layers = append(layers, "commit-message")
				items = append(items, evidence.New(evidence.KindCommitMessage, "subjects look like the same change").
					WithSource(p.Commit).WithTarget(overlapHit.Commit.Hash).WithWeight(6))
			}
			if !found && bestOverlap >= 0.8 && (fnScore >= 0.5 || subj) {
				match = overlapHit
				found = true
			}
		}

		bestTok := 0.0
		var tokHit UpstreamCommit
		for _, u := range upstream {
			if len(p.Tokens) == 0 || len(u.Tokens) == 0 {
				continue
			}
			score := TokenSimilarity(p.Tokens, u.Tokens)
			if score > bestTok {
				bestTok = score
				tokHit = u
			}
		}
		if bestTok >= 0.7 {
			layers = append(layers, "normalized-diff")
			items = append(items, evidence.New(evidence.KindNormalizedDiff, fmt.Sprintf("normalised diff tokens overlap %.0f%%", bestTok*100)).
				WithSource(p.Commit).WithTarget(tokHit.Commit.Hash).WithWeight(12))
			if !found {
				match = tokHit
				found = true
			}
		}

		for _, u := range upstream {
			if shared := intersectStrings(p.IssueIDs, u.IssueIDs); len(shared) > 0 {
				layers = append(layers, "issue-id")
				items = append(items, evidence.New(evidence.KindSharedIssueID, "shared issue identifiers: "+strings.Join(shared, ", ")).
					WithSource(p.Commit).WithTarget(u.Commit.Hash).WithWeight(18))
				if !found {
					match = u
					found = true
				}
				break
			}
		}

		if len(p.ASTFuncs) > 0 {
			bestAST := 0.0
			var astHit UpstreamCommit
			for _, u := range upstream {
				score := jaccard(p.ASTFuncs, u.ASTFuncs)
				if score > bestAST {
					bestAST = score
					astHit = u
				}
			}
			if bestAST >= 0.6 {
				layers = append(layers, "ast")
				items = append(items, evidence.New(evidence.KindASTSimilarity, fmt.Sprintf("Go function set overlap %.0f%%", bestAST*100)).
					WithSource(p.Commit).WithTarget(astHit.Commit.Hash).WithWeight(10))
				if !found && bestAST >= 0.8 {
					match = astHit
					found = true
				}
			}
		}

		if !found {
			continue
		}
		conf := evidence.FromWeights("Layers are independent evidence, not a vote by a model.", items)
		if !conf.Actionable() && conf.Level != evidence.LevelModerate {
			continue
		}
		status := StatusUpstreamEquivalent
		if conf.Level == evidence.LevelModerate {
			// Possible twin, not a licence to delete the patch.
		} else {
			local[i].Status = status
		}
		local[i].Equivalence = &Equivalence{
			UpstreamCommit:  match.Commit.Hash,
			UpstreamSubject: firstNonEmpty(match.Subject, match.Commit.Subject),
			Confidence:      conf,
			Layers:          unique(layers),
		}
		local[i].Evidence = append(local[i].Evidence, items...)
	}
	return local
}

func matchKnown(manual map[string]config.KnownEquivalent, hash string) (config.KnownEquivalent, bool) {
	h := strings.ToLower(hash)
	if k, ok := manual[h]; ok {
		return k, true
	}
	for key, k := range manual {
		if strings.HasPrefix(h, key) || strings.HasPrefix(key, h) {
			return k, true
		}
	}
	return config.KnownEquivalent{}, false
}

func fileNames(files []git.FileChange) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

func fileKey(files []string) string {
	return strings.Join(files, "|")
}

func jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	sa := map[string]struct{}{}
	for _, x := range a {
		sa[x] = struct{}{}
	}
	inter := 0
	sb := map[string]struct{}{}
	for _, x := range b {
		sb[x] = struct{}{}
		if _, ok := sa[x]; ok {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func similarSubject(a, b string) bool {
	na := normalizeSubject(a)
	nb := normalizeSubject(b)
	if na == "" || nb == "" {
		return false
	}
	return na == nb || strings.Contains(na, nb) || strings.Contains(nb, na)
}

func normalizeSubject(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"fix:", "feat:", "security:", "chore:", "patch:"} {
		s = strings.TrimPrefix(s, prefix)
		s = strings.TrimSpace(s)
	}
	return s
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func intersectStrings(a, b []string) []string {
	sb := map[string]struct{}{}
	for _, x := range b {
		sb[strings.ToUpper(x)] = struct{}{}
	}
	var out []string
	for _, x := range a {
		if _, ok := sb[strings.ToUpper(x)]; ok {
			out = append(out, x)
		}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func MarkConflicts(local []Patch, bothChanged []string) []Patch {
	set := map[string]struct{}{}
	for _, f := range bothChanged {
		set[f] = struct{}{}
	}
	for i, p := range local {
		var hit []string
		for _, f := range p.Files {
			if _, ok := set[f.Path]; ok {
				hit = append(hit, f.Path)
			}
		}
		if len(hit) == 0 {
			continue
		}
		local[i].ConflictLikely = true
		local[i].ConflictReason = "also modified upstream since the fork point: " + strings.Join(hit, ", ")
		if local[i].Status == StatusActive {
			local[i].Status = StatusBlocksUpgrade
		}
		local[i].Evidence = append(local[i].Evidence,
			evidence.New(evidence.KindMergeTree, local[i].ConflictReason).WithSource(p.Commit).WithWeight(5))
	}
	return local
}
