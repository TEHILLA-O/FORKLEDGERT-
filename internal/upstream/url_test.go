package upstream

import "testing"

func TestParseGitHub(t *testing.T) {
	cases := []string{
		"https://github.com/redis/redis.git",
		"git@github.com:redis/redis.git",
		"github.com/redis/redis",
		"redis/redis",
	}
	for _, c := range cases {
		g, ok := ParseGitHub(c)
		if !ok {
			t.Fatalf("failed %q", c)
		}
		if g.Owner != "redis" || g.Name != "redis" {
			t.Fatalf("%q -> %+v", c, g)
		}
	}
}

func TestParseGitHubRejectsFile(t *testing.T) {
	if _, ok := ParseGitHub("file:///tmp/repo"); ok {
		t.Fatal("file urls are not github repos")
	}
}
