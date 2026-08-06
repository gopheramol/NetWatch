"use client";

interface Payload {
  name?: string;
  value?: number | string;
  color?: string;
}

export function ChartTooltip({
  active,
  payload,
  label,
  labelFormatter,
  valueFormatter,
}: {
  active?: boolean;
  payload?: Payload[];
  label?: string | number;
  labelFormatter?: (label: string | number) => string;
  valueFormatter?: (value: number | string) => string;
}) {
  if (!active || !payload || payload.length === 0) return null;

  return (
    <div className="rounded-lg border border-border bg-surface px-3 py-2 text-xs shadow-md">
      {label !== undefined && (
        <p className="mb-1 font-medium text-foreground/70">
          {labelFormatter ? labelFormatter(label) : label}
        </p>
      )}
      {payload.map((p, i) => (
        <div key={i} className="flex items-center gap-2">
          {p.color && (
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: p.color }} />
          )}
          <span className="text-foreground/60">{p.name}</span>
          <span className="font-medium tabular-nums">
            {p.value !== undefined && (valueFormatter ? valueFormatter(p.value) : p.value)}
          </span>
        </div>
      ))}
    </div>
  );
}
