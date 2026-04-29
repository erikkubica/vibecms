import { useState, useEffect, useMemo } from "react";
import { Outlet, Link, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  FileText,
  Newspaper,
  Boxes,
  Square,
  LayoutTemplate,
  PanelTop,
  Palette,
  Component,
  ListTree,
  Globe,
  Image,
  Settings,
  LogOut,
  Menu,
  X,
  ChevronDown,
  Users as UsersIcon,
  Shield,
  ShoppingBag,
  Calendar,
  Users,
  Folder,
  Bookmark,
  Tag,
  Star,
  Heart,
  Puzzle,
  Mail,
  ScrollText,
  Gavel,
  Send,
  RefreshCw,
  ExternalLink,
  Key,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useAuth } from "@/hooks/use-auth";
import {
  getNodeAccess,
  clearCache,
  getNodeTypes,
  getTaxonomies,
  type NodeType,
  type Taxonomy,
} from "@/api/client";
import { toast } from "sonner";
import { useAdminLanguage } from "@/hooks/use-admin-language";
import { useExtensions } from "@/hooks/use-extensions";
import { PageMetaProvider, usePageMetaContext } from "./page-meta";

const iconMapRaw: Record<string, LucideIcon> = {
  "file-text": FileText,
  "newspaper": Newspaper,
  "shopping-bag": ShoppingBag,
  "calendar": Calendar,
  "users": Users,
  "folder": Folder,
  "bookmark": Bookmark,
  "tag": Tag,
  "star": Star,
  "heart": Heart,
  "boxes": Boxes,
  "image": Image,
  "mail": Mail,
  "layout-template": LayoutTemplate,
  "gavel": Gavel,
  "scroll-text": ScrollText,
  "settings": Settings,
  "send": Send,
  "puzzle": Puzzle,
};

const iconMap = new Proxy(iconMapRaw, {
  get(target, prop: string) {
    return target[prop] || target[prop.toLowerCase()] || target[prop.replace(/([a-z])([A-Z])/g, "$1-$2").toLowerCase()];
  },
});

interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  disabled?: boolean;
}

interface NavGroup {
  label: string;
  icon: LucideIcon;
  children: NavItem[];
}

interface NavSection {
  section: string;
}

type NavEntry = NavItem | NavGroup | NavSection;

function isNavGroup(entry: NavEntry): entry is NavGroup {
  return "children" in entry;
}
function isNavSection(entry: NavEntry): entry is NavSection {
  return "section" in entry;
}

const prettify = (s: string) =>
  s.replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

function computeAutoBreadcrumbs(
  pathname: string,
  customTypes: NodeType[],
): string[] {
  const parts = pathname.replace(/^\/admin\/?/, "").split("/").filter(Boolean);
  if (parts.length === 0) return [];

  // Known top-level sections with nicer labels.
  const sectionLabels: Record<string, string> = {
    dashboard: "Dashboard",
    pages: "Pages",
    posts: "Posts",
    content: "Content",
    "content-types": "Content Types",
    taxonomies: "Taxonomies",
    "block-types": "Block Types",
    templates: "Templates",
    layouts: "Layouts",
    "layout-blocks": "Layout Partials",
    menus: "Menus",
    themes: "Themes",
    extensions: "Extensions",
    users: "Users",
    roles: "Roles",
    languages: "Languages",
    settings: "Settings",
    "mcp-tokens": "MCP Tokens",
    ext: "Extensions",
  };

  const crumbs: string[] = [];
  const head = parts[0];

  // /admin/content/<slug>[/...] — map slug to node type label_plural
  if (head === "content" && parts[1]) {
    const slug = parts[1];
    const nt = customTypes.find((t) => t.slug === slug);
    crumbs.push(nt?.label_plural || nt?.label || prettify(slug));
    // /admin/content/<slug>/taxonomies/<tax>
    if (parts[2] === "taxonomies" && parts[3]) {
      crumbs.push(prettify(parts[3]));
    } else if (parts[2] === "new") {
      crumbs.push("New");
    } else if (parts[2] && parts[3] === "edit") {
      crumbs.push("Edit");
    }
    return crumbs;
  }

  // /admin/pages or /admin/posts — treat as node-type listings
  if (head === "pages" || head === "posts") {
    crumbs.push(sectionLabels[head]);
    if (parts[1] === "new") crumbs.push("New");
    else if (parts[1] && parts[2] === "edit") crumbs.push("Edit");
    return crumbs;
  }

  // Default: map each segment via sectionLabels or prettify
  for (let i = 0; i < parts.length; i++) {
    const seg = parts[i];
    // Skip numeric ID segments unless followed by nothing (don't show raw ids).
    if (/^\d+$/.test(seg)) continue;
    if (seg === "edit") {
      crumbs.push("Edit");
      continue;
    }
    if (seg === "new") {
      crumbs.push("New");
      continue;
    }
    crumbs.push(sectionLabels[seg] || prettify(seg));
  }
  return crumbs;
}

export default function AdminLayout() {
  return (
    <PageMetaProvider>
      <AdminLayoutInner />
    </PageMetaProvider>
  );
}

function AdminLayoutInner() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [customTypes, setCustomTypes] = useState<NodeType[]>([]);
  const [taxonomies, setTaxonomies] = useState<Taxonomy[]>([]);
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});
  const [clearingCache, setClearingCache] = useState(false);
  const { user, logout } = useAuth();
  const { languages: adminLangs, currentCode, currentLanguage, setCurrentCode } = useAdminLanguage();
  const { menus: extensionMenus, settingsMenuItems } = useExtensions();
  const location = useLocation();
  const { crumbs: overrideCrumbs } = usePageMetaContext();

  const autoCrumbs = useMemo(
    () => computeAutoBreadcrumbs(location.pathname, customTypes),
    [location.pathname, customTypes],
  );
  const breadcrumbs =
    overrideCrumbs && overrideCrumbs.length > 0 ? overrideCrumbs : autoCrumbs;

  useEffect(() => {
    const last = breadcrumbs[breadcrumbs.length - 1];
    document.title = last ? `${last} — Squilla` : "Squilla Admin";
  }, [breadcrumbs]);

  useEffect(() => {
    getNodeTypes()
      .then((typesData) => {
        const types = Array.isArray(typesData) ? typesData : (typesData as any)?.data || [];
        setCustomTypes(types);
      })
      .catch((err) => console.error("Node types load failed:", err));

    getTaxonomies()
      .then((taxesData) => {
        const taxes = Array.isArray(taxesData) ? taxesData : (taxesData as any)?.data || [];
        setTaxonomies(taxes);
      })
      .catch((err) => console.error("Taxonomies load failed:", err));
  }, []);

  const navEntries = useMemo(() => {
    const topBase: NavEntry[] = [
      { to: "/admin/dashboard", label: "Dashboard", icon: LayoutDashboard },
    ];

    const typeItems: NavEntry[] = [...customTypes]
      .filter((t) => getNodeAccess(user, t.slug).access !== "none")
      .sort((a, b) => {
        const order = (s: string) => (s === "page" ? 0 : s === "post" ? 1 : 2);
        const ra = order(a.slug);
        const rb = order(b.slug);
        if (ra !== rb) return ra - rb;
        return (a.label_plural || a.label).localeCompare(b.label_plural || b.label);
      })
      .map((t) => {
        const baseTo = t.slug === 'page' ? '/admin/pages' : t.slug === 'post' ? '/admin/posts' : `/admin/content/${t.slug}`;
        const icon = t.slug === 'page' ? FileText : t.slug === 'post' ? Newspaper : (iconMap[t.icon] || Boxes);

        const typeTaxes = taxonomies.filter(tax =>
          Array.isArray(tax.node_types) && tax.node_types.includes(t.slug)
        );

        const plural = t.label_plural || t.label;

        if (typeTaxes.length > 0) {
          return {
            label: plural,
            icon,
            children: [
              { to: baseTo, label: plural, icon },
              ...typeTaxes.map(tax => ({
                to: `/admin/content/${t.slug}/taxonomies/${tax.slug}`,
                label: tax.label,
                icon: Tag,
              })),
            ],
          };
        }
        return { to: baseTo, label: plural, icon };
      });

    const makeExtEntry = (menu: (typeof extensionMenus)[number]): NavEntry => {
      if (menu.children && menu.children.length > 0) {
        return {
          label: menu.label,
          icon: iconMap[menu.icon] || Puzzle,
          children: menu.children.map((child) => ({
            to: child.route.startsWith("/admin") ? child.route : `/admin/ext/${menu.slug}${child.route}`,
            label: child.label,
            icon: child.icon ? (iconMap[child.icon] || iconMap[menu.icon] || Puzzle) : (iconMap[menu.icon] || Puzzle),
          })),
        };
      }
      return {
        to: `/admin/ext/${menu.slug}/`,
        label: menu.label,
        icon: iconMap[menu.icon] || Puzzle,
      };
    };

    const extBySection: Record<string, NavEntry[]> = {
      content: [],
      design: [],
      development: [],
      settings: [],
    };
    for (const menu of extensionMenus) {
      const section = (menu.section && extBySection[menu.section]) ? menu.section : "content";
      extBySection[section].push(makeExtEntry(menu));
    }

    const designEntries: NavEntry[] = [
      { to: "/admin/themes", label: "Themes", icon: Palette },
      { to: "/admin/layouts", label: "Layouts", icon: PanelTop },
      { to: "/admin/layout-blocks", label: "Layout Blocks", icon: Component },
      { to: "/admin/menus", label: "Menus", icon: ListTree },
    ];

    const developmentEntries: NavEntry[] = [
      { to: "/admin/content-types", label: "Content Types", icon: Boxes },
      { to: "/admin/taxonomies", label: "Taxonomies", icon: Tag },
      { to: "/admin/block-types", label: "Block Types", icon: Square },
      { to: "/admin/templates", label: "Templates", icon: LayoutTemplate },
      { to: "/admin/extensions", label: "Extensions", icon: Puzzle },
    ];

    const settingsEntries: NavEntry[] = [
      { to: "/admin/users", label: "Users", icon: UsersIcon },
      { to: "/admin/roles", label: "Roles", icon: Shield },
      { to: "/admin/languages", label: "Languages", icon: Globe },
      { to: "/admin/settings/site", label: "Site Settings", icon: Settings },
      { to: "/admin/mcp-tokens", label: "MCP Tokens", icon: Key },
      ...settingsMenuItems.map((item) => ({
        to: item.route,
        label: item.label,
        icon: iconMap[item.icon || ""] || Settings,
      })),
    ];

    return [
      ...topBase,
      { section: "Content" },
      ...typeItems,
      ...extBySection.content,
      { section: "Design" },
      ...designEntries,
      ...extBySection.design,
      { section: "Development" },
      ...developmentEntries,
      ...extBySection.development,
      { section: "Settings" },
      ...settingsEntries,
      ...extBySection.settings,
    ];
  }, [customTypes, taxonomies, user, extensionMenus, settingsMenuItems]);

  useEffect(() => {
    const updates: Record<string, boolean> = {};
    for (const entry of navEntries) {
      if (isNavGroup(entry)) {
        if (entry.children.some((child) => location.pathname.startsWith(child.to))) {
          updates[entry.label] = true;
        }
      }
    }
    if (Object.keys(updates).length > 0) {
      setOpenGroups((prev) => ({ ...prev, ...updates }));
    }
  }, [location.pathname, navEntries]);

  async function handleClearCache() {
    setClearingCache(true);
    try {
      await clearCache();
      toast.success("All caches cleared");
    } catch {
      toast.error("Failed to clear caches");
    } finally {
      setClearingCache(false);
    }
  }

  const toggleGroup = (label: string) => {
    setOpenGroups((prev) => ({ ...prev, [label]: !prev[label] }));
  };

  const sidebarWidth = collapsed ? 52 : 232;

  // Collect every navigable path so isPathActive can exclude more-specific
  // siblings across groups (fixes highlight bleed between e.g. Media listing
  // and Image Optimizer settings).
  const allNavPaths: string[] = [];
  for (const entry of navEntries) {
    if (isNavSection(entry)) continue;
    if (isNavGroup(entry)) {
      for (const c of entry.children) allNavPaths.push(c.to);
    } else {
      allNavPaths.push(entry.to);
    }
  }

  const itemBase =
    "flex w-full items-center gap-2 transition-colors relative whitespace-nowrap cursor-pointer";
  const itemRest =
    "text-[color:var(--sb-fg)] hover:bg-[color:var(--sb-hover)] hover:text-[color:var(--sb-fg-active)]";

  const itemStyle = (
    isActive: boolean,
    depth: number
  ): React.CSSProperties => ({
    fontSize: depth > 0 ? 12.5 : 13,
    borderRadius: "var(--radius)",
    padding: collapsed ? "9px 0" : depth > 0 ? "7px 8px" : "9px 10px",
    justifyContent: collapsed ? "center" : "flex-start",
    fontWeight: isActive ? 600 : 500,
  });

  const isPathActive = (to: string, excludePrefixes: string[] = []): boolean => {
    const p = location.pathname;
    if (p === to) return true;
    if (!p.startsWith(to.endsWith("/") ? to : to + "/")) return false;
    // If this path would also be matched by a more-specific sibling, don't claim active
    for (const ex of excludePrefixes) {
      if (ex === to) continue;
      if (p === ex || p.startsWith(ex.endsWith("/") ? ex : ex + "/")) return false;
    }
    return true;
  };

  const renderNavItem = (entry: NavItem, depth = 0, excludePrefixes: string[] = []) => {
    const isActive = !entry.disabled && isPathActive(entry.to, excludePrefixes);
    return (
      <Link
        key={entry.to}
        to={entry.disabled ? "#" : entry.to}
        onClick={(e) => {
          if (entry.disabled) e.preventDefault();
          else setSidebarOpen(false);
        }}
        title={collapsed ? entry.label : undefined}
        className={`${itemBase} ${
          entry.disabled
            ? "cursor-not-allowed text-[color:var(--sb-fg-muted)]"
            : isActive
              ? "text-[color:var(--sb-fg-active)]"
              : itemRest
        }`}
        style={{
          ...itemStyle(isActive, depth),
          background: isActive ? "var(--sb-active)" : undefined,
        }}
      >
        {isActive && !collapsed && (
          <span
            className="absolute left-0 top-1 bottom-1 w-[2px] rounded"
            style={{ background: "var(--sb-active-border)" }}
          />
        )}
        <entry.icon
          className="shrink-0"
          size={15}
          style={{
            color: isActive ? "var(--sb-fg-active)" : "var(--sb-fg-muted)",
          }}
        />
        {!collapsed && <span className="flex-1 truncate text-left">{entry.label}</span>}
        {!collapsed && entry.disabled && (
          <span className="ml-auto rounded bg-white/5 px-1.5 py-0.5 text-[10px] text-[color:var(--sb-fg-muted)]">
            Soon
          </span>
        )}
      </Link>
    );
  };

  return (
    <div className="flex h-screen overflow-hidden" style={{ background: "var(--app-bg)" }}>
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex flex-col transition-all duration-200 lg:relative lg:z-auto ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        }`}
        style={{
          width: sidebarWidth,
          background: "var(--sb-bg)",
          borderRight: "1px solid var(--sb-border)",
          color: "var(--sb-fg)",
        }}
      >
        {/* Logo header */}
        <div
          className="flex h-11 items-center shrink-0"
          style={{
            padding: collapsed ? 0 : "0 10px",
            justifyContent: collapsed ? "center" : "space-between",
            borderBottom: "1px solid var(--sb-border)",
          }}
        >
          <div className="flex items-center gap-2">
            <div
              className="grid place-items-center shrink-0"
              style={{
                width: 22,
                height: 22,
                borderRadius: "var(--radius-sm)",
                background: "var(--sb-logo-bg)",
                border: "1px solid color-mix(in oklab, var(--accent) 40%, transparent)",
                color: "var(--accent)",
                fontSize: 12,
                fontWeight: 700,
                fontFamily: "var(--font-mono)",
              }}
            >
              V
            </div>
            {!collapsed && (
              <span
                className="font-semibold"
                style={{ fontSize: 13, color: "var(--sb-fg-active)", letterSpacing: "-0.01em" }}
              >
                Squilla
              </span>
            )}
          </div>
          {!collapsed && (
            <button
              onClick={() => setCollapsed(true)}
              className="grid place-items-center hidden lg:grid rounded"
              style={{ width: 22, height: 22, color: "var(--sb-fg-muted)", background: "transparent" }}
              title="Collapse"
            >
              <Menu size={14} />
            </button>
          )}
          <button
            className="lg:hidden rounded p-1"
            style={{ color: "var(--sb-fg-muted)" }}
            onClick={() => setSidebarOpen(false)}
          >
            <X size={16} />
          </button>
        </div>

        {/* Nav */}
        <nav
          className="flex-1 overflow-y-auto sb-scroll"
          style={{ padding: "6px 6px 0" }}
        >
          {collapsed && (
            <div className="flex justify-center mb-1">
              <button
                onClick={() => setCollapsed(false)}
                className="grid place-items-center rounded"
                style={{ width: 30, height: 30, color: "var(--sb-fg-muted)" }}
                title="Expand"
              >
                <Menu size={14} />
              </button>
            </div>
          )}

          {navEntries.map((entry, idx) => {
            if (isNavSection(entry)) {
              if (collapsed) {
                return (
                  <div
                    key={`sec-${idx}`}
                    className="h-px my-1.5 mx-2"
                    style={{ background: "var(--sb-border)" }}
                  />
                );
              }
              return (
                <div
                  key={`sec-${idx}`}
                  className="uppercase"
                  style={{
                    padding: "10px 10px 4px",
                    fontSize: 10.5,
                    fontWeight: 600,
                    color: "var(--sb-fg-muted)",
                    letterSpacing: "0.06em",
                  }}
                >
                  {entry.section}
                </div>
              );
            }
            if (isNavGroup(entry)) {
              const isOpen = openGroups[entry.label] || collapsed;
              const Ico = entry.icon;
              return (
                <div key={entry.label}>
                  <button
                    onClick={() => toggleGroup(entry.label)}
                    className={`${itemBase} ${itemRest}`}
                    style={itemStyle(false, 0)}
                    title={collapsed ? entry.label : undefined}
                  >
                    <Ico size={15} style={{ color: "var(--sb-fg-muted)" }} className="shrink-0" />
                    {!collapsed && (
                      <>
                        <span className="flex-1 truncate text-left">{entry.label}</span>
                        <ChevronDown
                          size={12}
                          style={{
                            color: "var(--sb-fg-muted)",
                            transform: isOpen ? "rotate(0deg)" : "rotate(-90deg)",
                            transition: "transform .15s",
                          }}
                        />
                      </>
                    )}
                  </button>
                  {isOpen && (
                    <div
                      className={collapsed ? "space-y-[2px]" : "ml-3 border-l pl-2 mt-0.5 mb-1 space-y-[1px]"}
                      style={!collapsed ? { borderColor: "var(--sb-border)" } : undefined}
                    >
                      {entry.children.map((child) =>
                        renderNavItem(child, 1, allNavPaths)
                      )}
                    </div>
                  )}
                </div>
              );
            }
            return renderNavItem(entry as NavItem, 0, allNavPaths);
          })}
        </nav>

        {/* Footer */}
        <div style={{ borderTop: "1px solid var(--sb-border)", padding: 6 }}>
          <button
            onClick={logout}
            className={`${itemBase} ${itemRest}`}
            style={itemStyle(false, 0)}
            title={collapsed ? "Log out" : undefined}
          >
            <LogOut size={15} style={{ color: "var(--sb-fg-muted)" }} className="shrink-0" />
            {!collapsed && <span className="flex-1 truncate text-left">Log out</span>}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header
          className="flex items-center justify-between shrink-0"
          style={{
            height: 44,
            padding: "0 16px",
            background: "var(--card-bg)",
            borderBottom: "1px solid var(--border)",
          }}
        >
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              className="lg:hidden h-7 w-7"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu className="h-4 w-4" />
            </Button>
            <nav
              className="flex items-center gap-1.5"
              style={{ fontSize: 12.5, color: "var(--fg-muted)" }}
            >
              <span>Admin</span>
              {breadcrumbs.map((crumb, i) => {
                const last = i === breadcrumbs.length - 1;
                return (
                  <span key={i} className="flex items-center gap-1.5">
                    <span style={{ color: "var(--fg-subtle)" }}>/</span>
                    <span
                      className="px-1 py-0.5 rounded"
                      style={{
                        color: last ? "var(--fg)" : "var(--fg-muted)",
                        fontWeight: last ? 500 : 400,
                      }}
                    >
                      {crumb}
                    </span>
                  </span>
                );
              })}
            </nav>
          </div>

          <div className="flex items-center gap-1.5">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-[12px]"
              onClick={handleClearCache}
              disabled={clearingCache}
              style={{ color: "var(--fg-2)" }}
            >
              <RefreshCw className={`mr-1 h-3.5 w-3.5 ${clearingCache ? "animate-spin" : ""}`} />
              {clearingCache ? "Clearing..." : "Clear Cache"}
            </Button>

            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-[12px]"
              asChild
              style={{ color: "var(--fg-2)" }}
            >
              <a href="/" target="_blank" rel="noopener noreferrer">
                <ExternalLink className="mr-1 h-3.5 w-3.5" />
                Visit Site
              </a>
            </Button>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 gap-1 text-[12px]"
                  style={{ color: "var(--fg-2)" }}
                >
                  <Globe className="h-3.5 w-3.5" style={{ color: "var(--fg-muted)" }} />
                  {currentCode === "all" ? (
                    "All languages"
                  ) : (
                    <>
                      {currentLanguage?.flag} {currentLanguage?.name || currentCode}
                    </>
                  )}
                  <ChevronDown className="h-3 w-3 opacity-60" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48 shadow-md">
                <DropdownMenuItem
                  onClick={() => setCurrentCode("all")}
                  className={currentCode === "all" ? "bg-indigo-50 text-indigo-700" : ""}
                >
                  <Globe className="mr-2 h-4 w-4" />
                  All languages
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                {adminLangs.map((lang) => (
                  <DropdownMenuItem
                    key={lang.code}
                    onClick={() => setCurrentCode(lang.code)}
                    className={currentCode === lang.code ? "bg-indigo-50 text-indigo-700" : ""}
                  >
                    {lang.name}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>

            <div className="w-px h-5 mx-1" style={{ background: "var(--border)" }} />

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  className="flex items-center gap-1.5 rounded-[var(--radius)] px-[9px] py-[3px]"
                  style={{ border: "1px solid var(--border)", background: "var(--card-bg)" }}
                >
                  <div
                    className="grid place-items-center shrink-0"
                    style={{
                      width: 20,
                      height: 20,
                      borderRadius: "var(--radius-sm)",
                      background: "color-mix(in oklab, var(--accent) 15%, var(--sub-bg))",
                      color: "var(--accent)",
                      fontSize: 10,
                      fontWeight: 600,
                    }}
                  >
                    {(user?.full_name || user?.email || "A").charAt(0).toUpperCase()}
                  </div>
                  <span className="text-[12px] font-medium hidden sm:inline" style={{ color: "var(--fg-2)" }}>
                    {user?.full_name || user?.email || "Admin"}
                  </span>
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48 shadow-md">
                <div className="px-2 py-1.5 text-sm">
                  <p className="font-medium">{user?.full_name}</p>
                  <p className="text-slate-500">{user?.email}</p>
                </div>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={logout} className="text-red-600 focus:text-red-600">
                  <LogOut className="mr-2 h-4 w-4" />
                  Log out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto" style={{ padding: "18px 20px" }}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
