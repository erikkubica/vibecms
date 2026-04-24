import { useSearchParams } from "react-router-dom";
import { SduiAdminShell } from "../sdui/admin-shell";
import { useLayout } from "../hooks/use-layout";
import { LayoutRenderer } from "../sdui/renderer";
import { getPageStore } from "../sdui/action-handler";

export function SduiTaxonomiesPage() {
  const [searchParams] = useSearchParams();
  const params: Record<string, string> = {};
  const page = searchParams.get("page");
  const tab = searchParams.get("tab");
  const search = searchParams.get("search");
  const sort = searchParams.get("sort");
  const order = searchParams.get("order");
  const per_page = searchParams.get("per_page");
  if (page) params.page = page;
  if (tab && tab !== "all") params.tab = tab;
  if (search) params.search = search;
  if (sort) params.sort = sort;
  if (order) params.order = order;
  if (per_page) params.per_page = per_page;

  const {
    data: layout,
    isLoading,
    isFetching,
    error,
  } = useLayout("taxonomies", params);
  const store = getPageStore("taxonomies");

  const showSpinner = isLoading && !layout;

  return (
    <SduiAdminShell>
      {showSpinner ? (
        <div className="flex h-64 items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
        </div>
      ) : error && !layout ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          <p className="font-medium">Failed to load taxonomies</p>
          <p className="mt-1 text-red-600">{error.message}</p>
        </div>
      ) : layout ? (
        <div
          className={isFetching ? "opacity-90 transition-opacity" : undefined}
        >
          <LayoutRenderer layout={layout} pageId="taxonomies" store={store} />
        </div>
      ) : null}
    </SduiAdminShell>
  );
}
