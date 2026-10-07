package git

import (
	"bufio"
	"strings"
	"strconv"
	"time"
)

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func parseRemotes(stdout string) []Remote {
	seen := map[string]Remote{}
	order := []string{}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		name, url := fields[0], fields[1]
		if _, ok := seen[name]; !ok {
			order = append(order, name)
			seen[name] = Remote{Name: name, URL: url}
			continue
		}
	}
	out := make([]Remote, 0, len(order))
	for _, name := range order {
		out = append(out, seen[name])
	}
	return out
}

func parseLog(stdout string) ([]Commit, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	records := strings.Split(stdout, "\x1e")
	out := make([]Commit, 0, len(records))
	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.Split(rec, "\x1f")
		if len(parts) < 6 {
			continue
		}
		parents := []string{}
		if strings.TrimSpace(parts[1]) != "" {
			parents = strings.Fields(parts[1])
		}
		var when time.Time
		if t, err := time.Parse(time.RFC3339, parts[4]); err == nil {
			when = t
		}
		body := ""
		if len(parts) > 6 {
			body = strings.TrimSpace(parts[6])
		}
		out = append(out, Commit{
			Hash:    parts[0],
			Parents: parents,
			Author:  parts[2],
			Email:   parts[3],
			Date:    when,
			Subject: parts[5],
			Body:    body,
		})
	}
	return out, nil
}

func parseRefs(stdout string) []Ref {
	var out []Ref
	for _, line := range splitLines(stdout) {
		hash, name, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		out = append(out, Ref{Name: name, Hash: hash})
	}
	return out
}

func parseNumStat(stdout string) []FileChange {
	var out []FileChange
	for _, line := range splitLines(stdout) {
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		added, _ := atoiOrDash(fields[0])
		deleted, _ := atoiOrDash(fields[1])
		path := fields[2]
		oldPath := ""
		status := "M"
		if strings.Contains(path, " => ") {
			status = "R"
			oldPath, path, _ = strings.Cut(path, " => ")
			oldPath = strings.Trim(oldPath, "{}")
			path = strings.Trim(path, "{}")
		} else if added > 0 && deleted == 0 && fields[0] != "-" {
			status = "A"
		} else if added == 0 && deleted > 0 {
			status = "D"
		}
		out = append(out, FileChange{
			Path:    path,
			OldPath: oldPath,
			Status:  status,
			Added:   added,
			Deleted: deleted,
		})
	}
	return out
}

func atoiOrDash(s string) (int, error) {
	if s == "-" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func parsePatchIDs(stdout string) []PatchID {
	var out []PatchID
	for _, line := range splitLines(stdout) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		id := PatchID{ID: fields[0]}
		if len(fields) > 1 {
			id.Commit = fields[1]
		}
		out = append(out, id)
	}
	return out
}

func parseCherry(stdout string) []Cherry {
	var out []Cherry
	for _, line := range splitLines(stdout) {
		if len(line) < 3 {
			continue
		}
		mark := CherryNotPicked
		if strings.HasPrefix(line, "- ") {
			mark = CherryEquivalent
		} else if strings.HasPrefix(line, "+ ") {
			mark = CherryNotPicked
		} else {
			continue
		}
		rest := strings.TrimSpace(line[2:])
		hash, subject, _ := strings.Cut(rest, " ")
		out = append(out, Cherry{Commit: hash, Mark: mark, Subject: strings.TrimSpace(subject)})
	}
	return out
}

func parseMergeTree(stdout, stderr string, clean bool) MergeTreeResult {
	res := MergeTreeResult{Clean: clean, Messages: strings.TrimSpace(stderr)}
	if stdout == "" {
		return res
	}
	parts := strings.Split(stdout, "\x00")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if looksLikeHash(p) {
			continue
		}
		if strings.Contains(p, "CONFLICT") {
			res.Clean = false
			res.Informational = joinParagraph(res.Informational, p)
			continue
		}
		if strings.Contains(p, "/") || strings.Contains(p, ".") || !strings.Contains(p, " ") {
			res.Conflicted = append(res.Conflicted, p)
			res.Clean = false
		}
	}
	return res
}

func parseLSRemote(stdout string) []Ref {
	var out []Ref
	for _, line := range splitLines(stdout) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		out = append(out, Ref{Hash: fields[0], Name: fields[1]})
	}
	return out
}

func looksLikeHash(s string) bool {
	if len(s) < 7 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func joinParagraph(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "\n" + b
}

// ParseHunks extracts function names Git already placed in hunk headers.
// This is not a parser for every language. It is Git being helpful, which we accept.
func ParseHunks(patch string) []Hunk {
	var hunks []Hunk
	currentFile := ""
	sc := bufio.NewScanner(strings.NewReader(patch))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "diff --git ") {
			currentFile = parseDiffGitPath(line)
			continue
		}
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			continue
		}
		if !strings.HasPrefix(line, "@@") {
			continue
		}
		h, ok := parseHunkHeader(line)
		if !ok {
			continue
		}
		h.File = currentFile
		hunks = append(hunks, h)
	}
	return hunks
}

func parseDiffGitPath(line string) string {
	// diff --git a/foo b/foo
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return ""
	}
	return strings.TrimPrefix(parts[3], "b/")
}

func parseHunkHeader(line string) (Hunk, bool) {
	// @@ -12,7 +14,9 @@ func Name(
	rest := strings.TrimPrefix(line, "@@")
	rest = strings.TrimSpace(rest)
	at := strings.Index(rest, "@@")
	if at < 0 {
		return Hunk{}, false
	}
	nums := strings.TrimSpace(rest[:at])
	fn := strings.TrimSpace(rest[at+2:])
	fields := strings.Fields(nums)
	h := Hunk{Function: fn}
	if len(fields) >= 1 {
		h.OldStart, h.OldCount = parseHunkPair(fields[0])
	}
	if len(fields) >= 2 {
		h.NewStart, h.NewCount = parseHunkPair(fields[1])
	}
	return h, true
}

func parseHunkPair(s string) (start, count int) {
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")
	a, b, ok := strings.Cut(s, ",")
	start, _ = strconv.Atoi(a)
	if ok {
		count, _ = strconv.Atoi(b)
	} else {
		count = 1
	}
	return start, count
}
