import type { SSEEventData } from "./types";

type Listener = (event: SSEEventData) => void;

/**
 * Lightweight pub-sub for components that need to react to SSE events
 * without converting their state to TanStack queries.
 *
 * Emitted from use-sse.ts. Consumers (e.g. useAuth) subscribe once on mount.
 */
class SseBus {
  private listeners = new Set<Listener>();

  emit(event: SSEEventData): void {
    for (const l of this.listeners) {
      try {
        l(event);
      } catch (err) {
        console.error("[SSE bus] listener threw:", err);
      }
    }
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }
}

export const sseBus = new SseBus();
