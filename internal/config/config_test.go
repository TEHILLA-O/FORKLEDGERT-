package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBadCategory(t *testing.T) {
	c := Default()
	c.PatchClassifications = []PatchClassification{{Commit: "abc", Category: "NOT_A_THING"}}
	if err := c.Validate(); err == nil {
		t.Fatal("expected invalid category")
	}
}

func TestValidateManualVulnNeedsReason(t *testing.T) {
	c := Default()
	c.ManualVulns = []ManualVuln{{ID: "CVE-1", State: "NOT_AFFECTED"}}
	if err := c.Validate(); err == nil {
		t.Fatal("expected missing reason")
	}
}

func TestLoadAndDuration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forkledger.yaml")
	body := "upstream: https://github.com/example/widget\nreview_interval: 90d\nowner:\n  team: platform\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Upstream != "https://github.com/example/widget" {
		t.Fatalf("upstream %q", cfg.Upstream)
	}
	if cfg.ReviewInterval.Hours() != 90*24 {
		t.Fatalf("interval %s", cfg.ReviewInterval.Duration)
	}
}

func TestBuildCommandRejectsShell(t *testing.T) {
	c := Default()
	c.BuildCommand = []string{"make", "test; curl evil.test"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected shell metacharacter rejection")
	}
}
