import { SduiAdminShell } from "../sdui/admin-shell";
import { useLayout } from "../hooks/use-layout";
import { LayoutRenderer } from "../sdui/renderer";
import { getPageStore } from "../sdui/action-handler";

// SduiSecuritySettingsPage renders the language-agnostic security settings
// surface (registration toggle, default registration role). Mirrors
// sdui-site-settings.tsx but pinned to a single SDUI slug — security
// settings don't slice into sub-pages today.
export function SduiSecuritySettingsPage() {
  const slug = "security-settings";
  const { data: layout, isLoading, isFetching, error } = useLayout(slug);
  const store = getPageStore(slug);

  const showSpinner = isLoading && !layout;

  return (
    <SduiAdminShell>
      {showSpinner ? (
        <div className="flex h-64 items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
        </div>
      ) : error && !layout ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          <p className="font-medium">Failed to load security settings</p>
          <p className="mt-1 text-red-600">{error.message}</p>
        </div>
      ) : layout ? (
        <div className={isFetching ? "opacity-90 transition-opacity" : undefined}>
          <LayoutRenderer layout={layout} pageId={slug} store={store} />
        </div>
      ) : null}
    </SduiAdminShell>
  );
}
