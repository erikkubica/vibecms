import { useQuery, keepPreviousData } from "@tanstack/react-query";
import type { LayoutNode } from "../sdui/types";
import { qk } from "../sdui/query-keys";

async function fetchLayout(
  page: string,
  params?: Record<string, string>,
): Promise<LayoutNode> {
  const searchParams = new URLSearchParams(params);
  const res = await fetch(`/admin/api/layout/${page}?${searchParams}`, {
    credentials: "include",
  });
  if (!res.ok) throw new Error(`Failed to fetch layout for ${page}`);
  const json = await res.json();
  return json.data;
}

export function useLayout(page: string, params?: Record<string, string>) {
  return useQuery({
    queryKey: qk.layout(page, params),
    queryFn: () => fetchLayout(page, params),
    enabled: !!page,
    placeholderData: keepPreviousData,
  });
}
