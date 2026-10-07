package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=Fork Ledger",
		"GIT_AUTHOR_EMAIL=fork@example.test",
		"GIT_COMMITTER_NAME=Fork Ledger",
		"GIT_COMMITTER_EMAIL=fork@example.test",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func InitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, dir, "init", "-b", "main")
	Git(t, dir, "config", "user.email", "fork@example.test")
	Git(t, dir, "config", "user.name", "Fork Ledger")
	Git(t, dir, "config", "commit.gpgsign", "false")
	Git(t, dir, "config", "core.autocrlf", "false")
}

func Write(t *testing.T, dir, rel, contents string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func CommitAll(t *testing.T, dir, message string) string {
	t.Helper()
	Git(t, dir, "add", "-A")
	Git(t, dir, "commit", "-m", message)
	return Hash(t, dir, "HEAD")
}

func Hash(t *testing.T, dir, rev string) string {
	t.Helper()
	out := Git(t, dir, "rev-parse", rev)
	return trim(out)
}

func Clone(t *testing.T, src, dest string) {
	t.Helper()
	Git(t, filepath.Dir(dest), "clone", src, dest)
	Git(t, dest, "config", "user.email", "fork@example.test")
	Git(t, dest, "config", "user.name", "Fork Ledger")
	Git(t, dest, "config", "commit.gpgsign", "false")
}

func AddRemote(t *testing.T, dir, name, url string) {
	t.Helper()
	Git(t, dir, "remote", "add", name, url)
	Git(t, dir, "fetch", name, "--tags")
}

func TempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		return dir
	}
	return dir
}

func trim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// BuildForkWorld creates an upstream history, clones it, then diverges both sides.
// One local patch is later copied upstream so patch-id equivalence has something real to chew on.
func BuildForkWorld(t *testing.T) (upstreamDir, forkDir string) {
	t.Helper()
	root := t.TempDir()
	upstreamDir = filepath.Join(root, "upstream")
	InitRepo(t, upstreamDir)
	Write(t, upstreamDir, "README.md", "# widget\n\nA small library used as a fixture.\n")
	Write(t, upstreamDir, "src/auth.go", "package src\n\nfunc TokenRefresh() string {\n\treturn \"v1\"\n}\n")
	Write(t, upstreamDir, "src/util.go", "package src\n\nfunc Helper() int { return 1 }\n")
	CommitAll(t, upstreamDir, "initial import")
	Git(t, upstreamDir, "tag", "v1.0.0")

	Write(t, upstreamDir, "src/util.go", "package src\n\nfunc Helper() int { return 2 }\n")
	CommitAll(t, upstreamDir, "fix helper off-by-one")
	Git(t, upstreamDir, "tag", "v1.0.1")

	forkDir = filepath.Join(root, "fork")
	Clone(t, upstreamDir, forkDir)
	Git(t, forkDir, "remote", "rename", "origin", "upstream")

	Write(t, forkDir, "src/auth.go", "package src\n\nfunc TokenRefresh() string {\n\treturn \"v1-internal\"\n}\n")
	CommitAll(t, forkDir, "compat: retain legacy token refresh behaviour")

	Write(t, forkDir, "branding.txt", "Acme Widget\n")
	CommitAll(t, forkDir, "brand: add internal product name")

	Write(t, forkDir, "src/util.go", "package src\n\nfunc Helper() int { return 2 }\n\nfunc Extra() int { return 9 }\n")
	CommitAll(t, forkDir, "feat: add Extra helper used by internal callers")

	Write(t, upstreamDir, "src/auth.go", "package src\n\nfunc TokenRefresh() string {\n\treturn \"v1-internal\"\n}\n")
	CommitAll(t, upstreamDir, "compat: retain legacy token refresh behaviour")

	Write(t, upstreamDir, "src/security.go", "package src\n\nfunc Sanitize(s string) string { return s }\n")
	CommitAll(t, upstreamDir, "security: sanitize untrusted input (CVE-2024-0001)")
	Git(t, upstreamDir, "tag", "v1.1.0")

	Git(t, forkDir, "fetch", "upstream", "--tags")
	return upstreamDir, forkDir
}
