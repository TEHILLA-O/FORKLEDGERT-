package git

import (
	"context"
	"os/exec"
)

func commandWithEnv(e *Executor, ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, e.GitPath, args...)
	cmd.Dir = dir
	cmd.Env = e.Env
	return cmd
}
