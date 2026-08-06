import type { Outage } from "@/lib/types";
import { formatDateTime, formatDurationNanos } from "@/lib/format";

export function DowntimeHistory({ outages }: { outages: Outage[] }) {
  if (outages.length === 0) {
    return (
      <p className="py-6 text-center text-sm text-foreground/40">
        No outages recorded — great sign!
      </p>
    );
  }

  return (
    <div className="max-h-[320px] overflow-y-auto">
      <table className="w-full text-left text-sm">
        <thead className="sticky top-0 bg-surface text-xs uppercase tracking-wide text-foreground/50">
          <tr>
            <th className="pb-2 pr-2 font-medium">Started</th>
            <th className="pb-2 pr-2 font-medium">Duration</th>
            <th className="pb-2 font-medium">Reason</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {outages.map((o) => (
            <tr key={o.id}>
              <td className="py-2 pr-2 whitespace-nowrap tabular-nums">
                {formatDateTime(o.start_time)}
              </td>
              <td className="py-2 pr-2 whitespace-nowrap tabular-nums">
                {o.resolved ? (
                  formatDurationNanos(o.duration)
                ) : (
                  <span className="font-medium" style={{ color: "var(--status-critical)" }}>
                    ongoing
                  </span>
                )}
              </td>
              <td className="py-2 max-w-[240px] truncate text-foreground/60" title={o.reason}>
                {o.reason || "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
