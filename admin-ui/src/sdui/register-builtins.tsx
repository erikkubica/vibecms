import React from "react";
import { useNavigate } from "react-router-dom";
import { registerComponents } from "./registry";
import {
  WelcomeBanner,
  StatCard as SduiStatCard,
  RecentContentTable,
  ActivityFeed,
  QuickActions,
} from "./sdui-components";
import { ContentTypeCard, TaxonomyCard } from "./list-components";
import {
  ContentNodeTable,
  TaxonomyTermsTable,
  PageHeader,
  SearchToolbar,
  TaxonomyFilterChips,
} from "./table-components";
import { GenericListTable } from "./generic-list-table";
import { ThemesGrid } from "./themes-grid";
import { ExtensionsGrid } from "./extensions-grid";
import { SettingsForm } from "./settings-form";
import { SchemaSettings } from "./schema-settings";

// ---------------------------------------------------------------------------
// Layout primitives — thin wrappers that map SDUI props to real DOM/React.
// These are the "Tier 4: Layout" and "Tier 2: Composites" components that
// the Go kernel can reference by name in layout trees.
// ---------------------------------------------------------------------------

/** Vertical flex stack with configurable gap and className. */
function VerticalStack({
  gap = 4,
  className = "",
  children,
}: {
  gap?: number;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className={`flex flex-col gap-${gap} ${className}`}>{children}</div>
  );
}

/** Horizontal flex stack. */
function HorizontalStack({
  gap = 4,
  className = "",
  align = "center",
  children,
}: {
  gap?: number;
  className?: string;
  align?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className={`flex flex-row items-${align} gap-${gap} ${className}`}>
      {children}
    </div>
  );
}

/** Admin page header with title and optional back link. */
function AdminHeader({
  title,
  back,
  children,
}: {
  title: string;
  back?: string;
  children?: React.ReactNode;
}) {
  const navigate = useNavigate();
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        {back && (
          <button
            onClick={() =>
              back.startsWith("/") ? navigate(back) : navigate(-1)
            }
            className="rounded-md p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="m15 18-6-6 6-6" />
            </svg>
          </button>
        )}
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">
          {title}
        </h1>
      </div>
      {children}
    </div>
  );
}

/** Simple card wrapper. */
function CardWrapper({
  className = "",
  children,
}: {
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div
      className={`rounded-xl border border-slate-200 bg-white shadow-sm ${className}`}
    >
      {children}
    </div>
  );
}

/** Stat card — delegates to the enhanced SDUI StatCard with colored icons. */
function StatCard(props: {
  label: string;
  value: string | number;
  icon?: string;
  color?: string;
  change?: string;
}) {
  return <SduiStatCard {...props} />;
}

/** Grid layout for dashboard widgets and card grids. */
function Grid({
  cols = 3,
  gap = 4,
  className = "",
  children,
}: {
  cols?: number;
  gap?: number;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div
      className={`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-${cols} gap-${gap} ${className}`}
    >
      {children}
    </div>
  );
}

/** Dashboard placeholder — queries stats via TanStack and renders stat cards. */
function DashboardWidgets() {
  // For now, render a static grid that will be replaced by data-driven widgets.
  // The real implementation will use TanStack Query to fetch stats.
  return (
    <Grid cols={4} gap={4}>
      <StatCard label="Total Pages" value="—" />
      <StatCard label="Published" value="—" />
      <StatCard label="Drafts" value="—" />
      <StatCard label="Users" value="—" />
    </Grid>
  );
}

/** List page header with title, count, and "New" button. */
function ListHeader({
  title,
  count,
  newPath,
}: {
  title: string;
  count?: number;
  newPath?: string;
}) {
  const navigate = useNavigate();
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">
          {title}
        </h1>
        {count !== undefined && (
          <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600">
            {count}
          </span>
        )}
      </div>
      {newPath && (
        <button
          onClick={() => navigate(newPath)}
          className="inline-flex items-center gap-2 rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M12 5v14M5 12h14" />
          </svg>
          New
        </button>
      )}
    </div>
  );
}

/** Search + filter toolbar for list pages. */
function ListToolbar({
  searchPlaceholder = "Search...",
}: {
  searchPlaceholder?: string;
  filters?: unknown[];
}) {
  // Placeholder — the full implementation will use the page store for
  // reactive search + filter state. For validation we just show a search input.
  return (
    <div className="flex items-center gap-3">
      <div className="relative flex-1 max-w-sm">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        >
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.3-4.3" />
        </svg>
        <input
          type="text"
          placeholder={searchPlaceholder}
          className="h-9 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-400 focus:outline-none"
        />
      </div>
    </div>
  );
}

/** Placeholder data table. The real one will accept column defs and data. */
function DataTable({
  nodeType,
}: {
  endpoint?: string;
  nodeType?: string;
  columns?: unknown[];
}) {
  return (
    <CardWrapper className="overflow-hidden">
      <div className="p-8 text-center text-sm text-slate-500">
        <p className="font-medium">DataTable: {nodeType || "unknown"}</p>
        <p className="mt-1 text-slate-400">
          Data-bound table will render here when connected to TanStack Query.
        </p>
      </div>
    </CardWrapper>
  );
}

/** Generic button that can trigger SDUI actions. */
function VibeButton({
  label,
  variant = "default",
  onClick,
}: {
  label: string;
  variant?: "default" | "destructive" | "outline" | "ghost";
  onClick?: () => void;
}) {
  const base =
    "inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors";
  const variants: Record<string, string> = {
    default: "bg-slate-900 text-white hover:bg-slate-800",
    destructive: "bg-red-600 text-white hover:bg-red-700",
    outline:
      "border border-slate-200 bg-white text-slate-700 hover:bg-slate-50",
    ghost: "text-slate-700 hover:bg-slate-100",
  };

  return (
    <button
      className={`${base} ${variants[variant] || variants.default}`}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

/** Plain text block — renders a string or children. */
function TextBlock({
  text,
  className = "",
}: {
  text?: string;
  className?: string;
}) {
  return <p className={`text-sm text-slate-700 ${className}`}>{text}</p>;
}

/** Divider line. */
function Divider({ className = "" }: { className?: string }) {
  return <hr className={`border-slate-200 ${className}`} />;
}

/** Spacer — adds vertical space. */
function Spacer({ height = 4 }: { height?: number }) {
  return <div style={{ height: `${height * 0.25}rem` }} />;
}

/** Sidebar layout — content area + sidebar panel. */
function SidebarLayout({
  sidebarWidth = 320,
  className = "",
  children,
}: {
  sidebarWidth?: number;
  className?: string;
  children?: React.ReactNode;
}) {
  // Expects exactly 2 children: [content, sidebar]
  const childArray = React.Children.toArray(children);
  const content = childArray[0] ?? null;
  const sidebar = childArray[1] ?? null;

  return (
    <div className={`flex ${className}`}>
      <div className="flex-1 min-w-0">{content}</div>
      <div
        className="flex-shrink-0 border-l border-slate-200 bg-slate-50"
        style={{ width: sidebarWidth }}
      >
        {sidebar}
      </div>
    </div>
  );
}

/** Tabs layout — renders children as tab panels. */
function TabLayout({
  tabs,
  children,
}: {
  tabs: Array<{ key: string; label: string }>;
  children?: React.ReactNode;
}) {
  const [active, setActive] = React.useState(tabs[0]?.key ?? "");
  const childArray = React.Children.toArray(children);

  return (
    <div>
      <div className="flex gap-1 border-b border-slate-200">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setActive(t.key)}
            className={`px-4 py-2 text-sm font-medium transition-colors ${
              active === t.key
                ? "border-b-2 border-slate-900 text-slate-900"
                : "text-slate-500 hover:text-slate-700"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>
      <div className="pt-4">
        {tabs.map((t, i) =>
          active === t.key ? (
            <React.Fragment key={t.key}>{childArray[i] ?? null}</React.Fragment>
          ) : null,
        )}
      </div>
    </div>
  );
}

/** Loading skeleton card. */
function LoadingCard() {
  return (
    <CardWrapper className="p-6">
      <div className="space-y-3 animate-pulse">
        <div className="h-4 w-1/3 rounded bg-slate-200" />
        <div className="h-4 w-2/3 rounded bg-slate-200" />
        <div className="h-4 w-1/2 rounded bg-slate-200" />
      </div>
    </CardWrapper>
  );
}

/** Empty state with icon, message, and optional action. */
function EmptyState({
  title,
  description,
  actionLabel,
  onAction,
}: {
  icon?: string;
  title?: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="rounded-full bg-slate-100 p-3">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-slate-400"
        >
          <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
          <polyline points="14 2 14 8 20 8" />
        </svg>
      </div>
      <h3 className="mt-4 text-sm font-semibold text-slate-900">
        {title || "No items yet"}
      </h3>
      {description && (
        <p className="mt-1 text-sm text-slate-500">{description}</p>
      )}
      {actionLabel && onAction && (
        <button
          onClick={onAction}
          className="mt-4 inline-flex items-center gap-2 rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800"
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}

/** Error card — shown when an SDUI component fails. */
function ErrorCard({ title, message }: { title?: string; message?: string }) {
  return (
    <div className="rounded-lg border border-red-200 bg-red-50 p-4">
      <p className="text-sm font-medium text-red-800">
        {title || "Something went wrong"}
      </p>
      {message && <p className="mt-1 text-sm text-red-600">{message}</p>}
    </div>
  );
}

/** Section wrapper with optional title. */
function Section({
  title,
  className = "",
  children,
}: {
  title?: string;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className={className}>
      {title && (
        <h2 className="mb-3 text-sm font-semibold text-slate-900">{title}</h2>
      )}
      {children}
    </div>
  );
}

/** Scroll region — wraps content in a scrollable container. */
function ScrollRegion({
  maxHeight = 400,
  className = "",
  children,
}: {
  maxHeight?: number;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className={`overflow-y-auto ${className}`} style={{ maxHeight }}>
      {children}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Registration — all built-in components are registered here so the
// RecursiveRenderer can look them up by type name.
// ---------------------------------------------------------------------------

export function registerBuiltinComponents() {
  registerComponents({
    // Layout primitives
    VerticalStack,
    HorizontalStack,
    Grid,
    SidebarLayout,
    TabLayout,
    Section,
    ScrollRegion,
    Spacer,
    Divider,

    // Page-level composites
    AdminHeader,
    DashboardWidgets,
    ListHeader,
    ListToolbar,
    DataTable,

    // UI primitives
    VibeButton,
    TextBlock,
    CardWrapper,
    StatCard,

    // Dashboard widgets
    WelcomeBanner,
    RecentContentTable,
    ActivityFeed,
    QuickActions,

    // Feedback
    LoadingCard,
    EmptyState,
    ErrorCard,

    // List components
    ContentTypeCard,
    TaxonomyCard,

    // Table-driven SDUI components
    ContentNodeTable,
    TaxonomyTermsTable,
    PageHeader,
    SearchToolbar,
    TaxonomyFilterChips,
    GenericListTable,
    ThemesGrid,
    ExtensionsGrid,

    // Settings (SDUI-driven form)
    SettingsForm,
    // Settings (registry-driven generic form)
    SchemaSettings,
  });
}
