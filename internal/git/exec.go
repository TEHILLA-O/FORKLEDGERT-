package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TEHILLA-O/forkledger/internal/logging"
)

const DefaultTimeout = 2 * time.Minute

// Result is the exact Git invocation ForkLedger relied on. If a conclusion
// cannot point at one of these, it is not a Git conclusion.
type Result struct {
	Args     []string      `json:"args"`
	Dir      string        `json:"dir"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Code     int           `json:"code"`
	Duration time.Duration `json:"duration"`
}

func (r Result) Output() string {
	return strings.TrimSpace(r.Stdout)
}

type Executor struct {
	GitPath string
	Timeout time.Duration
	Log     *logging.Logger
	Env     []string
}

func NewExecutor(log *logging.Logger) *Executor {
	return &Executor{
		GitPath: "git",
		Timeout: DefaultTimeout,
		Log:     log,
		Env:     safeEnv(),
	}
}

func (e *Executor) timeout() time.Duration {
	if e.Timeout <= 0 {
		return DefaultTimeout
	}
	return e.Timeout
}

// Run executes Git with argv only. No shell. The repository does not get to
// invent new command lines, which is how credential helpers and hook tricks start.
func (e *Executor) Run(ctx context.Context, dir string, args ...string) (Result, error) {
	if dir != "" {
		clean, err := sanitizeDir(dir)
		if err != nil {
			return Result{}, err
		}
		dir = clean
	}
	full := append(safetyArgs(), args...)
	timeout := e.timeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.GitPath, full...)
	cmd.Dir = dir
	cmd.Env = e.Env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	started := time.Now()
	err := cmd.Run()
	res := Result{
		Args:     full,
		Dir:      dir,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(started),
	}
	if cmd.ProcessState != nil {
		res.Code = cmd.ProcessState.ExitCode()
	}

	e.trace(res)

	if ctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("git timed out after %s: git %s", timeout, strings.Join(args, " "))
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			msg := strings.TrimSpace(res.Stderr)
			if msg == "" {
				msg = err.Error()
			}
			return res, &CommandError{Args: args, Code: res.Code, Stderr: msg}
		}
		return res, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return res, nil
}

func (e *Executor) trace(res Result) {
	if e.Log == nil {
		return
	}
	e.Log.Debug("git %s (code=%d, %s)", strings.Join(visibleArgs(res.Args), " "), res.Code, res.Duration)
	if strings.TrimSpace(res.Stderr) != "" {
		e.Log.Debug("git stderr: %s", strings.TrimSpace(res.Stderr))
	}
}

// CommandError is a failed Git process with the original wording preserved.
type CommandError struct {
	Args   []string
	Code   int
	Stderr string
}

func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), e.Stderr)
	}
	return fmt.Sprintf("git %s: exit %d", strings.Join(e.Args, " "), e.Code)
}

func sanitizeDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("repository path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("repository path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository path is not a directory: %s", abs)
	}
	return abs, nil
}

func visibleArgs(args []string) []string {
	out := make([]string, 0, len(args))
	skip := false
	for _, a := range args {
		if skip {
			skip = false
			continue
		}
		if a == "-c" {
			skip = true
			continue
		}
		out = append(out, a)
	}
	return out
}

func LookPath() (string, error) {
	return exec.LookPath("git")
}
