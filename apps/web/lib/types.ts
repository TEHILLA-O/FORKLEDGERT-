export type Confidence = {
  level: string;
  score: number;
  basis?: string;
  evidence?: Evidence[];
};

export type Evidence = {
  kind: string;
  summary: string;
  detail?: string;
  source?: string;
  target?: string;
};

export type Patch = {
  id: string;
  commit: string;
  subject: string;
  category: string;
  status: string;
  added: number;
  deleted: number;
  files?: string[];
  functions?: string[];
  conflict_likely?: boolean;
  conflict_reason?: string;
  security_relevant?: boolean;
  equivalence?: {
    upstream_commit: string;
    upstream_subject?: string;
    confidence: Confidence;
    layers: string[];
  };
};

export type Finding = {
  id: string;
  summary: string;
  severity?: string;
  applicability: string;
  recommended_action: string;
  upstream_fix_available?: boolean;
  fix_present_in_fork?: boolean;
  evidence?: Evidence[];
};

export type Report = {
  schema_version: string;
  generated_at: string;
  repository: { path: string; head: string; branch?: string };
  upstream: {
    selected?: string;
    uncertain: boolean;
    explanation: string;
    confidence: Confidence;
    candidates?: { identity: string; confidence: Confidence; reasoning: string }[];
  };
  lineage: {
    fork_point?: string;
    baseline?: string;
    current_upstream?: string;
    ahead: number;
    behind: number;
  };
  patches: {
    count: number;
    added: number;
    deleted: number;
    likely_equivalents: number;
    potentially_obsolete: number;
    potential_conflicts: number;
    items: Patch[];
  };
  security: {
    findings: Finding[];
    missing_upstream_fixes: number;
    needs_investigation: number;
  };
  health: { score: number; factors: { name: string; delta: number; reason: string }[] };
  debt: { level: string; summary: string };
  lifecycle: {
    recommended: string;
    confidence: Confidence;
    why: string[];
    risks: string[];
    benefit: string;
    required_work: string;
    alternatives?: string[];
  };
  summary: {
    detected_upstream?: string;
    baseline?: string;
    current_upstream?: string;
    local_modifications: number;
    changed_lines: number;
    likely_equivalents: number;
    potentially_obsolete: number;
    potential_conflicts: number;
    missing_security_fixes: number;
    security_investigations: number;
    suggested_action: string;
    confidence: string;
    headline: string;
  };
  warnings?: string[];
};
