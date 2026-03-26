import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  GripVertical,
  Trash2,
  ArrowUp,
  ArrowDown,
  ArrowRight,
  ArrowLeft,
  Globe,
  Link as LinkIcon,
  Hash,
  FileText,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import type { MenuItem } from "@/api/client";

const MAX_DEPTH = 3;

interface MenuTreeProps {
  items: MenuItem[];
  onChange: (items: MenuItem[]) => void;
}

function generateTempId(): string {
  return `temp_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}

function itemTypeBadge(type: MenuItem["item_type"]) {
  switch (type) {
    case "node":
      return (
        <Badge className="bg-blue-100 text-blue-700 hover:bg-blue-100 border-0 text-xs gap-1">
          <FileText className="h-3 w-3" />
          Page
        </Badge>
      );
    case "custom":
      return (
        <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100 border-0 text-xs gap-1">
          <Globe className="h-3 w-3" />
          Custom
        </Badge>
      );
  }
}

interface FlatItem {
  item: MenuItem;
  depth: number;
  path: number[]; // indices path to reach this item
  parentPath: number[];
  indexInParent: number;
  siblingCount: number;
}

function flattenItems(items: MenuItem[], depth: number = 0, parentPath: number[] = []): FlatItem[] {
  const result: FlatItem[] = [];
  items.forEach((item, idx) => {
    const path = [...parentPath, idx];
    result.push({
      item,
      depth,
      path,
      parentPath,
      indexInParent: idx,
      siblingCount: items.length,
    });
    if (item.children && item.children.length > 0) {
      result.push(...flattenItems(item.children, depth + 1, path));
    }
  });
  return result;
}

function cloneItems(items: MenuItem[]): MenuItem[] {
  return JSON.parse(JSON.stringify(items));
}

function getItemAtPath(items: MenuItem[], path: number[]): MenuItem {
  let current = items[path[0]];
  for (let i = 1; i < path.length; i++) {
    current = current.children![path[i]];
  }
  return current;
}


export default function MenuTree({ items, onChange }: MenuTreeProps) {
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [editingPath, setEditingPath] = useState<string | null>(null);

  const flat = flattenItems(items);

  function toggleExpanded(pathKey: string) {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(pathKey)) {
        next.delete(pathKey);
      } else {
        next.add(pathKey);
      }
      return next;
    });
  }

  function toggleEditing(pathKey: string) {
    setEditingPath((prev) => (prev === pathKey ? null : pathKey));
  }

  function updateItemField(path: number[], field: string, value: unknown) {
    const next = cloneItems(items);
    const item = getItemAtPath(next, path);
    (item as unknown as Record<string, unknown>)[field] = value;
    onChange(next);
  }

  function deleteItem(path: number[]) {
    const next = cloneItems(items);
    const parentPath = path.slice(0, -1);
    const idx = path[path.length - 1];
    const siblings = parentPath.length === 0 ? next : getItemAtPath(next, parentPath).children!;
    siblings.splice(idx, 1);
    onChange(next);
    setEditingPath(null);
  }

  function moveUp(path: number[]) {
    const idx = path[path.length - 1];
    if (idx === 0) return;
    const next = cloneItems(items);
    const parentPath = path.slice(0, -1);
    const siblings = parentPath.length === 0 ? next : getItemAtPath(next, parentPath).children!;
    [siblings[idx - 1], siblings[idx]] = [siblings[idx], siblings[idx - 1]];
    onChange(next);
  }

  function moveDown(path: number[]) {
    const idx = path[path.length - 1];
    const next = cloneItems(items);
    const parentPath = path.slice(0, -1);
    const siblings = parentPath.length === 0 ? next : getItemAtPath(next, parentPath).children!;
    if (idx >= siblings.length - 1) return;
    [siblings[idx], siblings[idx + 1]] = [siblings[idx + 1], siblings[idx]];
    onChange(next);
  }

  function indent(path: number[]) {
    const idx = path[path.length - 1];
    const depth = path.length - 1;
    if (idx === 0 || depth >= MAX_DEPTH - 1) return;
    const next = cloneItems(items);
    const parentPath = path.slice(0, -1);
    const siblings = parentPath.length === 0 ? next : getItemAtPath(next, parentPath).children!;
    const item = siblings.splice(idx, 1)[0];
    const prevSibling = siblings[idx - 1];
    if (!prevSibling.children) prevSibling.children = [];
    prevSibling.children.push(item);
    onChange(next);
    setEditingPath(null);
  }

  function outdent(path: number[]) {
    if (path.length <= 1) return; // already at root
    const next = cloneItems(items);
    const parentPath = path.slice(0, -1);
    const idx = path[path.length - 1];
    const parent = getItemAtPath(next, parentPath);
    const item = parent.children!.splice(idx, 1)[0];
    // Insert after parent in grandparent's children
    const grandParentPath = parentPath.slice(0, -1);
    const parentIdx = parentPath[parentPath.length - 1];
    const grandSiblings = grandParentPath.length === 0 ? next : getItemAtPath(next, grandParentPath).children!;
    grandSiblings.splice(parentIdx + 1, 0, item);
    onChange(next);
    setEditingPath(null);
  }

  function hasChildren(item: MenuItem): boolean {
    return !!(item.children && item.children.length > 0);
  }

  return (
    <div className="space-y-0">
      {flat.length === 0 ? (
        <div className="rounded-lg border border-dashed border-slate-300 p-8 text-center text-sm text-slate-400">
          No menu items yet. Add items using the buttons above.
        </div>
      ) : (
        <div className="rounded-lg border border-slate-200 overflow-hidden divide-y divide-slate-100">
          {flat.map((fi) => {
            const pathKey = fi.path.join("-");
            const isEditing = editingPath === pathKey;
            const isChildExpanded = hasChildren(fi.item) && expandedPaths.has(pathKey);

            return (
              <div key={pathKey} className="bg-white">
                {/* Collapsed row */}
                <div
                  className={`flex items-center gap-2 px-3 py-2.5 hover:bg-slate-50 transition-colors ${
                    isEditing ? "bg-indigo-50/50" : ""
                  }`}
                  style={{ paddingLeft: `${fi.depth * 24 + 12}px` }}
                >
                  {/* Expand toggle for children */}
                  <button
                    className="flex h-6 w-6 items-center justify-center rounded text-slate-400 hover:text-slate-600 hover:bg-slate-100 flex-shrink-0"
                    onClick={() => toggleExpanded(pathKey)}
                    title={hasChildren(fi.item) ? (isChildExpanded ? "Collapse" : "Expand") : "No children"}
                  >
                    {hasChildren(fi.item) ? (
                      isChildExpanded ? (
                        <ChevronDown className="h-4 w-4" />
                      ) : (
                        <ChevronRight className="h-4 w-4" />
                      )
                    ) : (
                      <GripVertical className="h-3.5 w-3.5 text-slate-300" />
                    )}
                  </button>

                  {/* Title */}
                  <span className="flex-1 text-sm font-medium text-slate-700 truncate">
                    {fi.item.title || "(untitled)"}
                  </span>

                  {/* Type badge */}
                  {itemTypeBadge(fi.item.item_type)}

                  {/* Edit toggle */}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2 text-xs"
                    onClick={() => toggleEditing(pathKey)}
                  >
                    {isEditing ? "Close" : "Edit"}
                  </Button>

                  {/* Delete */}
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-red-400 hover:text-red-600"
                    onClick={() => deleteItem(fi.path)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>

                {/* Expanded editing panel */}
                {isEditing && (
                  <div
                    className="border-t border-slate-100 bg-slate-50/70 px-4 py-4 space-y-3"
                    style={{ paddingLeft: `${fi.depth * 24 + 16}px` }}
                  >
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                      {/* Title */}
                      <div>
                        <label className="mb-1 block text-xs font-medium text-slate-600">
                          Title
                        </label>
                        <Input
                          value={fi.item.title}
                          onChange={(e) => updateItemField(fi.path, "title", e.target.value)}
                          placeholder="Menu item title"
                          className="h-9"
                        />
                      </div>

                      {/* Item Type */}
                      <div>
                        <label className="mb-1 block text-xs font-medium text-slate-600">
                          Type
                        </label>
                        <select
                          className="h-9 w-full rounded-md border border-slate-200 bg-white px-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20"
                          value={fi.item.item_type}
                          onChange={(e) => updateItemField(fi.path, "item_type", e.target.value)}
                        >
                          <option value="node">Page (Node)</option>
                          <option value="custom">Custom URL</option>
                        </select>
                      </div>

                      {/* Type-specific field */}
                      {fi.item.item_type === "custom" && (
                        <div>
                          <label className="mb-1 block text-xs font-medium text-slate-600">
                            URL
                          </label>
                          <div className="relative">
                            <LinkIcon className="absolute left-2.5 top-2 h-4 w-4 text-slate-400" />
                            <Input
                              value={fi.item.url || ""}
                              onChange={(e) => updateItemField(fi.path, "url", e.target.value)}
                              placeholder="https://example.com or #anchor"
                              className="h-9 pl-8"
                            />
                          </div>
                        </div>
                      )}

                      {fi.item.item_type === "node" && (
                        <div>
                          <label className="mb-1 block text-xs font-medium text-slate-600">
                            Node ID
                          </label>
                          <Input
                            type="number"
                            value={fi.item.node_id ?? ""}
                            onChange={(e) =>
                              updateItemField(
                                fi.path,
                                "node_id",
                                e.target.value ? Number(e.target.value) : null
                              )
                            }
                            placeholder="Enter node ID"
                            className="h-9"
                          />
                        </div>
                      )}

                      {/* Target */}
                      <div>
                        <label className="mb-1 block text-xs font-medium text-slate-600">
                          Target
                        </label>
                        <select
                          className="h-9 w-full rounded-md border border-slate-200 bg-white px-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20"
                          value={fi.item.target}
                          onChange={(e) => updateItemField(fi.path, "target", e.target.value)}
                        >
                          <option value="_self">Same Window (_self)</option>
                          <option value="_blank">New Window (_blank)</option>
                        </select>
                      </div>

                      {/* CSS Class */}
                      <div>
                        <label className="mb-1 block text-xs font-medium text-slate-600">
                          CSS Class
                        </label>
                        <Input
                          value={fi.item.css_class || ""}
                          onChange={(e) => updateItemField(fi.path, "css_class", e.target.value)}
                          placeholder="optional-class"
                          className="h-9"
                        />
                      </div>
                    </div>

                    {/* Move buttons */}
                    <div className="flex items-center gap-1.5 pt-1">
                      <span className="text-xs text-slate-500 mr-1">Move:</span>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 px-2 text-xs gap-1"
                        disabled={fi.indexInParent === 0}
                        onClick={() => moveUp(fi.path)}
                      >
                        <ArrowUp className="h-3 w-3" />
                        Up
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 px-2 text-xs gap-1"
                        disabled={fi.indexInParent >= fi.siblingCount - 1}
                        onClick={() => moveDown(fi.path)}
                      >
                        <ArrowDown className="h-3 w-3" />
                        Down
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 px-2 text-xs gap-1"
                        disabled={fi.indexInParent === 0 || fi.depth >= MAX_DEPTH - 1}
                        onClick={() => indent(fi.path)}
                      >
                        <ArrowRight className="h-3 w-3" />
                        Indent
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 px-2 text-xs gap-1"
                        disabled={fi.depth === 0}
                        onClick={() => outdent(fi.path)}
                      >
                        <ArrowLeft className="h-3 w-3" />
                        Outdent
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export { generateTempId };
export type { MenuTreeProps };
