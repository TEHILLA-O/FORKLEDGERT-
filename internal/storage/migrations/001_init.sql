-- Relational graph for the future control plane.
-- Confidence and evidence live beside every inferred relationship so a
-- dashboard never has to invent a reason after the fact.

CREATE TABLE organisations (
    id            UUID PRIMARY KEY,
    name          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE repositories (
    id            UUID PRIMARY KEY,
    organisation_id UUID REFERENCES organisations (id),
    host          TEXT NOT NULL,
    owner_name    TEXT NOT NULL,
    name          TEXT NOT NULL,
    default_branch TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host, owner_name, name)
);

CREATE TABLE repository_remotes (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    url           TEXT NOT NULL,
    UNIQUE (repository_id, name)
);

CREATE TABLE fork_relationships (
    id            UUID PRIMARY KEY,
    source_repository_id UUID NOT NULL REFERENCES repositories (id),
    target_repository_id UUID NOT NULL REFERENCES repositories (id),
    relationship_type TEXT NOT NULL,
    confidence    INTEGER,
    evidence      JSONB NOT NULL DEFAULT '[]',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_verified_at TIMESTAMPTZ
);

CREATE TABLE upstream_candidates (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    identity      TEXT NOT NULL,
    url           TEXT,
    confidence    INTEGER NOT NULL,
    evidence      JSONB NOT NULL DEFAULT '[]',
    selected      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scans (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ NOT NULL,
    finished_at   TIMESTAMPTZ,
    tool_version  TEXT NOT NULL,
    report        JSONB
);

CREATE TABLE scan_findings (
    id            UUID PRIMARY KEY,
    scan_id       UUID NOT NULL REFERENCES scans (id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,
    severity      TEXT,
    body          JSONB NOT NULL
);

CREATE TABLE patches (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    commit_sha    TEXT NOT NULL,
    patch_id      TEXT,
    category      TEXT NOT NULL,
    status        TEXT NOT NULL,
    subject       TEXT,
    author        TEXT,
    authored_at   TIMESTAMPTZ,
    added         INTEGER,
    deleted       INTEGER,
    UNIQUE (repository_id, commit_sha)
);

CREATE TABLE patch_evidence (
    id            UUID PRIMARY KEY,
    patch_id      UUID NOT NULL REFERENCES patches (id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,
    summary       TEXT NOT NULL,
    payload       JSONB
);

CREATE TABLE patch_equivalences (
    id            UUID PRIMARY KEY,
    local_patch_id UUID NOT NULL REFERENCES patches (id) ON DELETE CASCADE,
    upstream_commit TEXT NOT NULL,
    confidence    INTEGER NOT NULL,
    layers        TEXT[] NOT NULL DEFAULT '{}',
    evidence      JSONB NOT NULL DEFAULT '[]'
);

CREATE TABLE vulnerabilities (
    id            TEXT PRIMARY KEY,
    aliases       TEXT[] NOT NULL DEFAULT '{}',
    summary       TEXT,
    severity      TEXT,
    source        TEXT NOT NULL
);

CREATE TABLE vulnerability_fixes (
    id            UUID PRIMARY KEY,
    vulnerability_id TEXT NOT NULL REFERENCES vulnerabilities (id) ON DELETE CASCADE,
    commit_sha    TEXT,
    version       TEXT
);

CREATE TABLE fork_vulnerability_states (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    vulnerability_id TEXT NOT NULL REFERENCES vulnerabilities (id),
    applicability TEXT NOT NULL,
    confidence    INTEGER NOT NULL,
    evidence      JSONB NOT NULL DEFAULT '[]',
    UNIQUE (repository_id, vulnerability_id)
);

CREATE TABLE repository_owners (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    team          TEXT,
    contact       TEXT,
    UNIQUE (repository_id)
);

CREATE TABLE products (
    id            UUID PRIMARY KEY,
    organisation_id UUID REFERENCES organisations (id),
    name          TEXT NOT NULL
);

CREATE TABLE product_repositories (
    product_id    UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, repository_id)
);

CREATE TABLE policies (
    id            UUID PRIMARY KEY,
    organisation_id UUID REFERENCES organisations (id),
    repository_id UUID REFERENCES repositories (id),
    rule_id       TEXT NOT NULL,
    body          JSONB NOT NULL
);

CREATE TABLE policy_violations (
    id            UUID PRIMARY KEY,
    policy_id     UUID NOT NULL REFERENCES policies (id) ON DELETE CASCADE,
    scan_id       UUID REFERENCES scans (id),
    severity      TEXT NOT NULL,
    waived        BOOLEAN NOT NULL DEFAULT FALSE,
    waiver_reason TEXT,
    waiver_expires_at TIMESTAMPTZ,
    owner         TEXT
);

CREATE TABLE lifecycle_decisions (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    recommended   TEXT NOT NULL,
    confidence    INTEGER NOT NULL,
    evidence      JSONB NOT NULL DEFAULT '[]',
    decided_by    TEXT,
    decided_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE upgrade_simulations (
    id            UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    target_ref    TEXT NOT NULL,
    result        JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE build_results (
    id            UUID PRIMARY KEY,
    upgrade_simulation_id UUID REFERENCES upgrade_simulations (id) ON DELETE CASCADE,
    ok            BOOLEAN NOT NULL,
    log_ref       TEXT
);

CREATE TABLE test_results (
    id            UUID PRIMARY KEY,
    upgrade_simulation_id UUID REFERENCES upgrade_simulations (id) ON DELETE CASCADE,
    ok            BOOLEAN NOT NULL,
    failed_count  INTEGER,
    log_ref       TEXT
);

CREATE TABLE relationships (
    id            UUID PRIMARY KEY,
    source_type   TEXT NOT NULL,
    source_id     TEXT NOT NULL,
    target_type   TEXT NOT NULL,
    target_id     TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    confidence    INTEGER,
    evidence      JSONB NOT NULL DEFAULT '[]',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_verified_at TIMESTAMPTZ
);

CREATE INDEX relationships_source_idx ON relationships (source_type, source_id);
CREATE INDEX relationships_target_idx ON relationships (target_type, target_id);
CREATE INDEX patches_repo_status_idx ON patches (repository_id, status);
CREATE INDEX scans_repo_started_idx ON scans (repository_id, started_at DESC);
