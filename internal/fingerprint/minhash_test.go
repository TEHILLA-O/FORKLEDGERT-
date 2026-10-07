package fingerprint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMinHashIdenticalTrees(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	for _, dir := range []string{a, b} {
		if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sa, err := MinHash{}.Compute(nil, a)
	if err != nil {
		t.Fatal(err)
	}
	sb, err := MinHash{}.Compute(nil, b)
	if err != nil {
		t.Fatal(err)
	}
	sim, err := MinHashSimilarity(sa.Value, sb.Value)
	if err != nil {
		t.Fatal(err)
	}
	if sim < 0.99 {
		t.Fatalf("identical trees should match, got %f", sim)
	}
}
