package patches

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	cveRe  = regexp.MustCompile(`(?i)\bCVE-\d{4}-\d+\b`)
	ghsaRe = regexp.MustCompile(`(?i)\bGHSA-[a-z0-9]+-[a-z0-9]+-[a-z0-9]+\b`)
	hashRe = regexp.MustCompile(`(?i)\b(?:fixes|closes|resolves)[:\s]+#(\d{1,7})\b`)
	bareRe = regexp.MustCompile(`#(\d{2,7})`)
)

func IssueIDs(subject, body string) []string {
	text := subject + "\n" + body
	seen := map[string]struct{}{}
	var out []string
	add := func(id string) {
		id = strings.ToUpper(id)
		if strings.HasPrefix(id, "#") {
			id = "ISSUE" + id
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, m := range cveRe.FindAllString(text, -1) {
		add(m)
	}
	for _, m := range ghsaRe.FindAllString(text, -1) {
		add(m)
	}
	for _, m := range hashRe.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("#" + m[1])
		}
	}
	if len(out) == 0 {
		for _, m := range bareRe.FindAllStringSubmatch(text, 3) {
			if len(m) > 1 {
				add("#" + m[1])
			}
		}
	}
	return out
}

// TokenizePatch keeps added and removed lines, drops hunk coordinates, and
// squashes whitespace. Two patches that fix the same function in different
// wrapping still tend to share these tokens. False positives happen around
// mechanical reformats. Weight this layer below patch-id.
func TokenizePatch(patch string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "@@") {
			continue
		}
		if line[0] != '+' && line[0] != '-' {
			continue
		}
		tok := normalizeToken(line[1:])
		if tok == "" || tok == "{" || tok == "}" {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}

func normalizeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func TokenSimilarity(a, b []string) float64 {
	return jaccard(a, b)
}
