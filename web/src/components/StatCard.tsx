import type { LucideIcon } from "lucide-react";

export function StatCard({
  icon: Icon,
  label,
  value,
  sublabel,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  sublabel?: string;
}) {
  return (
    <div className="rounded-2xl border border-border bg-surface p-5 shadow-sm">
      <div className="flex items-center gap-2 text-foreground/50">
        <Icon size={16} />
        <p className="text-xs uppercase tracking-wide">{label}</p>
      </div>
      <p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p>
      {sublabel && <p className="mt-1 text-xs text-foreground/50">{sublabel}</p>}
    </div>
  );
}
