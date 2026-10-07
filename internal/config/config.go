package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const FileName = "forkledger.yaml"

type Config struct {
	Upstream          string                  `yaml:"upstream" json:"upstream,omitempty"`
	ExpectedBaseline  string                  `yaml:"expected_baseline" json:"expected_baseline,omitempty"`
	Owner             Owner                   `yaml:"owner" json:"owner"`
	ReviewInterval    Duration                `yaml:"review_interval" json:"review_interval,omitempty"`
	LastReviewed      string                  `yaml:"last_reviewed" json:"last_reviewed,omitempty"`
	SecuritySLA       Duration                `yaml:"security_sla" json:"security_sla,omitempty"`
	BuildCommand      []string                `yaml:"build_command" json:"build_command,omitempty"`
	TestCommand       []string                `yaml:"test_command" json:"test_command,omitempty"`
	IgnoredPaths      []string                `yaml:"ignored_paths" json:"ignored_paths,omitempty"`
	AllowedLicences   []string                `yaml:"allowed_licences" json:"allowed_licences,omitempty"`
	PatchClassifications []PatchClassification `yaml:"patch_classifications" json:"patch_classifications,omitempty"`
	KnownEquivalents  []KnownEquivalent       `yaml:"known_equivalent_patches" json:"known_equivalent_patches,omitempty"`
	ManualVulns       []ManualVuln            `yaml:"manual_vulnerability_decisions" json:"manual_vulnerability_decisions,omitempty"`
	Policy            PolicyConfig            `yaml:"policy" json:"policy"`
	GitHub            GitHubConfig            `yaml:"github" json:"github"`
}

type Owner struct {
	Team    string `yaml:"team" json:"team,omitempty"`
	Contact string `yaml:"contact" json:"contact,omitempty"`
}

type PatchClassification struct {
	Commit   string `yaml:"commit" json:"commit"`
	Category string `yaml:"category" json:"category"`
	Reason   string `yaml:"reason,omitempty" json:"reason,omitempty"`
	Status   string `yaml:"status,omitempty" json:"status,omitempty"`
}

type KnownEquivalent struct {
	Local    string `yaml:"local" json:"local"`
	Upstream string `yaml:"upstream" json:"upstream"`
	Note     string `yaml:"note,omitempty" json:"note,omitempty"`
}

type ManualVuln struct {
	ID       string `yaml:"id" json:"id"`
	State    string `yaml:"state" json:"state"`
	Reason   string `yaml:"reason" json:"reason"`
	Owner    string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Expires  string `yaml:"expires,omitempty" json:"expires,omitempty"`
}

type PolicyConfig struct {
	Rules []PolicyRule `yaml:"rules" json:"rules,omitempty"`
}

type PolicyRule struct {
	ID          string            `yaml:"id" json:"id"`
	Severity    string            `yaml:"severity" json:"severity"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Remediation string            `yaml:"remediation,omitempty" json:"remediation,omitempty"`
	When        map[string]string `yaml:"when" json:"when"`
}

type GitHubConfig struct {
	CommentMode string `yaml:"comment_mode" json:"comment_mode,omitempty"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	if strings.TrimSpace(s) == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := parseHumanDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	if d.Duration == 0 {
		return "", nil
	}
	return d.Duration.String(), nil
}

func parseHumanDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		days, err := time.ParseDuration(n + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}
	return time.ParseDuration(s)
}

func Load(path string) (Config, string, error) {
	cfg := Default()
	if path == "" {
		return cfg, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, path, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, path, fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, path, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, path, nil
}

func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, FileName)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func Default() Config {
	return Config{
		GitHub: GitHubConfig{CommentMode: "update"},
	}
}

func (c Config) HasOwner() bool {
	return strings.TrimSpace(c.Owner.Team) != "" || strings.TrimSpace(c.Owner.Contact) != ""
}
