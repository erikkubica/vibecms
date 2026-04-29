import type { ReactNode } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";

interface SidebarCardProps {
  title: string;
  icon?: ReactNode;
  children: ReactNode;
}

// SidebarCard is the standard right-rail card used by editors and settings
// pages: rounded card + section header + padded content. Everything that
// belongs in a "Publish"-style sidebar (language pickers, status, save
// action, clear cache, etc.) goes inside as children.
export function SidebarCard({ title, icon, children }: SidebarCardProps) {
  return (
    <Card className="rounded-xl border border-slate-200 shadow-sm">
      <SectionHeader title={title} icon={icon} />
      <CardContent className="space-y-4">{children}</CardContent>
    </Card>
  );
}

export default SidebarCard;
