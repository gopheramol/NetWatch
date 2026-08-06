import type { ReactNode } from "react";
import clsx from "clsx";

export function Card({
  children,
  className,
  title,
  action,
}: {
  children: ReactNode;
  className?: string;
  title?: string;
  action?: ReactNode;
}) {
  return (
    <div
      className={clsx(
        "rounded-2xl border border-border bg-surface p-5 shadow-sm",
        className,
      )}
    >
      {(title || action) && (
        <div className="mb-4 flex items-center justify-between">
          {title && <h2 className="text-sm font-semibold text-foreground/80">{title}</h2>}
          {action}
        </div>
      )}
      {children}
    </div>
  );
}
