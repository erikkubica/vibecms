import React, { useState, useMemo, useCallback } from "react";
import { useLocation, Link } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useBoot } from "../hooks/use-boot";
import { useAuth } from "../hooks/use-auth";
import { useAdminLanguage } from "../hooks/use-admin-language";
import { getThemeSettingsPages, clearCache } from "../api/client";
import type { NavItem } from "./types";
import * as Lucide from "lucide-react";
import {
  ChevronDown,
  Menu,
  X,
  LogOut,
  ChevronRight,
  ExternalLink,
  Home,
  Bell,
  RefreshCw,
  Globe,
} from "lucide-react";
import { toast } from "sonner";

// ---------------------------------------------------------------------------
// Icon resolution — look up any icon name against the full lucide-react
// export. Supports the PascalCase names lucide uses ("ImageDown") as well
// as the kebab-case aliases themes/extensions may pass ("image-down").
// ---------------------------------------------------------------------------

type IconComponent = React.ComponentType<{ className?: string; size?: number; style?: React.CSSProperties }>;
const lucideIcons = Lucide as unknown as Record<string, IconComponent>;

function toPascalCase(name: string): string {
  return name
    .split(/[-_]/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
    .join("");
}

function getIcon(name: string): IconComponent | null {
  if (!name) return null;
  return (
    lucideIcons[name] ||
    lucideIcons[toPascalCase(name)] ||
    lucideIcons["Puzzle"] ||
    null
  );
}

// ---------------------------------------------------------------------------
// Section ordering and labels
// ---------------------------------------------------------------------------

// No longer needed — section headers come from is_section items in the
// flat navigation list returned by the boot manifest.

// ---------------------------------------------------------------------------
// Breadcrumb computation
// ---------------------------------------------------------------------------

const prettify = (s: string) =>
  s.replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

const sectionBreadcrumbLabels: Record<string, string> = {
  dashboard: "Dashboard",
  pages: "Pages",
  posts: "Posts",
  content: "Content",
  "content-types": "Content Types",
  taxonomies: "Taxonomies",
  "block-types": "Block Types",
  templates: "Templates",
  layouts: "Layouts",
  "layout-blocks": "Layout Blocks",
  menus: "Menus",
  themes: "Themes",
  extensions: "Extensions",
  users: "Users",
  roles: "Roles",
  languages: "Languages",
  settings: "Settings",
  "mcp-tokens": "MCP Tokens",
  ext: "Extensions",
  sdui: "SDUI",
};

function computeBreadcrumbs(pathname: string): string[] {
  const parts = pathname
    .replace(/^\/admin\/?/, "")
    .split("/")
    .filter(Boolean);
  if (parts.length === 0) return [];

  const crumbs: string[] = [];

  for (let i = 0; i < parts.length; i++) {
    const seg = parts[i];
    if (/^\d+$/.test(seg)) continue;
    if (seg === "edit") {
      crumbs.push("Edit");
      continue;
    }
    if (seg === "new") {
      crumbs.push("New");
      continue;
    }
    crumbs.push(sectionBreadcrumbLabels[seg] || prettify(seg));
  }

  return crumbs;
}

// ---------------------------------------------------------------------------
// Sidebar Navigation
// ---------------------------------------------------------------------------

interface SidebarProps {
  navigation: NavItem[];
  collapsed: boolean;
  onToggleCollapse: () => void;
  onClose: () => void;
}

function hasActiveDescendant(items: NavItem[], pathname: string): boolean {
  for (const item of items) {
    if (item.path) {
      const p = pathname;
      if (
        p === item.path ||
        p.startsWith(item.path.endsWith("/") ? item.path : item.path + "/")
      )
        return true;
    }
    if (item.children && hasActiveDescendant(item.children, pathname))
      return true;
  }
  return false;
}

function SidebarNav({
  navigation,
  collapsed,
  onToggleCollapse,
  onClose,
}: SidebarProps) {
  const location = useLocation();
  // Tracks groups explicitly toggled by the user (overrides auto-open logic).
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});

  const toggleGroup = useCallback((id: string, currentlyOpen: boolean) => {
    setOpenGroups((prev) => ({ ...prev, [id]: !currentlyOpen }));
  }, []);

  const isActive = useCallback(
    (path: string | undefined): boolean => {
      if (!path) return false;
      const p = location.pathname;
      if (p === path) return true;
      const prefix = path.endsWith("/") ? path : path + "/";
      if (!p.startsWith(prefix)) return false;
      // Only activate via prefix if remaining segments are resource identifiers
      // (numeric IDs, UUIDs, "edit", "new") — not named sub-sections like "taxonomies".
      const remaining = p.slice(prefix.length);
      return remaining
        .split("/")
        .every(
          (seg) =>
            seg === "" ||
            seg === "edit" ||
            seg === "new" ||
            /^\d+$/.test(seg) ||
            /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i.test(seg),
        );
    },
    [location.pathname],
  );

  // A group is open if the user explicitly opened it, OR if it has an active
  // descendant and the user hasn't explicitly closed it.
  const isGroupOpen = useCallback(
    (item: NavItem): boolean => {
      if (item.id in openGroups) return openGroups[item.id];
      return hasActiveDescendant(item.children || [], location.pathname);
    },
    [openGroups, location.pathname],
  );

  const renderItem = (item: NavItem, depth: number = 0) => {
    const IconComp = getIcon(item.icon || "");
    const hasChildren = item.children && item.children.length > 0;
    const isOpen = isGroupOpen(item);
    // For parent groups, only apply the active style when a child is active —
    // but use a lighter treatment so the leaf item remains the primary highlight.
    const childActive =
      hasChildren && hasActiveDescendant(item.children!, location.pathname);
    // Leaf active: exact match or prefix match (for sub-routes like /edit/1)
    const active = !hasChildren && isActive(item.path);

    // Section header — non-clickable separator from boot manifest.
    if (item.is_section) {
      if (collapsed) {
        return <div key={item.id} className="my-2 mx-2 h-px" style={{background: "var(--sb-border)"}} />;
      }
      return (
        <div
          key={item.id}
          className="px-3 pt-4 pb-1 text-[10.5px] font-semibold uppercase tracking-[0.06em]"
          style={{color: "var(--sb-fg-muted)"}}
        >
          {item.label}
        </div>
      );
    }

    if (hasChildren) {
      return (
        <div key={item.id}>
          <button
            onClick={() => toggleGroup(item.id, isOpen)}
            className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-[13px] font-medium transition-colors`}
            style={{color: childActive ? "var(--sb-fg-active)" : "var(--sb-fg)"}}
            title={collapsed ? item.label : undefined}
          >
            {IconComp && (
              <IconComp
                size={15}
                className="shrink-0"
                style={{color: childActive ? "var(--accent)" : "var(--sb-fg-muted)"}}
              />
            )}
            {!collapsed && (
              <>
                <span className="flex-1 truncate">{item.label}</span>
                <ChevronDown
                  size={12}
                  className={`shrink-0 transition-transform duration-150 ${
                    isOpen ? "rotate-0" : "-rotate-90"
                  }`}
                  style={{color: "var(--sb-fg-muted)"}}
                />
              </>
            )}
          </button>
          {isOpen && !collapsed && (
            <div className="mt-0.5 ml-3 border-l space-y-[1px]" style={{borderColor: "var(--sb-border)"}}>
              {item.children!.map((child) => renderItem(child, depth + 1))}
            </div>
          )}
        </div>
      );
    }

    // Child rows live inside a parent group and visually attach to the
    // group's left border. They render with right-only rounding and a
    // full-height active indicator that sits flush against the border —
    // the rounded left edge looks detached when each child is supposed
    // to read as 'part of this group'.
    const isChild = depth > 0;
    const radiusClass = isChild ? "rounded-r-md rounded-l-none" : "rounded-lg";

    // Leaf item
    const linkContent = (
      <>
        {active && !collapsed && (
          <span
            className={`absolute left-0 w-[2px] ${
              isChild ? "top-0 bottom-0" : "top-1 bottom-1 rounded"
            }`}
            style={{background: "var(--accent)"}}
          />
        )}
        {IconComp && (
          <IconComp
            size={15}
            className="shrink-0"
            style={{color: active ? "var(--accent)" : "var(--sb-fg-muted)"}}
          />
        )}
        {!collapsed && <span className="flex-1 truncate">{item.label}</span>}
      </>
    );

    if (item.path) {
      return (
        <Link
          key={item.id}
          to={item.path}
          onClick={onClose}
          className={`relative flex w-full items-center gap-2.5 ${radiusClass} px-3 py-2 text-[13px] font-medium transition-colors`}
          style={{
            color: active ? "var(--sb-fg-active)" : "var(--sb-fg)",
            background: active ? "var(--sb-active)" : undefined,
            ...(isChild && depth > 1
              ? { paddingLeft: `${12 + (depth - 1) * 12}px` }
              : {}),
          }}
          title={collapsed ? item.label : undefined}
        >
          {linkContent}
        </Link>
      );
    }

    return (
      <button
        key={item.id}
        className={`relative flex w-full items-center gap-2.5 ${radiusClass} px-3 py-2 text-[13px] font-medium transition-colors`}
        style={{
          color: active ? "var(--sb-fg-active)" : "var(--sb-fg)",
          background: active ? "var(--sb-active)" : undefined,
        }}
        title={collapsed ? item.label : undefined}
      >
        {linkContent}
      </button>
    );
  };

  return (
    <nav className="flex-1 overflow-y-auto px-2 py-2 scrollbar-thin">
      {/* Expand button when collapsed */}
      {collapsed && (
        <div className="mb-2 flex justify-center">
          <button
            onClick={onToggleCollapse}
            className="flex h-7 w-7 items-center justify-center rounded"
            style={{color: "var(--sb-fg-muted)"}}
            title="Expand sidebar"
          >
            <ChevronRight size={14} className="rotate-180" />
          </button>
        </div>
      )}

      {/* Iterate the flat navigation list — is_section items render as
          non-clickable headers, regular items as links, items with children
          as expandable groups. */}
      {navigation.map((item) => renderItem(item))}
    </nav>
  );
}

// ---------------------------------------------------------------------------
// Main Admin Shell
// ---------------------------------------------------------------------------

interface SduiAdminShellProps {
  children: React.ReactNode;
  mainClassName?: string;
}

export function SduiAdminShell({ children, mainClassName }: SduiAdminShellProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const location = useLocation();
  const { user, logout } = useAuth();
  const { data: boot } = useBoot();
  const { languages, currentCode, setCurrentCode } = useAdminLanguage();
  const queryClient = useQueryClient();
  const [clearingCache, setClearingCache] = useState(false);
  const { data: themePages } = useQuery({
    queryKey: ["theme-settings-pages", currentCode],
    queryFn: getThemeSettingsPages,
    staleTime: 60_000,
  });

  const handleClearCache = useCallback(async () => {
    setClearingCache(true);
    try {
      await clearCache();
      toast.success("All caches cleared");
    } catch {
      toast.error("Failed to clear caches");
    } finally {
      setClearingCache(false);
    }
  }, []);

  const handleLanguageChange = useCallback(
    (code: string) => {
      setCurrentCode(code);
      // Invalidate every query so locale-aware fetches refresh under the new
      // X-Admin-Language header. queryClient.invalidateQueries() with no key
      // marks every cached query stale.
      queryClient.invalidateQueries();
    },
    [setCurrentCode, queryClient],
  );

  const breadcrumbs = useMemo(
    () => computeBreadcrumbs(location.pathname),
    [location.pathname],
  );

  const navigation = useMemo<NavItem[]>(() => {
    const base = boot?.navigation || [];
    const pages = themePages?.pages || [];
    if (pages.length === 0) return base;
    const themeSection: NavItem[] = [
      {
        id: "theme-settings-section",
        label: "Theme Settings",
        is_section: true,
      },
      ...pages.map<NavItem>((p) => ({
        id: `theme-settings-${p.slug}`,
        label: p.name,
        icon: p.icon || "Palette",
        path: `/admin/theme-settings/${p.slug}`,
      })),
    ];
    // Insert immediately before the kernel "Settings" section so theme
    // settings appear above core settings — they're touched far more often
    // (theme tweaks vs. one-time site config).
    const insertAt = base.findIndex((item) => item.id === "section-settings");
    if (insertAt === -1) {
      return [...base, ...themeSection];
    }
    return [...base.slice(0, insertAt), ...themeSection, ...base.slice(insertAt)];
  }, [boot?.navigation, themePages?.pages]);

  const sidebarWidth = collapsed ? 56 : 256;

  const handleLogout = useCallback(async () => {
    await logout();
  }, [logout]);

  return (
    <div className="flex h-screen overflow-hidden" style={{background: "var(--app-bg)"}}>
      {/* ----------------------------------------------------------------- */}
      {/* Mobile overlay                                                     */}
      {/* ----------------------------------------------------------------- */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* ----------------------------------------------------------------- */}
      {/* Sidebar                                                            */}
      {/* ----------------------------------------------------------------- */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex flex-col transition-all duration-200 lg:relative lg:z-auto ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        }`}
        style={{ width: sidebarWidth, background: "var(--sb-bg)" }}
      >
        {/* Logo header */}
        <div className="flex shrink-0 items-center" style={{ height: 48, padding: "0 14px" }}>
          <div className="flex items-center" style={{ gap: 10 }}>
            <div
              className="grid shrink-0 place-items-center"
              style={{
                width: 22,
                height: 22,
                borderRadius: 6,
                background: "var(--accent)",
                color: "var(--accent-fg)",
                fontFamily: "var(--font-mono)",
                fontSize: 11,
                fontWeight: 600,
              }}
            >
              S
            </div>
            {!collapsed && (
              <span style={{ fontSize: 14, fontWeight: 600, color: "var(--sb-fg-active)", letterSpacing: "-0.02em" }}>
                Squilla
              </span>
            )}
          </div>
          {!collapsed && (
            <button
              onClick={() => setCollapsed(true)}
              className="ml-auto hidden h-6 w-6 items-center justify-center rounded lg:grid"
              style={{color: "var(--sb-fg-muted)"}}
              title="Collapse sidebar"
            >
              <ChevronRight size={14} />
            </button>
          )}
          <button
            className="ml-auto rounded p-1 lg:hidden"
            style={{color: "var(--sb-fg-muted)"}}
            onClick={() => setSidebarOpen(false)}
          >
            <X size={16} />
          </button>
        </div>

        {/* Navigation */}
        <SidebarNav
          navigation={navigation}
          collapsed={collapsed}
          onToggleCollapse={() => setCollapsed(false)}
          onClose={() => setSidebarOpen(false)}
        />

        {/* User footer */}
        <div className="shrink-0 border-t p-2" style={{borderColor: "var(--sb-border)"}}>
          {collapsed ? (
            <button
              onClick={handleLogout}
              className="flex w-full items-center justify-center rounded-lg py-2"
              style={{color: "var(--sb-fg-muted)"}}
              title="Log out"
            >
              <LogOut size={15} />
            </button>
          ) : (
            <div className="flex items-center gap-2.5 rounded-lg px-3 py-2">
              <div className="grid h-7 w-7 shrink-0 place-items-center rounded-full text-xs font-semibold" style={{background: "rgba(99,102,241,0.2)", color: "var(--accent)"}}>
                {(user?.full_name || user?.email || "A")
                  .charAt(0)
                  .toUpperCase()}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-medium" style={{color: "var(--sb-fg)"}}>
                  {user?.full_name || "Admin"}
                </p>
                <p className="truncate text-[10px]" style={{color: "var(--sb-fg-muted)"}}>
                  {user?.email}
                </p>
              </div>
              <button
                onClick={handleLogout}
                className="shrink-0 rounded p-1"
                style={{color: "var(--sb-fg-muted)"}}
                title="Log out"
              >
                <LogOut size={14} />
              </button>
            </div>
          )}
        </div>
      </aside>

      {/* ----------------------------------------------------------------- */}
      {/* Main content area                                                  */}
      {/* ----------------------------------------------------------------- */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex h-12 shrink-0 items-center justify-between border-b border-border bg-card px-4">
          <div className="flex items-center gap-2">
            {/* Mobile hamburger */}
            <button
              className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground lg:hidden"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu size={18} />
            </button>

            {/* Breadcrumbs */}
            <nav className="flex items-center gap-1 text-xs text-muted-foreground">
              <Home size={12} style={{color: "var(--fg-subtle)"}} />
              {breadcrumbs.map((crumb, i) => {
                const last = i === breadcrumbs.length - 1;
                return (
                  <span key={i} className="flex items-center gap-1">
                    <ChevronRight size={10} style={{color: "var(--fg-subtle)"}} />
                    <span
                      className={`rounded px-1 py-0.5 ${
                        last ? "font-medium text-foreground" : "text-muted-foreground"
                      }`}
                    >
                      {crumb}
                    </span>
                  </span>
                );
              })}
            </nav>
          </div>

          <div className="flex items-center gap-2">
            {/* Admin language selector — picks which locale the admin is
                editing. Every translatable surface (settings pages, content
                lists) uses this as its default; the per-page selector can
                override it. */}
            {languages.length > 0 && (
              <label className="hidden h-7 items-center gap-1.5 rounded-md border border-border pl-2 pr-1 text-xs font-medium text-muted-foreground sm:flex">
                <Globe size={12} className="text-muted-foreground" />
                <select
                  value={currentCode}
                  onChange={(e) => handleLanguageChange(e.target.value)}
                  className="bg-transparent pr-1 text-xs font-medium text-foreground outline-none"
                  aria-label="Admin language"
                >
                  {languages.map((lang) => (
                    <option key={lang.code} value={lang.code}>
                      {lang.name || lang.code}
                    </option>
                  ))}
                </select>
              </label>
            )}

            {/* Clear cache — invalidates the public-render layout/asset
                caches; useful after editing templates or activating themes. */}
            <button
              onClick={handleClearCache}
              disabled={clearingCache}
              className="hidden h-7 items-center gap-1.5 rounded-md border border-border px-2.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted disabled:opacity-50 sm:flex"
              title="Clear all caches"
            >
              <RefreshCw
                size={12}
                className={clearingCache ? "animate-spin" : ""}
              />
              {clearingCache ? "Clearing..." : "Clear Cache"}
            </button>

            {/* Visit site */}
            <button
              onClick={() => window.open("/", "_blank")}
              className="hidden h-7 items-center gap-1.5 rounded-md border border-border px-2.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted sm:flex"
            >
              <ExternalLink size={12} />
              Visit Site
            </button>

            {/* Notifications placeholder */}
            <button className="relative flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground">
              <Bell size={15} />
            </button>

            {/* User avatar */}
            <button
              className="flex h-7 w-7 items-center justify-center rounded-full text-xs font-semibold"
              style={{background: "var(--accent-weak)", color: "var(--accent-strong)"}}
              title={user?.email}
            >
              {(user?.full_name || user?.email || "A").charAt(0).toUpperCase()}
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className={mainClassName ?? "flex-1 overflow-y-auto p-5"}>{children}</main>
      </div>
    </div>
  );
}
