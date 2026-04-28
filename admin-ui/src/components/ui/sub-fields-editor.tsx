import { useState, useEffect } from "react";
import { Plus, X, ChevronUp, ChevronDown, Pencil } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import FieldTypePicker from "@/components/ui/field-type-picker";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { NodeTypeField } from "@/api/client";

function fieldTypeBadgeClass(type: string): string {
  switch (type) {
    case "text":
      return "bg-blue-100 text-blue-700 hover:bg-blue-100";
    case "textarea":
      return "bg-purple-100 text-purple-700 hover:bg-purple-100";
    case "richtext":
      return "bg-purple-100 text-purple-700 hover:bg-purple-100";
    case "number":
      return "bg-amber-100 text-amber-700 hover:bg-amber-100";
    case "range":
      return "bg-amber-100 text-amber-700 hover:bg-amber-100";
    case "date":
      return "bg-teal-100 text-teal-700 hover:bg-teal-100";
    case "select":
      return "bg-indigo-100 text-indigo-700 hover:bg-indigo-100";
    case "radio":
      return "bg-indigo-100 text-indigo-700 hover:bg-indigo-100";
    case "checkbox":
      return "bg-indigo-100 text-indigo-700 hover:bg-indigo-100";
    case "image":
      return "bg-pink-100 text-pink-700 hover:bg-pink-100";
    case "gallery":
      return "bg-pink-100 text-pink-700 hover:bg-pink-100";
    case "file":
      return "bg-pink-100 text-pink-700 hover:bg-pink-100";
    case "toggle":
      return "bg-emerald-100 text-emerald-700 hover:bg-emerald-100";
    case "link":
      return "bg-cyan-100 text-cyan-700 hover:bg-cyan-100";
    case "color":
      return "bg-rose-100 text-rose-700 hover:bg-rose-100";
    case "email":
      return "bg-blue-100 text-blue-700 hover:bg-blue-100";
    case "url":
      return "bg-blue-100 text-blue-700 hover:bg-blue-100";
    case "node":
      return "bg-sky-100 text-sky-700 hover:bg-sky-100";
    case "term":
      return "bg-sky-100 text-sky-700 hover:bg-sky-100";
    default:
      return "bg-slate-100 text-slate-600 hover:bg-slate-100";
  }
}

interface SubFieldsEditorProps {
  value: NodeTypeField[];
  onChange: (fields: NodeTypeField[]) => void;
  label?: string;
}

function keyify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s]/g, "")
    .replace(/[\s]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

function TypeSpecificOptions({ field, updateField, size = "normal" }: { field: NodeTypeField; updateField: (updates: Partial<NodeTypeField>) => void; size?: "normal" | "compact" }) {
  const inputClass = size === "compact"
    ? "h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
    : "rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20";
  const labelClass = size === "compact" ? "text-xs font-medium text-slate-600" : "text-sm font-medium text-slate-700";

  return (
    <>
      {/* Options for select/radio/checkbox */}
      {(field.type === "select" || field.type === "radio" || field.type === "checkbox") && (
        <div className="space-y-1.5">
          <Label className={labelClass}>Options (comma-separated)</Label>
          <Input
            value={(field.options || []).join(", ")}
            onChange={(e) => updateField({ options: e.target.value.split(",").map((o) => o.trim()).filter(Boolean) })}
            placeholder="Option A, Option B, Option C"
            className={inputClass}
          />
        </div>
      )}

      {/* Placeholder */}
      {["text", "textarea", "number", "email", "url"].includes(field.type) && (
        <div className="space-y-1.5">
          <Label className={labelClass}>Placeholder</Label>
          <Input
            value={field.placeholder || ""}
            onChange={(e) => updateField({ placeholder: e.target.value || undefined })}
            placeholder="Placeholder text shown when empty"
            className={inputClass}
          />
        </div>
      )}

      {/* Default Value */}
      {!["group", "repeater"].includes(field.type) && (
        <div className="space-y-1.5">
          <Label className={labelClass}>Default Value</Label>
          <Input
            value={field.default_value || ""}
            onChange={(e) => updateField({ default_value: e.target.value || undefined })}
            placeholder="Default value for new content"
            className={inputClass}
          />
        </div>
      )}

      {/* Help Text */}
      <div className="space-y-1.5">
        <Label className={labelClass}>Help Text</Label>
        <Input
          value={field.help || ""}
          onChange={(e) => updateField({ help: e.target.value || undefined })}
          placeholder="Instructions shown below the field"
          className={inputClass}
        />
      </div>

      {/* Number/Range constraints */}
      {(field.type === "number" || field.type === "range") && (
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="space-y-1.5">
            <Label className={labelClass}>Min</Label>
            <Input type="number" value={field.min ?? ""} onChange={(e) => updateField({ min: e.target.value ? Number(e.target.value) : undefined })} placeholder="Min" className={inputClass} />
          </div>
          <div className="space-y-1.5">
            <Label className={labelClass}>Max</Label>
            <Input type="number" value={field.max ?? ""} onChange={(e) => updateField({ max: e.target.value ? Number(e.target.value) : undefined })} placeholder="Max" className={inputClass} />
          </div>
          <div className="space-y-1.5">
            <Label className={labelClass}>Step</Label>
            <Input type="number" value={field.step ?? ""} onChange={(e) => updateField({ step: e.target.value ? Number(e.target.value) : undefined })} placeholder="Step" className={inputClass} />
          </div>
        </div>
      )}

      {/* Text length constraints */}
      {(field.type === "text" || field.type === "textarea") && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label className={labelClass}>Min Length</Label>
            <Input type="number" value={field.min_length ?? ""} onChange={(e) => updateField({ min_length: e.target.value ? Number(e.target.value) : undefined })} placeholder="No minimum" className={inputClass} />
          </div>
          <div className="space-y-1.5">
            <Label className={labelClass}>Max Length</Label>
            <Input type="number" value={field.max_length ?? ""} onChange={(e) => updateField({ max_length: e.target.value ? Number(e.target.value) : undefined })} placeholder="No maximum" className={inputClass} />
          </div>
        </div>
      )}

      {/* Textarea rows */}
      {field.type === "textarea" && (
        <div className="space-y-1.5">
          <Label className={labelClass}>Rows</Label>
          <Input type="number" value={field.rows ?? ""} onChange={(e) => updateField({ rows: e.target.value ? Number(e.target.value) : undefined })} placeholder="4 (default)" className={inputClass} />
        </div>
      )}

      {/* Prepend / Append */}
      {["text", "number", "email", "url"].includes(field.type) && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label className={labelClass}>Prepend</Label>
            <Input value={field.prepend || ""} onChange={(e) => updateField({ prepend: e.target.value || undefined })} placeholder="e.g. $, https://" className={inputClass} />
          </div>
          <div className="space-y-1.5">
            <Label className={labelClass}>Append</Label>
            <Input value={field.append || ""} onChange={(e) => updateField({ append: e.target.value || undefined })} placeholder="e.g. px, kg, %" className={inputClass} />
          </div>
        </div>
      )}

      {/* File options */}
      {field.type === "file" && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label className={labelClass}>Allowed Types</Label>
            <Input value={field.allowed_types || ""} onChange={(e) => updateField({ allowed_types: e.target.value || undefined })} placeholder="pdf, doc, zip" className={inputClass} />
          </div>
          <div className="space-y-1.5">
            <Label className={labelClass}>&nbsp;</Label>
            <label className="flex items-center gap-2 h-9 cursor-pointer">
              <Switch checked={!!field.multiple} onCheckedChange={(c) => updateField({ multiple: c })} />
              <span className="text-sm text-slate-700">Multiple files</span>
            </label>
          </div>
        </div>
      )}

      {/* Term options */}
      {field.type === "term" && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label className={labelClass}>Taxonomy slug</Label>
            <Input value={field.taxonomy || ""} onChange={(e) => updateField({ taxonomy: e.target.value || undefined })} placeholder="e.g. trip_tag" className={inputClass} />
          </div>
          <div className="space-y-1.5">
            <Label className={labelClass}>Term node type</Label>
            <Input value={field.term_node_type || ""} onChange={(e) => updateField({ term_node_type: e.target.value || undefined })} placeholder="e.g. trip" className={inputClass} />
          </div>
          <div className="space-y-1.5 sm:col-span-2">
            <label className="flex items-center gap-2 h-9 cursor-pointer">
              <Switch checked={!!field.multiple} onCheckedChange={(c) => updateField({ multiple: c })} />
              <span className="text-sm text-slate-700">Allow multiple</span>
            </label>
          </div>
        </div>
      )}

      {/* Node options */}
      {field.type === "node" && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label className={labelClass}>Node Type Filter</Label>
            <Input value={field.node_type_filter || ""} onChange={(e) => updateField({ node_type_filter: e.target.value || undefined })} placeholder="e.g. page, post (empty = all)" className={inputClass} />
          </div>
          <div className="space-y-1.5">
            <Label className={labelClass}>&nbsp;</Label>
            <label className="flex items-center gap-2 h-9 cursor-pointer">
              <Switch checked={!!field.multiple} onCheckedChange={(c) => updateField({ multiple: c })} />
              <span className="text-sm text-slate-700">Allow multiple</span>
            </label>
          </div>
        </div>
      )}
    </>
  );
}

export default function SubFieldsEditor({ value, onChange, label }: SubFieldsEditorProps) {
  const [adding, setAdding] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [newFieldLabel, setNewFieldLabel] = useState("");
  const [newFieldKey, setNewFieldKey] = useState("");
  const [newFieldType, setNewFieldType] = useState<NodeTypeField["type"]>("text");
  const [newFieldRequired, setNewFieldRequired] = useState(false);
  const [autoKey, setAutoKey] = useState(true);

  // Temporary state for the "new field" type-specific options
  const [newFieldOptions, setNewFieldOptions] = useState("");
  const [newFieldPlaceholder, setNewFieldPlaceholder] = useState("");
  const [newFieldDefaultValue, setNewFieldDefaultValue] = useState("");
  const [newFieldHelpText, setNewFieldHelpText] = useState("");

  useEffect(() => {
    if (autoKey) setNewFieldKey(keyify(newFieldLabel));
  }, [newFieldLabel, autoKey]);

  function reset() {
    setNewFieldLabel("");
    setNewFieldKey("");
    setNewFieldType("text");
    setNewFieldRequired(false);
    setNewFieldOptions("");
    setNewFieldPlaceholder("");
    setNewFieldDefaultValue("");
    setNewFieldHelpText("");
    setAutoKey(true);
    setAdding(false);
  }

  function handleAdd() {
    if (!newFieldLabel.trim() || !newFieldKey.trim()) return;
    if (value.some((f) => f.key === newFieldKey)) return;
    const sf: NodeTypeField = {
      name: newFieldKey,
      key: newFieldKey,
      label: newFieldLabel,
      type: newFieldType,
      required: newFieldRequired || undefined,
    };
    if ((newFieldType === "select" || newFieldType === "radio" || newFieldType === "checkbox") && newFieldOptions.trim()) {
      sf.options = newFieldOptions.split(",").map((o) => o.trim()).filter(Boolean);
    }
    if (newFieldPlaceholder.trim()) sf.placeholder = newFieldPlaceholder.trim();
    if (newFieldDefaultValue.trim()) sf.default_value = newFieldDefaultValue.trim();
    if (newFieldHelpText.trim()) sf.help = newFieldHelpText.trim();
    onChange([...value, sf]);
    reset();
  }

  function handleRemove(index: number) {
    onChange(value.filter((_, i) => i !== index));
    if (editingIndex === index) setEditingIndex(null);
  }

  function updateField(index: number, updates: Partial<NodeTypeField>) {
    onChange(value.map((f, i) => i === index ? { ...f, ...updates } : f));
  }

  function handleMoveField(index: number, direction: "up" | "down") {
    const newFields = [...value];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newFields.length) return;
    [newFields[index], newFields[targetIndex]] = [newFields[targetIndex], newFields[index]];
    onChange(newFields);
    if (editingIndex === index) setEditingIndex(targetIndex);
    else if (editingIndex === targetIndex) setEditingIndex(index);
  }

  return (
    <div className="space-y-3">
      <Label className="text-sm font-medium text-slate-700">{label || "Sub-fields"}</Label>

      {value.length > 0 && (
        <div className="space-y-2">
          {value.map((sf, i) => (
            <div
              key={i}
              className={`rounded-lg border ${editingIndex === i ? "border-indigo-300 bg-indigo-50/30" : "border-slate-200 bg-slate-50"}`}
            >
              {/* Header row */}
              <div className="flex items-center gap-3 px-4 py-3">
                <div className="flex flex-col gap-0.5">
                  <button type="button" onClick={() => handleMoveField(i, "up")} disabled={i === 0} className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed">
                    <ChevronUp className="h-4 w-4" />
                  </button>
                  <button type="button" onClick={() => handleMoveField(i, "down")} disabled={i === value.length - 1} className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed">
                    <ChevronDown className="h-4 w-4" />
                  </button>
                </div>
                <button type="button" className="flex-1 min-w-0 text-left" onClick={() => setEditingIndex(editingIndex === i ? null : i)}>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-slate-800">{sf.label}</span>
                    <span className="text-xs text-slate-400 font-mono">{sf.key}</span>
                  </div>
                </button>
                <Badge className={`${fieldTypeBadgeClass(sf.type)} border-0 text-xs`}>{sf.type}</Badge>
                {sf.required && <Badge className="bg-red-100 text-red-600 hover:bg-red-100 border-0 text-xs">Required</Badge>}
                {sf.help && <Badge className="bg-slate-100 text-slate-500 hover:bg-slate-100 border-0 text-xs" title={sf.help}>?</Badge>}
                <Button type="button" variant="ghost" size="icon" className="h-8 w-8 text-slate-400 hover:text-indigo-600 shrink-0" onClick={() => setEditingIndex(editingIndex === i ? null : i)}>
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button type="button" variant="ghost" size="icon" className="h-8 w-8 text-red-500 hover:text-red-600 shrink-0" onClick={() => handleRemove(i)}>
                  <X className="h-4 w-4" />
                </Button>
              </div>

              {/* Inline edit form */}
              {editingIndex === i && (
                <div className="border-t border-indigo-200 px-4 py-3 space-y-3">
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="space-y-1.5">
                      <Label className="text-xs font-medium text-slate-600">Label</Label>
                      <Input value={sf.label} onChange={(e) => updateField(i, { label: e.target.value })} className="h-9 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs font-medium text-slate-600">Key</Label>
                      <Input value={sf.key} onChange={(e) => updateField(i, { key: e.target.value })} className="h-9 text-sm font-mono rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                    </div>
                  </div>
                  <div className="grid gap-3 sm:grid-cols-3">
                    <div className="space-y-1.5">
                      <Label className="text-xs font-medium text-slate-600">Type</Label>
                      <FieldTypePicker value={sf.type} onValueChange={(v) => updateField(i, { type: v as NodeTypeField["type"] })} compact />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs font-medium text-slate-600">Width</Label>
                      <Select
                        value={String(sf.width ?? 100)}
                        onValueChange={(v) => {
                          const n = Number(v);
                          updateField(i, { width: n === 100 ? undefined : n });
                        }}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="100">100%</SelectItem>
                          <SelectItem value="75">75%</SelectItem>
                          <SelectItem value="66">66%</SelectItem>
                          <SelectItem value="50">50%</SelectItem>
                          <SelectItem value="33">33%</SelectItem>
                          <SelectItem value="25">25%</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs font-medium text-slate-600">&nbsp;</Label>
                      <label className="flex items-center gap-2 h-9 cursor-pointer">
                        <Switch checked={!!sf.required} onCheckedChange={(c) => updateField(i, { required: c || undefined })} />
                        <span className="text-sm text-slate-700">Required</span>
                      </label>
                    </div>
                  </div>
                  <TypeSpecificOptions field={sf} updateField={(updates) => updateField(i, updates)} size="compact" />
                  {(sf.type === "group" || sf.type === "repeater") && (
                    <SubFieldsEditor
                      value={sf.sub_fields || []}
                      onChange={(subFields) => updateField(i, { sub_fields: subFields })}
                      label={sf.type === "group" ? "Group sub-fields" : "Repeater row fields"}
                    />
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Add sub-field form */}
      {adding ? (
        <>
          <Separator />
          <div className="space-y-4 rounded-lg border border-indigo-200 bg-indigo-50/50 p-4">
            <p className="text-sm font-semibold text-slate-700">New Sub-field</p>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Label</Label>
                <Input
                  value={newFieldLabel}
                  onChange={(e) => setNewFieldLabel(e.target.value)}
                  placeholder="Field label"
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label className="text-sm font-medium text-slate-700">Key</Label>
                  <button type="button" className="text-xs text-indigo-600 hover:underline" onClick={() => setAutoKey(!autoKey)}>
                    {autoKey ? "Edit manually" : "Auto-generate"}
                  </button>
                </div>
                <Input
                  value={newFieldKey}
                  onChange={(e) => { setAutoKey(false); setNewFieldKey(e.target.value); }}
                  disabled={autoKey}
                  placeholder="field_key"
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 font-mono text-sm"
                />
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Type</Label>
                <FieldTypePicker value={newFieldType} onValueChange={(v) => setNewFieldType(v as NodeTypeField["type"])} />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">&nbsp;</Label>
                <label className="flex items-center gap-2 h-9 cursor-pointer">
                  <Switch checked={newFieldRequired} onCheckedChange={setNewFieldRequired} />
                  <span className="text-sm font-medium text-slate-700">Required</span>
                </label>
              </div>
            </div>

            {/* Options for select/radio/checkbox */}
            {(newFieldType === "select" || newFieldType === "radio" || newFieldType === "checkbox") && (
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Options (comma-separated)</Label>
                <Input
                  value={newFieldOptions}
                  onChange={(e) => setNewFieldOptions(e.target.value)}
                  placeholder="Option A, Option B, Option C"
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
            )}

            {/* Placeholder */}
            {["text", "textarea", "number", "email", "url"].includes(newFieldType) && (
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Placeholder</Label>
                <Input
                  value={newFieldPlaceholder}
                  onChange={(e) => setNewFieldPlaceholder(e.target.value)}
                  placeholder="Placeholder text shown when empty"
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
            )}

            {/* Default Value */}
            <div className="space-y-2">
              <Label className="text-sm font-medium text-slate-700">Default Value</Label>
              <Input
                value={newFieldDefaultValue}
                onChange={(e) => setNewFieldDefaultValue(e.target.value)}
                placeholder="Default value for new content"
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
            </div>

            {/* Help Text */}
            <div className="space-y-2">
              <Label className="text-sm font-medium text-slate-700">Help Text</Label>
              <Input
                value={newFieldHelpText}
                onChange={(e) => setNewFieldHelpText(e.target.value)}
                placeholder="Instructions shown below the field"
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
            </div>

            <div className="flex gap-2">
              <Button
                type="button"
                className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg"
                onClick={handleAdd}
              >
                Add Sub-field
              </Button>
              <Button
                type="button"
                variant="outline"
                className="rounded-lg border-slate-300"
                onClick={reset}
              >
                Cancel
              </Button>
            </div>
          </div>
        </>
      ) : (
        <Button
          type="button"
          variant="outline"
          className="w-full rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
          onClick={() => setAdding(true)}
        >
          <Plus className="mr-2 h-4 w-4" />
          Add sub-field
        </Button>
      )}
    </div>
  );
}
