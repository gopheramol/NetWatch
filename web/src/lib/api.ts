import type {
  ConnectivityCheck,
  CurrentStatus,
  DailyStats,
  ListResponse,
  MonthlyStats,
  Outage,
  Settings,
  SpeedTestResult,
} from "./types";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, { cache: "no-store" });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new ApiError(res.status, body.error ?? res.statusText);
  }
  return res.json() as Promise<T>;
}

async function post<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({ error: res.statusText }));
    throw new ApiError(res.status, data.error ?? res.statusText);
  }
  return res.json() as Promise<T>;
}

export const api = {
  getStatus: () => request<CurrentStatus>("/api/status"),

  getConnectivity: (params?: { from?: string; to?: string; limit?: number }) =>
    request<ListResponse<ConnectivityCheck>>(`/api/connectivity${toQuery(params)}`),

  getSpeedLatest: async (): Promise<SpeedTestResult | null> => {
    try {
      return await request<SpeedTestResult>("/api/speed/latest");
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return null;
      throw err;
    }
  },

  getSpeedHistory: (params?: { from?: string; to?: string; limit?: number }) =>
    request<ListResponse<SpeedTestResult>>(`/api/speed/history${toQuery(params)}`),

  triggerSpeedTest: () => post<SpeedTestResult>("/api/speedtest"),

  getDowntime: (params?: { limit?: number }) =>
    request<ListResponse<Outage>>(`/api/downtime${toQuery(params)}`),

  getAnalyticsDailyRange: (params?: { from?: string; to?: string }) =>
    request<ListResponse<DailyStats>>(`/api/analytics/daily${toQuery(params)}`),

  getAnalyticsDailyByDate: (date: string) => request<DailyStats>(`/api/analytics/daily?date=${date}`),

  getAnalyticsMonthlyRange: (params?: { from?: string; to?: string }) =>
    request<ListResponse<MonthlyStats>>(`/api/analytics/monthly${toQuery(params)}`),

  getAnalyticsMonthlyByMonth: (month: string) =>
    request<MonthlyStats>(`/api/analytics/monthly?month=${month}`),

  getSettings: () => request<Settings>("/api/settings"),

  saveSettings: (settings: Settings) => post<Settings>("/api/settings", settings),

  testTelegram: () => post<{ sent: boolean }>("/api/telegram/test"),
};

function toQuery(params?: Record<string, string | number | undefined>): string {
  if (!params) return "";
  const entries = Object.entries(params).filter(([, v]) => v !== undefined);
  if (entries.length === 0) return "";
  const search = new URLSearchParams();
  for (const [k, v] of entries) search.set(k, String(v));
  return `?${search.toString()}`;
}

export { ApiError, API_BASE_URL };
