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
	var minhash, simhash, tlsh Signal
	for _, s := range sigs {
		switch s.Kind {
		case KindMinHash:
			minhash = s
		case KindSimHash:
			simhash = s
		case KindTLSH:
			tlsh = s
		}
	}
	if !minhash.Ready || minhash.Value == "" {
		t.Fatalf("MinHash should be ready: %+v", minhash)
	}
	if !simhash.Ready || simhash.Value == "" {
		t.Fatalf("SimHash should be ready: %+v", simhash)
	}
	if tlsh.Ready {
		t.Fatal("TLSH remains reserved until a real implementation exists")
	}
}
