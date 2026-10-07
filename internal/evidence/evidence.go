package evidence

import "time"

// Kind names a class of observation. Conclusions may cite several kinds.
// Heuristics should add evidence. They should not quietly promote themselves to facts.
type Kind string

const (
	KindGitAncestry          Kind = "GIT_ANCESTRY"
	KindTreeSimilarity       Kind = "TREE_SIMILARITY"
	KindMatchingPatchID      Kind = "MATCHING_PATCH_ID"
	KindFunctionOverlap      Kind = "FUNCTION_OVERLAP"
	KindASTSimilarity        Kind = "AST_SIMILARITY"
	KindSharedIssueID        Kind = "SHARED_ISSUE_ID"
	KindPackageMetadata      Kind = "PACKAGE_METADATA"
	KindSecurityAdvisory     Kind = "SECURITY_ADVISORY"
	KindSuccessfulBuild      Kind = "SUCCESSFUL_BUILD"
	KindTestResult           Kind = "TEST_RESULT"
	KindHumanReview          Kind = "HUMAN_REVIEW"
	KindManualOverride       Kind = "MANUAL_OVERRIDE"
	KindConfiguredRemote     Kind = "CONFIGURED_REMOTE"
	KindConfiguredUpstream   Kind = "CONFIGURED_UPSTREAM"
	KindGitHubForkMetadata   Kind = "GITHUB_FORK_METADATA"
	KindMatchingCommit       Kind = "MATCHING_COMMIT"
	KindMatchingTree         Kind = "MATCHING_TREE"
	KindReleaseTag           Kind = "RELEASE_TAG"
	KindCherryEquivalent     Kind = "CHERRY_EQUIVALENT"
	KindNormalizedDiff       Kind = "NORMALIZED_DIFF"
	KindFileOverlap          Kind = "FILE_OVERLAP"
	KindCommitMessage        Kind = "COMMIT_MESSAGE"
	KindMergeTree            Kind = "MERGE_TREE"
	KindRemoteName           Kind = "REMOTE_NAME"
	KindRepositoryName       Kind = "REPOSITORY_NAME"
	KindLicenseFile          Kind = "LICENSE_FILE"
	KindReadmeContent        Kind = "README_CONTENT"
	KindDirectorySignature   Kind = "DIRECTORY_SIGNATURE"
	KindFileHash             Kind = "FILE_HASH"
	KindMinHash              Kind = "MINHASH"
	KindSimHash              Kind = "SIMHASH"
	KindTLSH                 Kind = "TLSH"
	KindSsdeep               Kind = "SSDEEP"
	KindUpstreamActivity     Kind = "UPSTREAM_ACTIVITY"
	KindPolicyRule           Kind = "POLICY_RULE"
	KindOwnerRecord          Kind = "OWNER_RECORD"
	KindReviewFreshness      Kind = "REVIEW_FRESHNESS"
)

// Item is one observation that a conclusion is allowed to cite.
type Item struct {
	Kind       Kind      `json:"kind"`
	Summary    string    `json:"summary"`
	Detail     string    `json:"detail,omitempty"`
	Source     string    `json:"source,omitempty"`
	Target     string    `json:"target,omitempty"`
	Weight     int       `json:"weight,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

// New builds an item stamped at now.
func New(kind Kind, summary string) Item {
	return Item{
		Kind:       kind,
		Summary:    summary,
		ObservedAt: time.Now().UTC(),
	}
}

func (e Item) WithDetail(detail string) Item {
	e.Detail = detail
	return e
}

func (e Item) WithSource(source string) Item {
	e.Source = source
	return e
}

func (e Item) WithTarget(target string) Item {
	e.Target = target
	return e
}

func (e Item) WithWeight(weight int) Item {
	e.Weight = weight
	return e
}

// Conclusion is anything inferred rather than read from a primary source.
type Conclusion struct {
	Statement  string     `json:"statement"`
	Confidence Assessment `json:"confidence"`
}
