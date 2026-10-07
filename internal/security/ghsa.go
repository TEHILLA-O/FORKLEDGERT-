package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GHSA struct {
	HTTP    *http.Client
	Token   string
	BaseURL string
}

func NewGHSA(token string) *GHSA {
	return &GHSA{
		HTTP:    &http.Client{Timeout: 20 * time.Second},
		Token:   token,
		BaseURL: "https://api.github.com/advisories",
	}
}

type ghsaItem struct {
	GHSAID      string   `json:"ghsa_id"`
	CVEID       string   `json:"cve_id"`
	Summary     string   `json:"summary"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	HTMLURL     string   `json:"html_url"`
	Vulnerabilities []struct {
		Package struct {
			Ecosystem string `json:"ecosystem"`
			Name      string `json:"name"`
		} `json:"package"`
		VulnerableVersionRange string `json:"vulnerable_version_range"`
		FirstPatchedVersion    *struct {
			Identifier string `json:"identifier"`
		} `json:"first_patched_version"`
	} `json:"vulnerabilities"`
}

func (g *GHSA) QueryRepo(ctx context.Context, owner, name string) ([]Advisory, error) {
	if g == nil || g.HTTP == nil {
		return nil, fmt.Errorf("github advisories client not configured")
	}
	if owner == "" || name == "" {
		return nil, nil
	}
	endpoint := "https://api.github.com/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/security-advisories"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "forkledger")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	resp, err := g.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("github advisories rate limited")
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github advisories status %d", resp.StatusCode)
	}
	var items []ghsaItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	var out []Advisory
	for _, it := range items {
		a := Advisory{
			ID:       it.GHSAID,
			Summary:  it.Summary,
			Severity: strings.ToUpper(it.Severity),
		}
		if it.CVEID != "" {
			a.Aliases = append(a.Aliases, it.CVEID)
		}
		if it.HTMLURL != "" {
			a.References = append(a.References, it.HTMLURL)
		}
		for _, v := range it.Vulnerabilities {
			a.Affected = append(a.Affected, Affected{Package: v.Package.Name, Ecosystem: v.Package.Ecosystem})
			if v.FirstPatchedVersion != nil && v.FirstPatchedVersion.Identifier != "" {
				a.Fixed = append(a.Fixed, v.FirstPatchedVersion.Identifier)
			}
		}
		out = append(out, a)
	}
	return out, nil
}
