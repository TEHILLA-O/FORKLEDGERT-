package git

import (
	"fmt"
	"net/url"
	"strings"
)

func rejectUnsafeRev(rev string) error {
	if rev == "" {
		return fmt.Errorf("empty revision")
	}
	if strings.HasPrefix(rev, "-") {
		return fmt.Errorf("revision must not start with a dash")
	}
	if strings.ContainsAny(rev, "\n\r;|&$`") {
		return fmt.Errorf("revision contains unsafe characters")
	}
	if strings.Contains(rev, "..") && !strings.Contains(rev, "...") {
		// Range syntax is allowed. Bare path traversal is not a revision.
	}
	return nil
}

func rejectUnsafePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	if strings.Contains(path, "\x00") || strings.ContainsAny(path, "\n\r") {
		return fmt.Errorf("path contains unsafe characters")
	}
	if strings.HasPrefix(path, "-") {
		return fmt.Errorf("path must not start with a dash")
	}
	return nil
}

func rejectUnsafeURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	if strings.ContainsAny(raw, "\n\r;|&$`") {
		return fmt.Errorf("url contains unsafe characters")
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "file:") || strings.HasPrefix(lower, "ext::") {
		return fmt.Errorf("refusing file or ext Git transport")
	}
	if strings.Contains(lower, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid url")
		}
		switch strings.ToLower(u.Scheme) {
		case "https", "http", "ssh", "git":
		default:
			return fmt.Errorf("unsupported git url scheme %q", u.Scheme)
		}
	}
	return nil
}
