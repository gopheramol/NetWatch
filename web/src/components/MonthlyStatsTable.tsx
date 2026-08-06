import type { MonthlyStats } from "@/lib/types";
import { formatDurationSeconds, formatLatency, formatPct } from "@/lib/format";

export function MonthlyStatsTable({ months }: { months: MonthlyStats[] }) {
  if (months.length === 0) {
    return <p className="py-6 text-center text-sm text-foreground/40">No monthly data yet</p>;
  }

  const sorted = [...months].sort((a, b) => b.month.localeCompare(a.month));

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-sm">
        <thead className="text-xs uppercase tracking-wide text-foreground/50">
          <tr>
            <th className="pb-2 pr-4 font-medium">Month</th>
            <th className="pb-2 pr-4 font-medium">Availability</th>
            <th className="pb-2 pr-4 font-medium">Avg latency</th>
            <th className="pb-2 pr-4 font-medium">Outages</th>
            <th className="pb-2 pr-4 font-medium">Longest outage</th>
            <th className="pb-2 font-medium">Total downtime</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border tabular-nums">
          {sorted.map((m) => (
            <tr key={m.month}>
              <td className="py-2 pr-4 font-medium">{m.month}</td>
              <td className="py-2 pr-4">{formatPct(m.availability_pct)}</td>
              <td className="py-2 pr-4">{formatLatency(m.avg_latency_ms)}</td>
              <td className="py-2 pr-4">{m.outage_count}</td>
              <td className="py-2 pr-4">{formatDurationSeconds(m.longest_outage_seconds)}</td>
              <td className="py-2">{formatDurationSeconds(m.downtime_seconds)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
