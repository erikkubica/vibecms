import { useEffect } from "react";
import type { QueryKey } from "@tanstack/react-query";
import { queryClient } from "../sdui/query-client";
import { qk } from "../sdui/query-keys";
import { sseBus } from "../sdui/sse-bus";
import { toast } from "sonner";
import type { SSEEventData } from "../sdui/types";

/**
 * SSE → query-invalidation router with cross-tab leader election.
 *
 * Browsers cap concurrent HTTP/1.1 connections per origin (~6). Each open
 * admin tab holding its own EventSource quickly exhausts that pool and
 * stalls every other request. To stay power-user-friendly (ctrl+clicking
 * many admin pages and hopping between them), we elect ONE leader tab
 * per origin via the Web Locks API. The leader holds the only SSE
 * connection and rebroadcasts events to peers via a BroadcastChannel.
 * When the leader closes, the next tab in the lock queue takes over.
 *
 * Result: exactly one EventSource per browser regardless of tab count.
 */

const LOCK_NAME = "squilla-sse-leader";
const CHANNEL_NAME = "squilla-sse";

function routeEvent(event: SSEEventData): QueryKey[] {
  switch (event.type) {
    case "NAV_STALE":
      return [qk.boot(), ["layout"]];

    case "SETTING_CHANGED":
      return [qk.settings(), qk.boot(), ["layout"]];

    case "ENTITY_CHANGED": {
      if (!event.entity) return [];
      const keys: QueryKey[] = [qk.list(event.entity), ["layout"]];
      if (event.id !== undefined && event.id !== null) {
        keys.push(qk.entity(event.entity, event.id));
      }
      if (
        event.entity === "user" ||
        event.entity === "node_type" ||
        event.entity === "menu" ||
        event.entity === "role"
      ) {
        keys.push(qk.boot());
      }
      return keys;
    }

    case "UI_STALE":
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

function dispatchEvent(event: SSEEventData): void {
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
}

export function useSSE(): void {
  useEffect(() => {
    const channel = new BroadcastChannel(CHANNEL_NAME);
    let cancelled = false;
    let releaseLock: (() => void) | null = null;
    let currentSource: EventSource | null = null;
    let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

    channel.addEventListener("message", (e) => {
      dispatchEvent(e.data as SSEEventData);
    });

    function openLeaderSource(): void {
      if (cancelled) return;
      const es = new EventSource("/admin/api/events", { withCredentials: true });
      currentSource = es;

      es.addEventListener("message", (e) => {
        let event: SSEEventData;
        try {
          event = JSON.parse(e.data);
        } catch (err) {
          console.error("[SSE] Failed to parse event:", err);
          return;
        }
        dispatchEvent(event);
        // Fan out to follower tabs. postMessage only reaches OTHER
        // BroadcastChannel listeners, so we won't re-receive our own emit.
        channel.postMessage(event);
      });

      es.onerror = () => {
        console.debug("[SSE] Connection lost, reconnecting in 3s");
        es.close();
        currentSource = null;
        if (cancelled) return;
        reconnectTimeout = setTimeout(openLeaderSource, 3000);
      };
    }

    // Fallback for environments without Web Locks (very old browsers,
    // some test runners). Behaves like the legacy per-tab connection.
    if (!("locks" in navigator) || typeof navigator.locks?.request !== "function") {
      console.warn("[SSE] navigator.locks unavailable — using per-tab connection");
      openLeaderSource();
    } else {
      navigator.locks.request(LOCK_NAME, async () => {
        if (cancelled) return;
        console.debug("[SSE] Became leader");
        openLeaderSource();
        // Hold the lock for the lifetime of this hook. Releasing the
        // lock returns from this callback, which lets the next queued
        // tab take leadership.
        await new Promise<void>((resolve) => {
          releaseLock = resolve;
        });
      });
    }

    return () => {
      cancelled = true;
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
      if (currentSource) currentSource.close();
      channel.close();
      if (releaseLock) releaseLock();
    };
  }, []);
}
