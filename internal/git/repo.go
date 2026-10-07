package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Remote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Role string `json:"role,omitempty"`
}

type Ref struct {
	Name   string `json:"name"`
	Hash   string `json:"hash"`
	Peeled string `json:"peeled,omitempty"`
}

type Commit struct {
	Hash    string    `json:"hash"`
	Parents []string  `json:"parents,omitempty"`
	Author  string    `json:"author,omitempty"`
	Email   string    `json:"email,omitempty"`
	Date    time.Time `json:"date"`
	Subject string    `json:"subject"`
	Body    string    `json:"body,omitempty"`
}

type CherryMark string

const (
	CherryEquivalent CherryMark = "equivalent"
	CherryNotPicked  CherryMark = "not_picked"
)

type Cherry struct {
	Commit string     `json:"commit"`
	Mark   CherryMark `json:"mark"`
	Subject string    `json:"subject,omitempty"`
}

type FileChange struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Added     int    `json:"added"`
	Deleted   int    `json:"deleted"`
}

type Hunk struct {
	OldStart int    `json:"old_start"`
	OldCount int    `json:"old_count"`
	NewStart int    `json:"new_start"`
	NewCount int    `json:"new_count"`
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
}

type PatchID struct {
	ID     string `json:"id"`
	Commit string `json:"commit"`
}

type MergeTreeResult struct {
	Clean         bool     `json:"clean"`
	Conflicted    []string `json:"conflicted,omitempty"`
	Messages      string   `json:"messages,omitempty"`
	Informational string   `json:"informational,omitempty"`
}

type Count struct {
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`
}

type Repository struct {
	WorkTree string
	GitDir   string
	Exec     *Executor
}

func Open(ctx context.Context, path string, exec *Executor) (*Repository, error) {
	if exec == nil {
		exec = NewExecutor(nil)
	}
	if err := EnsureEmptyHooks(); err != nil {
		return nil, err
	}
	res, err := exec.Run(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}
	root := filepath.Clean(res.Output())
	gd, err := exec.Run(ctx, root, "rev-parse", "--git-dir")
	if err != nil {
		return nil, fmt.Errorf("cannot resolve git dir: %w", err)
	}
	gitDir := gd.Output()
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	return &Repository{WorkTree: root, GitDir: gitDir, Exec: exec}, nil
}

func (r *Repository) run(ctx context.Context, args ...string) (Result, error) {
	return r.Exec.Run(ctx, r.WorkTree, args...)
}

func (r *Repository) Head(ctx context.Context) (string, error) {
	res, err := r.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return res.Output(), nil
}

func (r *Repository) SymbolicHead(ctx context.Context) (string, error) {
	res, err := r.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return res.Output(), nil
}

func (r *Repository) RevParse(ctx context.Context, rev string) (string, error) {
	if err := rejectUnsafeRev(rev); err != nil {
		return "", err
	}
	res, err := r.run(ctx, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("unknown revision %q", rev)
	}
	out := res.Output()
	if out == "" {
		return "", fmt.Errorf("unknown revision %q", rev)
	}
	return out, nil
}

func (r *Repository) Remotes(ctx context.Context) ([]Remote, error) {
	res, err := r.run(ctx, "remote", "-v")
	if err != nil {
		return nil, err
	}
	return parseRemotes(res.Stdout), nil
}

func (r *Repository) MergeBase(ctx context.Context, a, b string) (string, error) {
	if err := rejectUnsafeRev(a); err != nil {
		return "", err
	}
	if err := rejectUnsafeRev(b); err != nil {
		return "", err
	}
	res, err := r.run(ctx, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return res.Output(), nil
}

func (r *Repository) MergeBases(ctx context.Context, a, b string) ([]string, error) {
	if err := rejectUnsafeRev(a); err != nil {
		return nil, err
	}
	if err := rejectUnsafeRev(b); err != nil {
		return nil, err
	}
	res, err := r.run(ctx, "merge-base", "--all", a, b)
	if err != nil {
		return nil, err
	}
	return splitLines(res.Stdout), nil
}

func (r *Repository) CountLeftRight(ctx context.Context, left, right string) (Count, error) {
	if err := rejectUnsafeRev(left); err != nil {
		return Count{}, err
	}
	if err := rejectUnsafeRev(right); err != nil {
		return Count{}, err
	}
	spec := left + "..." + right
	res, err := r.run(ctx, "rev-list", "--left-right", "--count", spec)
	if err != nil {
		return Count{}, err
	}
	return parseLeftRightCount(res.Output())
}

func (r *Repository) RevList(ctx context.Context, revRange string) ([]string, error) {
	if err := rejectUnsafeRev(revRange); err != nil {
		return nil, err
	}
	res, err := r.run(ctx, "rev-list", "--reverse", revRange)
	if err != nil {
		return nil, err
	}
	return splitLines(res.Stdout), nil
}

func (r *Repository) Log(ctx context.Context, revRange string) ([]Commit, error) {
	if err := rejectUnsafeRev(revRange); err != nil {
		return nil, err
	}
	const pretty = "%H%x1f%P%x1f%an%x1f%ae%x1f%aI%x1f%s%x1f%b%x1e"
	res, err := r.run(ctx, "log", "--reverse", "--format="+pretty, revRange)
	if err != nil {
		return nil, err
	}
	return parseLog(res.Stdout)
}

func (r *Repository) ForEachRef(ctx context.Context, pattern string) ([]Ref, error) {
	args := []string{"for-each-ref", "--format=%(objectname)%09%(refname)"}
	if pattern != "" {
		if strings.ContainsAny(pattern, ";|&$`\n") {
			return nil, fmt.Errorf("invalid ref pattern")
		}
		args = append(args, pattern)
	}
	res, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseRefs(res.Stdout), nil
}

func (r *Repository) DescribeNearestTag(ctx context.Context, commit string) (string, error) {
	if err := rejectUnsafeRev(commit); err != nil {
		return "", err
	}
	res, err := r.run(ctx, "describe", "--tags", "--abbrev=0", commit)
	if err != nil {
		return "", err
	}
	return res.Output(), nil
}

func (r *Repository) NameOnlyDiff(ctx context.Context, a, b string) ([]string, error) {
	if err := rejectUnsafeRev(a); err != nil {
		return nil, err
	}
	if err := rejectUnsafeRev(b); err != nil {
		return nil, err
	}
	res, err := r.run(ctx, "diff", "--name-only", a, b)
	if err != nil {
		return nil, err
	}
	return splitLines(res.Stdout), nil
}

func (r *Repository) NumStat(ctx context.Context, a, b string) ([]FileChange, error) {
	if err := rejectUnsafeRev(a); err != nil {
		return nil, err
	}
	if err := rejectUnsafeRev(b); err != nil {
		return nil, err
	}
	res, err := r.run(ctx, "diff", "--numstat", a, b)
	if err != nil {
		return nil, err
	}
	return parseNumStat(res.Stdout), nil
}

func (r *Repository) CommitNumStat(ctx context.Context, commit string) ([]FileChange, error) {
	if err := rejectUnsafeRev(commit); err != nil {
		return nil, err
	}
	res, err := r.run(ctx, "show", "--format=", "--numstat", "--find-renames", commit)
	if err != nil {
		return nil, err
	}
	return parseNumStat(res.Stdout), nil
}

func (r *Repository) CommitPatch(ctx context.Context, commit string) (string, error) {
	if err := rejectUnsafeRev(commit); err != nil {
		return "", err
	}
	res, err := r.run(ctx, "show", "--format=", "--patch", "--find-renames", commit)
	if err != nil {
		return "", err
	}
	return res.Stdout, nil
}

func (r *Repository) PatchIDs(ctx context.Context, commits []string) ([]PatchID, error) {
	if len(commits) == 0 {
		return nil, nil
	}
	out := make([]PatchID, 0, len(commits))
	for _, c := range commits {
		if err := rejectUnsafeRev(c); err != nil {
			return nil, err
		}
		patch, err := r.CommitPatch(ctx, c)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(patch) == "" {
			out = append(out, PatchID{Commit: c})
			continue
		}
		ids, err := r.patchIDStdin(ctx, patch, []string{c})
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			out = append(out, PatchID{Commit: c})
			continue
		}
		ids[0].Commit = c
		out = append(out, ids[0])
	}
	return out, nil
}

func (r *Repository) patchIDStdin(ctx context.Context, patches string, commits []string) ([]PatchID, error) {
	if strings.TrimSpace(patches) == "" {
		return nil, nil
	}
	timeout := r.Exec.timeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := commandWithEnv(r.Exec, ctx, r.WorkTree, append(safetyArgs(), "patch-id", "--stable")...)
	cmd.Stdin = strings.NewReader(patches)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git patch-id: %w", err)
	}
	ids := parsePatchIDs(string(out))
	if len(ids) != len(commits) && len(ids) > 0 {
		// Git emits one id per detected patch. Merge commits can produce none.
		// We align by order for the ones we have and leave the rest unmatched.
	}
	for i := range ids {
		if i < len(commits) && ids[i].Commit == "" {
			ids[i].Commit = commits[i]
		}
	}
	return ids, nil
}

func (r *Repository) Cherry(ctx context.Context, upstream, head string) ([]Cherry, error) {
	if err := rejectUnsafeRev(upstream); err != nil {
		return nil, err
	}
	if err := rejectUnsafeRev(head); err != nil {
		return nil, err
	}
	res, err := r.run(ctx, "cherry", "-v", upstream, head)
	if err != nil {
		return nil, err
	}
	return parseCherry(res.Stdout), nil
}

func (r *Repository) MergeTree(ctx context.Context, ours, theirs string) (MergeTreeResult, error) {
	if err := rejectUnsafeRev(ours); err != nil {
		return MergeTreeResult{}, err
	}
	if err := rejectUnsafeRev(theirs); err != nil {
		return MergeTreeResult{}, err
	}
	res, err := r.run(ctx, "merge-tree", "-z", "--name-only", "--messages", ours, theirs)
	if err != nil {
		var ce *CommandError
		if !asCommandError(err, &ce) {
			return MergeTreeResult{}, err
		}
		return parseMergeTree(res.Stdout, res.Stderr, false), nil
	}
	return parseMergeTree(res.Stdout, res.Stderr, true), nil
}

func (r *Repository) TreeHash(ctx context.Context, rev string) (string, error) {
	if err := rejectUnsafeRev(rev); err != nil {
		return "", err
	}
	res, err := r.run(ctx, "rev-parse", rev+"^{tree}")
	if err != nil {
		return "", err
	}
	return res.Output(), nil
}

func (r *Repository) FileContents(ctx context.Context, rev, path string) (string, error) {
	if err := rejectUnsafeRev(rev); err != nil {
		return "", err
	}
	if err := rejectUnsafePath(path); err != nil {
		return "", err
	}
	spec := rev + ":" + filepath.ToSlash(path)
	res, err := r.run(ctx, "cat-file", "-p", spec)
	if err != nil {
		return "", err
	}
	return res.Stdout, nil
}

func (r *Repository) FileExists(ctx context.Context, rev, path string) bool {
	_, err := r.FileContents(ctx, rev, path)
	return err == nil
}

func (r *Repository) LSRemoteHeads(ctx context.Context, url string) ([]Ref, error) {
	if err := rejectUnsafeURL(url); err != nil {
		return nil, err
	}
	res, err := r.run(ctx, "ls-remote", "--heads", "--tags", url)
	if err != nil {
		return nil, err
	}
	return parseLSRemote(res.Stdout), nil
}

func (r *Repository) WorktreeAddDetached(ctx context.Context, dest, commit string) error {
	if err := rejectUnsafeRev(commit); err != nil {
		return err
	}
	_, err := r.run(ctx, "worktree", "add", "--detach", dest, commit)
	return err
}

func (r *Repository) WorktreeRemove(ctx context.Context, dest string) error {
	_, err := r.run(ctx, "worktree", "remove", "--force", dest)
	return err
}

func asCommandError(err error, target **CommandError) bool {
	if err == nil {
		return false
	}
	ce, ok := err.(*CommandError)
	if ok {
		*target = ce
		return true
	}
	return false
}

func parseLeftRightCount(s string) (Count, error) {
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return Count{}, fmt.Errorf("unexpected rev-list count %q", s)
	}
	left, err := strconv.Atoi(parts[0])
	if err != nil {
		return Count{}, err
	}
	right, err := strconv.Atoi(parts[1])
	if err != nil {
		return Count{}, err
	}
	return Count{Ahead: right, Behind: left}, nil
}
