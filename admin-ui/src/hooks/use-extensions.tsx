import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import {
  fetchExtensionManifests,
  loadExtension,
  type ExtensionManifestEntry,
  type LoadedExtension,
  type AdminUIRoute,
  type AdminUIMenu,
} from "@/lib/extension-loader";

export interface ResolvedFieldType {
  type: string;
  label: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  group: string;
  Component: React.ComponentType<unknown>;
  supports?: string[];
  extensionSlug: string;
}

interface ExtensionsContextValue {
  manifests: ExtensionManifestEntry[];
  loaded: Map<string, LoadedExtension>;
  loading: boolean;
  getSlotExtensions: (
    slotName: string,
  ) => Array<{ slug: string; label: string; Component: React.ComponentType<unknown> }>;
  getFieldTypes: () => ResolvedFieldType[];
  getFieldComponent: (fieldType: string) => { Component: React.ComponentType<unknown>; extensionSlug: string } | null;
  routes: Array<AdminUIRoute & { slug: string }>;
  menus: Array<AdminUIMenu & { slug: string }>;
}

const ExtensionsContext = createContext<ExtensionsContextValue>({
  manifests: [],
  loaded: new Map(),
  loading: true,
  getSlotExtensions: () => [],
  getFieldTypes: () => [],
  getFieldComponent: () => null,
  routes: [],
  menus: [],
});

export function ExtensionsProvider({ children }: { children: ReactNode }) {
  const [manifests, setManifests] = useState<ExtensionManifestEntry[]>([]);
  const [loaded, setLoaded] = useState<Map<string, LoadedExtension>>(new Map());
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function init() {
      const entries = await fetchExtensionManifests();
      if (cancelled) return;
      setManifests(entries);

      const loadedMap = new Map<string, LoadedExtension>();
      await Promise.allSettled(
        entries.map(async (entry) => {
          const ext = await loadExtension(entry);
          if (ext) loadedMap.set(entry.slug, ext);
        }),
      );

      if (!cancelled) {
        setLoaded(loadedMap);
        setLoading(false);
      }
    }

    init();
    return () => { cancelled = true; };
  }, []);

  function getSlotExtensions(slotName: string) {
    const results: Array<{
      slug: string;
      label: string;
      Component: React.ComponentType<unknown>;
    }> = [];

    for (const [slug, ext] of loaded) {
      const adminUI = ext.entry.manifest.admin_ui;
      if (!adminUI?.slots?.[slotName]) continue;

      const slotDef = adminUI.slots[slotName];
      const Component = ext.module[slotDef.component];
      if (Component) {
        results.push({ slug, label: slotDef.label, Component });
      }
    }

    return results;
  }

  function getFieldTypes(): ResolvedFieldType[] {
    const results: ResolvedFieldType[] = [];
    const icons = (window as any).__VIBECMS_SHARED__?.icons || {};

    for (const [slug, ext] of loaded) {
      const fieldTypes = ext.entry.manifest.admin_ui?.field_types;
      if (!fieldTypes) continue;

      for (const ft of fieldTypes) {
        const Component = ext.module[ft.component];
        if (!Component) continue;

        const IconComponent = icons[ft.icon] || icons["Puzzle"];
        results.push({
          type: ft.type,
          label: ft.label,
          description: ft.description,
          icon: IconComponent,
          group: ft.group,
          Component,
          supports: ft.supports,
          extensionSlug: slug,
        });
      }
    }

    return results;
  }

  function getFieldComponent(fieldType: string): { Component: React.ComponentType<unknown>; extensionSlug: string } | null {
    for (const [slug, ext] of loaded) {
      const fieldTypes = ext.entry.manifest.admin_ui?.field_types;
      if (!fieldTypes) continue;

      for (const ft of fieldTypes) {
        const Component = ext.module[ft.component];
        if (!Component) continue;

        if (ft.type === fieldType || ft.supports?.includes(fieldType)) {
          return { Component, extensionSlug: slug };
        }
      }
    }
    return null;
  }

  const routes: Array<AdminUIRoute & { slug: string }> = [];
  const menus: Array<AdminUIMenu & { slug: string }> = [];

  for (const [slug, ext] of loaded) {
    const adminUI = ext.entry.manifest.admin_ui;
    if (!adminUI) continue;

    if (adminUI.routes) {
      for (const route of adminUI.routes) {
        routes.push({ ...route, slug });
      }
    }

    if (adminUI.menu) {
      menus.push({ ...adminUI.menu, slug });
    }
  }

  return (
    <ExtensionsContext.Provider
      value={{ manifests, loaded, loading, getSlotExtensions, getFieldTypes, getFieldComponent, routes, menus }}
    >
      {children}
    </ExtensionsContext.Provider>
  );
}

export function useExtensions() {
  return useContext(ExtensionsContext);
}
