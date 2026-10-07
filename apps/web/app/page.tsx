import Link from "next/link";
import { attentionRows } from "@/lib/example-report";

function pill(value: string) {
  if (value === "REBASE" || value === "HIGH" || value === "SEVERE") return "pill warn";
  if (value === "RETIRE" || Number(value) > 0) return "pill danger";
  if (value === "RETAIN") return "pill";
  return "pill ok";
}

export default function HomePage() {
  const missingFixes = attentionRows.filter((r) => r.missing > 0);
  const noOwner = attentionRows.filter((r) => !r.owner);
  const stale = attentionRows.filter((r) => !r.reviewed);
  const retire = attentionRows.filter((r) => r.action === "RETIRE");

  return (
    <main>
      <h1>Which forks need attention?</h1>
      <p className="lede">
        This is not a dashboard of vanity charts. Each row is a fork that still
        requires a human decision. Open a repository to see the evidence.
      </p>

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
            {attentionRows.map((row) => (
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
    </main>
  );
}
