import type { QueryKey } from "@tanstack/react-query";

/**
 * Central query-key factory. Every TanStack Query in the admin UI — and every
 * extension admin UI — should produce its key through this factory, so the
 * SSE → invalidation router in use-sse.ts can reliably target caches.
 *
 * Shapes:
 *   ['boot']
 *   ['layout', page, params]
 *   ['list', entity, filters]
 *   ['entity', entity, id]
 *   ['settings']
 *   ['ext', slug, ...rest]      — extension-owned sub-keys
 *
 * Prefix-match is TanStack's default for invalidateQueries, so invalidating
 * ['list', 'user'] also invalidates ['list', 'user', { search: 'a' }] etc.
 */
export const qk = {
  boot: (): QueryKey => ["boot"] as const,

  layout: (page: string, params?: Record<string, string>): QueryKey =>
    params && Object.keys(params).length > 0
      ? (["layout", page, params] as const)
      : (["layout", page] as const),

  list: (entity: string, filters?: Record<string, unknown>): QueryKey =>
    filters && Object.keys(filters).length > 0
      ? (["list", entity, filters] as const)
      : (["list", entity] as const),

  entity: (entity: string, id: string | number): QueryKey =>
    ["entity", entity, String(id)] as const,

  settings: (): QueryKey => ["settings"] as const,

  ext: (slug: string, ...rest: ReadonlyArray<string | number>): QueryKey =>
    ["ext", slug, ...rest] as const,
} as const;
