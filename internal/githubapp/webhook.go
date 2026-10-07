package githubapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Result struct {
	Delivery  string `json:"delivery"`
	Event     string `json:"event"`
	Duplicate bool   `json:"duplicate"`
	Action    string `json:"action,omitempty"`
	Repo      string `json:"repo,omitempty"`
}

func Verify(secret, header string, body []byte) error {
	if secret == "" {
		return fmt.Errorf("webhook secret is empty")
	}
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "sha256=") {
		return fmt.Errorf("missing sha256 signature")
	}
	got, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return fmt.Errorf("invalid signature encoding")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	sum := mac.Sum(nil)
	if !hmac.Equal(sum, got) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func Handle(storeDir, event, delivery string, body []byte) (Result, error) {
	res := Result{Delivery: delivery, Event: event}
	if delivery == "" {
		return res, fmt.Errorf("missing delivery id")
	}
	seenDir := filepath.Join(storeDir, "deliveries")
	_ = os.MkdirAll(seenDir, 0o755)
	marker := filepath.Join(seenDir, delivery)
	if _, err := os.Stat(marker); err == nil {
		res.Duplicate = true
		res.Action = "ignored"
		return res, nil
	}
	if err := os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644); err != nil {
		return res, err
	}

	switch event {
	case "ping":
		res.Action = "pong"
	case "push", "pull_request", "installation", "installation_repositories", "repository":
		res.Action = "accepted"
		res.Repo = extractRepo(body)
	default:
		res.Action = "ignored_event"
	}
	return res, nil
}

func extractRepo(body []byte) string {
	var payload struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.Repository.FullName
}
