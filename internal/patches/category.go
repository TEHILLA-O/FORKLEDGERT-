package patches

import (
	"regexp"
	"strings"
)

type categoryRule struct {
	category Category
	reason   string
	pattern  *regexp.Regexp
}

var categoryRules = []categoryRule{
	{CategorySecurity, "commit message mentions a vulnerability or CVE", regexp.MustCompile(`(?i)\b(cve-\d{4}-\d+|vulnerab|security|advisory|exploit|authn|authz)\b`)},
	{CategoryTemporaryWorkaround, "commit message describes a workaround", regexp.MustCompile(`(?i)\b(workaround|hack|tmp|temporary|band-aid|bandaid)\b`)},
	{CategoryCompliance, "commit message mentions licence or compliance", regexp.MustCompile(`(?i)\b(licen[sc]e|compliance|spdx|export control)\b`)},
	{CategoryBranding, "commit message mentions branding", regexp.MustCompile(`(?i)\b(brand|logo|rename product|white[- ]label)\b`)},
	{CategoryPerformance, "commit message mentions performance", regexp.MustCompile(`(?i)\b(perf|performance|optimis|optimiz|latency|throughput)\b`)},
	{CategoryConfiguration, "commit message mentions configuration", regexp.MustCompile(`(?i)\b(config|setting|default value)\b`)},
	{CategoryCompatibility, "commit message mentions compatibility", regexp.MustCompile(`(?i)\b(compat|backport|downstream)\b`)},
	{CategoryFeature, "commit message describes a feature", regexp.MustCompile(`(?i)\b(feat(\(.+\))?|feature|add support)\b`)},
	{CategoryExperimental, "commit message looks experimental", regexp.MustCompile(`(?i)\b(wip|experiment|poc|prototype)\b`)},
	{CategoryBugFix, "commit message describes a fix", regexp.MustCompile(`(?i)\b(fix|bug|issue|defect|regression)\b`)},
}

// ClassifyMessage is a first pass, not a courtroom. The commit message is a
// noisy signal. False positives happen around words like "security group" and
// "fixed typo in docs". Operators can override in forkledger.yaml.
func ClassifyMessage(subject, body string) (Category, string) {
	text := subject + "\n" + body
	for _, rule := range categoryRules {
		if rule.pattern.MatchString(text) {
			return rule.category, rule.reason
		}
	}
	return CategoryUnknown, "no message heuristic matched"
}

func LooksSecurity(p Patch) bool {
	if p.Category == CategorySecurity {
		return true
	}
	blob := strings.ToLower(p.Subject + " " + p.Body)
	return strings.Contains(blob, "cve-") || strings.Contains(blob, "vulnerab")
}
