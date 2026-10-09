package store

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/TEHILLA-O/fork_ledger/internal/report"
)

type Attention struct {
	ID        string `json:"id"`
	Repo      string `json:"repo"`
	Upstream  string `json:"upstream"`
	Action    string `json:"action"`
	Debt      string `json:"debt"`
	Missing   int    `json:"missing"`
	Owner     string `json:"owner,omitempty"`
	Reviewed  string `json:"reviewed,omitempty"`
	Health    int    `json:"health"`
	UpdatedAt string `json:"updated_at"`
}

type Store struct {
	Dir string
	mu  sync.Mutex
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "scans"), 0o755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

func (s *Store) Put(rep *report.Report) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := scanID(rep)
	path := filepath.Join(s.Dir, "scans", id+".json")
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	idx, err := s.loadIndex()
	if err != nil {
		return "", err
	}
	row := Attention{
		ID:        id,
		Repo:      rep.Repository.Path,
		Upstream:  rep.Upstream.Selected,
		Action:    string(rep.Lifecycle.Recommended),
		Debt:      rep.Debt.Level,
		Missing:   rep.Security.MissingUpstreamFixes,
		Health:    rep.Health.Score,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	idx = upsert(idx, row)
	if err := s.saveIndex(idx); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) Get(id string) (*report.Report, error) {
	data, err := os.ReadFile(filepath.Join(s.Dir, "scans", id+".json"))
	if err != nil {
		return nil, err
	}
	var rep report.Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func (s *Store) Attention() ([]Attention, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	sort.Slice(idx, func(i, j int) bool {
		if idx[i].Missing != idx[j].Missing {
			return idx[i].Missing > idx[j].Missing
		}
		return idx[i].Health < idx[j].Health
	})
	return idx, nil
}

func (s *Store) loadIndex() ([]Attention, error) {
	path := filepath.Join(s.Dir, "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var idx []Attention
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func (s *Store) saveIndex(idx []Attention) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir, "index.json"), append(data, '\n'), 0o644)
}

func scanID(rep *report.Report) string {
	h := sha1.Sum([]byte(rep.Repository.Path + "|" + rep.Repository.Head + "|" + rep.GeneratedAt.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h[:8])
}

func upsert(idx []Attention, row Attention) []Attention {
	for i, r := range idx {
		if r.Repo == row.Repo {
			idx[i] = row
			return idx
		}
	}
	return append(idx, row)
}

func DefaultDir(repo string) string {
	return filepath.Join(repo, ".forkledger", "store")
}

func MustOpen(dir string) *Store {
	s, err := Open(dir)
	if err != nil {
		panic(fmt.Sprintf("store: %v", err))
	}
	return s
}
