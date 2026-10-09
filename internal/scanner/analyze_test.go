package scanner_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/logging"
	"github.com/TEHILLA-O/forkledger/internal/scanner"
	"github.com/TEHILLA-O/forkledger/internal/testutil"
)

func TestAnalyzeForkWorld(t *testing.T) {
	_, fork := testutil.BuildForkWorld(t)
	log := logging.New(logging.LevelQuiet)
	eng := scanner.New(log)
	cfg := config.Default()
	cfg.Owner.Team = "platform"

	rep, err := eng.Analyze(context.Background(), scanner.Options{
		Path:    fork,
		Config:  cfg,
		Offline: true,
		Log:     log,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Patches.Count != 3 {
		t.Fatalf("expected 3 local patches, got %d", rep.Patches.Count)
	}
	if rep.Lineage.ForkPoint == "" {
		t.Fatal("expected a fork point")
	}
	if rep.Lineage.Behind < 1 {
		t.Fatalf("expected to be behind upstream, behind=%d", rep.Lineage.Behind)
	}
	if rep.Patches.LikelyEquivalents < 1 && rep.Patches.PossibleEquivalents < 1 {
		t.Fatalf("expected the token refresh patch to look equivalent upstream")
	}
	if rep.Summary.SuggestedAction == "" {
		t.Fatal("expected a lifecycle action")
	}
	if rep.Health.Score == 0 && len(rep.Health.Factors) == 0 {
		t.Fatal("health should explain itself")
	}
}

func TestAnalyzeNotGit(t *testing.T) {
	dir := t.TempDir()
	eng := scanner.New(logging.New(logging.LevelQuiet))
	// The developer home directory is itself a git repo on this machine.
	// Ceiling stops Git walking out of the empty temp directory and claiming a false positive.
	eng.Git.Env = append(append([]string{}, eng.Git.Env...), "GIT_CEILING_DIRECTORIES="+filepath.Dir(dir))
	_, err := eng.Analyze(context.Background(), scanner.Options{Path: dir, Offline: true})
	if err == nil {
		t.Fatal("expected error")
	}
}
