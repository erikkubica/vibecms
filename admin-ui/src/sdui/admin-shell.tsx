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

type IconComponent = React.ComponentType<{ className?: string; size?: number; style?: React.CSSProperties; strokeWidth?: number | string }>;
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
                style={{color: childActive ? "var(--sb-fg-active)" : "var(--sb-fg-muted)", opacity: childActive ? 1 : 0.7}}
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
        {IconComp && (
          <IconComp
            size={14.5}
            strokeWidth={1.6}
            className="shrink-0"
            style={{
              color: active ? "var(--sb-fg-active)" : "var(--sb-fg-muted)",
              opacity: active ? 1 : 0.7,
            }}
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
        <header
          className="flex shrink-0 items-center justify-between"
          style={{ height: 48, padding: "0 22px", background: "var(--app-bg)", borderBottom: "1px solid var(--divider)" }}
        >
          <div className="flex items-center" style={{ gap: 8 }}>
            <button
              className="flex h-7 w-7 items-center justify-center rounded-md lg:hidden"
              style={{ color: "var(--fg-muted)" }}
              onClick={() => setSidebarOpen(true)}
            >
              <Menu size={18} />
            </button>
            <nav className="flex items-center" style={{ gap: 6, fontSize: 12.5, color: "var(--fg-muted)", letterSpacing: "-0.005em" }}>
              <Home size={12} style={{ color: "var(--fg-subtle)" }} />
              {breadcrumbs.map((crumb, i) => {
                const last = i === breadcrumbs.length - 1;
                return (
                  <span key={i} className="flex items-center" style={{ gap: 6 }}>
                    <span style={{ color: "var(--fg-hint)", fontSize: 12, opacity: 0.7 }}>/</span>
                    <span style={{ color: last ? "var(--fg)" : "var(--fg-muted)", fontWeight: last ? 500 : 400 }}>
                      {crumb}
                    </span>
                  </span>
                );
              })}
            </nav>
          </div>

          <div className="flex items-center" style={{ gap: 4 }}>
            {languages.length > 0 && (
              <label
                className="hidden sm:flex items-center"
                style={{ height: 28, padding: "0 10px", borderRadius: 6, gap: 6, fontSize: 12.5, color: "var(--fg-2)", background: "transparent", letterSpacing: "-0.005em", transition: "background 0.12s" }}
                onMouseEnter={(e) => (e.currentTarget.style.background = "var(--hover-bg)")}
                onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
              >
                <Globe size={12} style={{ opacity: 0.6 }} />
                <select
                  value={currentCode}
                  onChange={(e) => handleLanguageChange(e.target.value)}
                  className="bg-transparent outline-none"
                  style={{ fontSize: 12.5, color: "var(--fg-2)" }}
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

            <button
              onClick={handleClearCache}
              disabled={clearingCache}
              className="hidden sm:flex items-center"
              style={{ height: 28, padding: "0 10px", borderRadius: 6, gap: 6, fontSize: 12.5, color: "var(--fg-2)", background: "transparent", letterSpacing: "-0.005em", opacity: clearingCache ? 0.5 : 1, transition: "background 0.12s" }}
              onMouseEnter={(e) => (e.currentTarget.style.background = "var(--hover-bg)")}
              onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
              title="Clear all caches"
            >
              <RefreshCw size={12} style={{ opacity: 0.6 }} className={clearingCache ? "animate-spin" : ""} />
              {clearingCache ? "Clearing…" : "Clear cache"}
            </button>

            <button
              onClick={() => window.open("/", "_blank")}
              className="hidden sm:flex items-center"
              style={{ height: 28, padding: "0 10px", borderRadius: 6, gap: 6, fontSize: 12.5, color: "var(--fg-2)", background: "transparent", letterSpacing: "-0.005em", transition: "background 0.12s" }}
              onMouseEnter={(e) => (e.currentTarget.style.background = "var(--hover-bg)")}
              onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
            >
              <ExternalLink size={12} style={{ opacity: 0.6 }} />
              Visit site
            </button>

            <button
              className="flex items-center justify-center"
              style={{ width: 28, height: 28, borderRadius: 6, color: "var(--fg-muted)", background: "transparent", border: "none", transition: "background 0.12s" }}
              onMouseEnter={(e) => (e.currentTarget.style.background = "var(--hover-bg)")}
              onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
            >
              <Bell size={14} style={{ opacity: 0.7 }} />
            </button>

            <div style={{ width: 1, height: 18, background: "var(--divider)", margin: "0 4px" }} />

            <button
              className="flex items-center"
              style={{ height: 28, padding: "0 10px 0 4px", borderRadius: 14, gap: 8, background: "transparent", border: "none", transition: "background 0.1s" }}
              onMouseEnter={(e) => (e.currentTarget.style.background = "var(--hover-bg)")}
              onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
              title={user?.email}
            >
              <span
                className="grid place-items-center shrink-0"
                style={{ width: 22, height: 22, borderRadius: "50%", background: "var(--accent)", color: "var(--accent-fg)", fontSize: 10, fontWeight: 600 }}
              >
                {(user?.full_name || user?.email || "A").charAt(0).toUpperCase()}
              </span>
              <span className="hidden sm:inline" style={{ fontSize: 12.5, fontWeight: 500, color: "var(--fg-2)" }}>
                {user?.full_name || (user?.email ? user.email.split("@")[0] : "Admin")}
              </span>
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className={mainClassName ?? "flex-1 overflow-y-auto"} style={{ padding: "18px 22px 40px" }}>
          <div style={{ maxWidth: 1200, margin: "0 auto" }}>{children}</div>
        </main>
      </div>
    </div>
  );
}
