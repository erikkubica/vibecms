import { SduiAdminShell } from "../sdui/admin-shell";
import { useLayout } from "../hooks/use-layout";
import { LayoutRenderer } from "../sdui/renderer";
import { getPageStore } from "../sdui/action-handler";

export function SduiContentTypesPage() {
  const {
    data: layout,
    isLoading,
    isFetching,
    error,
  } = useLayout("content-types");
  const store = getPageStore("content-types");

  // Only show full-page spinner on initial load (no data yet).
  // When refetching, keep the previous layout mounted so interactive
  // elements don't lose focus.
  const showSpinner = isLoading && !layout;

  return (
    <SduiAdminShell>
      {showSpinner ? (
        <div className="flex h-64 items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
        </div>
      ) : error && !layout ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          <p className="font-medium">Failed to load content types</p>
          <p className="mt-1 text-red-600">{error.message}</p>
        </div>
      ) : layout ? (
        <div
          className={isFetching ? "opacity-90 transition-opacity" : undefined}
        >
          <LayoutRenderer
            layout={layout}
            pageId="content-types"
            store={store}
          />
        </div>
      ) : null}
    </SduiAdminShell>
  );
}
