package git

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (r *Repository) Fetch(ctx context.Context, remote string) error {
	if err := rejectUnsafeRev(remote); err != nil {
		return err
	}
	// Fetch updates remote-tracking refs. It does not checkout the working tree.
	// Large histories still have to download objects, so this call gets a longer leash.
	old := r.Exec.Timeout
	r.Exec.Timeout = 5 * time.Minute
	defer func() { r.Exec.Timeout = old }()
	_, err := r.run(ctx, "fetch", "--tags", "--prune", remote)
	return err
}

type CherryPickResult struct {
	Clean     bool     `json:"clean"`
	Empty     bool     `json:"empty"`
	Conflict  bool     `json:"conflict"`
	Files     []string `json:"files,omitempty"`
	Message   string   `json:"message,omitempty"`
}

func (r *Repository) CherryPick(ctx context.Context, commit string) CherryPickResult {
	if err := rejectUnsafeRev(commit); err != nil {
		return CherryPickResult{Conflict: true, Message: err.Error()}
	}
	res, err := r.run(ctx, "cherry-pick", "--allow-empty", "--keep-redundant-commits", commit)
	if err == nil {
		msg := strings.ToLower(res.Stderr + res.Stdout)
		empty := strings.Contains(msg, "empty") || strings.Contains(msg, "previously applied")
		return CherryPickResult{Clean: true, Empty: empty, Message: strings.TrimSpace(res.Stderr)}
	}
	files := r.unmerged(ctx)
	_ = r.CherryPickAbort(ctx)
	return CherryPickResult{
		Conflict: true,
		Files:    files,
		Message:  strings.TrimSpace(res.Stderr),
	}
}

func (r *Repository) CherryPickAbort(ctx context.Context) error {
	_, err := r.run(ctx, "cherry-pick", "--abort")
	return err
}

func (r *Repository) ResetHard(ctx context.Context, rev string) error {
	if err := rejectUnsafeRev(rev); err != nil {
		return err
	}
	_, err := r.run(ctx, "reset", "--hard", rev)
	return err
}

func (r *Repository) unmerged(ctx context.Context) []string {
	res, err := r.run(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil
	}
	return splitLines(res.Stdout)
}

func (r *Repository) RemoteExists(ctx context.Context, name string) bool {
	rems, err := r.Remotes(ctx)
	if err != nil {
		return false
	}
	for _, rem := range rems {
		if rem.Name == name {
			return true
		}
	}
	return false
}

func LooksEmptyCherry(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "empty") || strings.Contains(m, "nothing to commit")
}

func FetchRemoteName(remotes []Remote, preferred string) string {
	if preferred != "" {
		for _, r := range remotes {
			if r.Name == preferred {
				return r.Name
			}
		}
	}
	for _, r := range remotes {
		if strings.EqualFold(r.Name, "upstream") {
			return r.Name
		}
	}
	return ""
}

func DescribeFetch(remote string) string {
	return fmt.Sprintf("fetched remote %s (remote-tracking refs only)", remote)
}
