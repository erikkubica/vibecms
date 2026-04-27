import React from "react";
import { useNavigate } from "react-router-dom";
import {
  LayoutDashboard,
  Database,
  Palette,
  Code,
  Settings,
  FileText,
  Newspaper,
  Boxes,
  PanelTop,
  Component,
  ListTree,
  Tag,
  Square,
  LayoutTemplate,
  Puzzle,
  Users,
  Shield,
  Globe,
  Key,
  Bell,
  MessageCircle,
  Map,
  Camera,
  Star,
  Heart,
  Zap,
} from "lucide-react";

// ---------------------------------------------------------------------------
// Icon map — shared across dashboard components
// ---------------------------------------------------------------------------

export const iconMap: Record<
  string,
  React.ComponentType<{ className?: string }>
> = {
  LayoutDashboard,
  Database,
  Palette,
  Code,
  Settings,
  FileText,
  Newspaper,
  Boxes,
  PanelTop,
  Component,
  ListTree,
  Tag,
  Square,
  LayoutTemplate,
  Puzzle,
  Users,
  Shield,
  Globe,
  Key,
  Bell,
  FileCode: LayoutTemplate,
  LayoutPanelTop: PanelTop,
  Languages: Globe,
  Brush: Palette,
  Blocks: Boxes,
  Menu: ListTree,
  Mail: Bell,
  FormInput: Square,
  Image: Boxes,
  ScrollText: FileText,
  Gavel: Shield,
  Layout: PanelTop,
  Send: Bell,
  Tags: Tag,
  Shapes: Boxes,
  MessageCircle,
  Map,
  Camera: Camera,
  Star,
  Heart,
  Bookmark: Tag,
  Calendar: FileText,
  Clock: FileText,
  Hash: Tag,
  Type: FileText,
  Zap,
};

// ---------------------------------------------------------------------------
// WelcomeBanner — gradient banner for the dashboard hero area
// ---------------------------------------------------------------------------

export function WelcomeBanner({
  title,
  subtitle,
  actionLabel,
  actionPath,
}: {
  title: string;
  subtitle: string;
  actionLabel?: string;
  actionPath?: string;
}) {
  const navigate = useNavigate();

  return (
    <div className="rounded-2xl bg-gradient-to-r from-indigo-600 to-indigo-800 p-6 text-white">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">{title}</h1>
          <p className="mt-1 text-sm text-indigo-200">{subtitle}</p>
        </div>
        {actionLabel && actionPath && (
          <button
            onClick={() => navigate(actionPath)}
            className="rounded-lg border border-white/30 bg-white/10 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-white/20"
          >
            + {actionLabel}
          </button>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// StatCard — dashboard stat with colored icon background
// ---------------------------------------------------------------------------

const colorMap: Record<string, { text: string; bg: string }> = {
  indigo: { text: "text-indigo-600", bg: "bg-indigo-50" },
  emerald: { text: "text-emerald-600", bg: "bg-emerald-50" },
  amber: { text: "text-amber-600", bg: "bg-amber-50" },
  violet: { text: "text-violet-600", bg: "bg-violet-50" },
  sky: { text: "text-sky-600", bg: "bg-sky-50" },
  rose: { text: "text-rose-600", bg: "bg-rose-50" },
  slate: { text: "text-slate-600", bg: "bg-slate-50" },
  blue: { text: "text-blue-600", bg: "bg-blue-50" },
};

export function StatCard({
  label,
  value,
  icon,
  color,
  change,
}: {
  label: string;
  value: string | number;
  icon?: string;
  color?: string;
  change?: string;
}) {
  const IconComp = icon ? iconMap[icon] : null;
  const c = colorMap[color || "indigo"] || colorMap.indigo;

  return (
    <div className="rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="flex items-center gap-4 p-6">
        {IconComp && (
          <div
            className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-lg ${c.bg}`}
          >
            <IconComp className={`h-6 w-6 ${c.text}`} />
          </div>
        )}
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-500">{label}</p>
          <p className="text-2xl font-bold text-slate-900">{value}</p>
          {change && <p className="mt-0.5 text-xs text-slate-500">{change}</p>}
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// RecentContentTable — table showing recent content nodes
// ---------------------------------------------------------------------------

interface RecentContentItem {
  id: number;
  title: string;
  node_type: string;
  status: string;
  updated_at: string;
}

const statusColors: Record<string, string> = {
  published: "bg-emerald-100 text-emerald-700",
  draft: "bg-amber-100 text-amber-700",
  archived: "bg-slate-100 text-slate-600",
  scheduled: "bg-blue-100 text-blue-700",
};

function editPathForNode(node: RecentContentItem): string {
  if (node.node_type === "post") return `/admin/posts/${node.id}/edit`;
  if (node.node_type === "page") return `/admin/pages/${node.id}/edit`;
  return `/admin/content/${node.node_type}/${node.id}/edit`;
}

export function RecentContentTable({ items }: { items: RecentContentItem[] }) {
  const navigate = useNavigate();

  if (!items || items.length === 0) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-500">
        No content yet. Create your first page to get started.
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="px-6 py-4">
        <h2 className="text-lg font-semibold text-slate-900">Recent Content</h2>
      </div>
      <table className="w-full">
        <thead>
          <tr className="border-t border-slate-100 bg-slate-50">
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
              Title
            </th>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
              Type
            </th>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
              Status
            </th>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
              Updated
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((node) => (
            <tr
              key={node.id}
              className="border-t border-slate-100 transition-colors hover:bg-slate-50"
            >
              <td className="px-6 py-4 text-sm">
                <button
                  onClick={() => navigate(editPathForNode(node))}
                  className="font-medium text-indigo-600 hover:text-indigo-700 hover:underline"
                >
                  {node.title}
                </button>
              </td>
              <td className="px-6 py-4 text-sm capitalize text-slate-500">
                {node.node_type?.replace(/_/g, " ")}
              </td>
              <td className="px-6 py-4 text-sm">
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                    statusColors[node.status] || statusColors.draft
                  }`}
                >
                  {node.status}
                </span>
              </td>
              <td className="px-6 py-4 text-sm text-slate-500">
                {node.updated_at}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ActivityFeed — simple list of recent activity items
// ---------------------------------------------------------------------------

export function ActivityFeed({
  items,
  title = "Recent Activity",
  emptyMessage = "No recent activity.",
}: {
  items: Array<{
    id: number | string;
    message: string;
    time: string;
    type?: string;
  }>;
  title?: string;
  emptyMessage?: string;
}) {
  if (!items || items.length === 0) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="px-6 py-4">
          <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
        </div>
        <p className="px-6 pb-6 text-center text-sm text-slate-500">
          {emptyMessage}
        </p>
      </div>
    );
  }

  const typeColors: Record<string, string> = {
    create: "bg-emerald-400",
    update: "bg-blue-400",
    delete: "bg-red-400",
    publish: "bg-indigo-400",
    default: "bg-slate-300",
  };

  return (
    <div className="rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="px-6 py-4">
        <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
      </div>
      <ul className="divide-y divide-slate-100">
        {items.map((item) => (
          <li key={item.id} className="flex items-start gap-3 px-6 py-3">
            <div
              className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${
                typeColors[item.type || "default"] || typeColors.default
              }`}
            />
            <div className="min-w-0 flex-1">
              <p className="text-sm text-slate-700">{item.message}</p>
              <p className="mt-0.5 text-xs text-slate-400">{item.time}</p>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

// ---------------------------------------------------------------------------
// QuickActions — grid of shortcut action buttons
// ---------------------------------------------------------------------------

export function QuickActions({
  actions,
}: {
  actions: Array<{
    label: string;
    path: string;
    icon?: string;
  }>;
}) {
  const navigate = useNavigate();

  if (!actions || actions.length === 0) return null;

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
      <h2 className="mb-4 text-lg font-semibold text-slate-900">
        Quick Actions
      </h2>
      <div className="grid grid-cols-2 gap-3">
        {actions.map((action, i) => {
          const IconComp = action.icon ? iconMap[action.icon] : null;
          return (
            <button
              key={i}
              onClick={() => navigate(action.path)}
              className="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2.5 text-sm font-medium text-slate-700 transition-colors hover:border-slate-300 hover:bg-slate-50"
            >
              {IconComp && <IconComp className="h-4 w-4 text-slate-400" />}
              {action.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}
