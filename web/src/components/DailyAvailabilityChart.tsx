"use client";

import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { DailyStats } from "@/lib/types";
import { ChartTooltip } from "./ChartTooltip";

export function DailyAvailabilityChart({ days }: { days: DailyStats[] }) {
  const data = days.map((d) => ({
    date: d.date,
    availability: Math.round(d.availability_pct * 100) / 100,
  }));

  if (data.length === 0) {
    return (
      <div className="flex h-[220px] items-center justify-center text-sm text-foreground/40">
        No daily analytics yet
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={220}>
      <BarChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
        <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
        <XAxis
          dataKey="date"
          tickFormatter={(d: string) => d.slice(5)}
          stroke="var(--chart-axis)"
          fontSize={11}
          tickLine={false}
          axisLine={false}
          minTickGap={20}
        />
        <YAxis
          domain={[0, 100]}
          stroke="var(--chart-axis)"
          fontSize={11}
          tickLine={false}
          axisLine={false}
          width={36}
          unit="%"
        />
        <Tooltip content={<ChartTooltip valueFormatter={(v) => `${v}%`} />} />
        <Bar dataKey="availability" fill="var(--chart-series-1)" radius={[4, 4, 0, 0]} maxBarSize={18} />
      </BarChart>
    </ResponsiveContainer>
  );
}
