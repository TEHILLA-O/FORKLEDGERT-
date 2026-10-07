package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Candidate struct {
	FullName      string `json:"full_name"`
	Fork          bool   `json:"fork"`
	Parent        string `json:"parent,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	Reason        string `json:"reason"`
}

type Client struct {
	HTTP    *http.Client
	Token   string
	BaseURL string
}

func New(token string) *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Token:   token,
		BaseURL: "https://api.github.com",
	}
}

func (c *Client) OrgRepos(ctx context.Context, org string) ([]Candidate, error) {
	if org == "" {
		return nil, fmt.Errorf("org is required")
	}
	var out []Candidate
	page := 1
	for page <= 10 {
		u := fmt.Sprintf("%s/orgs/%s/repos?type=all&per_page=100&page=%d", c.BaseURL, url.PathEscape(org), page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "forkledger")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		var repos []ghRepo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("github org repos status %d", resp.StatusCode)
		}
		if len(repos) == 0 {
			break
		}
		for _, r := range repos {
			cand := Candidate{
				FullName:      r.FullName,
				Fork:          r.Fork,
				DefaultBranch: r.DefaultBranch,
				Private:       r.Private,
				HTMLURL:       r.HTMLURL,
			}
			if r.Fork && r.Parent != nil {
				cand.Parent = r.Parent.FullName
				cand.Reason = "GitHub records this repository as a fork of " + r.Parent.FullName
			} else if r.Fork {
				cand.Reason = "GitHub fork flag is set but parent metadata is missing"
			} else {
				cand.Reason = "listed in the organisation. Not an explicit GitHub fork."
			}
			out = append(out, cand)
		}
		page++
	}
	return out, nil
}

type ghRepo struct {
	FullName      string `json:"full_name"`
	Fork          bool   `json:"fork"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	Parent        *struct {
		FullName string `json:"full_name"`
	} `json:"parent"`
}
