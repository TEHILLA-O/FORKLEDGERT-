package upstream

import (
	"net/url"
	"strings"
)

type GitHubRepo struct {
	Host  string
	Owner string
	Name  string
}

func (g GitHubRepo) String() string {
	if g.Owner == "" || g.Name == "" {
		return ""
	}
	host := g.Host
	if host == "" {
		host = "github.com"
	}
	return host + "/" + g.Owner + "/" + g.Name
}

func (g GitHubRepo) HTTPS() string {
	if g.Owner == "" {
		return ""
	}
	host := g.Host
	if host == "" {
		host = "github.com"
	}
	return "https://" + host + "/" + g.Owner + "/" + g.Name + ".git"
}

func ParseGitHub(raw string) (GitHubRepo, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if raw == "" {
		return GitHubRepo{}, false
	}

	if strings.Contains(raw, "github.com:") && strings.Contains(raw, "@") {
		_, rest, ok := strings.Cut(raw, "github.com:")
		if !ok {
			return GitHubRepo{}, false
		}
		owner, name, ok := cutOwnerName(rest)
		if !ok {
			return GitHubRepo{}, false
		}
		return GitHubRepo{Host: "github.com", Owner: owner, Name: name}, true
	}

	if !strings.Contains(raw, "://") {
		if strings.HasPrefix(raw, "github.com/") {
			raw = "https://" + raw
		} else if strings.Count(raw, "/") == 1 {
			raw = "https://github.com/" + raw
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return GitHubRepo{}, false
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	if host != "github.com" && host != "www.github.com" {
		if host == "" {
			return GitHubRepo{}, false
		}
	}
	path := strings.Trim(u.Path, "/")
	owner, name, ok := cutOwnerName(path)
	if !ok {
		return GitHubRepo{}, false
	}
	if host == "" {
		host = "github.com"
	}
	return GitHubRepo{Host: host, Owner: owner, Name: name}, true
}

func cutOwnerName(path string) (string, string, bool) {
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func NormalizeRemote(raw string) string {
	if g, ok := ParseGitHub(raw); ok {
		return g.String()
	}
	return strings.TrimSuffix(strings.TrimSpace(raw), ".git")
}
