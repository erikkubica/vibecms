import type { ReactNode } from "react";

interface AccordionRowProps {
  headerLeft: ReactNode;
  headerRight?: ReactNode;
  children?: ReactNode;
  open: boolean;
  onToggle: () => void;
  depth?: number;
}

export function AccordionRow({
  headerLeft,
  headerRight,
  children,
  open,
  onToggle,
  depth = 0,
}: AccordionRowProps) {
  return (
    <div
      className="overflow-hidden"
      style={{
        border: "1px solid var(--border)",
        borderRadius: "var(--radius-lg)",
        background: "var(--card-bg)",
        marginLeft: `${depth * 16}px`,
      }}
    >
      <div
        className="flex items-center gap-2 cursor-pointer select-none"
        style={{
          padding: "8px 10px",
          background: "var(--sub-bg)",
          borderBottom: open ? "1px solid var(--border)" : "none",
        }}
        onClick={onToggle}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">{headerLeft}</div>
        {headerRight && (
          <div className="flex items-center gap-0.5" onClick={(e) => e.stopPropagation()}>
            {headerRight}
          </div>
        )}
      </div>
      {open && children && (
        <div className="bg-slate-50/70 px-4 py-4 space-y-3">{children}</div>
      )}
    </div>
  );
}
