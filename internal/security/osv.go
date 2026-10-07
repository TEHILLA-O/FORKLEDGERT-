package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OSV struct {
	HTTP    *http.Client
	BaseURL string
}

func NewOSV() *OSV {
	return &OSV{
		HTTP:    &http.Client{Timeout: 20 * time.Second},
		BaseURL: "https://api.osv.dev/v1/query",
	}
}

type osvRequest struct {
	Commit  string     `json:"commit,omitempty"`
	Version string     `json:"version,omitempty"`
	Package *osvPkg    `json:"package,omitempty"`
}

type osvPkg struct {
	Name      string `json:"name,omitempty"`
	Ecosystem string `json:"ecosystem,omitempty"`
}

type osvResponse struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvVuln struct {
	ID         string        `json:"id"`
	Summary    string        `json:"summary"`
	Aliases    []string      `json:"aliases"`
	Severity   []osvSeverity `json:"severity"`
	Affected   []osvAffected `json:"affected"`
	References []osvRef      `json:"references"`
}

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type osvAffected struct {
	Package osvPkg       `json:"package"`
	Ranges  []osvRange   `json:"ranges"`
}

type osvRange struct {
	Type   string     `json:"type"`
	Repo   string     `json:"repo,omitempty"`
	Events []osvEvent `json:"events"`
}

type osvEvent struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
	LastAffected string `json:"last_affected,omitempty"`
}

type osvRef struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (o *OSV) Query(ctx context.Context, q Query) ([]Advisory, error) {
	if o == nil || o.HTTP == nil {
		return nil, fmt.Errorf("osv client not configured")
	}
	reqBody := osvRequest{}
	if q.Commit != "" {
		reqBody.Commit = q.Commit
	} else if q.Package != "" {
		reqBody.Version = q.Version
		reqBody.Package = &osvPkg{Name: q.Package, Ecosystem: q.Ecosystem}
	} else if q.RepoURL != "" && q.Version != "" {
		reqBody.Version = q.Version
		reqBody.Package = &osvPkg{Name: q.RepoURL, Ecosystem: "GIT"}
	} else {
		return nil, nil
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "forkledger")
	resp, err := o.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("osv status %d", resp.StatusCode)
	}
	var parsed osvResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	out := make([]Advisory, 0, len(parsed.Vulns))
	for _, v := range parsed.Vulns {
		out = append(out, toAdvisory(v))
	}
	return out, nil
}

func toAdvisory(v osvVuln) Advisory {
	a := Advisory{
		ID:      v.ID,
		Aliases: v.Aliases,
		Summary: v.Summary,
	}
	if len(v.Severity) > 0 {
		a.Severity = v.Severity[0].Score
		if v.Severity[0].Type != "" {
			a.Severity = v.Severity[0].Type + " " + v.Severity[0].Score
		}
	}
	for _, aff := range v.Affected {
		a.Affected = append(a.Affected, Affected{Package: aff.Package.Name, Ecosystem: aff.Package.Ecosystem})
		for _, rg := range aff.Ranges {
			for _, ev := range rg.Events {
				if ev.Fixed != "" {
					a.Fixed = append(a.Fixed, ev.Fixed)
				}
				if ev.Introduced != "" && ev.Introduced != "0" {
					a.Introduced = append(a.Introduced, ev.Introduced)
				}
			}
		}
	}
	for _, r := range v.References {
		a.References = append(a.References, r.URL)
	}
	return a
}

func NormalizeSeverity(s string) string {
	u := strings.ToUpper(s)
	switch {
	case strings.Contains(u, "CRIT"):
		return "CRITICAL"
	case strings.Contains(u, "HIGH") || strings.Contains(u, "9.") || strings.Contains(u, "8."):
		return "HIGH"
	case strings.Contains(u, "MED") || strings.Contains(u, "7.") || strings.Contains(u, "6.") || strings.Contains(u, "5."):
		return "MEDIUM"
	case strings.Contains(u, "LOW"):
		return "LOW"
	default:
		return strings.TrimSpace(s)
	}
}
