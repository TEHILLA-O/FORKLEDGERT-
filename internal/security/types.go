package security

import (
	"context"

	"github.com/TEHILLA-O/forkledger/internal/evidence"
)

type Finding struct {
	ID                     string              `json:"id"`
	Aliases                []string            `json:"aliases,omitempty"`
	Summary                string              `json:"summary"`
	Severity               string              `json:"severity,omitempty"`
	UpstreamAffected       bool                `json:"upstream_affected"`
	ForkBaselineAffected   bool                `json:"fork_baseline_affected"`
	UpstreamFixAvailable   bool                `json:"upstream_fix_available"`
	FixPresentInFork       bool                `json:"fix_present_in_fork"`
	VulnerableCodePresent  *bool               `json:"vulnerable_code_present,omitempty"`
	InternalOverlap        bool                `json:"internal_modification_overlaps"`
	Applicability          string              `json:"applicability"`
	Confidence             evidence.Assessment `json:"confidence"`
	RecommendedAction      string              `json:"recommended_action"`
	Evidence               []evidence.Item     `json:"evidence"`
	Fixed                  []string            `json:"fixed,omitempty"`
	Introduced             []string            `json:"introduced,omitempty"`
}

const (
	ApplicabilityAffected          = "AFFECTED"
	ApplicabilityNotAffected       = "NOT_AFFECTED"
	ApplicabilityFixed             = "FIXED"
	ApplicabilityUnderInvestigation = "UNDER_INVESTIGATION"
)

type Query struct {
	Commit     string
	Version    string
	Package    string
	Ecosystem  string
	RepoURL    string
}

type Client interface {
	Query(ctx context.Context, q Query) ([]Advisory, error)
}

type Advisory struct {
	ID         string
	Aliases    []string
	Summary    string
	Severity   string
	Fixed      []string
	Introduced []string
	References []string
	Affected   []Affected
}

type Affected struct {
	Package   string
	Ecosystem string
	Versions  []string
}
