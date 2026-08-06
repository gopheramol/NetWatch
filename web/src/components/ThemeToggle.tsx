"use client";

import { Moon, Sun } from "lucide-react";

// Both icons render unconditionally and CSS (the `dark` class already set
// pre-paint by the inline script in layout.tsx) decides which is visible.
// This keeps server and client markup identical, so there is nothing to
// hydrate-mismatch on.
export function ThemeToggle() {
  function toggle() {
    const next = !document.documentElement.classList.contains("dark");
    document.documentElement.classList.toggle("dark", next);
    localStorage.setItem("netwatch-theme", next ? "dark" : "light");
  }

  return (
    <button
      onClick={toggle}
      aria-label="Toggle dark mode"
      className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-surface text-foreground/70 transition hover:text-foreground"
    >
      <Moon size={16} className="dark:hidden" />
      <Sun size={16} className="hidden dark:inline" />
    </button>
  );
}
