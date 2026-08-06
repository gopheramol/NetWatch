"use client";

import { Activity, Settings } from "lucide-react";
import { ThemeToggle } from "./ThemeToggle";

export function Header({ onOpenSettings }: { onOpenSettings: () => void }) {
  return (
    <header className="sticky top-0 z-10 border-b border-border bg-background/80 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4 sm:px-6">
        <div className="flex items-center gap-2">
          <span
            className="flex h-9 w-9 items-center justify-center rounded-lg"
            style={{
              backgroundColor: "color-mix(in srgb, var(--chart-series-1) 15%, transparent)",
              color: "var(--chart-series-1)",
            }}
          >
            <Activity size={18} />
          </span>
          <div>
            <h1 className="text-base font-semibold leading-none">NetWatch</h1>
            <p className="text-xs text-foreground/50">Internet Monitor</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={onOpenSettings}
            aria-label="Settings"
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-surface text-foreground/70 transition hover:text-foreground"
          >
            <Settings size={16} />
          </button>
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
