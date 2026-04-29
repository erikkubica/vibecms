import { useParams, useSearchParams } from "react-router-dom";
import { SduiAdminShell } from "../sdui/admin-shell";
import { useLayout } from "../hooks/use-layout";
import { LayoutRenderer } from "../sdui/renderer";
import { getPageStore } from "../sdui/action-handler";

export function SduiTaxonomyTermsPage() {
  const { nodeType, taxonomy } = useParams<{
    nodeType: string;
    taxonomy: string;
  }>();

  const [searchParams] = useSearchParams();
  const params: Record<string, string> = {
    nodeType: nodeType || "page",
    taxonomy: taxonomy || "",
  };
  const search = searchParams.get("search");
  const sort = searchParams.get("sort");
  const order = searchParams.get("order");
  const page = searchParams.get("page");
  const language = searchParams.get("language");
  if (search) params.search = search;
  if (sort) params.sort = sort;
  const per_page = searchParams.get("per_page");
  if (per_page) params.per_page = per_page;
  if (order) params.order = order;
  if (page) params.page = page;
  if (language) params.language = language;

  const {
    data: layout,
    isLoading,
    isFetching,
    error,
  } = useLayout("taxonomy-terms", params);
  const store = getPageStore(`taxonomy-terms-${nodeType}-${taxonomy}`);

  // Only show full-page spinner on initial load (no data yet).
  // When params change and we're refetching, keep the previous layout mounted
  // so interactive elements (search input) don't lose focus.
  const showSpinner = isLoading && !layout;

  return (
    <SduiAdminShell>
      {showSpinner ? (
        <div className="flex h-64 items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
        </div>
      ) : error && !layout ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          <p className="font-medium">Failed to load taxonomy terms</p>
          <p className="mt-1 text-red-600">{error.message}</p>
        </div>
      ) : layout ? (
        <div
          className={isFetching ? "opacity-90 transition-opacity" : undefined}
        >
          <LayoutRenderer
            layout={layout}
            pageId={`taxonomy-terms-${nodeType}-${taxonomy}`}
            params={params}
            store={store}
          />
        </div>
      ) : null}
    </SduiAdminShell>
  );
}
