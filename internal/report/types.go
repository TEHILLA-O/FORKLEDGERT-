package report

import (
	"time"

	"github.com/TEHILLA-O/fork_ledger/internal/evidence"
	"github.com/TEHILLA-O/fork_ledger/internal/security"
)

const SchemaVersion = "0.1.0"

type Report struct {
	SchemaVersion   string            `json:"schema_version"`
	GeneratedAt     time.Time         `json:"generated_at"`
	Duration        string            `json:"duration,omitempty"`
	Repository      RepositoryInfo    `json:"repository"`
	Upstream        UpstreamSection   `json:"upstream"`
	Lineage         Lineage           `json:"lineage"`
	Patches         PatchSection      `json:"patches"`
	Security        SecuritySection   `json:"security"`
	Upgrade         UpgradeHint       `json:"upgrade"`
	Health          Health            `json:"health"`
	Debt            Debt              `json:"debt"`
	Lifecycle       Lifecycle         `json:"lifecycle"`
	UpstreamActivity Activity         `json:"upstream_activity"`
	Fingerprints    []Fingerprint     `json:"fingerprints,omitempty"`
	Policy          PolicySection     `json:"policy"`
	Warnings        []string          `json:"warnings,omitempty"`
	Summary         Summary           `json:"summary"`
}

type RepositoryInfo struct {
	Path    string   `json:"path"`
	Head    string   `json:"head"`
	Branch  string   `json:"branch,omitempty"`
	Remotes []Remote `json:"remotes,omitempty"`
}

type Remote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type UpstreamSection struct {
	Selected    string              `json:"selected,omitempty"`
	Ref         string              `json:"ref,omitempty"`
	Uncertain   bool                `json:"uncertain"`
	Explanation string              `json:"explanation"`
	Confidence  evidence.Assessment `json:"confidence"`
	Candidates  []UpstreamCandidate `json:"candidates,omitempty"`
}

type UpstreamCandidate struct {
	Identity   string              `json:"identity"`
	URL        string              `json:"url,omitempty"`
	Remote     string              `json:"remote,omitempty"`
	Confidence evidence.Assessment `json:"confidence"`
	Reasoning  string              `json:"reasoning"`
}

type Lineage struct {
	ForkPoint         string              `json:"fork_point,omitempty"`
	MergeBases        []string            `json:"merge_bases,omitempty"`
	Baseline          string              `json:"baseline,omitempty"`
	ExpectedBaseline  string              `json:"expected_baseline,omitempty"`
	CurrentUpstream   string              `json:"current_upstream,omitempty"`
	Ahead             int                 `json:"ahead"`
	Behind            int                 `json:"behind"`
	Confidence        evidence.Assessment `json:"confidence"`
}

type PatchSection struct {
	Count                 int     `json:"count"`
	Added                 int     `json:"added"`
	Deleted               int     `json:"deleted"`
	LikelyEquivalents     int     `json:"likely_equivalents"`
	PossibleEquivalents   int     `json:"possible_equivalents"`
	PotentiallyObsolete   int     `json:"potentially_obsolete"`
	PotentialConflicts    int     `json:"potential_conflicts"`
	Items                 []Patch `json:"items"`
}

type Patch struct {
	ID               string              `json:"id"`
	Commit           string              `json:"commit"`
	Author           string              `json:"author,omitempty"`
	Date             time.Time           `json:"date"`
	Subject          string              `json:"subject"`
	PatchID          string              `json:"patch_id,omitempty"`
	Files            []string            `json:"files,omitempty"`
	Functions        []string            `json:"functions,omitempty"`
	Added            int                 `json:"added"`
	Deleted          int                 `json:"deleted"`
	Category         string              `json:"category"`
	CategoryReason   string              `json:"category_reason,omitempty"`
	Status           string              `json:"status"`
	Equivalence      *PatchEquivalence   `json:"equivalence,omitempty"`
	ConflictLikely   bool                `json:"conflict_likely"`
	ConflictReason   string              `json:"conflict_reason,omitempty"`
	SecurityRelevant bool                `json:"security_relevant"`
	Evidence         []evidence.Item     `json:"evidence,omitempty"`
}

type PatchEquivalence struct {
	UpstreamCommit  string              `json:"upstream_commit"`
	UpstreamSubject string              `json:"upstream_subject,omitempty"`
	Confidence      evidence.Assessment `json:"confidence"`
	Layers          []string            `json:"layers"`
}

type SecuritySection struct {
	Findings             []security.Finding      `json:"findings"`
	MissingUpstreamFixes int                     `json:"missing_upstream_fixes"`
	NeedsInvestigation   int                     `json:"needs_investigation"`
	VEX                  security.VEXDocument    `json:"vex"`
	OpenVEX              security.OpenVEXDoc     `json:"openvex,omitempty"`
	CycloneDX            security.CycloneDXDoc   `json:"cyclonedx,omitempty"`
	SPDX                 security.SPDXDoc        `json:"spdx,omitempty"`
}

type UpgradeHint struct {
	MergeTreeClean  bool     `json:"merge_tree_clean"`
	ConflictedFiles []string `json:"conflicted_files,omitempty"`
	Notes           []string `json:"notes,omitempty"`
}

type Factor struct {
	Name    string `json:"name"`
	Delta   int    `json:"delta"`
	Reason  string `json:"reason"`
}

type Health struct {
	Score   int      `json:"score"`
	Factors []Factor `json:"factors"`
}

type Debt struct {
	Level   string   `json:"level"`
	Signals []Factor `json:"signals"`
	Summary string   `json:"summary"`
}

type LifecycleAction string

const (
	ActionRetain   LifecycleAction = "RETAIN"
	ActionUpstream LifecycleAction = "UPSTREAM"
	ActionRebase   LifecycleAction = "REBASE"
	ActionReplace  LifecycleAction = "REPLACE"
	ActionRetire   LifecycleAction = "RETIRE"
)

type Lifecycle struct {
	Recommended LifecycleAction     `json:"recommended"`
	Confidence  evidence.Assessment `json:"confidence"`
	Why         []string            `json:"why"`
	Risks       []string            `json:"risks"`
	Benefit     string              `json:"benefit"`
	RequiredWork string             `json:"required_work"`
	Alternatives []LifecycleAction  `json:"alternatives,omitempty"`
}

type Activity struct {
	Commits            int       `json:"commits"`
	Oldest             time.Time `json:"oldest,omitempty"`
	Newest             time.Time `json:"newest,omitempty"`
	SecurityMentioning int       `json:"security_mentioning"`
}

type Fingerprint struct {
	Kind    string         `json:"kind"`
	Value   string         `json:"value,omitempty"`
	Note    string         `json:"note,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type PolicySection struct {
	Violations []PolicyViolation `json:"violations,omitempty"`
}

type PolicyViolation struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
	Waived      bool   `json:"waived,omitempty"`
	WaiverNote  string `json:"waiver_note,omitempty"`
}

type Summary struct {
	DetectedUpstream       string `json:"detected_upstream,omitempty"`
	Baseline               string `json:"baseline,omitempty"`
	CurrentUpstream        string `json:"current_upstream,omitempty"`
	LocalModifications     int    `json:"local_modifications"`
	ChangedLines           int    `json:"changed_lines"`
	LikelyEquivalents      int    `json:"likely_equivalents"`
	PotentiallyObsolete    int    `json:"potentially_obsolete"`
	PotentialConflicts     int    `json:"potential_conflicts"`
	MissingSecurityFixes   int    `json:"missing_security_fixes"`
	SecurityInvestigations int    `json:"security_investigations"`
	SuggestedAction        string `json:"suggested_action"`
	Confidence             string `json:"confidence"`
	Headline               string `json:"headline"`
}
