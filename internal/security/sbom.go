package security

import (
	"time"

	"github.com/TEHILLA-O/fork_ledger/internal/config"
)

type SBOMInput struct {
	Name     string
	Version  string
	Upstream string
	Findings []Finding
	Licences []string
}

type CycloneDXDoc struct {
	BOMFormat       string           `json:"bomFormat"`
	SpecVersion     string           `json:"specVersion"`
	Version         int              `json:"version"`
	Metadata        map[string]any   `json:"metadata"`
	Components      []map[string]any `json:"components"`
	Vulnerabilities []map[string]any `json:"vulnerabilities,omitempty"`
}

type SPDXDoc struct {
	SPDXVersion     string           `json:"spdxVersion"`
	DataLicense     string           `json:"dataLicense"`
	SPDXID          string           `json:"SPDXID"`
	Name            string           `json:"name"`
	Created         string           `json:"created"`
	Packages        []map[string]any `json:"packages"`
	Relationships   []map[string]any `json:"relationships,omitempty"`
}

type OpenVEXDoc struct {
	Context    string        `json:"@context"`
	Author     string        `json:"author"`
	Timestamp  time.Time     `json:"timestamp"`
	Version    int           `json:"version"`
	Statements []OpenVEXStmt `json:"statements"`
}

type OpenVEXStmt struct {
	Vulnerability map[string]string   `json:"vulnerability"`
	Products      []map[string]string `json:"products,omitempty"`
	Status        string              `json:"status"`
	Justification string              `json:"justification,omitempty"`
	Action        string              `json:"action_statement,omitempty"`
}

func CycloneDX(in SBOMInput, cfg config.Config) CycloneDXDoc {
	doc := CycloneDXDoc{
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.5",
		Version:     1,
		Metadata: map[string]any{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"tools":     []map[string]string{{"name": "forkledger", "vendor": "ForkLedger"}},
			"component": map[string]string{"type": "application", "name": in.Name, "version": in.Version},
		},
	}
	comp := map[string]any{"type": "library", "name": in.Name, "version": in.Version}
	if in.Upstream != "" {
		comp["bom-ref"] = in.Upstream
		comp["purl"] = "pkg:github/" + in.Upstream
	}
	if len(cfg.AllowedLicences) > 0 {
		comp["licenses"] = cfg.AllowedLicences
	}
	doc.Components = []map[string]any{comp}
	for _, f := range in.Findings {
		doc.Vulnerabilities = append(doc.Vulnerabilities, map[string]any{
			"id":          f.ID,
			"description": f.Summary,
			"ratings":     []map[string]string{{"severity": f.Severity}},
			"analysis": map[string]string{
				"state":  vexState(f.Applicability),
				"detail": f.RecommendedAction,
			},
		})
	}
	return doc
}

func SPDX(in SBOMInput, cfg config.Config) SPDXDoc {
	doc := SPDXDoc{
		SPDXVersion: "SPDX-2.3",
		DataLicense: "CC0-1.0",
		SPDXID:      "SPDXRef-DOCUMENT",
		Name:        in.Name,
		Created:     time.Now().UTC().Format(time.RFC3339),
		Packages: []map[string]any{{
			"SPDXID":           "SPDXRef-Package",
			"name":             in.Name,
			"versionInfo":      in.Version,
			"downloadLocation": "NOASSERTION",
			"licenseDeclared":  firstLicence(cfg),
		}},
	}
	if in.Upstream != "" {
		doc.Relationships = []map[string]any{{
			"spdxElementId":      "SPDXRef-Package",
			"relationshipType":   "DESCENDANT_OF",
			"relatedSpdxElement": in.Upstream,
		}}
	}
	return doc
}

func OpenVEX(product string, findings []Finding) OpenVEXDoc {
	doc := OpenVEXDoc{
		Context:   "https://openvex.dev/ns/v0.2.0",
		Author:    "forkledger",
		Timestamp: time.Now().UTC(),
		Version:   1,
	}
	for _, f := range findings {
		st := OpenVEXStmt{
			Vulnerability: map[string]string{"name": f.ID},
			Products:      []map[string]string{{"@id": product}},
			Status:        openVEXStatus(f.Applicability),
			Action:        f.RecommendedAction,
		}
		if f.Applicability == ApplicabilityNotAffected {
			st.Justification = firstEvidence(f)
			if st.Justification == "" {
				st.Justification = "vulnerable_code_not_present"
			}
		}
		if f.Applicability == ApplicabilityFixed {
			st.Justification = "fork contains equivalent fix"
		}
		doc.Statements = append(doc.Statements, st)
	}
	return doc
}

func vexState(app string) string {
	switch app {
	case ApplicabilityAffected:
		return "exploitable"
	case ApplicabilityFixed:
		return "resolved"
	case ApplicabilityNotAffected:
		return "not_affected"
	default:
		return "in_triage"
	}
}

func openVEXStatus(app string) string {
	switch app {
	case ApplicabilityAffected:
		return "affected"
	case ApplicabilityFixed:
		return "fixed"
	case ApplicabilityNotAffected:
		return "not_affected"
	default:
		return "under_investigation"
	}
}

func firstLicence(cfg config.Config) string {
	if len(cfg.AllowedLicences) == 0 {
		return "NOASSERTION"
	}
	return cfg.AllowedLicences[0]
}

func InputFrom(name, version, upstream string, findings []Finding) SBOMInput {
	return SBOMInput{Name: name, Version: version, Upstream: upstream, Findings: findings}
}
