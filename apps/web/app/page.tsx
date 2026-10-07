import { AttentionQueue } from "@/components/AttentionQueue";

export default function HomePage() {
  return (
    <main>
      <h1>Which forks need attention?</h1>
      <p className="lede">
        This is not a dashboard of vanity charts. Each row is a fork that still
        requires a human decision. Open a repository to see the evidence. If a
        local ForkLedger API is running, this page uses stored scans. Otherwise
        it shows the sample queue.
      </p>
      <AttentionQueue />
    </main>
  );
}
