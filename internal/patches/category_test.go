package patches

import "testing"

func TestClassifyMessage(t *testing.T) {
	cases := []struct {
		subject string
		want    Category
	}{
		{"security: fix CVE-2024-0001 in parser", CategorySecurity},
		{"fix helper off-by-one", CategoryBugFix},
		{"feat: add Extra helper", CategoryFeature},
		{"brand: add internal product name", CategoryBranding},
		{"perf: reduce allocations", CategoryPerformance},
		{"temporary workaround for vendor API", CategoryTemporaryWorkaround},
		{"totally mysterious", CategoryUnknown},
	}
	for _, tc := range cases {
		got, _ := ClassifyMessage(tc.subject, "")
		if got != tc.want {
			t.Errorf("%q: got %s want %s", tc.subject, got, tc.want)
		}
	}
}

func TestClassifyFalsePositiveDocsFix(t *testing.T) {
	got, reason := ClassifyMessage("fix typo in README", "")
	if got != CategoryBugFix {
		t.Fatalf("got %s", got)
	}
	if reason == "" {
		t.Fatal("expected a reason even when the heuristic is crude")
	}
}
