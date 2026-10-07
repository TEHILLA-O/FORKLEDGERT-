package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forkledger/forkledger/internal/config"
	"github.com/forkledger/forkledger/internal/git"
	"github.com/forkledger/forkledger/internal/logging"
	"github.com/forkledger/forkledger/internal/policy"
	"github.com/forkledger/forkledger/internal/report"
	"github.com/forkledger/forkledger/internal/scanner"
	"github.com/forkledger/forkledger/internal/upgrade"
	"github.com/forkledger/forkledger/internal/version"
)

type Globals struct {
	JSON    bool
	Quiet   bool
	Verbose bool
	Debug   bool
	Config  string
	Output  string
	Offline bool
	Path    string
	Log     *logging.Logger
}

func Run(args []string) int {
	if len(args) == 0 {
		printHelp(os.Stderr)
		return ExitUsage
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "analyze":
		return cmdAnalyze(rest)
	case "upstream":
		return cmdUpstream(rest)
	case "patches":
		return cmdPatches(rest)
	case "security":
		return cmdSecurity(rest)
	case "upgrade":
		return cmdUpgrade(rest)
	case "report":
		return cmdReport(rest)
	case "policy":
		return cmdPolicy(rest)
	case "explain":
		return cmdExplain(rest)
	case "doctor":
		return cmdDoctor(rest)
	case "version", "--version", "-v":
		return cmdVersion(rest)
	case "help", "--help", "-h":
		printHelp(os.Stdout)
		return ExitOK
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		printHelp(os.Stderr)
		return ExitUsage
	}
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `ForkLedger understands modified upstream software.

Usage:
  forkledger <command> [flags]

Commands:
  analyze         Inspect a repository and produce a fork report
  upstream        Show upstream candidates and evidence
  patches         List the local patch ledger
  security        Show vulnerability applicability in fork context
  upgrade         Simulate an upgrade against a target ref
  report          Render an existing JSON report
  policy          Check configured policy rules
  explain         Explain a conclusion from the latest report
  doctor          Check Git, configuration, and environment
  version         Print the version

Global flags:
  --json          Machine readable output
  --quiet         Suppress diagnostics
  --verbose       Extra diagnostics
  --debug         Include Git command traces
  --config PATH   Configuration file
  --output PATH   Write JSON report to PATH
  --offline       Do not call network services
  --path PATH     Repository path (default .)

Exit codes:
  0   completed without blocking findings
  1   findings present
  2   usage error
  3   not a git repository
  4   analysis failed
  5   invalid configuration
  20  security findings requiring attention
  30  policy violations
`)
}

func parseGlobals(args []string) (Globals, []string, error) {
	fs := flag.NewFlagSet("forkledger", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	g := Globals{Path: "."}
	fs.BoolVar(&g.JSON, "json", false, "")
	fs.BoolVar(&g.Quiet, "quiet", false, "")
	fs.BoolVar(&g.Verbose, "verbose", false, "")
	fs.BoolVar(&g.Debug, "debug", false, "")
	fs.StringVar(&g.Config, "config", "", "")
	fs.StringVar(&g.Output, "output", "", "")
	fs.BoolVar(&g.Offline, "offline", false, "")
	fs.StringVar(&g.Path, "path", ".", "")
	if err := fs.Parse(args); err != nil {
		return g, nil, err
	}
	level := logging.LevelInfo
	switch {
	case g.Quiet:
		level = logging.LevelQuiet
	case g.Debug:
		level = logging.LevelDebug
	case g.Verbose:
		level = logging.LevelVerbose
	}
	g.Log = logging.New(level)
	return g, fs.Args(), nil
}

func loadConfig(g Globals) (config.Config, int) {
	path := g.Config
	if path == "" {
		found, err := config.Find(g.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			return config.Config{}, ExitConfig
		}
		path = found
	}
	if path == "" {
		return config.Default(), ExitOK
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return config.Config{}, ExitConfig
	}
	g.Log.Verbose("loaded %s", path)
	return cfg, ExitOK
}

func cmdAnalyze(args []string) int {
	g, extra, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	if len(extra) > 0 && g.Path == "." {
		g.Path = extra[0]
	}
	cfg, code := loadConfig(g)
	if code != ExitOK {
		return code
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	eng := scanner.New(g.Log)
	rep, err := eng.Analyze(ctx, scanner.Options{
		Path:        g.Path,
		Config:      cfg,
		Offline:     g.Offline,
		Log:         g.Log,
		GitHubToken: os.Getenv("FORKLEDGER_GITHUB_TOKEN"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if strings.Contains(err.Error(), "not a git repository") {
			return ExitNotGit
		}
		return ExitAnalysis
	}
	policy.Check(cfg, rep)
	if err := emit(g, rep); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	return exitFromReport(rep)
}

func cmdUpstream(args []string) int {
	return withReport(args, func(rep *report.Report) int {
		if gJSON(args) {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(rep.Upstream)
			return ExitOK
		}
		if rep.Upstream.Selected != "" {
			fmt.Printf("Upstream candidate\n%s\n\nConfidence\n%s (%d)\n\nEvidence\n",
				rep.Upstream.Selected, rep.Upstream.Confidence.Level, rep.Upstream.Confidence.Score)
			for _, e := range rep.Upstream.Confidence.Evidence {
				fmt.Printf("- %s\n", e.Summary)
			}
		} else {
			fmt.Println(rep.Upstream.Explanation)
		}
		if len(rep.Upstream.Candidates) > 1 {
			fmt.Println("\nAlternative candidates")
			for i, c := range rep.Upstream.Candidates {
				if i == 0 && rep.Upstream.Selected == c.Identity {
					continue
				}
				fmt.Printf("- %s (%s, %d)\n", c.Identity, c.Confidence.Level, c.Confidence.Score)
			}
		}
		return ExitOK
	})
}

func cmdPatches(args []string) int {
	return withReport(args, func(rep *report.Report) int {
		if gJSON(args) {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(rep.Patches)
			return ExitOK
		}
		fmt.Printf("%d local patches\n\n", rep.Patches.Count)
		for _, p := range rep.Patches.Items {
			fmt.Printf("%s  %-24s  %-28s  %s\n", short(p.Commit), p.Category, p.Status, p.Subject)
			if p.Equivalence != nil {
				fmt.Printf("  equivalent %s (%s)\n", short(p.Equivalence.UpstreamCommit), p.Equivalence.Confidence.Level)
			}
			if p.ConflictLikely {
				fmt.Printf("  conflict: %s\n", p.ConflictReason)
			}
		}
		return ExitOK
	})
}

func cmdSecurity(args []string) int {
	return withReport(args, func(rep *report.Report) int {
		if gJSON(args) {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(rep.Security)
			return ExitOK
		}
		if len(rep.Security.Findings) == 0 {
			fmt.Println("No vulnerability findings were associated with this fork.")
			return ExitOK
		}
		for _, f := range rep.Security.Findings {
			fmt.Printf("%s\nSeverity\n%s\nApplicability\n%s\nRecommended action\n%s\n\n",
				f.ID, f.Severity, f.Applicability, f.RecommendedAction)
			fmt.Println("Evidence")
			for _, e := range f.Evidence {
				fmt.Printf("- %s\n", e.Summary)
			}
			fmt.Println()
		}
		return exitFromReport(rep)
	})
}

func cmdUpgrade(args []string) int {
	g, extra, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	if len(extra) == 0 {
		fmt.Fprintln(os.Stderr, "usage: forkledger upgrade <ref>")
		return ExitUsage
	}
	target := extra[0]
	cfg, code := loadConfig(g)
	if code != ExitOK {
		return code
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	res, err := upgrade.Simulate(ctx, g.Path, target, cfg, g.Log, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	if g.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return ExitOK
	}
	fmt.Println(res.Recommendation)
	return ExitOK
}

func cmdReport(args []string) int {
	g, extra, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	path := g.Output
	if len(extra) > 0 {
		path = extra[0]
	}
	if path == "" {
		path = defaultReportPath(g.Path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	var rep report.Report
	if err := json.Unmarshal(data, &rep); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	if g.JSON {
		return writeJSON("", &rep)
	}
	return writeTerm(&rep)
}

func cmdPolicy(args []string) int {
	if len(args) > 0 && args[0] == "check" {
		args = args[1:]
	}
	return withReport(args, func(rep *report.Report) int {
		if gJSON(args) {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(rep.Policy)
			return exitFromReport(rep)
		}
		if len(rep.Policy.Violations) == 0 {
			fmt.Println("No policy violations.")
			return ExitOK
		}
		for _, v := range rep.Policy.Violations {
			fmt.Printf("%s [%s]\n%s\n%s\n\n", v.RuleID, v.Severity, v.Description, v.Remediation)
		}
		return ExitPolicy
	})
}

func cmdExplain(args []string) int {
	return withReport(args, func(rep *report.Report) int {
		fmt.Println("What changed")
		fmt.Println(rep.Summary.Headline)
		fmt.Println()
		fmt.Println("Why it matters")
		for _, w := range rep.Lifecycle.Why {
			fmt.Println(w)
		}
		fmt.Println()
		fmt.Println("What you should do next")
		fmt.Println(rep.Lifecycle.RequiredWork)
		fmt.Println()
		fmt.Println("Evidence for the lifecycle recommendation")
		for _, e := range rep.Lifecycle.Confidence.Evidence {
			fmt.Printf("- %s (%s)\n", e.Summary, e.Kind)
		}
		return ExitOK
	})
}

func cmdDoctor(args []string) int {
	g, _, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	ok := true
	if _, err := git.LookPath(); err != nil {
		fmt.Println("git: not found on PATH")
		ok = false
	} else {
		fmt.Println("git: ok")
	}
	if _, err := git.Open(context.Background(), g.Path, git.NewExecutor(g.Log)); err != nil {
		fmt.Printf("repository: %v\n", err)
		ok = false
	} else {
		fmt.Println("repository: ok")
	}
	if path, err := config.Find(g.Path); err != nil {
		fmt.Printf("config: %v\n", err)
		ok = false
	} else if path == "" {
		fmt.Println("config: none found (optional)")
	} else if _, _, err := config.Load(path); err != nil {
		fmt.Printf("config: %v\n", err)
		ok = false
	} else {
		fmt.Println("config: ok " + path)
	}
	if os.Getenv("FORKLEDGER_GITHUB_TOKEN") == "" {
		fmt.Println("github token: not set (public metadata only)")
	} else {
		fmt.Println("github token: present")
	}
	if !ok {
		return ExitAnalysis
	}
	return ExitOK
}

func cmdVersion(args []string) int {
	g, _, _ := parseGlobals(args)
	if g.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]string{"version": version.Version, "commit": version.Commit})
		return ExitOK
	}
	fmt.Printf("forkledger %s (%s)\n", version.Version, version.Commit)
	return ExitOK
}

func withReport(args []string, fn func(*report.Report) int) int {
	g, extra, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	if len(extra) > 0 && g.Path == "." {
		g.Path = extra[0]
	}
	cfg, code := loadConfig(g)
	if code != ExitOK {
		return code
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	eng := scanner.New(g.Log)
	rep, err := eng.Analyze(ctx, scanner.Options{Path: g.Path, Config: cfg, Offline: g.Offline, Log: g.Log})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if strings.Contains(err.Error(), "not a git repository") {
			return ExitNotGit
		}
		return ExitAnalysis
	}
	policy.Check(cfg, rep)
	return fn(rep)
}

func emit(g Globals, rep *report.Report) error {
	out := g.Output
	if out == "" {
		out = defaultReportPath(g.Path)
		_ = os.MkdirAll(filepath.Dir(out), 0o755)
		if err := report.WriteJSON(out, rep); err != nil {
			return err
		}
		g.Log.Verbose("wrote %s", out)
	} else {
		if err := report.WriteJSON(out, rep); err != nil {
			return err
		}
	}
	if g.JSON {
		return report.WriteJSON("-", rep)
	}
	return report.WriteTerminal(os.Stdout, rep)
}

func writeJSON(path string, rep *report.Report) int {
	if err := report.WriteJSON(path, rep); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	return ExitOK
}

func writeTerm(rep *report.Report) int {
	if err := report.WriteTerminal(os.Stdout, rep); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	return ExitOK
}

func defaultReportPath(repo string) string {
	return filepath.Join(repo, ".forkledger", "analysis.json")
}

func exitFromReport(rep *report.Report) int {
	if len(rep.Policy.Violations) > 0 {
		for _, v := range rep.Policy.Violations {
			if strings.EqualFold(v.Severity, "critical") {
				return ExitPolicy
			}
		}
		return ExitPolicy
	}
	if rep.Security.MissingUpstreamFixes > 0 {
		return ExitSecurity
	}
	if rep.Summary.PotentialConflicts > 0 || rep.Summary.MissingSecurityFixes > 0 {
		return ExitFindings
	}
	if len(rep.Security.Findings) > 0 && rep.Security.NeedsInvestigation > 0 {
		return ExitFindings
	}
	return ExitOK
}

func gJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
