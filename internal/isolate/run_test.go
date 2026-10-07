package isolate

import (
	"context"
	"testing"
)

func TestRejectsShell(t *testing.T) {
	_, err := Run(context.Background(), t.TempDir(), []string{"echo", "hi; rm -rf /"})
	if err == nil {
		t.Fatal("expected rejection")
	}
}

func TestEmpty(t *testing.T) {
	_, err := Run(context.Background(), t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected empty command error")
	}
}
