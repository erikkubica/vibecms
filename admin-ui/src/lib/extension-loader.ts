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
