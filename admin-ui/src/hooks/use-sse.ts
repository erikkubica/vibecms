import { useEffect, useRef, useCallback } from "react";
import { queryClient } from "../sdui/query-client";
import { toast } from "sonner";
import type { SSEEventData } from "../sdui/types";

export function useSSE() {
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
      try {
        const event: SSEEventData = JSON.parse(e.data);

        switch (event.type) {
          case "CONNECTED":
            console.log("[SSE] Connected to event stream");
            break;

          case "UI_STALE":
            console.log("[SSE] UI state changed, invalidating queries");
            queryClient.invalidateQueries({ queryKey: ["boot"] });
            queryClient.invalidateQueries({ queryKey: ["layout"] });
            break;

          case "NODE_TYPE_CHANGED":
            console.log("[SSE] Node types changed, invalidating");
            queryClient.invalidateQueries({ queryKey: ["boot"] });
            queryClient.invalidateQueries({ queryKey: ["node-types"] });
            queryClient.invalidateQueries({ queryKey: ["nodes"] });
            break;

          case "NOTIFY":
            const data = event.data as { message?: string; variant?: string };
            if (data?.message) {
              if (data.variant === "error") {
                toast.error(data.message);
              } else {
                toast.info(data.message);
              }
            }
            break;
        }
      } catch (err) {
        console.error("[SSE] Failed to parse event:", err);
      }
    });

    es.onerror = () => {
      console.log("[SSE] Connection lost, reconnecting in 3s...");
      es.close();
      reconnectTimeoutRef.current = setTimeout(connect, 3000);
    };
  }, []);

  useEffect(() => {
    connect();
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [connect]);
}
