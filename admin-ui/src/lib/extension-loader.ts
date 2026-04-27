export interface AdminUISlot {
  component: string;
  label: string;
}

export interface AdminUIRoute {
  path: string;
  component: string;
}

export interface AdminUIMenuItem {
  label: string;
  route: string;
  icon?: string;
}

export interface AdminUIMenu {
  label: string;
  icon: string;
  position: string;
  section?: "content" | "design" | "development" | "settings";
  children: AdminUIMenuItem[];
}

export interface AdminUIFieldType {
  type: string;
  label: string;
  description: string;
  icon: string;
  group: string;
  component: string;
  supports?: string[];
}

export interface AdminUIManifest {
  entry: string;
  slots: Record<string, AdminUISlot>;
  routes: AdminUIRoute[];
  menu: AdminUIMenu | null;
  settings_menu?: AdminUIMenuItem[];
  field_types?: AdminUIFieldType[];
}

export interface ExtensionManifestEntry {
  slug: string;
  name: string;
  manifest: {
    admin_ui?: AdminUIManifest;
    provides?: string[];
    settings_schema?: Record<string, unknown>;
  };
}

export interface LoadedExtension {
  entry: ExtensionManifestEntry;
  module: Record<string, React.ComponentType<unknown>>;
}

const extensionCache = new Map<string, LoadedExtension>();

export async function fetchExtensionManifests(): Promise<ExtensionManifestEntry[]> {
  const res = await fetch("/admin/api/extensions/manifests", {
    credentials: "include",
  });
  if (!res.ok) return [];
  const json = await res.json();
  return json.data || [];
}

const injectedStylesheets = new Set<string>();

function injectExtensionStylesheet(slug: string, entry: string): void {
  const cleanEntry = entry.replace(/^admin-ui\/dist\//, "");
  // Sibling CSS file next to the JS entry — Vite lib build with cssFileName: "index"
  // emits <name>.css alongside <name>.js. Extensions that ship no CSS get a 404
  // which is harmless (the <link> just fails to load).
  const cssEntry = cleanEntry.replace(/\.(m?js)$/, ".css");
  if (cssEntry === cleanEntry) return;

  const href = `/admin/api/extensions/${encodeURIComponent(slug)}/assets/${cssEntry}`;
  if (injectedStylesheets.has(href)) return;
  injectedStylesheets.add(href);

  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = href;
  link.dataset.extensionSlug = slug;
  // Insert BEFORE the first <link rel="stylesheet"> in <head> so admin-ui's
  // own stylesheet stays later in source order and wins the cascade. Without
  // this, the extension's utility rules (e.g. `.fixed`) — which sit in the
  // same `@layer utilities` as admin-ui's — beat admin-ui's responsive
  // overrides like `lg:relative` and break the shell layout.
  const firstLink = document.head.querySelector('link[rel="stylesheet"]');
  if (firstLink) {
    document.head.insertBefore(link, firstLink);
  } else {
    // No admin-ui stylesheet found yet — prepend so any later link still wins.
    document.head.insertBefore(link, document.head.firstChild);
  }
}

export async function loadExtensionModule(
  slug: string,
  entry: string,
): Promise<Record<string, React.ComponentType<unknown>>> {
  const cleanEntry = entry.replace(/^admin-ui\/dist\//, "");
  const url = `/admin/api/extensions/${encodeURIComponent(slug)}/assets/${cleanEntry}`;

  // Validate the URL is a safe relative path (no protocol, no double dots).
  if (url.includes("..") || /^[a-z]+:/i.test(url)) {
    throw new Error(`Invalid extension entry path for ${slug}`);
  }

  injectExtensionStylesheet(slug, entry);

  try {
    const mod = await import(/* @vite-ignore */ url);
    return mod;
  } catch (err) {
    console.error(`[extensions] Failed to load module for ${slug}:`, err);
    throw err;
  }
}

export async function loadExtension(
  entry: ExtensionManifestEntry,
): Promise<LoadedExtension | null> {
  if (extensionCache.has(entry.slug)) {
    return extensionCache.get(entry.slug)!;
  }

  const adminUI = entry.manifest.admin_ui;
  if (!adminUI?.entry) return null;

  try {
    const module = await loadExtensionModule(entry.slug, adminUI.entry);
    const loaded: LoadedExtension = { entry, module };
    extensionCache.set(entry.slug, loaded);
    return loaded;
  } catch {
    return null;
  }
}

export function getExtensionCache(): Map<string, LoadedExtension> {
  return extensionCache;
}
