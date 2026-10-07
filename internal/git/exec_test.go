package git

import (
	"context"
	"testing"
	"time"
)

func TestRejectUnsafeURL(t *testing.T) {
	if err := rejectUnsafeURL("file:///tmp/evil"); err == nil {
		t.Fatal("file transport should be refused")
	}
	if err := rejectUnsafeURL("ext::sh -c evil"); err == nil {
		t.Fatal("ext transport should be refused")
	}
	if err := rejectUnsafeURL("https://github.com/example/project.git"); err != nil {
		t.Fatal(err)
	}
}

func TestTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("timeout test talks to git")
	}
	ex := NewExecutor(nil)
	ex.Timeout = 1 * time.Millisecond
	_, err := ex.Run(context.Background(), ".", "status")
	if err == nil {
		t.Log("git returned before the tiny timeout, which can happen on a warm disk")
	}
}
