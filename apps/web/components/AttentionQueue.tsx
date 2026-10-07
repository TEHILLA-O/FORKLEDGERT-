"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { attentionRows } from "@/lib/example-report";

type Row = {
  repo: string;
  upstream: string;
  action: string;
  debt: string;
  missing: number;
  owner: string;
  reviewed: string;
  health: number;
};

function pill(value: string) {
  if (value === "REBASE" || value === "HIGH" || value === "SEVERE") return "pill warn";
  if (value === "RETIRE") return "pill danger";
  if (value === "RETAIN") return "pill";
  return "pill ok";
}

export function AttentionQueue() {
  const [rows, setRows] = useState<Row[]>(attentionRows);
  const [source, setSource] = useState("sample");

  useEffect(() => {
    const base = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    fetch(base + "/v1/attention")
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((data) => {
        const items = (data.items || []).map((it: Record<string, unknown>) => ({
          repo: String(it.repo || ""),
          upstream: String(it.upstream || ""),
          action: String(it.action || ""),
          debt: String(it.debt || ""),
          missing: Number(it.missing || 0),
          owner: String(it.owner || ""),
          reviewed: String(it.reviewed || ""),
          health: Number(it.health || 0),
        }));
        if (items.length > 0) {
          setRows(items);
          setSource("local api");
        }
      })
      .catch(() => {
        setSource("sample");
      });
  }, []);

  const missingFixes = rows.filter((r) => r.missing > 0);
  const noOwner = rows.filter((r) => !r.owner);
  const stale = rows.filter((r) => !r.reviewed);
  const retire = rows.filter((r) => r.action === "RETIRE");

  return (
    <>
      <p className="muted">Source: {source}</p>
      <div className="grid">
        <div className="metric">
          <span>Missing upstream fixes</span>
          <b>{missingFixes.length}</b>
        </div>
        <div className="metric">
          <span>No owner</span>
          <b>{noOwner.length}</b>
        </div>
        <div className="metric">
          <span>No recent review</span>
          <b>{stale.length}</b>
        </div>
        <div className="metric">
          <span>Retire candidates</span>
          <b>{retire.length}</b>
        </div>
      </div>

      <h2>Attention queue</h2>
      {rows.length === 0 ? (
        <div className="empty">No stored scans yet. Run forkledger analyze, then forkledger serve.</div>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Repository</th>
                <th>Upstream</th>
                <th>Action</th>
                <th>Debt</th>
                <th>Missing fixes</th>
                <th>Owner</th>
                <th>Last review</th>
                <th>Health</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.repo} tabIndex={0}>
                  <td>
                    <Link href="/repository">{row.repo}</Link>
                  </td>
                  <td className="mono">{row.upstream}</td>
                  <td>
                    <span className={pill(row.action)}>{row.action}</span>
                  </td>
                  <td>
                    <span className={pill(row.debt)}>{row.debt}</span>
                  </td>
                  <td className={row.missing ? "neg" : ""}>{row.missing}</td>
                  <td>{row.owner || <span className="muted">unassigned</span>}</td>
                  <td>{row.reviewed || <span className="muted">never</span>}</td>
                  <td className="mono">{row.health}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
