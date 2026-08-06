import type { ConnectivityStatus } from "./types";

export const STATUS_COLOR: Record<ConnectivityStatus, string> = {
  up: "var(--status-good)",
  degraded: "var(--status-warning)",
  down: "var(--status-critical)",
};

export const STATUS_LABEL: Record<ConnectivityStatus, string> = {
  up: "Online",
  degraded: "Degraded",
  down: "Offline",
};
