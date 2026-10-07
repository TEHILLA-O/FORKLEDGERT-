package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GitHubClient struct {
	HTTP    *http.Client
	Token   string
	BaseURL string
}

type GitHubRepository struct {
	FullName      string           `json:"full_name"`
	Fork          bool             `json:"fork"`
	DefaultBranch string           `json:"default_branch"`
	Parent        *GitHubParent    `json:"parent"`
	Source        *GitHubParent    `json:"source"`
	License       *GitHubLicense   `json:"license"`
	Description   string           `json:"description"`
}

type GitHubParent struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	CloneURL      string `json:"clone_url"`
}

type GitHubLicense struct {
	SPDX string `json:"spdx_id"`
}

func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		HTTP:    &http.Client{Timeout: 15 * time.Second},
		Token:   token,
		BaseURL: "https://api.github.com",
	}
}

func (c *GitHubClient) Repo(ctx context.Context, owner, name string) (GitHubRepository, error) {
	if c == nil || c.HTTP == nil {
		return GitHubRepository{}, fmt.Errorf("github client not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/repos/"+owner+"/"+name, nil)
	if err != nil {
		return GitHubRepository{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "forkledger")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return GitHubRepository{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return GitHubRepository{}, fmt.Errorf("github repository %s/%s not found", owner, name)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return GitHubRepository{}, fmt.Errorf("github api rate limited")
	}
	if resp.StatusCode >= 300 {
		return GitHubRepository{}, fmt.Errorf("github api status %d", resp.StatusCode)
	}
	var repo GitHubRepository
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return GitHubRepository{}, err
	}
	return repo, nil
}
