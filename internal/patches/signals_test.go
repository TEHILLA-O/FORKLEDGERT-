package patches

import "testing"

func TestIssueIDs(t *testing.T) {
	ids := IssueIDs("security: fix CVE-2024-0001", "Also closes #42 and GHSA-abcd-efgh-ijkl")
	if len(ids) < 2 {
		t.Fatalf("ids %v", ids)
	}
}

func TestTokenizePatchSimilarity(t *testing.T) {
	a := TokenizePatch("@@ -1 +1 @@\n-return v1\n+return v1-internal\n")
	b := TokenizePatch("@@ -4 +4 @@\n-return v1\n+return v1-internal\n")
	if TokenSimilarity(a, b) < 0.9 {
		t.Fatalf("expected high similarity, tokens a=%v b=%v", a, b)
	}
}

func TestParseGoFunctions(t *testing.T) {
	src := "package src\nfunc TokenRefresh() string { return \"x\" }\nfunc Helper() int { return 1 }\n"
	got := ParseGoFunctions(src, "auth.go")
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}
