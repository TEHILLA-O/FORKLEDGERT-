package fingerprint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Kind is a future detached-copy signal. Most of these are not implemented yet.
// The registry exists so a later MinHash does not require rewriting detection.
type Kind string

const (
	KindNormalizedFileHash Kind = "NORMALIZED_FILE_HASH"
	KindDirectorySignature Kind = "DIRECTORY_SIGNATURE"
	KindMinHash            Kind = "MINHASH"
	KindSimHash            Kind = "SIMHASH"
	KindTLSH               Kind = "TLSH"
	KindSsdeep             Kind = "SSDEEP"
	KindASTFingerprint     Kind = "AST_FINGERPRINT"
	KindPackageMetadata    Kind = "PACKAGE_METADATA"
	KindLicense            Kind = "LICENSE"
	KindReadme             Kind = "README"
	KindSemanticSimilarity Kind = "SEMANTIC_SIMILARITY"
)

type Signal struct {
	Kind    Kind           `json:"kind"`
	Value   string         `json:"value,omitempty"`
	Weight  int            `json:"weight"`
	Ready   bool           `json:"ready"`
	Note    string         `json:"note,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type Computer interface {
	Kind() Kind
	Compute(ctx context.Context, root string) (Signal, error)
}

type Registry struct {
	computers []Computer
}

func NewRegistry(computers ...Computer) *Registry {
	return &Registry{computers: computers}
}

func DefaultRegistry() *Registry {
	return NewRegistry(
		NormalizedTree{},
		DirectorySignature{},
		MinHash{},
		SimHash{},
		placeholder{KindTLSH, "TLSH is reserved for fuzzy hashing of larger blobs."},
		placeholder{KindSsdeep, "ssdeep is reserved for fuzzy hashing when the operator enables it."},
		placeholder{KindASTFingerprint, "AST fingerprints need language parsers. Git hunk headers stand in for now."},
		placeholder{KindSemanticSimilarity, "Semantic comparison is an interpretation aid, not a source of truth."},
	)
}

func (r *Registry) Compute(ctx context.Context, root string) ([]Signal, error) {
	var out []Signal
	for _, c := range r.computers {
		sig, err := c.Compute(ctx, root)
		if err != nil {
			return nil, err
		}
		out = append(out, sig)
	}
	return out, nil
}

type placeholder struct {
	kind Kind
	note string
}

func (p placeholder) Kind() Kind { return p.kind }

func (p placeholder) Compute(ctx context.Context, root string) (Signal, error) {
	_ = ctx
	_ = root
	return Signal{Kind: p.kind, Ready: false, Note: p.note}, nil
}

type NormalizedTree struct{}

func (NormalizedTree) Kind() Kind { return KindNormalizedFileHash }

func (NormalizedTree) Compute(ctx context.Context, root string) (Signal, error) {
	_ = ctx
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == "vendor" || base == ".forkledger" {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipFile(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(data) > 1<<20 {
			return nil
		}
		norm := normalizeBytes(data)
		sum := sha256.Sum256(norm)
		files[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		return Signal{}, err
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s %s\n", files[k], k)
	}
	return Signal{
		Kind:   KindNormalizedFileHash,
		Value:  hex.EncodeToString(h.Sum(nil)),
		Weight: 10,
		Ready:  true,
		Note:   "SHA-256 of normalized file contents. Similarity is not ancestry.",
		Details: map[string]any{
			"files": len(files),
		},
	}, nil
}

type DirectorySignature struct{}

func (DirectorySignature) Kind() Kind { return KindDirectorySignature }

func (DirectorySignature) Compute(ctx context.Context, root string) (Signal, error) {
	_ = ctx
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		base := d.Name()
		if base == ".git" || base == "node_modules" || base == "vendor" {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if rel != "." {
			dirs = append(dirs, rel)
		}
		return nil
	})
	if err != nil {
		return Signal{}, err
	}
	sort.Strings(dirs)
	h := sha256.Sum256([]byte(strings.Join(dirs, "\n")))
	return Signal{
		Kind:   KindDirectorySignature,
		Value:  hex.EncodeToString(h[:]),
		Weight: 6,
		Ready:  true,
		Note:   "Directory layout fingerprint. Useful when someone copied a tree and started a new Git history.",
		Details: map[string]any{"directories": len(dirs)},
	}, nil
}

func shouldSkipFile(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	if strings.HasSuffix(base, ".png") || strings.HasSuffix(base, ".jpg") || strings.HasSuffix(base, ".pdf") {
		return true
	}
	return false
}

func normalizeBytes(data []byte) []byte {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	s = strings.TrimSpace(s)
	return []byte(s)
}
