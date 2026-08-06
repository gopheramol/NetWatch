"use client";

import { useEffect, useState } from "react";
import { X } from "lucide-react";
import { api } from "@/lib/api";
import type { Settings } from "@/lib/types";

export function SettingsPanel({ onClose }: { onClose: () => void }) {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [message, setMessage] = useState<{ kind: "success" | "error"; text: string } | null>(null);

  useEffect(() => {
    api.getSettings().then(setSettings).catch((err: Error) => setMessage({ kind: "error", text: err.message }));
  }, []);

  async function save() {
    if (!settings) return;
    setSaving(true);
    setMessage(null);
    try {
      const saved = await api.saveSettings(settings);
      setSettings(saved);
      setMessage({ kind: "success", text: "Settings saved." });
    } catch (err) {
      setMessage({ kind: "error", text: (err as Error).message });
    } finally {
      setSaving(false);
    }
  }

  async function sendTest() {
    setTesting(true);
    setMessage(null);
    try {
      await api.testTelegram();
      setMessage({ kind: "success", text: "Test notification sent (check Telegram)." });
    } catch (err) {
      setMessage({ kind: "error", text: (err as Error).message });
    } finally {
      setTesting(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md rounded-2xl border border-border bg-surface p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-base font-semibold">Settings</h2>
          <button onClick={onClose} className="text-foreground/50 hover:text-foreground">
            <X size={18} />
          </button>
        </div>

        {!settings ? (
          <p className="text-sm text-foreground/50">Loading…</p>
        ) : (
          <div className="space-y-4">
            <label className="flex items-center justify-between text-sm">
              <span>Telegram alerts enabled</span>
              <input
                type="checkbox"
                checked={settings.telegram_enabled}
                onChange={(e) => setSettings({ ...settings, telegram_enabled: e.target.checked })}
                className="h-4 w-4"
              />
            </label>

            <Field
              label="Bot token"
              value={settings.telegram_bot_token ?? ""}
              onChange={(v) => setSettings({ ...settings, telegram_bot_token: v })}
              type="password"
            />
            <Field
              label="Chat ID"
              value={settings.telegram_chat_id ?? ""}
              onChange={(v) => setSettings({ ...settings, telegram_chat_id: v })}
            />

            <div className="grid grid-cols-2 gap-3">
              <NumberField
                label="Monitor (sec)"
                value={settings.monitor_interval_sec}
                onChange={(v) => setSettings({ ...settings, monitor_interval_sec: v })}
              />
              <NumberField
                label="Speed test (hr)"
                value={settings.speed_interval_hours}
                onChange={(v) => setSettings({ ...settings, speed_interval_hours: v })}
              />
              <NumberField
                label="Retention (days)"
                value={settings.retention_days}
                onChange={(v) => setSettings({ ...settings, retention_days: v })}
              />
              <NumberField
                label="Battery alert (%)"
                value={settings.battery_low_threshold_pct}
                onChange={(v) => setSettings({ ...settings, battery_low_threshold_pct: v })}
              />
            </div>

            {message && (
              <p
                className="text-sm"
                style={{ color: message.kind === "success" ? "var(--status-good)" : "var(--status-critical)" }}
              >
                {message.text}
              </p>
            )}

            <div className="flex items-center justify-between gap-2 pt-2">
              <button
                onClick={sendTest}
                disabled={testing}
                className="rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground/70 transition hover:text-foreground disabled:opacity-50"
              >
                {testing ? "Sending…" : "Send test message"}
              </button>
              <button
                onClick={save}
                disabled={saving}
                className="rounded-lg bg-[var(--chart-series-1)] px-4 py-2 text-sm font-medium text-white transition disabled:opacity-50"
              >
                {saving ? "Saving…" : "Save"}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  type = "text",
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block text-foreground/60">{label}</span>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none focus:border-[var(--chart-series-1)]"
      />
    </label>
  );
}

function NumberField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block text-foreground/60">{label}</span>
      <input
        type="number"
        min={1}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none focus:border-[var(--chart-series-1)]"
      />
    </label>
  );
}
