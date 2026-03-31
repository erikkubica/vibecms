import { useState, useEffect, useRef, useCallback } from "react";
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
  FileText,
  Search,
  Loader2,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { getNodes, type MenuItem, type ContentNode } from "@/api/client";

function NodeSearchInput({
  value,
  onChange,
}: {
  value: number | null;
  onChange: (nodeId: number | null, title?: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<ContentNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [selectedLabel, setSelectedLabel] = useState<string>("");
  const wrapperRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Load initial label for existing node_id
  useEffect(() => {
    if (value && !selectedLabel) {
      getNodes({ search: "", per_page: 100 })
        .then((res) => {
          const node = res.data.find((n) => n.id === value);
          if (node) setSelectedLabel(`${node.title} (/${node.slug})`);
          else setSelectedLabel(`Node #${value}`);
        })
        .catch(() => setSelectedLabel(`Node #${value}`));
    }
  }, [value]);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const doSearch = useCallback((q: string) => {
    setLoading(true);
    getNodes({ search: q, per_page: 20 })
      .then((res) => setResults(res.data))
      .catch(() => setResults([]))
      .finally(() => setLoading(false));
  }, []);

  function handleInput(val: string) {
    setQuery(val);
    setOpen(true);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(val), 250);
  }

  function handleSelect(node: ContentNode) {
    onChange(node.id, node.title);
    setSelectedLabel(`${node.title} (/${node.slug})`);
    setQuery("");
    setOpen(false);
  }

  function handleClear() {
    onChange(null);
    setSelectedLabel("");
    setQuery("");
  }

  if (value && selectedLabel) {
    return (
      <div className="flex items-center gap-1 h-9 rounded-md border border-slate-200 bg-slate-50 px-3 text-sm">
        <FileText className="h-3.5 w-3.5 text-slate-400 shrink-0" />
        <span className="flex-1 truncate text-slate-700">{selectedLabel}</span>
        <button type="button" onClick={handleClear} className="text-slate-400 hover:text-red-500">
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
    );
  }

  return (
    <div ref={wrapperRef} className="relative">
      <div className="relative">
        <Search className="absolute left-2.5 top-2 h-4 w-4 text-slate-400" />
        <Input
          value={query}
          onChange={(e) => handleInput(e.target.value)}
          onFocus={() => { if (!open) { setOpen(true); doSearch(query); } }}
          placeholder="Search pages..."
          className="h-9 pl-8"
        />
        {loading && <Loader2 className="absolute right-2.5 top-2 h-4 w-4 animate-spin text-slate-400" />}
      </div>
      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-slate-200 bg-white shadow-lg max-h-48 overflow-y-auto">
          {results.length === 0 && !loading && (
            <p className="px-3 py-2 text-xs text-slate-400">
              {query ? "No results found" : "Type to search pages"}
            </p>
          )}
          {results.map((node) => (
            <button
              key={node.id}
              type="button"
              onClick={() => handleSelect(node)}
              className="flex items-center gap-2 w-full px-3 py-2 text-left text-sm hover:bg-indigo-50 transition-colors"
            >
              <FileText className="h-3.5 w-3.5 text-slate-400 shrink-0" />
              <span className="font-medium text-slate-800 truncate">{node.title}</span>
              <span className="text-xs text-slate-400 truncate">/{node.slug}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

const MAX_DEPTH = 3;

interface MenuTreeProps {
  items: MenuItem[];
  onChange: (items: MenuItem[]) => void;
  autoEditId?: string | null;
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


// Ensure every item has a stable _uid for tracking UI state
function ensureUids(items: MenuItem[]): MenuItem[] {
  for (const item of items) {
    if (!(item as Record<string, unknown>)._uid) {
      (item as Record<string, unknown>)._uid = generateTempId();
    }
    if (item.children && item.children.length > 0) {
      ensureUids(item.children);
    }
  }
  return items;
}

function getItemUid(item: MenuItem): string {
  return ((item as Record<string, unknown>)._uid as string) || "";
}

export default function MenuTree({ items, onChange, autoEditId }: MenuTreeProps) {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [editingId, setEditingId] = useState<string | null>(null);

  // Ensure all items have stable UIDs
  useEffect(() => {
    const hasAllUids = flattenItems(items).every((fi) => !!(fi.item as Record<string, unknown>)._uid);
    if (!hasAllUids) {
      const next = cloneItems(items);
      ensureUids(next);
      onChange(next);
    }
  }, [items]);

  // Auto-open newly added items for editing
  useEffect(() => {
    if (autoEditId) {
      setEditingId(autoEditId);
    }
  }, [autoEditId]);

  const flat = flattenItems(items);

  function toggleExpanded(uid: string) {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(uid)) {
        next.delete(uid);
      } else {
        next.add(uid);
      }
      return next;
    });
  }

  function toggleEditing(uid: string) {
    setEditingId((prev) => (prev === uid ? null : uid));
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
    setEditingId(null);
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
            const uid = getItemUid(fi.item);
            const isEditing = editingId === uid;
            const isChildExpanded = hasChildren(fi.item) && expandedIds.has(uid);

            return (
              <div key={uid || fi.path.join("-")} className="bg-white">
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
                    onClick={() => toggleExpanded(uid)}
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
                    onClick={() => toggleEditing(uid)}
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
                            Page / Content Node
                          </label>
                          <NodeSearchInput
                            value={fi.item.node_id ?? null}
                            onChange={(nodeId, title) => {
                              const next = cloneItems(items);
                              const item = getItemAtPath(next, fi.path);
                              (item as unknown as Record<string, unknown>)["node_id"] = nodeId;
                              if (title && !item.title) {
                                item.title = title;
                              }
                              onChange(next);
                            }}
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
