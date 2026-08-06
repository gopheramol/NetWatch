"use client";

import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { ConnectivityCheck } from "@/lib/types";
import { formatTime } from "@/lib/format";
import { ChartTooltip } from "./ChartTooltip";

export function LatencyChart({ checks }: { checks: ConnectivityCheck[] }) {
  const data = checks.map((c) => ({
    time: c.timestamp,
    latency: c.status === "down" ? null : Math.round(c.latency_ms),
  }));

  if (data.length === 0) {
    return <EmptyState />;
  }

  return (
    <ResponsiveContainer width="100%" height={240}>
      <LineChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
        <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
        <XAxis
          dataKey="time"
          tickFormatter={formatTime}
          stroke="var(--chart-axis)"
          fontSize={11}
          tickLine={false}
          axisLine={false}
          minTickGap={40}
        />
        <YAxis
          stroke="var(--chart-axis)"
          fontSize={11}
          tickLine={false}
          axisLine={false}
          width={40}
          unit="ms"
        />
        <Tooltip
          content={
            <ChartTooltip
              labelFormatter={(v) => formatTime(String(v))}
              valueFormatter={(v) => `${v} ms`}
            />
          }
        />
        <Line
          type="monotone"
          dataKey="latency"
          stroke="var(--chart-series-1)"
          strokeWidth={2}
          dot={false}
          activeDot={{ r: 4 }}
          isAnimationActive={false}
          connectNulls={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

function EmptyState() {
  return (
    <div className="flex h-[240px] items-center justify-center text-sm text-foreground/40">
      No connectivity data yet
    </div>
  );
}
