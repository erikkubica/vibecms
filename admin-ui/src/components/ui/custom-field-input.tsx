import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import RichTextEditor from "@/components/ui/rich-text-editor";
import {
  X,
  Plus,
  ChevronUp,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import {
  searchNodes,
  getNodeTypes,
  type NodeTypeField,
  type NodeSearchResult,
  type NodeType,
} from "@/api/client";

// Link field: text, url, alt, open in new tab
function LinkFieldInput({
  value,
  onChange,
}: {
  value: Record<string, unknown> | null;
  onChange: (val: unknown) => void;
}) {
  const defaultLink = { text: "", url: "", alt: "", target: "" };
  const link = (value && typeof value === "object" && !Array.isArray(value) && "url" in value)
    ? value
    : defaultLink;
  const update = (key: string, val: unknown) => onChange({ ...link, [key]: val });
  const inputClass = "rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20";

  return (
    <div className="space-y-3 rounded-lg border border-slate-200 bg-slate-50 p-3">
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <Label className="text-xs font-medium text-slate-600">Link Text</Label>
          <Input value={String(link.text || "")} onChange={(e) => update("text", e.target.value)} placeholder="Click here" className={inputClass} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs font-medium text-slate-600">URL</Label>
          <Input value={String(link.url || "")} onChange={(e) => update("url", e.target.value)} placeholder="https://example.com" className={inputClass} />
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <Label className="text-xs font-medium text-slate-600">Alt / Title</Label>
          <Input value={String(link.alt || "")} onChange={(e) => update("alt", e.target.value)} placeholder="Link description" className={inputClass} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs font-medium text-slate-600">&nbsp;</Label>
          <div className="flex items-center gap-2 h-9">
            <input
              type="checkbox"
              checked={link.target === "_blank"}
              onChange={(e) => update("target", e.target.checked ? "_blank" : "")}
              className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            <span className="text-sm text-slate-600">Open in new tab</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// Group field: renders sub-fields as a nested form
function GroupFieldInput({
  field,
  value,
  onChange,
}: {
  field: NodeTypeField;
  value: Record<string, unknown> | null;
  onChange: (val: unknown) => void;
}) {
  const group = (value && typeof value === "object" && !Array.isArray(value)) ? value as Record<string, unknown> : {};
  const subFields = field.sub_fields || [];

  if (subFields.length === 0) {
    return <p className="text-sm text-slate-400 italic">No sub-fields defined for this group.</p>;
  }

  return (
    <div className="space-y-3 rounded-lg border border-slate-200 bg-slate-50 p-3">
      {subFields.map((sf) => (
        <div key={sf.key} className="space-y-1">
          <Label className="text-xs font-medium text-slate-600">
            {sf.label}
            {sf.required && <span className="ml-1 text-red-500">*</span>}
          </Label>
          <CustomFieldInput
            field={sf}
            value={group[sf.key]}
            onChange={(val) => onChange({ ...group, [sf.key]: val })}
          />
        </div>
      ))}
    </div>
  );
}

// Repeater field: add/remove rows of sub-fields
function getRowSummary(row: Record<string, unknown>, subFields: NodeTypeField[]): string {
  const parts: string[] = [];
  for (const sf of subFields) {
    const val = row[sf.key];
    if (val == null || val === "") continue;
    if (sf.type === "richtext" && typeof val === "string") {
      // Strip HTML tags for summary
      const text = val.replace(/<[^>]*>/g, "").trim();
      if (text) parts.push(text.length > 40 ? text.slice(0, 40) + "..." : text);
    } else if (sf.type === "toggle") {
      if (val) parts.push(sf.label);
    } else if (sf.type === "repeater" || sf.type === "group") {
      const count = Array.isArray(val) ? val.length : val ? 1 : 0;
      if (count > 0) parts.push(`${count} ${sf.label}`);
    } else if (typeof val === "string" || typeof val === "number") {
      const s = String(val);
      if (s) parts.push(s.length > 30 ? s.slice(0, 30) + "..." : s);
    }
    if (parts.length >= 3) break;
  }
  return parts.join(" / ") || "Empty row";
}

function RepeaterFieldInput({
  field,
  value,
  onChange,
}: {
  field: NodeTypeField;
  value: unknown[] | null;
  onChange: (val: unknown) => void;
}) {
  const rows = (Array.isArray(value) ? value : []) as Record<string, unknown>[];
  const subFields = field.sub_fields || [];
  const [collapsedRows, setCollapsedRows] = useState<Set<number>>(() => new Set());

  if (subFields.length === 0) {
    return <p className="text-sm text-slate-400 italic">No sub-fields defined for this repeater.</p>;
  }

  function addRow() {
    const emptyRow: Record<string, unknown> = {};
    subFields.forEach((sf) => { emptyRow[sf.key] = sf.default_value || ""; });
    onChange([...rows, emptyRow]);
  }

  function removeRow(index: number) {
    onChange(rows.filter((_, i) => i !== index));
    setCollapsedRows((prev) => {
      const next = new Set<number>();
      prev.forEach((i) => {
        if (i < index) next.add(i);
        else if (i > index) next.add(i - 1);
      });
      return next;
    });
  }

  function updateRow(index: number, key: string, val: unknown) {
    const updated = rows.map((row, i) =>
      i === index ? { ...row, [key]: val } : row
    );
    onChange(updated);
  }

  function moveRow(index: number, direction: "up" | "down") {
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= rows.length) return;
    const newRows = [...rows];
    [newRows[index], newRows[targetIndex]] = [newRows[targetIndex], newRows[index]];
    onChange(newRows);
    // Swap collapsed state along with the rows
    setCollapsedRows((prev) => {
      const next = new Set<number>();
      for (const i of prev) {
        if (i === index) next.add(targetIndex);
        else if (i === targetIndex) next.add(index);
        else next.add(i);
      }
      return next;
    });
  }

  function toggleCollapse(index: number) {
    setCollapsedRows((prev) => {
      const next = new Set(prev);
      if (next.has(index)) next.delete(index);
      else next.add(index);
      return next;
    });
  }

  function collapseAll() {
    setCollapsedRows(new Set(rows.map((_, i) => i)));
  }

  function expandAll() {
    setCollapsedRows(new Set());
  }

  return (
    <div className="space-y-2">
      {rows.length > 1 && (
        <div className="flex justify-end gap-2">
          <button type="button" onClick={expandAll} className="text-xs text-indigo-600 hover:underline">Expand all</button>
          <button type="button" onClick={collapseAll} className="text-xs text-indigo-600 hover:underline">Collapse all</button>
        </div>
      )}
      {rows.map((row, rowIndex) => {
        const isCollapsed = collapsedRows.has(rowIndex);
        return (
          <div key={rowIndex} className={`rounded-lg border ${isCollapsed ? "border-slate-200 bg-white" : "border-slate-200 bg-slate-50"}`}>
            {/* Row header - always visible */}
            <div className="flex items-center gap-2 px-3 py-2">
              {/* Move up/down */}
              <div className="flex flex-col gap-0.5">
                <button type="button" onClick={() => moveRow(rowIndex, "up")} disabled={rowIndex === 0} className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed">
                  <ChevronUp className="h-3.5 w-3.5" />
                </button>
                <button type="button" onClick={() => moveRow(rowIndex, "down")} disabled={rowIndex === rows.length - 1} className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed">
                  <ChevronDown className="h-3.5 w-3.5" />
                </button>
              </div>

              {/* Collapse toggle + summary */}
              <button
                type="button"
                onClick={() => toggleCollapse(rowIndex)}
                className="flex flex-1 items-center gap-2 text-left min-w-0"
              >
                <ChevronRight className={`h-4 w-4 text-slate-400 shrink-0 transition-transform ${!isCollapsed ? "rotate-90" : ""}`} />
                <span className="text-xs font-semibold text-slate-500 shrink-0">#{rowIndex + 1}</span>
                {isCollapsed && (
                  <span className="text-xs text-slate-400 truncate">{getRowSummary(row, subFields)}</span>
                )}
              </button>

              {/* Delete */}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-7 w-7 text-red-400 hover:text-red-600 shrink-0"
                onClick={() => removeRow(rowIndex)}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>

            {/* Row fields - collapsible */}
            {!isCollapsed && (
              <div className="border-t border-slate-200 px-3 py-3 space-y-3">
                {subFields.map((sf) => (
                  <div key={sf.key} className="space-y-1.5">
                    <Label className="text-xs font-medium text-slate-600">
                      {sf.label}
                      {sf.required && <span className="ml-1 text-red-500">*</span>}
                    </Label>
                    <CustomFieldInput
                      field={sf}
                      value={row[sf.key]}
                      onChange={(val) => updateRow(rowIndex, sf.key, val)}
                    />
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })}
      <Button
        type="button"
        variant="outline"
        className="w-full rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
        onClick={addRow}
      >
        <Plus className="mr-2 h-4 w-4" />
        Add Row
      </Button>
    </div>
  );
}

// Node selector: search and select content nodes
function NodeSelectorInput({
  field,
  value,
  onChange,
}: {
  field: NodeTypeField;
  value: unknown;
  onChange: (val: unknown) => void;
}) {
  const [searchQuery, setSearchQuery] = useState("");
  const [results, setResults] = useState<NodeSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [showResults, setShowResults] = useState(false);

  const isMultiple = !!field.multiple;
  const selected: NodeSearchResult[] = isMultiple
    ? (Array.isArray(value) ? value as NodeSearchResult[] : [])
    : (value && typeof value === "object" && "id" in (value as Record<string, unknown>))
      ? [value as NodeSearchResult]
      : [];

  useEffect(() => {
    if (!searchQuery.trim()) {
      setResults([]);
      return;
    }
    const timer = setTimeout(async () => {
      setSearching(true);
      try {
        const res = await searchNodes({
          q: searchQuery,
          node_type: field.node_type_filter || undefined,
          limit: 10,
        });
        // Filter out already selected
        const selectedIds = new Set(selected.map((s) => s.id));
        setResults(res.filter((r) => !selectedIds.has(r.id)));
      } catch {
        setResults([]);
      } finally {
        setSearching(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery, field.node_type_filter]);

  function handleSelect(node: NodeSearchResult) {
    if (isMultiple) {
      onChange([...selected, node]);
    } else {
      onChange(node);
    }
    setSearchQuery("");
    setResults([]);
    setShowResults(false);
  }

  function handleRemove(id: number) {
    if (isMultiple) {
      onChange(selected.filter((s) => s.id !== id));
    } else {
      onChange(null);
    }
  }

  return (
    <div className="space-y-2">
      {/* Selected nodes */}
      {selected.length > 0 && (
        <div className="space-y-1.5">
          {selected.map((node) => (
            <div key={node.id} className="flex items-center gap-2 rounded-lg border border-indigo-200 bg-indigo-50 px-3 py-2">
              <span className="flex-1 text-sm font-medium text-slate-800">{node.title}</span>
              <span className="text-xs text-slate-400 font-mono">{node.node_type}</span>
              <Button type="button" variant="ghost" size="icon" className="h-6 w-6 text-slate-400 hover:text-red-500" onClick={() => handleRemove(node.id)}>
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>
          ))}
        </div>
      )}

      {/* Search input */}
      {(isMultiple || selected.length === 0) && (
        <div className="relative">
          <Input
            placeholder={`Search ${field.node_type_filter || "content"}...`}
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setShowResults(true);
            }}
            onFocus={() => setShowResults(true)}
            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
          />
          {/* Dropdown results */}
          {showResults && (searchQuery.trim() || searching) && (
            <div className="absolute z-10 mt-1 w-full rounded-lg border border-slate-200 bg-white shadow-lg max-h-48 overflow-y-auto">
              {searching ? (
                <div className="px-3 py-2 text-sm text-slate-400">Searching...</div>
              ) : results.length === 0 ? (
                <div className="px-3 py-2 text-sm text-slate-400">
                  {searchQuery.trim() ? "No results found" : "Type to search..."}
                </div>
              ) : (
                results.map((node) => (
                  <button
                    key={node.id}
                    type="button"
                    className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-indigo-50 transition-colors"
                    onClick={() => handleSelect(node)}
                  >
                    <span className="font-medium text-slate-800">{node.title}</span>
                    <span className="text-xs text-slate-400 font-mono ml-auto">{node.node_type}</span>
                    <span className={`text-xs px-1.5 py-0.5 rounded ${node.status === "published" ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"}`}>
                      {node.status}
                    </span>
                  </button>
                ))
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Node type selector: fetches registered node types and renders a select dropdown
function NodeTypeSelectInput({
  value,
  onChange,
}: {
  value: string;
  onChange: (val: string) => void;
}) {
  const [nodeTypes, setNodeTypes] = useState<NodeType[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getNodeTypes()
      .then(setNodeTypes)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <Select disabled>
        <SelectTrigger className="rounded-lg border-slate-300">
          <SelectValue placeholder="Loading node types..." />
        </SelectTrigger>
      </Select>
    );
  }

  return (
    <Select key={nodeTypes.length} value={value || undefined} onValueChange={onChange}>
      <SelectTrigger className="rounded-lg border-slate-300">
        <SelectValue placeholder="Select content type" />
      </SelectTrigger>
      <SelectContent>
        {nodeTypes.map((nt) => (
          <SelectItem key={nt.slug} value={nt.slug}>
            {nt.label || nt.slug}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function CustomFieldInput({
  field,
  value,
  onChange,
}: {
  field: NodeTypeField;
  value: unknown;
  onChange: (val: unknown) => void;
}) {
  const inputClass =
    "rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20";
  const strVal = value == null ? (field.default_value ?? "") : String(value);

  const input = (() => { switch (field.type) {
    case "text":
      return (
        <div className="flex">
          {field.prepend && <span className="inline-flex items-center rounded-l-lg border border-r-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.prepend}</span>}
          <Input
            placeholder={field.placeholder || `Enter ${field.label.toLowerCase()}`}
            value={strVal}
            onChange={(e) => onChange(e.target.value)}
            required={field.required}
            maxLength={field.max_length}
            className={`${inputClass} ${field.prepend ? "rounded-l-none" : ""} ${field.append ? "rounded-r-none" : ""}`}
          />
          {field.append && <span className="inline-flex items-center rounded-r-lg border border-l-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.append}</span>}
        </div>
      );
    case "textarea":
      return (
        <Textarea
          placeholder={field.placeholder || `Enter ${field.label.toLowerCase()}`}
          value={strVal}
          onChange={(e) => onChange(e.target.value)}
          rows={field.rows || 4}
          required={field.required}
          className={inputClass}
        />
      );
    case "number":
      return (
        <div className="flex">
          {field.prepend && <span className="inline-flex items-center rounded-l-lg border border-r-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.prepend}</span>}
          <Input
            type="number"
            placeholder={field.placeholder || "0"}
            value={strVal}
            onChange={(e) => onChange(e.target.value ? Number(e.target.value) : "")}
            required={field.required}
            min={field.min}
            max={field.max}
            step={field.step}
            className={`${inputClass} ${field.prepend ? "rounded-l-none" : ""} ${field.append ? "rounded-r-none" : ""}`}
          />
          {field.append && <span className="inline-flex items-center rounded-r-lg border border-l-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.append}</span>}
        </div>
      );
    case "date":
      return (
        <Input
          type="date"
          value={strVal}
          onChange={(e) => onChange(e.target.value)}
          required={field.required}
          className={inputClass}
        />
      );
    case "select":
      return (
        <Select value={strVal} onValueChange={(v) => onChange(v)}>
          <SelectTrigger className="rounded-lg border-slate-300">
            <SelectValue placeholder={`Select ${field.label.toLowerCase()}`} />
          </SelectTrigger>
          <SelectContent>
            {(field.options || []).map((opt) => (
              <SelectItem key={opt} value={opt}>
                {opt}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    case "node_type_select":
      return <NodeTypeSelectInput value={strVal} onChange={(v) => onChange(v)} />;
    case "toggle":
      return (
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => onChange(!value)}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500/20 ${
              value ? "bg-indigo-600" : "bg-slate-300"
            }`}
          >
            <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${value ? "translate-x-6" : "translate-x-1"}`} />
          </button>
          <span className="text-sm text-slate-600">{value ? "Yes" : "No"}</span>
        </div>
      );
    case "image":
      return (
        <Input
          placeholder={field.placeholder || "Image URL"}
          value={strVal}
          onChange={(e) => onChange(e.target.value)}
          required={field.required}
          className={inputClass}
        />
      );
    case "link":
      return <LinkFieldInput value={value as Record<string, unknown> | null} onChange={onChange} />;
    case "group":
      return <GroupFieldInput field={field} value={value as Record<string, unknown> | null} onChange={onChange} />;
    case "repeater":
      return <RepeaterFieldInput field={field} value={value as unknown[] | null} onChange={onChange} />;
    case "node":
      return <NodeSelectorInput field={field} value={value} onChange={onChange} />;
    case "richtext":
      return (
        <RichTextEditor
          value={strVal}
          onChange={(v) => onChange(v)}
          placeholder={field.placeholder || `Enter ${field.label.toLowerCase()}`}
        />
      );
    case "email":
      return (
        <div className="flex">
          {field.prepend && <span className="inline-flex items-center rounded-l-lg border border-r-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.prepend}</span>}
          <Input
            type="email"
            placeholder={field.placeholder || "email@example.com"}
            value={strVal}
            onChange={(e) => onChange(e.target.value)}
            required={field.required}
            className={`${inputClass} ${field.prepend ? "rounded-l-none" : ""} ${field.append ? "rounded-r-none" : ""}`}
          />
          {field.append && <span className="inline-flex items-center rounded-r-lg border border-l-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.append}</span>}
        </div>
      );
    case "url":
      return (
        <div className="flex">
          {field.prepend && <span className="inline-flex items-center rounded-l-lg border border-r-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.prepend}</span>}
          <Input
            type="url"
            placeholder={field.placeholder || "https://example.com"}
            value={strVal}
            onChange={(e) => onChange(e.target.value)}
            required={field.required}
            className={`${inputClass} ${field.prepend ? "rounded-l-none" : ""} ${field.append ? "rounded-r-none" : ""}`}
          />
          {field.append && <span className="inline-flex items-center rounded-r-lg border border-l-0 border-slate-300 bg-slate-100 px-3 text-sm text-slate-500">{field.append}</span>}
        </div>
      );
    case "color":
      return (
        <div className="flex items-center gap-3">
          <input
            type="color"
            value={strVal || "#000000"}
            onChange={(e) => onChange(e.target.value)}
            className="h-10 w-14 cursor-pointer rounded-lg border border-slate-300 p-1"
          />
          <Input
            placeholder="#000000"
            value={strVal}
            onChange={(e) => onChange(e.target.value)}
            className={`${inputClass} max-w-[140px] font-mono text-sm`}
          />
          {strVal && (
            <div
              className="h-8 w-8 rounded-md border border-slate-200 shrink-0"
              style={{ backgroundColor: strVal }}
            />
          )}
        </div>
      );
    case "range":
      return (
        <div className="space-y-2">
          <div className="flex items-center gap-4">
            <input
              type="range"
              min={field.min ?? 0}
              max={field.max ?? 100}
              step={field.step ?? 1}
              value={strVal || String(field.min ?? 0)}
              onChange={(e) => onChange(Number(e.target.value))}
              className="flex-1 h-2 rounded-lg appearance-none bg-slate-200 accent-indigo-600 cursor-pointer"
            />
            <span className="min-w-[3rem] text-center text-sm font-medium text-slate-700 bg-slate-100 px-2 py-1 rounded-md">
              {strVal || (field.min ?? 0)}
            </span>
          </div>
          <div className="flex justify-between text-xs text-slate-400">
            <span>{field.min ?? 0}</span>
            <span>{field.max ?? 100}</span>
          </div>
        </div>
      );
    case "file":
      return (
        <Input
          placeholder={field.placeholder || "File URL"}
          value={strVal}
          onChange={(e) => onChange(e.target.value)}
          required={field.required}
          className={inputClass}
        />
      );
    case "gallery":
      return (
        <Textarea
          placeholder="Enter image URLs, one per line"
          value={Array.isArray(value) ? (value as string[]).join("\n") : strVal}
          onChange={(e) => onChange(e.target.value.split("\n").filter(Boolean))}
          rows={4}
          className={inputClass}
        />
      );
    case "radio":
      return (
        <div className="space-y-2">
          {(field.options || []).map((opt) => (
            <label key={opt} className="flex items-center gap-2.5 cursor-pointer group" onClick={() => onChange(opt)}>
              <div className={`flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors ${strVal === opt ? "border-indigo-600 bg-indigo-600" : "border-slate-300 group-hover:border-slate-400"}`}>
                {strVal === opt && <div className="h-2 w-2 rounded-full bg-white" />}
              </div>
              <span className="text-sm text-slate-700">{opt}</span>
            </label>
          ))}
        </div>
      );
    case "checkbox": {
      const checked: string[] = Array.isArray(value) ? (value as string[]) : (value == null && field.default_value ? [field.default_value] : []);
      return (
        <div className="space-y-2">
          {(field.options || []).map((opt) => (
            <label key={opt} className="flex items-center gap-2.5 cursor-pointer">
              <input
                type="checkbox"
                checked={checked.includes(opt)}
                onChange={(e) => {
                  if (e.target.checked) {
                    onChange([...checked, opt]);
                  } else {
                    onChange(checked.filter((v) => v !== opt));
                  }
                }}
                className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-sm text-slate-700">{opt}</span>
            </label>
          ))}
        </div>
      );
    }
    default:
      return (
        <Input
          value={strVal}
          onChange={(e) => onChange(e.target.value)}
          className={inputClass}
        />
      );
  } })();

  return (
    <div>
      {input}
      {field.help_text && (
        <p className="mt-1 text-xs text-slate-400">{field.help_text}</p>
      )}
    </div>
  );
}

export default CustomFieldInput;
