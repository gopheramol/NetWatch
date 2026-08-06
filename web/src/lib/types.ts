// Mirrors the JSON shapes produced by the Go backend's internal/models package.

export type ConnectivityStatus = "up" | "down" | "degraded";

export interface ConnectivityCheck {
  timestamp: string;
  status: ConnectivityStatus;
  latency_ms: number;
  dns_ok: boolean;
  http_ok: boolean;
  ping_ok: boolean;
  failure_reason?: string;
  packet_loss: number;
}

export interface Outage {
  id: string;
  start_time: string;
  end_time?: string;
  duration: number; // nanoseconds, as emitted by Go's time.Duration JSON encoding
  reason: string;
  resolved: boolean;
}

export interface SpeedTestResult {
  id: string;
  timestamp: string;
  download_mbps: number;
  upload_mbps: number;
  ping_ms: number;
  jitter_ms: number;
  isp: string;
  server: string;
  provider: string;
}

export interface DailyStats {
  date: string;
  avg_latency_ms: number;
  min_speed_mbps: number;
  max_speed_mbps: number;
  avg_speed_mbps: number;
  downtime_seconds: number;
  availability_pct: number;
  outage_count: number;
  check_count: number;
  speed_test_count: number;
}

export interface MonthlyStats {
  month: string;
  avg_latency_ms: number;
  avg_speed_mbps: number;
  downtime_seconds: number;
  availability_pct: number;
  outage_count: number;
  longest_outage_seconds: number;
  avg_outage_seconds: number;
  check_count: number;
  speed_test_count: number;
}

export interface Settings {
  telegram_enabled: boolean;
  telegram_bot_token?: string;
  telegram_chat_id?: string;
  monitor_interval_sec: number;
  speed_interval_minutes: number;
  speed_report_enabled: boolean;
  retention_days: number;
  battery_low_threshold_pct: number;
}

export interface CurrentStatus {
  status: ConnectivityStatus;
  last_check: string;
  latency_ms: number;
  isp: string;
  current_uptime_start?: string;
  current_streak_seconds: number;
  today_downtime_seconds: number;
  month_availability_pct: number;
  ongoing_outage?: Outage;
}

export interface ListResponse<T> {
  from?: string;
  to?: string;
  count: number;
  data: T[] | null;
}
