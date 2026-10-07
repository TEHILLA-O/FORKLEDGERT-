package evidence

import "testing"

func TestFromWeightsCategories(t *testing.T) {
	a := FromWeights("test", []Item{{Weight: 95}})
	if a.Level != LevelVeryHigh || a.Score != 95 {
		t.Fatalf("%+v", a)
	}
	b := FromWeights("test", []Item{{Weight: 40}, {Weight: 5}})
	if b.Level != LevelModerate {
		t.Fatalf("%+v", b)
	}
	c := FromWeights("test", nil)
	if c.Level != LevelNone {
		t.Fatalf("%+v", c)
	}
}

func TestActionable(t *testing.T) {
	if FromWeights("", []Item{{Weight: 50}}).Actionable() {
		t.Fatal("moderate should not be silently actionable")
	}
	if !FromWeights("", []Item{{Weight: 80}}).Actionable() {
		t.Fatal("high should be actionable")
	}
}
