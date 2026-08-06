"use client";

import { useEffect, useState } from "react";
import { Card } from "./Card";
import { api } from "../lib/api";
import type { SystemMetrics } from "../lib/types";

export function SystemMetricsCard() {
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;

    async function load() {
      try {
        const data = await api.getMetricsLatest();
        if (mounted) {
          setMetrics(data);
          setLoading(false);
        }
      } catch {
        if (mounted) setLoading(false);
      }
    }

    load();
    const interval = setInterval(load, 15000);
    return () => {
      mounted = false;
      clearInterval(interval);
    };
  }, []);

  if (loading) {
    return (
      <Card title="System Resources">
        <div className="h-28 animate-pulse rounded-xl bg-border/40" />
      </Card>
    );
  }

  if (!metrics) {
    return (
      <Card title="System Resources">
        <p className="text-sm text-foreground/60">No system metrics available.</p>
      </Card>
    );
  }

  return (
    <Card title="System Resources">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {/* CPU Usage */}
        <div className="rounded-xl border border-border/50 bg-background/50 p-4">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-foreground/60">CPU Usage</span>
            <span className="text-sm font-bold text-emerald-500">{metrics.cpu_percent.toFixed(1)}%</span>
          </div>
          <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-border">
            <div
              className="h-full bg-emerald-500 transition-all duration-500"
              style={{ width: `${Math.min(100, Math.max(0, metrics.cpu_percent))}%` }}
            />
          </div>
        </div>

        {/* RAM Usage */}
        <div className="rounded-xl border border-border/50 bg-background/50 p-4">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-foreground/60">Memory (RAM)</span>
            <span className="text-sm font-bold text-sky-500">{metrics.ram_percent.toFixed(1)}%</span>
          </div>
          <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-border">
            <div
              className="h-full bg-sky-500 transition-all duration-500"
              style={{ width: `${Math.min(100, Math.max(0, metrics.ram_percent))}%` }}
            />
          </div>
          <p className="mt-1 text-[11px] text-foreground/50">
            {metrics.ram_used_mb.toFixed(0)} MB / {metrics.ram_total_mb.toFixed(0)} MB
          </p>
        </div>

        {/* Disk Usage */}
        <div className="rounded-xl border border-border/50 bg-background/50 p-4">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-foreground/60">Disk Usage (/)</span>
            <span className="text-sm font-bold text-amber-500">{metrics.disk_percent.toFixed(1)}%</span>
          </div>
          <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-border">
            <div
              className="h-full bg-amber-500 transition-all duration-500"
              style={{ width: `${Math.min(100, Math.max(0, metrics.disk_percent))}%` }}
            />
          </div>
          <p className="mt-1 text-[11px] text-foreground/50">
            {metrics.disk_used_gb.toFixed(1)} GB / {metrics.disk_total_gb.toFixed(1)} GB
          </p>
        </div>

        {/* CPU Temp */}
        <div className="rounded-xl border border-border/50 bg-background/50 p-4">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-foreground/60">CPU Temperature</span>
            <span className="text-sm font-bold text-indigo-500">
              {metrics.cpu_temp_c && metrics.cpu_temp_c > 0
                ? `${metrics.cpu_temp_c.toFixed(1)} °C`
                : "N/A"}
            </span>
          </div>
          <div className="mt-2 flex items-center gap-2">
            <span className="text-xs text-foreground/50">Sensor:</span>
            <span className="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-500">
              {metrics.cpu_temp_c && metrics.cpu_temp_c > 0 ? "Active" : "Normal"}
            </span>
          </div>
        </div>

        {/* Battery / UPS */}
        <div className="rounded-xl border border-border/50 bg-background/50 p-4">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-foreground/60">Battery / UPS</span>
            <span className="text-sm font-bold text-purple-500">
              {metrics.battery_present ? `${metrics.battery_percent?.toFixed(0)}%` : "AC Power"}
            </span>
          </div>
          {metrics.battery_present ? (
            <>
              <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-border">
                <div
                  className={`h-full transition-all duration-500 ${
                    (metrics.battery_percent ?? 100) <= 20
                      ? "bg-rose-500"
                      : (metrics.battery_percent ?? 100) <= 50
                      ? "bg-amber-500"
                      : "bg-purple-500"
                  }`}
                  style={{ width: `${Math.min(100, Math.max(0, metrics.battery_percent ?? 0))}%` }}
                />
              </div>
              <div className="mt-1 flex items-center gap-1.5 text-[11px] text-foreground/50">
                <span>{metrics.battery_charging ? "⚡️ Charging" : "🔋 Discharging"}</span>
              </div>
            </>
          ) : (
            <div className="mt-2 flex items-center gap-2">
              <span className="inline-flex items-center gap-1 rounded-full bg-purple-500/10 px-2 py-0.5 text-[11px] font-medium text-purple-400">
                🔌 Always on AC
              </span>
            </div>
          )}
        </div>
      </div>
    </Card>
  );
}
