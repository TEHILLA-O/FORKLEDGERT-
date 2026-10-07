package fingerprint

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const minHashSize = 128

type MinHash struct{}

func (MinHash) Kind() Kind { return KindMinHash }

func (MinHash) Compute(ctx context.Context, root string) (Signal, error) {
	_ = ctx
	values, err := fileHashes(root)
	if err != nil {
		return Signal{}, err
	}
	sig := minhash(values, minHashSize)
	return Signal{
		Kind:   KindMinHash,
		Value:  hex.EncodeToString(sig),
		Weight: 12,
		Ready:  true,
		Note:   "MinHash over normalised file hashes. Useful when two trees share no Git history. Collision is possible. Ancestry is not.",
		Details: map[string]any{
			"files":  len(values),
			"width":  minHashSize,
			"method": "minhash-sha256",
		},
	}, nil
}

func fileHashes(root string) ([][]byte, error) {
	var hashes [][]byte
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == "vendor" || base == ".forkledger" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if shouldSkipFile(filepath.ToSlash(rel)) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(data) > 1<<20 {
			return nil
		}
		sum := sha256.Sum256(normalizeBytes(data))
		hashes = append(hashes, sum[:])
		return nil
	})
	return hashes, err
}

func minhash(values [][]byte, width int) []byte {
	mins := make([]uint64, width)
	for i := range mins {
		mins[i] = ^uint64(0)
	}
	for _, v := range values {
		for i := 0; i < width; i++ {
			h := keyedHash(v, uint64(i+1))
			if h < mins[i] {
				mins[i] = h
			}
		}
	}
	out := make([]byte, width*8)
	for i, m := range mins {
		binary.BigEndian.PutUint64(out[i*8:], m)
	}
	return out
}

func keyedHash(v []byte, key uint64) uint64 {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], key)
	h := sha256.New()
	h.Write(buf[:])
	h.Write(v)
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

func MinHashSimilarity(a, b string) (float64, error) {
	da, err := hex.DecodeString(a)
	if err != nil {
		return 0, err
	}
	db, err := hex.DecodeString(b)
	if err != nil {
		return 0, err
	}
	if len(da) != len(db) || len(da)%8 != 0 || len(da) == 0 {
		return 0, fmt.Errorf("minhash width mismatch")
	}
	match := 0
	n := len(da) / 8
	for i := 0; i < n; i++ {
		if binary.BigEndian.Uint64(da[i*8:]) == binary.BigEndian.Uint64(db[i*8:]) {
			match++
		}
	}
	return float64(match) / float64(n), nil
}

func SortedHex(values []string) string {
	sort.Strings(values)
	return stringsJoin(values)
}

func stringsJoin(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}
