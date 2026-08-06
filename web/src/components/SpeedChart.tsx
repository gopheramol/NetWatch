"use client";

import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { SpeedTestResult } from "@/lib/types";
import { formatDateTime } from "@/lib/format";
import { ChartTooltip } from "./ChartTooltip";

export function SpeedChart({ results }: { results: SpeedTestResult[] }) {
  const data = results.map((r) => ({
    time: r.timestamp,
    download: Math.round(r.download_mbps * 10) / 10,
    upload: Math.round(r.upload_mbps * 10) / 10,
  }));

  if (data.length === 0) {
    return (
      <div className="flex h-[240px] items-center justify-center text-sm text-foreground/40">
        No speed test results yet
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={240}>
      <LineChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
        <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
        <XAxis
          dataKey="time"
          tickFormatter={formatDateTime}
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
          width={44}
          unit=" Mb"
        />
        <Tooltip
          content={
            <ChartTooltip
              labelFormatter={(v) => formatDateTime(String(v))}
              valueFormatter={(v) => `${v} Mbps`}
            />
          }
        />
        <Legend
          iconType="circle"
          wrapperStyle={{ fontSize: 12, color: "var(--chart-axis)" }}
        />
        <Line
          type="monotone"
          dataKey="download"
          name="Download"
          stroke="var(--chart-series-1)"
          strokeWidth={2}
          dot={false}
          activeDot={{ r: 4 }}
          isAnimationActive={false}
        />
        <Line
          type="monotone"
          dataKey="upload"
          name="Upload"
          stroke="var(--chart-series-2)"
          strokeWidth={2}
          dot={false}
          activeDot={{ r: 4 }}
          isAnimationActive={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}
