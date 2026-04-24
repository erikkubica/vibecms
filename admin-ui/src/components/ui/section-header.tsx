import type { ReactNode } from "react";

interface SectionHeaderProps {
  title: string;
  icon?: ReactNode;
  actions?: ReactNode;
}

export function SectionHeader({ title, icon, actions }: SectionHeaderProps) {
  return (
    <div
      className="flex items-center justify-between px-4 py-3"
      style={{ background: "var(--sub-bg)", borderBottom: "1px solid var(--border)" }}
    >
      <div className="flex items-center gap-2">
        {icon}
        <h2 className="font-semibold" style={{ fontSize: 13, color: "var(--fg)" }}>
          {title}
        </h2>
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}
