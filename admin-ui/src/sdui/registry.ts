// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyComponent = React.ComponentType<any>;

import type React from "react";

// Global component registry — maps type name strings to React components.
// The renderer looks up components by the "type" field in layout tree nodes.
const registry = new Map<string, AnyComponent>();

// Pre-register a component
export function registerComponent(name: string, component: AnyComponent) {
  registry.set(name, component);
}

// Get a component by type name
export function getComponent(name: string): AnyComponent | null {
  return registry.get(name) ?? null;
}

// Check if a component is registered
export function hasComponent(name: string): boolean {
  return registry.has(name);
}

// Get all registered component names
export function getRegisteredNames(): string[] {
  return Array.from(registry.keys());
}

// Batch register components
export function registerComponents(components: Record<string, AnyComponent>) {
  for (const [name, component] of Object.entries(components)) {
    registry.set(name, component);
  }
}
