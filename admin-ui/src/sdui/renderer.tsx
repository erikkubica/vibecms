import React, { Suspense, useCallback } from "react";
import type { LayoutNode } from "./types";
import { getComponent } from "./registry";
import { executeActionDef } from "./action-handler";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyComponent = React.ComponentType<any>;

interface RendererProps {
  node: LayoutNode;
  pageId?: string;
  params?: Record<string, string>;
  store?: Map<string, unknown>;
}

// Error fallback component
function VibeErrorCard({ type, error }: { type: string; error?: Error }) {
  return (
    <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
      <p className="font-medium">Failed to render: {type}</p>
      {error && <p className="mt-1 text-red-600">{error.message}</p>}
    </div>
  );
}

// Loading fallback
function VibeLoadingCard() {
  return (
    <div className="rounded-lg border bg-muted/50 p-4 animate-pulse">
      <div className="h-4 w-2/3 rounded bg-muted" />
    </div>
  );
}

// Resolve prop values that reference store/params
function resolveProps(
  props: Record<string, unknown> | undefined,
  params?: Record<string, string>,
  store?: Map<string, unknown>,
): Record<string, unknown> {
  if (!props) return {};

  const resolved: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(props)) {
    resolved[key] = resolvePropValue(value, params, store);
  }
  return resolved;
}

function resolvePropValue(
  value: unknown,
  params?: Record<string, string>,
  store?: Map<string, unknown>,
): unknown {
  if (typeof value === "string" && value.startsWith("$")) {
    const path = value.slice(1);
    const dotIndex = path.indexOf(".");
    if (dotIndex === -1) return value;

    const root = path.slice(0, dotIndex);
    const rest = path.slice(dotIndex + 1);

    switch (root) {
      case "params":
        return params?.[rest] ?? value;
      case "store":
        return store?.get(rest) ?? value;
      default:
        return value;
    }
  }
  return value;
}

// Lazy component loader for extensions
const extensionCache = new Map<string, AnyComponent>();

async function loadExtensionComponent(
  extensionSlug: string,
  componentName: string,
): Promise<AnyComponent | null> {
  const cacheKey = `${extensionSlug}:${componentName}`;
  if (extensionCache.has(cacheKey)) {
    return extensionCache.get(cacheKey)!;
  }

  const url = `/admin/api/extensions/${extensionSlug}/assets/index.js`;
  try {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const mod: any = await import(/* @vite-ignore */ url);
    const Component: AnyComponent | undefined = mod[componentName];
    if (Component) {
      extensionCache.set(cacheKey, Component);
      return Component;
    }
  } catch (err) {
    console.error(
      `[SDUI] Failed to load extension component ${extensionSlug}:${componentName}`,
      err,
    );
  }
  return null;
}

// RemoteComponent wrapper — loads extension components dynamically
function RemoteComponent({
  extension,
  component,
  context,
  ...rest
}: {
  extension: string;
  component: string;
  context?: Record<string, unknown>;
  [key: string]: unknown;
}) {
  const [Comp, setComp] = React.useState<AnyComponent | null>(null);
  const [error, setError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    loadExtensionComponent(extension, component)
      .then((c) => setComp(() => c))
      .catch(setError);
  }, [extension, component]);

  if (error) {
    return (
      <VibeErrorCard
        type={`RemoteComponent(${extension}/${component})`}
        error={error}
      />
    );
  }
  if (!Comp) {
    return <VibeLoadingCard />;
  }

  return <Comp context={context} {...rest} />;
}

// Main recursive renderer
export function RecursiveRenderer({
  node,
  pageId,
  params,
  store,
}: RendererProps) {
  const { type, props, children, actions } = node;

  const handleAction = useCallback(
    async (actionName: string, eventData?: unknown) => {
      const action = actions?.[actionName];
      if (!action) return;

      try {
        await executeActionDef(action, {
          pageId,
          params,
          store: store ? Object.fromEntries(store) : undefined,
          event: eventData,
        });
      } catch (err) {
        console.error("[SDUI] Action error:", err);
        const message = err instanceof Error ? err.message : "Action failed";
        const { toast } = await import("sonner");
        toast.error(message);
      }
    },
    [actions, pageId, params, store],
  );

  // Special type: RemoteComponent
  if (type === "RemoteComponent") {
    const resolved = resolveProps(props, params, store);
    return (
      <RemoteComponent
        extension={resolved.extension as string}
        component={resolved.component as string}
        context={resolved.context as Record<string, unknown> | undefined}
      />
    );
  }

  // Look up in component registry
  const Registered = getComponent(type) as AnyComponent | null;

  if (!Registered) {
    return (
      <VibeErrorCard
        type={type}
        error={new Error(`Unknown component type: "${type}"`)}
      />
    );
  }

  // Resolve props
  const resolvedProps = resolveProps(props, params, store);

  // Add action handlers as callbacks if actions are defined
  const actionProps: Record<string, unknown> = {};
  if (actions) {
    for (const actionName of Object.keys(actions)) {
      actionProps[actionName] = (eventData?: unknown) =>
        handleAction(actionName, eventData);
    }
  }

  // Render children
  const childElements = children?.map((child: LayoutNode, i: number) => (
    <RecursiveRenderer
      key={`${child.type}-${i}`}
      node={child}
      pageId={pageId}
      params={params}
      store={store}
    />
  ));

  return React.createElement(
    Registered,
    { ...resolvedProps, ...actionProps },
    childElements?.length ? childElements : undefined,
  );
}

// Top-level layout renderer with Suspense boundary
export function LayoutRenderer({
  layout,
  pageId,
  params,
  store,
}: {
  layout: LayoutNode;
  pageId?: string;
  params?: Record<string, string>;
  store?: Map<string, unknown>;
}) {
  return (
    <Suspense fallback={<VibeLoadingCard />}>
      <RecursiveRenderer
        node={layout}
        pageId={pageId}
        params={params}
        store={store}
      />
    </Suspense>
  );
}
