package git

import "testing"

func TestParseRemotes(t *testing.T) {
	in := "origin\tgit@github.com:acme/widget.git (fetch)\norigin\tgit@github.com:acme/widget.git (push)\nupstream\thttps://github.com/example/widget.git (fetch)\n"
	got := parseRemotes(in)
	if len(got) != 2 {
		t.Fatalf("got %d remotes", len(got))
	}
	if got[0].Name != "origin" || got[1].Name != "upstream" {
		t.Fatalf("names: %+v", got)
	}
}

func TestParseCherry(t *testing.T) {
	in := "+ 1111111 unique patch\n- 2222222 already upstream\n"
	got := parseCherry(in)
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Mark != CherryNotPicked || got[1].Mark != CherryEquivalent {
		t.Fatalf("%+v", got)
	}
}

func TestParseHunks(t *testing.T) {
	patch := `diff --git a/src/auth.go b/src/auth.go
index 111..222 100644
--- a/src/auth.go
+++ b/src/auth.go
@@ -1,3 +1,4 @@ func TokenRefresh() string {
 package src
`
	h := ParseHunks(patch)
	if len(h) != 1 {
		t.Fatalf("hunks %d", len(h))
	}
	if h[0].Function != "func TokenRefresh() string {" {
		t.Fatalf("function %q", h[0].Function)
	}
	if h[0].File != "src/auth.go" {
		t.Fatalf("file %q", h[0].File)
	}
}

func TestRejectUnsafeRev(t *testing.T) {
	if err := rejectUnsafeRev("-c"); err == nil {
		t.Fatal("expected dash rejection")
	}
	if err := rejectUnsafeRev("HEAD"); err != nil {
		t.Fatal(err)
	}
}

func TestParseLeftRightCount(t *testing.T) {
	c, err := parseLeftRightCount("12\t4")
	if err != nil {
		t.Fatal(err)
	}
	if c.Behind != 12 || c.Ahead != 4 {
		t.Fatalf("%+v", c)
	}
}
