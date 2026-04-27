import { useEffect, useRef, useCallback } from "react";
import type { QueryKey } from "@tanstack/react-query";
import { queryClient } from "../sdui/query-client";
import { qk } from "../sdui/query-keys";
import { sseBus } from "../sdui/sse-bus";
import { toast } from "sonner";
import type { SSEEventData } from "../sdui/types";

/**
 * SSE → query-invalidation router.
 *
 * The backend emits a narrow set of typed events (see internal/sdui/types.go).
 * This hook maps each event onto the specific query keys it should invalidate,
 * so we never nuke more cache than necessary.
 *
 * Adding a new SSE event type: extend SSEEventData in sdui/types.ts, then add a
 * case below. Prefer `qk.*` over string-literal keys so the invalidation stays
 * aligned with how queries are registered elsewhere.
 */
function routeEvent(event: SSEEventData): QueryKey[] {
  switch (event.type) {
    case "NAV_STALE":
      // Sidebar + any layout that renders nav (= all of them).
      return [qk.boot(), ["layout"]];

    case "SETTING_CHANGED":
      // Settings live in boot (site title, etc.) and on the settings page.
      return [qk.settings(), qk.boot(), ["layout"]];

    case "ENTITY_CHANGED": {
      if (!event.entity) return [];
      const keys: QueryKey[] = [
        qk.list(event.entity),
        ["layout"], // any open list layout needs to refetch its data
      ];
      if (event.id !== undefined && event.id !== null) {
        keys.push(qk.entity(event.entity, event.id));
      }
      // These entities are embedded in the boot manifest.
      if (
        event.entity === "user" ||
        event.entity === "node_type" ||
        event.entity === "menu"
      ) {
        keys.push(qk.boot());
      }
      return keys;
    }

    case "UI_STALE":
      // Coarse fallback — should only fire for events we haven't mapped yet.
      return [qk.boot(), ["layout"]];

    case "CONNECTED":
    case "NOTIFY":
      return [];

    default:
      return [];
  }
}

function handleNotify(event: SSEEventData): void {
  const data = event.data as { message?: string; variant?: string } | undefined;
  if (!data?.message) return;
  if (data.variant === "error") toast.error(data.message);
  else if (data.variant === "warning") toast.warning(data.message);
  else toast.info(data.message);
}

export function useSSE(): void {
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const es = new EventSource("/admin/api/events", { withCredentials: true });
    eventSourceRef.current = es;

    es.addEventListener("message", (e) => {
      let event: SSEEventData;
      try {
        event = JSON.parse(e.data);
      } catch (err) {
        console.error("[SSE] Failed to parse event:", err);
        return;
      }

      if (event.type === "CONNECTED") {
        console.debug("[SSE] Connected");
        return;
      }

      if (event.type === "NOTIFY") {
        handleNotify(event);
        return;
      }

      const keys = routeEvent(event);
      if (keys.length > 0) {
        console.debug("[SSE]", event.type, event.entity ?? event.key ?? "", "→ invalidate", keys);
        for (const key of keys) {
          queryClient.invalidateQueries({ queryKey: key });
        }
      }
      sseBus.emit(event);
    });

    es.onerror = () => {
      console.debug("[SSE] Connection lost, reconnecting in 3s");
      es.close();
      reconnectTimeoutRef.current = setTimeout(connect, 3000);
    };
  }, []);

  useEffect(() => {
    connect();
    return () => {
      if (eventSourceRef.current) eventSourceRef.current.close();
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
    };
  }, [connect]);
}
