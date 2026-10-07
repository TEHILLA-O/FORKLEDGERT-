package fingerprint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizedTreeReady(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sig, err := NormalizedTree{}.Compute(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !sig.Ready || sig.Value == "" {
		t.Fatalf("%+v", sig)
	}
}

func TestPlaceholdersNotReady(t *testing.T) {
	reg := DefaultRegistry()
	sigs, err := reg.Compute(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var minhash Signal
	for _, s := range sigs {
		if s.Kind == KindMinHash {
			minhash = s
		}
	}
	if minhash.Ready {
		t.Fatal("MinHash must remain a reserved signal until it is implemented")
	}
}
