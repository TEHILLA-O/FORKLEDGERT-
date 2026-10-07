package storage

import "time"

// Typed records matching 001_init.sql. The CLI does not persist these yet.
// They exist so the hosted control plane does not invent a second vocabulary.

type Organisation struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type Repository struct {
	ID             string
	OrganisationID string
	Host           string
	OwnerName      string
	Name           string
	DefaultBranch  string
	CreatedAt      time.Time
}

type ForkRelationship struct {
	ID                   string
	SourceRepositoryID   string
	TargetRepositoryID   string
	RelationshipType     string
	Confidence           int
	Evidence             []byte
	CreatedAt            time.Time
	LastVerifiedAt       *time.Time
}

type Scan struct {
	ID           string
	RepositoryID string
	StartedAt    time.Time
	FinishedAt   *time.Time
	ToolVersion  string
	Report       []byte
}

type PatchRecord struct {
	ID           string
	RepositoryID string
	CommitSHA    string
	PatchID      string
	Category     string
	Status       string
	Subject      string
	Author       string
	AuthoredAt   time.Time
	Added        int
	Deleted      int
}

type ForkVulnerabilityState struct {
	ID               string
	RepositoryID     string
	VulnerabilityID  string
	Applicability    string
	Confidence       int
	Evidence         []byte
}

type LifecycleDecision struct {
	ID           string
	RepositoryID string
	Recommended  string
	Confidence   int
	Evidence     []byte
	DecidedBy    string
	DecidedAt    time.Time
}

type UpgradeSimulation struct {
	ID           string
	RepositoryID string
	TargetRef    string
	Result       []byte
	CreatedAt    time.Time
}
