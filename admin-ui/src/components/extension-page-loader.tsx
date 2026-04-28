import { Suspense } from "react";
import { useParams, Routes, Route, Navigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useExtensions } from "@/hooks/use-extensions";
import { ExtensionErrorBoundary } from "@/components/extension-error-boundary";

export function ExtensionPageLoader() {
  const { slug } = useParams<{ slug: string }>();
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

  // Build nested React Router routes from the extension manifest so that
  // useParams() inside extension components can access :id and other params.
  return (
    <ExtensionErrorBoundary key={slug} extensionName={ext.entry.name}>
      <Routes>
        {adminUI.routes.map((route) => {
          const Component = ext.module[route.component];
          if (!Component) return null;
          // Normalize path: ensure it starts with /
          const routePath = route.path.startsWith("/") ? route.path.slice(1) : route.path;
          return (
            <Route
              key={route.path}
              path={routePath}
              element={
                <Suspense fallback={<div className="flex h-64 items-center justify-center"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div>}>
                  <Component />
                </Suspense>
              }
            />
          );
        })}
        {/* Fallback: redirect to the first route */}
        {adminUI.routes[0] && (
          <Route path="*" element={<Navigate to={adminUI.routes[0].path.startsWith("/") ? adminUI.routes[0].path.slice(1) : adminUI.routes[0].path} replace />} />
        )}
      </Routes>
    </ExtensionErrorBoundary>
  );
}
