"use client";

import { AlertTriangle, CheckCircle2, XCircle } from "lucide-react";
import type { CurrentStatus } from "@/lib/types";
import { STATUS_COLOR, STATUS_LABEL } from "@/lib/status";
import { formatDurationSeconds, formatLatency, formatRelative } from "@/lib/format";

const ICON = { up: CheckCircle2, degraded: AlertTriangle, down: XCircle };

export function StatusBanner({ status }: { status: CurrentStatus }) {
  const Icon = ICON[status.status];
  const color = STATUS_COLOR[status.status];

  return (
    <div
      className="flex flex-col gap-4 rounded-2xl border border-border bg-surface p-6 shadow-sm sm:flex-row sm:items-center sm:justify-between"
      style={{ borderLeft: `4px solid ${color}` }}
    >
      <div className="flex items-center gap-4">
        <span
          className="relative flex h-12 w-12 items-center justify-center rounded-full"
          style={{ backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)`, color }}
        >
          <Icon size={26} />
          {status.status === "up" && (
            <span
              className="absolute inline-flex h-full w-full animate-ping rounded-full opacity-20"
              style={{ backgroundColor: color }}
            />
          )}
        </span>
        <div>
          <p className="text-lg font-semibold" style={{ color }}>
            {STATUS_LABEL[status.status]}
          </p>
          <p className="text-sm text-foreground/60">
            Last check {formatRelative(status.last_check)} · latency {formatLatency(status.latency_ms)}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-6 sm:flex sm:gap-8">
        <Metric label="Current streak" value={formatDurationSeconds(status.current_streak_seconds)} />
        <Metric label="Today's downtime" value={formatDurationSeconds(status.today_downtime_seconds)} />
        <Metric label="ISP" value={status.isp || "—"} />
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs uppercase tracking-wide text-foreground/50">{label}</p>
      <p className="text-base font-medium">{value}</p>
    </div>
  );
}
