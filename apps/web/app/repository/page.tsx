"use client";

import { useMemo, useState } from "react";
import { exampleReport } from "@/lib/example-report";
import type { Report } from "@/lib/types";

const emptyHelp =
  "Load a ForkLedger JSON report to replace the sample. The sample exists so the page has evidence to show before an API exists.";

export default function RepositoryPage() {
  const [report, setReport] = useState<Report>(exampleReport);
  const [raw, setRaw] = useState("");
  const [error, setError] = useState("");
  const [filter, setFilter] = useState("");

  const patches = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return report.patches.items;
    return report.patches.items.filter((p) =>
      [p.subject, p.category, p.status, p.commit].join(" ").toLowerCase().includes(q),
    );
  }, [filter, report.patches.items]);

  function load() {
    try {
      setReport(JSON.parse(raw) as Report);
      setError("");
    } catch {
      setError("That is not valid JSON.");
    }
  }

  const s = report.summary;
  return (
    <main>
      <h1>{report.repository.path}</h1>
      <p className="lede">{s.headline}</p>

      <div className="grid">
        <div className="metric">
          <span>Baseline</span>
          <b>{s.baseline || "n/a"}</b>
        </div>
        <div className="metric">
          <span>Current upstream</span>
          <b>{s.current_upstream || "n/a"}</b>
        </div>
        <div className="metric">
          <span>Local patches</span>
          <b>{s.local_modifications}</b>
        </div>
        <div className="metric">
          <span>Health</span>
          <b>{report.health.score}</b>
        </div>
      </div>

      <div className="panel">
        <h2 style={{ marginTop: 0 }}>Lifecycle recommendation</h2>
        <dl className="kv">
          <dt>Recommended</dt>
          <dd>
            <span className="pill warn">{report.lifecycle.recommended}</span>{" "}
            <span className="muted">
              {report.lifecycle.confidence.level} ({report.lifecycle.confidence.score})
            </span>
          </dd>
          <dt>Why</dt>
          <dd>{report.lifecycle.why.join(" ")}</dd>
          <dt>Risks</dt>
          <dd>{report.lifecycle.risks.join(" ")}</dd>
          <dt>Required work</dt>
          <dd>{report.lifecycle.required_work}</dd>
          <dt>Benefit</dt>
          <dd>{report.lifecycle.benefit}</dd>
        </dl>
      </div>

      <h2>Upstream lineage</h2>
      <div className="panel">
        <dl className="kv">
          <dt>Selected upstream</dt>
          <dd className="mono">{report.upstream.selected || "not settled"}</dd>
          <dt>Fork point</dt>
          <dd className="mono">{report.lineage.fork_point || "unknown"}</dd>
          <dt>Ahead / behind</dt>
          <dd>
            {report.lineage.ahead} / {report.lineage.behind}
          </dd>
          <dt>Evidence</dt>
          <dd>
            {(report.upstream.confidence.evidence || []).map((e) => (
              <div key={e.summary}>{e.summary}</div>
            ))}
          </dd>
        </dl>
      </div>

      <h2>Fork health</h2>
      <p className="muted">Health is never shown as a bare number.</p>
      <ul className="factors">
        {report.health.factors.map((f) => (
          <li key={f.name}>
            <span className={f.delta < 0 ? "neg" : "pos"}>
              {f.delta > 0 ? "+" : ""}
              {f.delta}
            </span>
            <span>
              {f.name}. {f.reason}
            </span>
          </li>
        ))}
      </ul>
      <p>{report.debt.summary}</p>

      <h2>Security state</h2>
      {report.security.findings.length === 0 ? (
        <div className="empty">No vulnerability findings were associated with this fork.</div>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Severity</th>
                <th>Applicability</th>
                <th>Next step</th>
              </tr>
            </thead>
            <tbody>
              {report.security.findings.map((f) => (
                <tr key={f.id} tabIndex={0}>
                  <td className="mono">{f.id}</td>
                  <td>{f.severity}</td>
                  <td>
                    <span className={f.applicability === "AFFECTED" ? "pill danger" : "pill warn"}>
                      {f.applicability}
                    </span>
                  </td>
                  <td>{f.recommended_action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <h2>Patch ledger</h2>
      <p className="muted">
        Filter by commit, category, or subject. Equivalents are candidates until someone confirms them.
      </p>
      <p>
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="filter patches"
          aria-label="filter patches"
          style={{
            background: "var(--fl-bg-sunken)",
            color: "var(--fl-text)",
            border: "1px solid var(--fl-line)",
            padding: "6px 8px",
            fontFamily: "var(--fl-mono)",
            width: "280px",
          }}
        />
      </p>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Commit</th>
              <th>Category</th>
              <th>Status</th>
              <th>Subject</th>
              <th>Lines</th>
            </tr>
          </thead>
          <tbody>
            {patches.map((p) => (
              <tr key={p.commit} tabIndex={0}>
                <td className="mono">{p.commit.slice(0, 7)}</td>
                <td>{p.category}</td>
                <td>{p.status}</td>
                <td>
                  {p.subject}
                  {p.equivalence ? (
                    <div className="muted">
                      possible upstream equivalent {p.equivalence.upstream_commit} (
                      {p.equivalence.confidence.level})
                    </div>
                  ) : null}
                  {p.conflict_likely ? <div className="neg">{p.conflict_reason}</div> : null}
                </td>
                <td className="mono">
                  +{p.added} / -{p.deleted}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h2>Load a local report</h2>
      <p className="muted">{emptyHelp}</p>
      <textarea
        value={raw}
        onChange={(e) => setRaw(e.target.value)}
        placeholder='paste the JSON from forkledger analyze --json'
        spellCheck={false}
      />
      <p>
        <button type="button" onClick={load}>
          Replace sample
        </button>
        {error ? <span className="err"> {error}</span> : null}
      </p>
    </main>
  );
}
