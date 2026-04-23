import type { ActionDef } from "./types";
import { queryClient } from "./query-client";
import { toast } from "sonner";

// Simple page store per rendered SDUI page
const pageStores = new Map<string, Map<string, unknown>>();

export function getPageStore(pageId: string): Map<string, unknown> {
  if (!pageStores.has(pageId)) {
    pageStores.set(pageId, new Map());
  }
  return pageStores.get(pageId)!;
}

export function clearPageStore(pageId: string) {
  pageStores.delete(pageId);
}

// Navigate function — set externally to avoid circular deps
let navigateFn: (to: string) => void;
export function setNavigate(fn: (to: string) => void) {
  navigateFn = fn;
}

// Resolve $event.xxx references in a single string value
function resolveActionField(
  value: string,
  context?: Record<string, unknown>,
): string {
  if (!context?.event || typeof context.event !== "object") return value;

  // Handle $event.xxx references
  if (value.startsWith("$event.")) {
    const path = value.slice(7); // strip "$event."
    const parts = path.split(".");
    let current: unknown = context.event;
    for (const p of parts) {
      if (current && typeof current === "object" && current !== null) {
        current = (current as Record<string, unknown>)[p];
      } else {
        return value;
      }
    }
    return current != null ? String(current) : value;
  }
  return value;
}

// Execute a single action
async function executeAction(
  action: ActionDef,
  context?: Record<string, unknown>,
): Promise<unknown> {
  switch (action.type) {
    case "CORE_API": {
      const method = action.method || "GET";
      const params = action.params || {};

      // Resolve $store.*, $params.*, $event.* references in params
      const resolved = resolveParams(params, context);

      // Map method to endpoint
      const endpoint = buildEndpoint(method, resolved);
      const init: RequestInit = {
        method: getHTTPMethod(method),
        credentials: "include",
        headers: { "Content-Type": "application/json" },
      };
      if (init.method !== "GET" && init.method !== "HEAD") {
        init.body = JSON.stringify(resolved);
      }

      const res = await fetch(`/admin/api${endpoint}`, init);
      if (!res.ok) {
        const err = await res
          .json()
          .catch(() => ({ error: { message: res.statusText } }));
        throw new Error(err.error?.message || `API error: ${res.status}`);
      }

      const json = await res.json();
      return json.data;
    }

    case "NAVIGATE": {
      const to = action.to ? resolveActionField(action.to, context) : undefined;
      if (to && navigateFn) {
        navigateFn(to);
      }
      return;
    }

    case "TOAST": {
      const variant = action.variant || "success";
      const message = action.message
        ? resolveActionField(action.message, context)
        : "Done";
      if (variant === "error") {
        toast.error(message);
      } else if (variant === "warning") {
        toast.warning(message);
      } else {
        toast.success(message);
      }
      return;
    }

    case "INVALIDATE": {
      if (action.keys) {
        for (const key of action.keys) {
          await queryClient.invalidateQueries({ queryKey: [key] });
        }
      } else {
        await queryClient.invalidateQueries();
      }
      return;
    }

    case "SET_STORE": {
      if (action.key && context?.pageId) {
        const store = getPageStore(context.pageId as string);
        store.set(action.key, action.value);
      }
      return;
    }

    case "CONFIRM": {
      return new Promise((resolve) => {
        const msg = action.message
          ? resolveActionField(action.message, context)
          : "Are you sure?";
        const ok = window.confirm(msg);
        resolve(ok);
      });
    }

    default:
      console.warn(`[SDUI] Unknown action type: ${action.type}`);
  }
}

// Execute an action (possibly a SEQUENCE)
export async function executeActionDef(
  action: ActionDef,
  context?: Record<string, unknown>,
): Promise<unknown> {
  if (action.type === "SEQUENCE" && action.steps) {
    let result: unknown;
    for (const step of action.steps) {
      result = await executeAction(step, context);
      // If CONFIRM step returns false, stop the sequence
      if (step.type === "CONFIRM" && result === false) {
        return false;
      }
    }
    return result;
  }

  return executeAction(action, context);
}

// Resolve $store.*, $params.*, $event.* references
function resolveParams(
  params: Record<string, unknown>,
  context?: Record<string, unknown>,
): Record<string, unknown> {
  const resolved: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(params)) {
    resolved[key] = resolveValue(value, context);
  }
  return resolved;
}

function resolveValue(
  value: unknown,
  context?: Record<string, unknown>,
): unknown {
  if (typeof value === "string" && value.startsWith("$")) {
    const parts = value.slice(1).split(".");
    const root = parts[0];
    const path = parts.slice(1);

    let current: unknown;
    switch (root) {
      case "store":
        current = context?.store;
        break;
      case "params":
        current = context?.params;
        break;
      case "event":
        current = context?.event;
        break;
      default:
        return value;
    }

    for (const p of path) {
      if (current && typeof current === "object" && current !== null) {
        current = (current as Record<string, unknown>)[p];
      } else {
        return value; // can't resolve, return as-is
      }
    }
    return current ?? value;
  }
  return value;
}

// Map SDUI method names to API endpoints
function buildEndpoint(
  method: string,
  params: Record<string, unknown>,
): string {
  const parts = method.split(":");
  if (parts.length === 2) {
    // e.g. "nodes:delete" → /nodes/{id}
    // e.g. "templates:detach" → /templates/{id}/detach
    const resource = parts[0];
    const action = parts[1];

    // Actions that map to a sub-path suffix (POST endpoints)
    const subPathActions = new Set([
      "detach",
      "reattach",
      "activate",
      "deactivate",
      "pull",
      "upload",
      "preview",
      "items",
    ]);

    const id = params["id"];
    const slug = params["slug"];

    const basePath = id
      ? `/${resource}/${id}`
      : slug
        ? `/${resource}/${slug}`
        : `/${resource}`;

    if (subPathActions.has(action)) {
      return `${basePath}/${action}`;
    }
    return basePath;
  }
  return `/${method}`;
}

function getHTTPMethod(sduiMethod: string): string {
  const parts = sduiMethod.split(":");
  const action = parts.length > 1 ? parts[1] : parts[0];
  switch (action) {
    case "create":
      return "POST";
    case "update":
      return "PATCH";
    case "delete":
      return "DELETE";
    case "detach":
      return "POST";
    default:
      return "GET";
  }
}
