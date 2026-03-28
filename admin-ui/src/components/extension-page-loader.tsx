import { Suspense } from "react";
import { useParams, useLocation } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useExtensions } from "@/hooks/use-extensions";
import { ExtensionErrorBoundary } from "@/components/extension-error-boundary";

export function ExtensionPageLoader() {
  const { slug } = useParams<{ slug: string }>();
  const location = useLocation();
  const { loaded, loading } = useExtensions();

  if (loading || !slug) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  const ext = loaded.get(slug);
  if (!ext) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-2 text-slate-400">
        <p className="text-lg font-medium">Extension not found</p>
        <p className="text-sm">The extension &quot;{slug}&quot; is not loaded.</p>
      </div>
    );
  }

  const adminUI = ext.entry.manifest.admin_ui;
  if (!adminUI?.routes) return null;

  // Extract the path after /admin/ext/:slug
  const basePath = `/admin/ext/${slug}`;
  let subPath = location.pathname.startsWith(basePath)
    ? location.pathname.slice(basePath.length)
    : "";
  // Ensure subPath starts with /
  if (!subPath.startsWith("/")) {
    subPath = "/" + subPath;
  }
  // Normalize: remove trailing slash (except for root /)
  if (subPath.length > 1 && subPath.endsWith("/")) {
    subPath = subPath.slice(0, -1);
  }

  // Find the matching route
  const matchedRoute = adminUI.routes.find((r) => {
    const routePath = r.path.replace(/^\/+/, "/"); // normalize
    // Convert :param patterns to regex
    const pattern = routePath.replace(/:\w+/g, "[^/]+");
    return new RegExp(`^${pattern}$`).test(subPath);
  });

  // Default to first route if no match
  const componentName = matchedRoute?.component || adminUI.routes[0]?.component;
  const Component = componentName ? ext.module[componentName] : null;
  if (!Component) return null;

  return (
    <ExtensionErrorBoundary extensionName={ext.entry.name}>
      <Suspense fallback={<div className="flex h-64 items-center justify-center"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div>}>
        <Component />
      </Suspense>
    </ExtensionErrorBoundary>
  );
}
