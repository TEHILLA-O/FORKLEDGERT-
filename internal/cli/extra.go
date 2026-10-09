package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/TEHILLA-O/forkledger/internal/api"
	"github.com/TEHILLA-O/forkledger/internal/discover"
	"github.com/TEHILLA-O/forkledger/internal/report"
	"github.com/TEHILLA-O/forkledger/internal/security"
	"github.com/TEHILLA-O/forkledger/internal/store"
)

func cmdServe(args []string) int {
	g, extra, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	addr := ":8080"
	if len(extra) > 0 {
		addr = extra[0]
	}
	st, err := store.Open(store.DefaultDir(g.Path))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	srv := &api.Server{
		Store:         st,
		WebhookSecret: os.Getenv("FORKLEDGER_WEBHOOK_SECRET"),
	}
	fmt.Fprintf(os.Stderr, "forkledger api listening on %s\n", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	return ExitOK
}

func cmdIngest(args []string) int {
	g, extra, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	path := ""
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
	st, err := store.Open(store.DefaultDir(g.Path))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	id, err := st.Put(&rep)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	fmt.Println(id)
	return ExitOK
}

func cmdDiscover(args []string) int {
	g, extra, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitUsage
	}
	if len(extra) == 0 {
		fmt.Fprintln(os.Stderr, "usage: forkledger discover <org>")
		return ExitUsage
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cands, err := discover.New(os.Getenv("FORKLEDGER_GITHUB_TOKEN")).OrgRepos(ctx, extra[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitAnalysis
	}
	if g.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(cands)
		return ExitOK
	}
	fmt.Printf("%d repositories in %s\n\n", len(cands), extra[0])
	for _, c := range cands {
		line := c.FullName
		if c.Fork {
			line += "  fork of " + c.Parent
		}
		fmt.Println(line)
	}
	return ExitOK
}

func cmdSBOM(args []string) int {
	return withReport(args, func(rep *report.Report) int {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"cyclonedx": rep.Security.CycloneDX,
			"spdx":      rep.Security.SPDX,
		})
		return ExitOK
	})
}

func cmdVEX(args []string) int {
	return withReport(args, func(rep *report.Report) int {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if rep.Security.OpenVEX.Author == "" {
			rep.Security.OpenVEX = security.OpenVEX(rep.Repository.Path, rep.Security.Findings)
		}
		_ = enc.Encode(rep.Security.OpenVEX)
		return ExitOK
	})
}

func persist(g Globals, rep *report.Report) {
	dir := store.DefaultDir(g.Path)
	st, err := store.Open(dir)
	if err != nil {
		g.Log.Verbose("store skipped: %v", err)
		return
	}
	if _, err := st.Put(rep); err != nil {
		g.Log.Verbose("store put: %v", err)
		return
	}
	g.Log.Verbose("stored scan under %s", dir)
}

