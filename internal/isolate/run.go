package isolate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const DefaultTimeout = 3 * time.Minute

type Result struct {
	OK       bool   `json:"ok"`
	Code     int    `json:"code"`
	Output   string `json:"output,omitempty"`
	Duration string `json:"duration"`
}

// Run executes an explicit argv inside dir. No shell. The repository does not
// get to invent the command line. Build and test stay off until the operator
// listed the program and FORKLEDGER_ALLOW_BUILD=1 is set by the caller.
func Run(ctx context.Context, dir string, argv []string) (Result, error) {
	if len(argv) == 0 {
		return Result{}, fmt.Errorf("empty command")
	}
	joined := strings.Join(argv, " ")
	if strings.ContainsAny(joined, ";|&`$") {
		return Result{}, fmt.Errorf("refusing shell metacharacters")
	}
	if strings.HasPrefix(argv[0], "-") {
		return Result{}, fmt.Errorf("program must not start with a dash")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = abs
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	started := time.Now()
	err = cmd.Run()
	res := Result{Output: buf.String(), Duration: time.Since(started).String()}
	if cmd.ProcessState != nil {
		res.Code = cmd.ProcessState.ExitCode()
	}
	if ctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("command timed out after %s", DefaultTimeout)
	}
	if err != nil {
		return res, err
	}
	res.OK = true
	return res, nil
}
