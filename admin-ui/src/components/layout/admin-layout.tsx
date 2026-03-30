import { useState, useEffect } from "react";
import { Outlet, NavLink, useLocation } from "react-router-dom";
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
  ChevronRight,
  ChevronDown,
  User,
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
import { Separator } from "@/components/ui/separator";
import { useAuth } from "@/hooks/use-auth";
import { getNodeAccess } from "@/api/client";
import { useAdminLanguage } from "@/hooks/use-admin-language";
import { useExtensions } from "@/hooks/use-extensions";
import { getNodeTypes, type NodeType } from "@/api/client";

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

// Case-insensitive lookup: extensions can use "Mail", "mail", or "mail" and it works.
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

type NavEntry = NavItem | NavGroup;

function isNavGroup(entry: NavEntry): entry is NavGroup {
  return "children" in entry;
}

const staticNavTopBase: NavItem[] = [
  { to: "/admin/dashboard", label: "Dashboard", icon: LayoutDashboard },
];

// Pages/Posts entries with their node type slugs for access filtering.
const nodeTypeNavItems: (NavItem & { nodeType: string })[] = [
  { to: "/admin/pages", label: "Pages", icon: FileText, nodeType: "page" },
  { to: "/admin/posts", label: "Posts", icon: Newspaper, nodeType: "post" },
];

const staticNavBottom: NavEntry[] = [
  {
    label: "Appearance",
    icon: Palette,
    children: [
      { to: "/admin/themes", label: "Themes", icon: Palette },
      { to: "/admin/layouts", label: "Layouts", icon: PanelTop },
      { to: "/admin/layout-blocks", label: "Layout Blocks", icon: Component },
      { to: "/admin/menus", label: "Menus", icon: ListTree },
    ],
  },
  {
    label: "Content",
    icon: Boxes,
    children: [
      { to: "/admin/content-types", label: "Content Types", icon: Boxes },
      { to: "/admin/block-types", label: "Block Types", icon: Square },
      { to: "/admin/templates", label: "Templates", icon: LayoutTemplate },
    ],
  },
  { to: "/admin/extensions", label: "Extensions", icon: Puzzle },
  {
    label: "Users",
    icon: UsersIcon,
    children: [
      { to: "/admin/users", label: "Users", icon: UsersIcon },
      { to: "/admin/roles", label: "Roles", icon: Shield },
    ],
  },
  {
    label: "Settings",
    icon: Settings,
    children: [
      { to: "/admin/settings/site", label: "Site", icon: Settings },
      { to: "/admin/languages", label: "Languages", icon: Globe },
      { to: "/admin/settings/api", label: "API", icon: Settings, disabled: true },
      { to: "/admin/settings/ai", label: "AI", icon: Settings, disabled: true },
      { to: "/admin/settings/mcp", label: "MCP", icon: Settings, disabled: true },
    ],
  },
];

function getBreadcrumb(pathname: string): string[] {
  const parts = pathname.replace("/admin/", "").split("/").filter(Boolean);
  return parts.map((p) => p.charAt(0).toUpperCase() + p.slice(1));
}

export default function AdminLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [customTypes, setCustomTypes] = useState<NodeType[]>([]);
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});
  const { user, logout } = useAuth();
  const { languages: adminLangs, currentCode, currentLanguage, setCurrentCode } = useAdminLanguage();
  const location = useLocation();
  const breadcrumbs = getBreadcrumb(location.pathname);

  const toggleGroup = (label: string) => {
    setOpenGroups((prev) => ({ ...prev, [label]: !prev[label] }));
  };

  useEffect(() => {
    getNodeTypes()
      .then((types) => {
        setCustomTypes(types.filter((t) => t.slug !== "page" && t.slug !== "post"));
      })
      .catch(() => {});
  }, []);

  // Auto-expand groups whose children match current path
  useEffect(() => {
    const updates: Record<string, boolean> = {};
    for (const entry of staticNavBottom) {
      if (isNavGroup(entry)) {
        if (entry.children.some((child) => location.pathname.startsWith(child.to))) {
          updates[entry.label] = true;
        }
      }
    }
    if (Object.keys(updates).length > 0) {
      setOpenGroups((prev) => ({ ...prev, ...updates }));
    }
  }, [location.pathname]);

  // Filter node type nav items by access.
  const visibleNodeTypeItems = nodeTypeNavItems.filter(
    (item) => getNodeAccess(user, item.nodeType).access !== "none"
  );
  const staticNavTop: NavItem[] = [...staticNavTopBase, ...visibleNodeTypeItems];

  const customNavItems: NavItem[] = customTypes
    .filter((t) => getNodeAccess(user, t.slug).access !== "none")
    .map((t) => ({
      to: `/admin/content/${t.slug}`,
      label: t.label,
      icon: iconMap[t.icon] || FileText,
    }));

  const { menus: extensionMenus } = useExtensions();

  // Build extension nav groups
  const extensionNavEntries: NavEntry[] = extensionMenus.map((menu) => {
    if (menu.children && menu.children.length > 0) {
      // Group with children (e.g., Email → Templates, Rules, Logs, Settings)
      return {
        label: menu.label,
        icon: iconMap[menu.icon] || Puzzle,
        children: menu.children.map((child) => ({
          to: child.route.startsWith("/admin") ? child.route : `/admin/ext/${menu.slug}${child.route}`,
          label: child.label,
          icon: child.icon ? (iconMap[child.icon] || iconMap[menu.icon] || Puzzle) : (iconMap[menu.icon] || Puzzle),
        })),
      } as NavEntry;
    }
    // Direct nav item (e.g., Media → /admin/ext/media-manager/)
    return {
      to: `/admin/ext/${menu.slug}/`,
      label: menu.label,
      icon: iconMap[menu.icon] || Puzzle,
    } as NavEntry;
  });

  // Insert extension menus before the "Appearance" group
  const appearanceIdx = staticNavBottom.findIndex(
    (e) => "label" in e && e.label === "Appearance"
  );
  const bottomWithExtensions = [...staticNavBottom];
  if (appearanceIdx >= 0) {
    bottomWithExtensions.splice(appearanceIdx, 0, ...extensionNavEntries);
  } else {
    bottomWithExtensions.unshift(...extensionNavEntries);
  }

  const navEntries: NavEntry[] = [...staticNavTop, ...customNavItems, ...bottomWithExtensions];

  const sidebarWidth = collapsed ? "w-16" : "w-64";

  return (
    <div className="flex h-screen overflow-hidden bg-slate-50">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex flex-col bg-slate-800 text-white transition-all duration-200 lg:relative lg:z-auto ${sidebarWidth} ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        }`}
      >
        {/* Logo */}
        <div className="flex h-16 items-center justify-between px-4">
          {!collapsed && (
            <div className="flex items-center gap-2.5">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600">
                <span className="text-sm font-bold text-white">V</span>
              </div>
              <span className="text-lg font-bold tracking-tight text-white">VibeCMS</span>
            </div>
          )}
          {collapsed && (
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 mx-auto">
              <span className="text-sm font-bold text-white">V</span>
            </div>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="text-white hover:bg-slate-700/50 lg:flex hidden"
            onClick={() => setCollapsed(!collapsed)}
          >
            <Menu className="h-5 w-5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="text-white hover:bg-slate-700/50 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          >
            <X className="h-5 w-5" />
          </Button>
        </div>

        <Separator className="bg-slate-700" />

        {/* Nav */}
        <nav className="flex-1 space-y-1 px-2 py-4 overflow-y-auto">
          {navEntries.map((entry) =>
            isNavGroup(entry) ? (
              <div key={entry.label}>
                <button
                  onClick={() => toggleGroup(entry.label)}
                  className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-700/50 hover:text-white ${
                    collapsed ? "justify-center" : ""
                  }`}
                >
                  <entry.icon className="h-5 w-5 shrink-0" />
                  {!collapsed && (
                    <>
                      <span>{entry.label}</span>
                      <ChevronDown
                        className={`ml-auto h-4 w-4 transition-transform duration-200 ${
                          openGroups[entry.label] ? "rotate-0" : "-rotate-90"
                        }`}
                      />
                    </>
                  )}
                </button>
                {(openGroups[entry.label] || collapsed) && (
                  <div className={collapsed ? "space-y-1" : "space-y-0.5 ml-3 border-l border-slate-700 pl-2 mt-0.5 mb-1"}>
                    {entry.children.map((child) => (
                      <NavLink
                        key={child.to}
                        to={child.disabled ? "#" : child.to}
                        onClick={(e) => {
                          if (child.disabled) e.preventDefault();
                          else setSidebarOpen(false);
                        }}
                        className={({ isActive }) =>
                          `flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                            child.disabled
                              ? "cursor-not-allowed text-slate-500"
                              : isActive
                                ? "bg-slate-700/50 text-white"
                                : "text-slate-300 hover:bg-slate-700/50 hover:text-white"
                          } ${collapsed ? "justify-center" : ""}`
                        }
                      >
                        <child.icon className="h-4 w-4 shrink-0" />
                        {!collapsed && <span>{child.label}</span>}
                        {!collapsed && child.disabled && (
                          <span className="ml-auto rounded bg-slate-700 px-1.5 py-0.5 text-[10px] text-slate-400">
                            Soon
                          </span>
                        )}
                      </NavLink>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <NavLink
                key={entry.to}
                to={entry.disabled ? "#" : entry.to}
                onClick={(e) => {
                  if (entry.disabled) e.preventDefault();
                  else setSidebarOpen(false);
                }}
                className={({ isActive }) =>
                  `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    entry.disabled
                      ? "cursor-not-allowed text-slate-500"
                      : isActive
                        ? "bg-slate-700/50 text-white"
                        : "text-slate-300 hover:bg-slate-700/50 hover:text-white"
                  } ${collapsed ? "justify-center" : ""}`
                }
              >
                <entry.icon className="h-5 w-5 shrink-0" />
                {!collapsed && <span>{entry.label}</span>}
                {!collapsed && entry.disabled && (
                  <span className="ml-auto rounded bg-slate-700 px-1.5 py-0.5 text-[10px] text-slate-400">
                    Soon
                  </span>
                )}
              </NavLink>
            )
          )}
        </nav>

        {/* Bottom */}
        <div className="border-t border-slate-700 p-2">
          <button
            onClick={logout}
            className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-700/50 hover:text-white ${
              collapsed ? "justify-center" : ""
            }`}
          >
            <LogOut className="h-5 w-5 shrink-0" />
            {!collapsed && <span>Log out</span>}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex h-16 items-center justify-between border-b border-slate-200 bg-white px-4 lg:px-6">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              size="icon"
              className="lg:hidden"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu className="h-5 w-5" />
            </Button>
            <nav className="flex items-center gap-1 text-sm text-slate-500">
              <span>Admin</span>
              {breadcrumbs.map((crumb, i) => (
                <span key={i} className="flex items-center gap-1">
                  <ChevronRight className="h-3 w-3" />
                  <span
                    className={
                      i === breadcrumbs.length - 1
                        ? "font-medium text-slate-800"
                        : ""
                    }
                  >
                    {crumb}
                  </span>
                </span>
              ))}
            </nav>
          </div>

          <div className="flex items-center gap-3">
            {/* Language picker */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5 rounded-lg border-slate-200 text-sm font-medium text-slate-600 hover:bg-slate-50">
                  <Globe className="h-4 w-4 text-slate-400" />
                  {currentCode === "all" ? "All languages" : (
                    <>{currentLanguage?.flag} {currentLanguage?.name || currentCode}</>
                  )}
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
                    <span className="mr-2">{lang.flag}</span>
                    {lang.name}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-indigo-100 text-indigo-700">
                  <User className="h-4 w-4" />
                </div>
                <span className="hidden text-sm font-medium text-slate-700 sm:inline">
                  {user?.full_name || user?.email}
                </span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48 shadow-md">
              <div className="px-2 py-1.5 text-sm">
                <p className="font-medium">{user?.full_name}</p>
                <p className="text-slate-500">{user?.email}</p>
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={logout}
                className="text-red-600 focus:text-red-600"
              >
                <LogOut className="mr-2 h-4 w-4" />
                Log out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-4 lg:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
