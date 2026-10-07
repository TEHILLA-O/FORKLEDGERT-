package security

import (
	"time"
)

type VEXDocument struct {
	Format    string       `json:"format"`
	Generated time.Time    `json:"generated"`
	Product   string       `json:"product,omitempty"`
	Statements []VEXStatement `json:"statements"`
}

type VEXStatement struct {
	Vulnerability string `json:"vulnerability"`
	Status        string `json:"status"`
	Justification string `json:"justification,omitempty"`
	Impact        string `json:"impact,omitempty"`
	Action        string `json:"action,omitempty"`
}

func FromFindings(product string, findings []Finding) VEXDocument {
	doc := VEXDocument{
		Format:    "forkledger-vex-complement",
		Generated: time.Now().UTC(),
		Product:   product,
	}
	for _, f := range findings {
		st := VEXStatement{
			Vulnerability: f.ID,
			Status:        f.Applicability,
			Action:        f.RecommendedAction,
		}
		if f.Applicability == ApplicabilityNotAffected {
			st.Justification = firstEvidence(f)
		}
		if f.Applicability == ApplicabilityFixed {
			st.Justification = "fork contains equivalent fix"
		}
		doc.Statements = append(doc.Statements, st)
	}
	return doc
}

func firstEvidence(f Finding) string {
	if len(f.Evidence) == 0 {
		return ""
	}
	return f.Evidence[0].Summary
}
