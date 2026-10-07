package security

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TrivyReport is the subset of Trivy JSON ForkLedger is willing to ingest.
// ForkLedger does not execute Trivy. The operator runs it and hands us the file.
type TrivyReport struct {
	Results []struct {
		Target          string `json:"Target"`
		Vulnerabilities []struct {
			VulnerabilityID string `json:"VulnerabilityID"`
			PkgName         string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			FixedVersion    string `json:"FixedVersion"`
			Severity        string `json:"Severity"`
			Title           string `json:"Title"`
			PrimaryURL      string `json:"PrimaryURL"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

func IngestTrivy(path string) ([]Advisory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var trivy TrivyReport
	if err := json.Unmarshal(data, &trivy); err != nil {
		return nil, fmt.Errorf("trivy json: %w", err)
	}
	var out []Advisory
	seen := map[string]struct{}{}
	for _, r := range trivy.Results {
		for _, v := range r.Vulnerabilities {
			id := strings.TrimSpace(v.VulnerabilityID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			a := Advisory{
				ID:       id,
				Summary:  v.Title,
				Severity: strings.ToUpper(v.Severity),
				Affected: []Affected{{Package: v.PkgName}},
			}
			if v.FixedVersion != "" {
				a.Fixed = []string{v.FixedVersion}
			}
			if v.PrimaryURL != "" {
				a.References = []string{v.PrimaryURL}
			}
			out = append(out, a)
		}
	}
	return out, nil
}
