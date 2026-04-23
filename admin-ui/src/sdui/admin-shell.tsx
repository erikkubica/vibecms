import React, { useState, useMemo, useCallback } from "react";
import { useLocation, Link } from "react-router-dom";
import { useBoot } from "../hooks/use-boot";
import { useSSE } from "../hooks/use-sse";
import { useAuth } from "../hooks/use-auth";
import type { NavItem } from "./types";
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
  ChevronDown,
  Menu,
  X,
  LogOut,
  Bell,
  ChevronRight,
  Home,
  ExternalLink,
  MessageCircle,
  Map,
  Star,
  Heart,
  Zap,
} from "lucide-react";

// ---------------------------------------------------------------------------
// Icon map — maps string icon names from boot manifest to Lucide components
// ---------------------------------------------------------------------------

const iconMap: Record<
  string,
  React.ComponentType<{ className?: string; size?: number }>
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
  Mail: Bell,
  FormInput: Square,
  Image: Boxes,
  ScrollText: FileText,
  Gavel: Shield,
  Layout: PanelTop,
  Send: Bell,
  Tags: Tag,
  Shapes: Boxes,
  Home: LayoutDashboard,
  Menu: ListTree,
  MessageCircle,
  Map,
  Camera: Boxes,
  Star,
  Heart,
  Bookmark: Tag,
  Calendar: FileText,
  Clock: FileText,
  Hash: Tag,
  Type: FileText,
  Zap,
};

function toPascalCase(name: string): string {
  return name
    .split(/[-_]/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
    .join("");
}

function getIcon(
  name: string,
): React.ComponentType<{ className?: string; size?: number }> | null {
  if (!name) return null;
  return iconMap[name] || iconMap[toPascalCase(name)] || null;
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
        return <div key={item.id} className="my-2 mx-2 h-px bg-slate-700/50" />;
      }
      return (
        <div
          key={item.id}
          className="px-3 pt-4 pb-1 text-[10.5px] font-semibold uppercase tracking-[0.06em] text-slate-500"
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
            className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-[13px] font-medium transition-colors ${
              childActive
                ? "text-white"
                : "text-slate-300 hover:bg-slate-800/60 hover:text-white"
            }`}
            style={{
              paddingLeft: depth > 0 ? `${12 + depth * 12}px` : undefined,
            }}
            title={collapsed ? item.label : undefined}
          >
            {IconComp && (
              <IconComp
                size={15}
                className={`shrink-0 ${childActive ? "text-indigo-400" : "text-slate-400"}`}
              />
            )}
            {!collapsed && (
              <>
                <span className="flex-1 truncate">{item.label}</span>
                <ChevronDown
                  size={12}
                  className={`shrink-0 text-slate-500 transition-transform duration-150 ${
                    isOpen ? "rotate-0" : "-rotate-90"
                  }`}
                />
              </>
            )}
          </button>
          {isOpen && !collapsed && (
            <div className="mt-0.5 ml-3 border-l border-slate-700 pl-2 space-y-[1px]">
              {item.children!.map((child) => renderItem(child, depth + 1))}
            </div>
          )}
        </div>
      );
    }

    // Leaf item
    const linkContent = (
      <>
        {active && !collapsed && (
          <span className="absolute left-0 top-1 bottom-1 w-[2px] rounded bg-indigo-400" />
        )}
        {IconComp && (
          <IconComp
            size={15}
            className={`shrink-0 ${active ? "text-indigo-400" : "text-slate-400"}`}
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
          className={`relative flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors ${
            active
              ? "bg-slate-800 text-white"
              : "text-slate-300 hover:bg-slate-800/60 hover:text-white"
          }`}
          style={{
            paddingLeft: depth > 0 ? `${12 + depth * 12}px` : undefined,
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
        className={`relative flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors ${
          active
            ? "bg-slate-800 text-white"
            : "text-slate-300 hover:bg-slate-800/60 hover:text-white"
        }`}
        style={{ paddingLeft: depth > 0 ? `${12 + depth * 12}px` : undefined }}
        title={collapsed ? item.label : undefined}
      >
        {linkContent}
      </button>
    );
  };

  return (
    <nav className="flex-1 overflow-y-auto px-2 py-2 scrollbar-thin scrollbar-thumb-slate-700">
      {/* Expand button when collapsed */}
      {collapsed && (
        <div className="mb-2 flex justify-center">
          <button
            onClick={onToggleCollapse}
            className="flex h-7 w-7 items-center justify-center rounded text-slate-400 hover:bg-slate-800 hover:text-white"
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

  // Connect SSE for real-time updates
  useSSE();

  const breadcrumbs = useMemo(
    () => computeBreadcrumbs(location.pathname),
    [location.pathname],
  );

  const navigation = boot?.navigation || [];

  const sidebarWidth = collapsed ? 56 : 256;

  const handleLogout = useCallback(async () => {
    await logout();
  }, [logout]);

  return (
    <div className="flex h-screen overflow-hidden bg-slate-50">
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
        style={{ width: sidebarWidth, background: "#0f172a" }}
      >
        {/* Logo header */}
        <div className="flex h-12 shrink-0 items-center border-b border-slate-700/50 px-4">
          <div className="flex items-center gap-2.5">
            <div className="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-indigo-600 text-xs font-bold text-white">
              V
            </div>
            {!collapsed && (
              <span className="text-sm font-semibold text-white tracking-tight">
                VibeCMS
              </span>
            )}
          </div>
          {!collapsed && (
            <button
              onClick={() => setCollapsed(true)}
              className="ml-auto hidden h-6 w-6 items-center justify-center rounded text-slate-400 hover:bg-slate-800 hover:text-white lg:grid"
              title="Collapse sidebar"
            >
              <ChevronRight size={14} />
            </button>
          )}
          <button
            className="ml-auto rounded p-1 text-slate-400 hover:text-white lg:hidden"
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
        <div className="shrink-0 border-t border-slate-700/50 p-2">
          {collapsed ? (
            <button
              onClick={handleLogout}
              className="flex w-full items-center justify-center rounded-lg py-2 text-slate-400 hover:bg-slate-800 hover:text-white"
              title="Log out"
            >
              <LogOut size={15} />
            </button>
          ) : (
            <div className="flex items-center gap-2.5 rounded-lg px-3 py-2">
              <div className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-indigo-500/20 text-xs font-semibold text-indigo-400">
                {(user?.full_name || user?.email || "A")
                  .charAt(0)
                  .toUpperCase()}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-medium text-slate-300">
                  {user?.full_name || "Admin"}
                </p>
                <p className="truncate text-[10px] text-slate-500">
                  {user?.email}
                </p>
              </div>
              <button
                onClick={handleLogout}
                className="shrink-0 rounded p-1 text-slate-500 hover:bg-slate-800 hover:text-red-400"
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
        <header className="flex h-12 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4">
          <div className="flex items-center gap-2">
            {/* Mobile hamburger */}
            <button
              className="flex h-8 w-8 items-center justify-center rounded-md text-slate-500 hover:bg-slate-100 hover:text-slate-700 lg:hidden"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu size={18} />
            </button>

            {/* Breadcrumbs */}
            <nav className="flex items-center gap-1 text-xs text-slate-500">
              <Home size={12} className="text-slate-400" />
              {breadcrumbs.map((crumb, i) => {
                const last = i === breadcrumbs.length - 1;
                return (
                  <span key={i} className="flex items-center gap-1">
                    <ChevronRight size={10} className="text-slate-300" />
                    <span
                      className={`rounded px-1 py-0.5 ${
                        last ? "font-medium text-slate-800" : "text-slate-500"
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
            {/* Visit site */}
            <button
              onClick={() => window.open("/", "_blank")}
              className="hidden h-7 items-center gap-1.5 rounded-md border border-slate-200 px-2.5 text-xs font-medium text-slate-600 transition-colors hover:bg-slate-50 sm:flex"
            >
              <ExternalLink size={12} />
              Visit Site
            </button>

            {/* Notifications placeholder */}
            <button className="relative flex h-7 w-7 items-center justify-center rounded-md text-slate-400 hover:bg-slate-100 hover:text-slate-600">
              <Bell size={15} />
            </button>

            {/* User avatar */}
            <button
              className="flex h-7 w-7 items-center justify-center rounded-full bg-indigo-100 text-xs font-semibold text-indigo-600"
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
