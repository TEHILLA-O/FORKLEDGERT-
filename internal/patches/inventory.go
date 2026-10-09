package patches

import (
	"context"
	"fmt"
	"strings"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/evidence"
	"github.com/TEHILLA-O/forkledger/internal/git"
	"github.com/TEHILLA-O/forkledger/internal/logging"
)

type InventoryOptions struct {
	IgnoredPaths []string
}

func Build(ctx context.Context, repo *git.Repository, commits []git.Commit, cfg config.Config, log *logging.Logger) ([]Patch, error) {
	out := make([]Patch, 0, len(commits))
	overrides := map[string]config.PatchClassification{}
	for _, pc := range cfg.PatchClassifications {
		overrides[strings.ToLower(pc.Commit)] = pc
	}

	for _, c := range commits {
		if len(c.Parents) > 1 {
			if log != nil {
				log.Verbose("skipping merge commit %s", short(c.Hash))
			}
			continue
		}
		files, err := repo.CommitNumStat(ctx, c.Hash)
		if err != nil {
			return nil, fmt.Errorf("numstat %s: %w", short(c.Hash), err)
		}
		files = filterIgnored(files, cfg.IgnoredPaths)
		patch, err := repo.CommitPatch(ctx, c.Hash)
		if err != nil {
			return nil, fmt.Errorf("show %s: %w", short(c.Hash), err)
		}
		hunks := git.ParseHunks(patch)
		fns := UniqueFunctions(hunks)
		added, deleted := 0, 0
		for _, f := range files {
			added += f.Added
			deleted += f.Deleted
		}
		cat, reason := ClassifyMessage(c.Subject, c.Body)
		status := StatusActive
		ev := []evidence.Item{
			evidence.New(evidence.KindCommitMessage, reason).WithSource(c.Hash),
		}
		if ov, ok := matchOverride(overrides, c.Hash); ok {
			if ov.Category != "" {
				cat = Category(ov.Category)
				reason = "configured classification"
			}
			if ov.Status != "" {
				status = Status(ov.Status)
			}
			if ov.Reason != "" {
				reason = ov.Reason
			}
			ev = append(ev, evidence.New(evidence.KindManualOverride, "forkledger.yaml classified this commit").WithSource(c.Hash))
		}
		p := Patch{
			ID:               "sha:" + c.Hash,
			Commit:           c.Hash,
			Parents:          c.Parents,
			Author:           c.Author,
			Email:            c.Email,
			Date:             c.Date,
			Subject:          c.Subject,
			Body:             c.Body,
			Files:            files,
			Functions:        fns,
			Added:            added,
			Deleted:          deleted,
			Category:         cat,
			CategoryReason:   reason,
			Status:           status,
			SecurityRelevant: cat == CategorySecurity,
			Evidence:         ev,
			IssueIDs:         IssueIDs(c.Subject, c.Body),
			Tokens:           TokenizePatch(patch),
			ASTFuncs:         goFuncsAt(ctx, repo, c.Hash, files),
		}
		if LooksSecurity(p) {
			p.SecurityRelevant = true
		}
		out = append(out, p)
	}

	ids, err := repo.PatchIDs(ctx, hashes(out))
	if err != nil {
		if log != nil {
			log.Verbose("patch-id computation failed: %v", err)
		}
	} else {
		byCommit := map[string]string{}
		for i, id := range ids {
			commit := id.Commit
			if commit == "" && i < len(out) {
				commit = out[i].Commit
			}
			if id.ID != "" {
				byCommit[commit] = id.ID
			}
		}
		for i := range out {
			if pid, ok := byCommit[out[i].Commit]; ok {
				out[i].PatchID = pid
				out[i].ID = pid
			} else if i < len(ids) && ids[i].ID != "" {
				out[i].PatchID = ids[i].ID
				out[i].ID = ids[i].ID
			}
		}
	}
	return out, nil
}

func hashes(patches []Patch) []string {
	out := make([]string, len(patches))
	for i, p := range patches {
		out[i] = p.Commit
	}
	return out
}

func filterIgnored(files []git.FileChange, ignored []string) []git.FileChange {
	if len(ignored) == 0 {
		return files
	}
	var out []git.FileChange
	for _, f := range files {
		if ignoredPath(f.Path, ignored) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func ignoredPath(path string, ignored []string) bool {
	p := strings.ReplaceAll(path, "\\", "/")
	for _, ig := range ignored {
		ig = strings.TrimSuffix(strings.ReplaceAll(ig, "\\", "/"), "/")
		if ig == "" {
			continue
		}
		if p == ig || strings.HasPrefix(p, ig+"/") {
			return true
		}
	}
	return false
}

func matchOverride(overrides map[string]config.PatchClassification, hash string) (config.PatchClassification, bool) {
	h := strings.ToLower(hash)
	if ov, ok := overrides[h]; ok {
		return ov, true
	}
	for key, ov := range overrides {
		if strings.HasPrefix(h, key) || strings.HasPrefix(key, h) {
			return ov, true
		}
	}
	return config.PatchClassification{}, false
}

func goFuncsAt(ctx context.Context, repo *git.Repository, commit string, files []git.FileChange) []string {
	var out []string
	n := 0
	for _, f := range files {
		if n >= 8 {
			break
		}
		if !strings.HasSuffix(strings.ToLower(f.Path), ".go") {
			continue
		}
		src, err := repo.FileContents(ctx, commit, f.Path)
		if err != nil {
			continue
		}
		out = append(out, ParseGoFunctions(src, f.Path)...)
		n++
	}
	return unique(out)
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
