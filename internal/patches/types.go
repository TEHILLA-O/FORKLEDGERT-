package patches

import (
	"strings"
	"time"

	"github.com/forkledger/forkledger/internal/evidence"
	"github.com/forkledger/forkledger/internal/git"
)

type Category string

const (
	CategorySecurity             Category = "SECURITY"
	CategoryBugFix               Category = "BUG_FIX"
	CategoryCompatibility        Category = "COMPATIBILITY"
	CategoryFeature              Category = "FEATURE"
	CategoryPerformance          Category = "PERFORMANCE"
	CategoryConfiguration        Category = "CONFIGURATION"
	CategoryBranding             Category = "BRANDING"
	CategoryCompliance           Category = "COMPLIANCE"
	CategoryTemporaryWorkaround  Category = "TEMPORARY_WORKAROUND"
	CategoryExperimental         Category = "EXPERIMENTAL"
	CategoryUnknown              Category = "UNKNOWN"
)

func AllCategories() []Category {
	return []Category{
		CategorySecurity,
		CategoryBugFix,
		CategoryCompatibility,
		CategoryFeature,
		CategoryPerformance,
		CategoryConfiguration,
		CategoryBranding,
		CategoryCompliance,
		CategoryTemporaryWorkaround,
		CategoryExperimental,
		CategoryUnknown,
	}
}

func ValidCategory(s string) bool {
	for _, c := range AllCategories() {
		if string(c) == s {
			return true
		}
	}
	return false
}

type Status string

const (
	StatusActive                 Status = "ACTIVE"
	StatusUpstreamSubmitted      Status = "UPSTREAM_SUBMITTED"
	StatusUpstreamAccepted       Status = "UPSTREAM_ACCEPTED"
	StatusUpstreamEquivalent     Status = "UPSTREAM_EQUIVALENT_EXISTS"
	StatusRedundant              Status = "REDUNDANT"
	StatusBlocksUpgrade          Status = "BLOCKS_UPGRADE"
	StatusMustRetain             Status = "MUST_RETAIN"
	StatusDeprecated             Status = "DEPRECATED"
	StatusUnknown                Status = "UNKNOWN"
)

func AllStatuses() []Status {
	return []Status{
		StatusActive,
		StatusUpstreamSubmitted,
		StatusUpstreamAccepted,
		StatusUpstreamEquivalent,
		StatusRedundant,
		StatusBlocksUpgrade,
		StatusMustRetain,
		StatusDeprecated,
		StatusUnknown,
	}
}

func ValidStatus(s string) bool {
	for _, st := range AllStatuses() {
		if string(st) == s {
			return true
		}
	}
	return false
}

type Patch struct {
	ID               string           `json:"id"`
	Commit           string           `json:"commit"`
	Parents          []string         `json:"parents,omitempty"`
	Author           string           `json:"author,omitempty"`
	Email            string           `json:"email,omitempty"`
	Date             time.Time        `json:"date"`
	Subject          string           `json:"subject"`
	Body             string           `json:"body,omitempty"`
	PatchID          string           `json:"patch_id,omitempty"`
	Files            []git.FileChange `json:"files"`
	Functions        []string         `json:"functions,omitempty"`
	Added            int              `json:"added"`
	Deleted          int              `json:"deleted"`
	Category         Category         `json:"category"`
	CategoryReason   string           `json:"category_reason,omitempty"`
	Status           Status           `json:"status"`
	Equivalence      *Equivalence     `json:"equivalence,omitempty"`
	ConflictLikely   bool             `json:"conflict_likely"`
	ConflictReason   string           `json:"conflict_reason,omitempty"`
	SecurityRelevant bool             `json:"security_relevant"`
	Evidence         []evidence.Item  `json:"evidence,omitempty"`
	IssueIDs         []string         `json:"issue_ids,omitempty"`
	Tokens           []string         `json:"-"`
	ASTFuncs         []string         `json:"ast_functions,omitempty"`
}

type Equivalence struct {
	UpstreamCommit string               `json:"upstream_commit"`
	UpstreamSubject string              `json:"upstream_subject,omitempty"`
	Confidence     evidence.Assessment  `json:"confidence"`
	Layers         []string             `json:"layers"`
}

func (p Patch) LineImpact() int {
	return p.Added + p.Deleted
}

func UniqueFunctions(hunks []git.Hunk) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, h := range hunks {
		fn := strings.TrimSpace(h.Function)
		if fn == "" {
			continue
		}
		if _, ok := seen[fn]; ok {
			continue
		}
		seen[fn] = struct{}{}
		out = append(out, fn)
	}
	return out
}
