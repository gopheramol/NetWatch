"use client";

import { useState } from "react";
import { ArrowDownToLine, ArrowUpFromLine, Gauge, ShieldCheck, Zap } from "lucide-react";
import { api } from "@/lib/api";
import { usePolling } from "@/hooks/usePolling";
import { Card } from "@/components/Card";
import { Header } from "@/components/Header";
import { StatCard } from "@/components/StatCard";
import { StatusBanner } from "@/components/StatusBanner";
import { LatencyChart } from "@/components/LatencyChart";
import { SpeedChart } from "@/components/SpeedChart";
import { DailyAvailabilityChart } from "@/components/DailyAvailabilityChart";
import { DowntimeHistory } from "@/components/DowntimeHistory";
import { MonthlyStatsTable } from "@/components/MonthlyStatsTable";
import { SettingsPanel } from "@/components/SettingsPanel";
import { SystemMetricsCard } from "@/components/SystemMetricsCard";
import { formatSpeed, formatDateTime } from "@/lib/format";

export default function Dashboard() {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [runningSpeedTest, setRunningSpeedTest] = useState(false);

  const status = usePolling(api.getStatus, 5000);
  const connectivity = usePolling(() => api.getConnectivity({ limit: 200 }), 15000);
  const speedLatest = usePolling(api.getSpeedLatest, 30000);
  const speedHistory = usePolling(() => api.getSpeedHistory({ limit: 80 }), 30000);
  const downtime = usePolling(() => api.getDowntime({ limit: 25 }), 15000);
  const daily = usePolling(() => api.getAnalyticsDailyRange(), 60000);
  const monthly = usePolling(() => api.getAnalyticsMonthlyRange(), 60000);

  async function runSpeedTestNow() {
    setRunningSpeedTest(true);
    try {
      await api.triggerSpeedTest();
      speedLatest.refresh();
      speedHistory.refresh();
    } catch {
      // surfaced implicitly via next poll's error state
    } finally {
      setRunningSpeedTest(false);
    }
  }

  return (
    <>
      <Header onOpenSettings={() => setSettingsOpen(true)} />

      <main className="mx-auto flex max-w-6xl flex-1 flex-col gap-6 px-4 py-6 sm:px-6">
        {status.data && <StatusBanner status={status.data} />}

        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <StatCard
            icon={Zap}
            label="Latency"
            value={status.data ? `${Math.round(status.data.latency_ms)} ms` : "—"}
          />
          <StatCard
            icon={ArrowDownToLine}
            label="Download"
            value={speedLatest.data ? formatSpeed(speedLatest.data.download_mbps) : "—"}
            sublabel={speedLatest.data ? formatDateTime(speedLatest.data.timestamp) : undefined}
          />
          <StatCard
            icon={ArrowUpFromLine}
            label="Upload"
            value={speedLatest.data ? formatSpeed(speedLatest.data.upload_mbps) : "—"}
          />
          <StatCard
            icon={ShieldCheck}
            label="Monthly availability"
            value={status.data ? `${status.data.month_availability_pct.toFixed(2)}%` : "—"}
          />
        </div>

        <SystemMetricsCard />

        <Card title="Latency (last 24h)">
          <LatencyChart checks={connectivity.data?.data ?? []} />
        </Card>

        <Card
          title="Speed history"
          action={
            <button
              onClick={runSpeedTestNow}
              disabled={runningSpeedTest}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground/70 transition hover:text-foreground disabled:opacity-50"
            >
              <Gauge size={14} />
              {runningSpeedTest ? "Testing…" : "Run test now"}
            </button>
          }
        >
          <SpeedChart results={speedHistory.data?.data ?? []} />
        </Card>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <Card title="Daily availability (30d)">
            <DailyAvailabilityChart days={daily.data?.data ?? []} />
          </Card>
          <Card title="Recent outages">
            <DowntimeHistory outages={downtime.data?.data ?? []} />
          </Card>
        </div>

        <Card title="Monthly statistics">
          <MonthlyStatsTable months={monthly.data?.data ?? []} />
        </Card>

        <footer className="pb-6 text-center text-xs text-foreground/40">
          NetWatch · self-hosted Internet monitor
        </footer>
      </main>

      {settingsOpen && <SettingsPanel onClose={() => setSettingsOpen(false)} />}
    </>
  );
}
