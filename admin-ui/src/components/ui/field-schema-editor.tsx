import { useEffect, useState } from "react";
import {
  Plus,
  ChevronUp,
  ChevronDown,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { AccordionRow } from "@/components/ui/accordion-row";
import SubFieldsEditor from "@/components/ui/sub-fields-editor";
import FieldTypePicker from "@/components/ui/field-type-picker";
import { toast } from "sonner";
import type { NodeTypeField } from "@/api/client";

function keyify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s]/g, "")
    .replace(/[\s]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export function fieldTypeBadgeClass(type: string): string {
  switch (type) {
    case "text":
      return "bg-blue-100 text-blue-700 hover:bg-blue-100";
    case "textarea":
      return "bg-purple-100 text-purple-700 hover:bg-purple-100";
    case "number":
      return "bg-amber-100 text-amber-700 hover:bg-amber-100";
    case "date":
      return "bg-teal-100 text-teal-700 hover:bg-teal-100";
    case "select":
      return "bg-accent text-accent-foreground";
    case "image":
      return "bg-pink-100 text-pink-700 hover:bg-pink-100";
    case "toggle":
      return "bg-emerald-100 text-emerald-700 hover:bg-emerald-100";
    case "link":
      return "bg-cyan-100 text-cyan-700 hover:bg-cyan-100";
    case "group":
      return "bg-violet-100 text-violet-700 hover:bg-violet-100";
    case "repeater":
      return "bg-orange-100 text-orange-700 hover:bg-orange-100";
    case "node":
      return "bg-sky-100 text-sky-700 hover:bg-sky-100";
    case "term":
      return "bg-teal-100 text-teal-700 hover:bg-teal-100";
    case "color":
      return "bg-rose-100 text-rose-700 hover:bg-rose-100";
    case "email":
      return "bg-blue-100 text-blue-700 hover:bg-blue-100";
    case "url":
      return "bg-blue-100 text-blue-700 hover:bg-blue-100";
    case "richtext":
      return "bg-purple-100 text-purple-700 hover:bg-purple-100";
    case "range":
      return "bg-amber-100 text-amber-700 hover:bg-amber-100";
    case "file":
      return "bg-pink-100 text-pink-700 hover:bg-pink-100";
    case "gallery":
      return "bg-pink-100 text-pink-700 hover:bg-pink-100";
    case "radio":
      return "bg-accent text-accent-foreground";
    case "checkbox":
      return "bg-accent text-accent-foreground";
    default:
      return "bg-muted text-muted-foreground";
  }
}

export interface FieldSchemaEditorProps {
  fields: NodeTypeField[];
  onChange: (fields: NodeTypeField[]) => void;
  title?: string;
  description?: string;
  addLabel?: string;
  disabled?: boolean;
}

export default function FieldSchemaEditor({
  fields,
  onChange,
  title: _title = "Custom Fields",
  description: _description,
  addLabel = "Add Field",
  disabled = false,
}: FieldSchemaEditorProps) {
  const [editingFieldIndex, setEditingFieldIndex] = useState<number | null>(null);

  // Add field form state
  const [showAddField, setShowAddField] = useState(false);
  const [newFieldLabel, setNewFieldLabel] = useState("");
  const [newFieldKey, setNewFieldKey] = useState("");
  const [newFieldType, setNewFieldType] = useState<NodeTypeField["type"]>("text");
  const [newFieldRequired, setNewFieldRequired] = useState(false);
  const [newFieldOptions, setNewFieldOptions] = useState("");
  const [newFieldSubFields, setNewFieldSubFields] = useState<NodeTypeField[]>([]);
  const [newFieldNodeTypeFilter, setNewFieldNodeTypeFilter] = useState("");
  const [newFieldTaxonomy, setNewFieldTaxonomy] = useState("");
  const [newFieldTermNodeType, setNewFieldTermNodeType] = useState("");
  const [newFieldMultiple, setNewFieldMultiple] = useState(false);
  const [newFieldPlaceholder, setNewFieldPlaceholder] = useState("");
  const [newFieldDefaultValue, setNewFieldDefaultValue] = useState("");
  const [newFieldHelpText, setNewFieldHelpText] = useState("");
  const [newFieldMin, setNewFieldMin] = useState("");
  const [newFieldMax, setNewFieldMax] = useState("");
  const [newFieldStep, setNewFieldStep] = useState("");
  const [newFieldMinLength, setNewFieldMinLength] = useState("");
  const [newFieldMaxLength, setNewFieldMaxLength] = useState("");
  const [newFieldRows, setNewFieldRows] = useState("");
  const [newFieldPrepend, setNewFieldPrepend] = useState("");
  const [newFieldAppend, setNewFieldAppend] = useState("");
  const [newFieldAllowedTypes, setNewFieldAllowedTypes] = useState("");
  const [autoFieldKey, setAutoFieldKey] = useState(true);

  useEffect(() => {
    if (autoFieldKey) {
      setNewFieldKey(keyify(newFieldLabel));
    }
  }, [newFieldLabel, autoFieldKey]);

  function resetAddFieldForm() {
    setNewFieldLabel("");
    setNewFieldKey("");
    setNewFieldType("text");
    setNewFieldRequired(false);
    setNewFieldOptions("");
    setNewFieldSubFields([]);
    setNewFieldNodeTypeFilter("");
    setNewFieldTaxonomy("");
    setNewFieldTermNodeType("");
    setNewFieldMultiple(false);
    setNewFieldPlaceholder("");
    setNewFieldDefaultValue("");
    setNewFieldHelpText("");
    setNewFieldMin("");
    setNewFieldMax("");
    setNewFieldStep("");
    setNewFieldMinLength("");
    setNewFieldMaxLength("");
    setNewFieldRows("");
    setNewFieldPrepend("");
    setNewFieldAppend("");
    setNewFieldAllowedTypes("");
    setAutoFieldKey(true);
    setShowAddField(false);
  }

  function handleAddField() {
    if (!newFieldLabel.trim() || !newFieldKey.trim()) {
      toast.error("Field label and key are required");
      return;
    }

    if (fields.some((f) => f.key === newFieldKey)) {
      toast.error("A field with this key already exists");
      return;
    }

    const sf: NodeTypeField = {
      name: newFieldKey,
      key: newFieldKey,
      title: newFieldLabel,
      label: newFieldLabel,
      type: newFieldType,
      required: newFieldRequired || undefined,
    };

    if (newFieldPlaceholder.trim()) sf.placeholder = newFieldPlaceholder.trim();
    if (newFieldDefaultValue.trim()) sf.default_value = newFieldDefaultValue.trim();
    if (newFieldHelpText.trim()) sf.help = newFieldHelpText.trim();

    if ((newFieldType === "select" || newFieldType === "radio" || newFieldType === "checkbox") && newFieldOptions.trim()) {
      sf.options = newFieldOptions.split(",").map((o) => o.trim()).filter(Boolean);
    }
    if ((newFieldType === "group" || newFieldType === "repeater") && newFieldSubFields.length > 0) {
      sf.sub_fields = newFieldSubFields;
    }
    if (newFieldType === "node") {
      if (newFieldNodeTypeFilter.trim()) sf.node_type_filter = newFieldNodeTypeFilter.trim();
      if (newFieldMultiple) sf.multiple = true;
    }
    if (newFieldType === "term") {
      if (newFieldTaxonomy.trim()) sf.taxonomy = newFieldTaxonomy.trim();
      if (newFieldTermNodeType.trim()) sf.term_node_type = newFieldTermNodeType.trim();
      if (newFieldMultiple) sf.multiple = true;
    }
    if (newFieldType === "number" || newFieldType === "range") {
      if (newFieldMin.trim()) sf.min = Number(newFieldMin);
      if (newFieldMax.trim()) sf.max = Number(newFieldMax);
      if (newFieldStep.trim()) sf.step = Number(newFieldStep);
    }
    if (newFieldType === "text" || newFieldType === "textarea") {
      if (newFieldMinLength.trim()) sf.min_length = Number(newFieldMinLength);
      if (newFieldMaxLength.trim()) sf.max_length = Number(newFieldMaxLength);
    }
    if (newFieldType === "textarea" && newFieldRows.trim()) {
      sf.rows = Number(newFieldRows);
    }
    if (["text", "number", "email", "url"].includes(newFieldType)) {
      if (newFieldPrepend.trim()) sf.prepend = newFieldPrepend.trim();
      if (newFieldAppend.trim()) sf.append = newFieldAppend.trim();
    }
    if (newFieldType === "file") {
      if (newFieldAllowedTypes.trim()) sf.allowed_types = newFieldAllowedTypes.trim();
      if (newFieldMultiple) sf.multiple = true;
    }

    onChange([...fields, sf]);
    resetAddFieldForm();
  }

  function handleRemoveField(index: number) {
    onChange(fields.filter((_, i) => i !== index));
    if (editingFieldIndex === index) setEditingFieldIndex(null);
  }

  function updateField(index: number, updates: Partial<NodeTypeField>) {
    onChange(fields.map((f, i) => i === index ? { ...f, ...updates } : f));
  }

  function handleMoveField(index: number, direction: "up" | "down") {
    const newFields = [...fields];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newFields.length) return;
    [newFields[index], newFields[targetIndex]] = [newFields[targetIndex], newFields[index]];
    onChange(newFields);
  }

  return (
    <div className="space-y-3">
      {fields.length === 0 && !showAddField && (
        <p className="text-sm text-center py-4">
          No fields defined yet. Add fields to define the structure.
        </p>
      )}

      {fields.length > 0 && (
        <div className="space-y-2">
          {fields.map((field, index) => (
            <AccordionRow
              key={index}
              open={editingFieldIndex === index}
              onToggle={() => !disabled && setEditingFieldIndex(editingFieldIndex === index ? null : index)}
              headerLeft={
                <>
                  <span className="font-semibold min-w-0 truncate" style={{ fontSize: 12.5, color: "var(--fg)" }}>
                    {field.label}
                  </span>
                  <span className="font-mono shrink-0" style={{ fontSize: 11, color: "var(--fg-muted)" }}>
                    {field.key}
                  </span>
                  <Badge className={`${fieldTypeBadgeClass(field.type)} border-0 text-[10px] shrink-0`}>{field.type}</Badge>
                  {field.required && <Badge className="border-0 text-[10px] shrink-0" style={{ background: "var(--danger-bg)", color: "var(--danger)" }}>Required</Badge>}
                </>
              }
              headerRight={
                <>
                  <button
                    type="button"
                    onClick={() => handleMoveField(index, "up")}
                    disabled={disabled || index === 0}
                    className="p-1 rounded disabled:opacity-30 disabled:cursor-not-allowed hover:bg-black/5"
                    style={{ color: "var(--fg-muted)" }}
                    title="Move up"
                  >
                    <ChevronUp className="h-3.5 w-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => handleMoveField(index, "down")}
                    disabled={disabled || index === fields.length - 1}
                    className="p-1 rounded disabled:opacity-30 disabled:cursor-not-allowed hover:bg-black/5"
                    style={{ color: "var(--fg-muted)" }}
                    title="Move down"
                  >
                    <ChevronDown className="h-3.5 w-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => handleRemoveField(index)}
                    disabled={disabled}
                    className="p-1 rounded hover:bg-muted disabled:opacity-30 disabled:cursor-not-allowed"
                    style={{ color: "var(--danger)" }}
                    title="Delete"
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </>
              }
            >
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Label</Label>
                  <Input value={field.label} onChange={(e) => updateField(index, { label: e.target.value })} disabled={disabled} className="h-8 text-sm" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Key</Label>
                  <Input value={field.key} onChange={(e) => updateField(index, { key: e.target.value })} disabled={disabled} className="h-8 text-sm font-mono" />
                </div>
              </div>
              <div className="grid gap-3 sm:grid-cols-3">
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Type</Label>
                  <FieldTypePicker value={field.type} onValueChange={(v) => updateField(index, { type: v as NodeTypeField["type"] })} compact />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Width</Label>
                  <Select
                    value={String(field.width ?? 100)}
                    onValueChange={(v) => {
                      const n = Number(v);
                      updateField(index, { width: n === 100 ? undefined : n });
                    }}
                    disabled={disabled}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="100">100% — full row</SelectItem>
                      <SelectItem value="75">75%</SelectItem>
                      <SelectItem value="66">66% (2/3)</SelectItem>
                      <SelectItem value="50">50% (half)</SelectItem>
                      <SelectItem value="33">33% (1/3)</SelectItem>
                      <SelectItem value="25">25% (quarter)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">&nbsp;</Label>
                  <label className={`flex items-center gap-2 h-8 ${disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                    <Switch checked={!!field.required} disabled={disabled} onCheckedChange={(c) => updateField(index, { required: c || undefined })} />
                    <span className="text-sm text-foreground">Required</span>
                  </label>
                </div>
              </div>
              {field.type === "select" && (
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Options (comma-separated)</Label>
                  <Input value={(field.options || []).join(", ")} disabled={disabled} onChange={(e) => updateField(index, { options: e.target.value.split(",").map((o) => o.trim()).filter(Boolean) })} className="h-8 text-sm" />
                </div>
              )}
              {(field.type === "group" || field.type === "repeater") && (
                <SubFieldsEditor
                  value={field.sub_fields || []}
                  onChange={(sf) => updateField(index, { sub_fields: sf })}
                  label={field.type === "group" ? "Group sub-fields" : "Repeater row fields"}
                />
              )}
              {field.type === "node" && (
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Node Type Filter</Label>
                    <Input value={field.node_type_filter || ""} disabled={disabled} onChange={(e) => updateField(index, { node_type_filter: e.target.value })} placeholder="e.g. page, post (empty = all)" className="h-8 text-sm" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">&nbsp;</Label>
                    <label className={`flex items-center gap-2 h-8 ${disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                      <Switch checked={!!field.multiple} disabled={disabled} onCheckedChange={(c) => updateField(index, { multiple: c })} />
                      <span className="text-sm text-foreground">Allow multiple</span>
                    </label>
                  </div>
                </div>
              )}
              {["text", "textarea", "number", "email", "url"].includes(field.type) && (
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Placeholder</Label>
                  <Input value={field.placeholder || ""} disabled={disabled} onChange={(e) => updateField(index, { placeholder: e.target.value || undefined })} placeholder="Placeholder text" className="h-8 text-sm" />
                </div>
              )}
              {!["group", "repeater"].includes(field.type) && (
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Default Value</Label>
                  <Input value={field.default_value || ""} disabled={disabled} onChange={(e) => updateField(index, { default_value: e.target.value || undefined })} placeholder="Default value" className="h-8 text-sm" />
                </div>
              )}
              <div className="space-y-1">
                <Label className="text-xs font-medium text-muted-foreground">Help Text</Label>
                <Input value={field.help || ""} disabled={disabled} onChange={(e) => updateField(index, { help: e.target.value || undefined })} placeholder="Instructions for content editors" className="h-8 text-sm" />
              </div>
              {(field.type === "radio" || field.type === "checkbox") && (
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Options (comma-separated)</Label>
                  <Input value={(field.options || []).join(", ")} disabled={disabled} onChange={(e) => updateField(index, { options: e.target.value.split(",").map((o) => o.trim()).filter(Boolean) })} className="h-8 text-sm" />
                </div>
              )}
              {(field.type === "number" || field.type === "range") && (
                <div className="grid gap-3 sm:grid-cols-3">
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Min</Label>
                    <Input type="number" value={field.min ?? ""} disabled={disabled} onChange={(e) => updateField(index, { min: e.target.value ? Number(e.target.value) : undefined })} className="h-8 text-sm" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Max</Label>
                    <Input type="number" value={field.max ?? ""} disabled={disabled} onChange={(e) => updateField(index, { max: e.target.value ? Number(e.target.value) : undefined })} className="h-8 text-sm" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Step</Label>
                    <Input type="number" value={field.step ?? ""} disabled={disabled} onChange={(e) => updateField(index, { step: e.target.value ? Number(e.target.value) : undefined })} className="h-8 text-sm" />
                  </div>
                </div>
              )}
              {(field.type === "text" || field.type === "textarea") && (
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Min Length</Label>
                    <Input type="number" value={field.min_length ?? ""} disabled={disabled} onChange={(e) => updateField(index, { min_length: e.target.value ? Number(e.target.value) : undefined })} placeholder="No min" className="h-8 text-sm" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Max Length</Label>
                    <Input type="number" value={field.max_length ?? ""} disabled={disabled} onChange={(e) => updateField(index, { max_length: e.target.value ? Number(e.target.value) : undefined })} placeholder="No max" className="h-8 text-sm" />
                  </div>
                </div>
              )}
              {field.type === "textarea" && (
                <div className="space-y-1">
                  <Label className="text-xs font-medium text-muted-foreground">Rows</Label>
                  <Input type="number" value={field.rows ?? ""} disabled={disabled} onChange={(e) => updateField(index, { rows: e.target.value ? Number(e.target.value) : undefined })} placeholder="4 (default)" className="h-8 text-sm" />
                </div>
              )}
              {["text", "number", "email", "url"].includes(field.type) && (
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Prepend</Label>
                    <Input value={field.prepend || ""} disabled={disabled} onChange={(e) => updateField(index, { prepend: e.target.value || undefined })} placeholder="e.g. $" className="h-8 text-sm" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Append</Label>
                    <Input value={field.append || ""} disabled={disabled} onChange={(e) => updateField(index, { append: e.target.value || undefined })} placeholder="e.g. px" className="h-8 text-sm" />
                  </div>
                </div>
              )}
              {field.type === "file" && (
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">Allowed Types</Label>
                    <Input value={field.allowed_types || ""} disabled={disabled} onChange={(e) => updateField(index, { allowed_types: e.target.value || undefined })} placeholder="pdf, doc, zip" className="h-8 text-sm" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs font-medium text-muted-foreground">&nbsp;</Label>
                    <label className={`flex items-center gap-2 h-8 ${disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                      <Switch checked={!!field.multiple} disabled={disabled} onCheckedChange={(c) => updateField(index, { multiple: c })} />
                      <span className="text-sm text-foreground">Multiple files</span>
                    </label>
                  </div>
                </div>
              )}
            </AccordionRow>
          ))}
        </div>
      )}

      {/* Add field form */}
      {!disabled && showAddField && (
        <>
          <Separator />
          <div className="space-y-4 rounded-lg border border-border bg-muted p-4">
            <p className="text-sm font-semibold text-foreground">New Field</p>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">Label</Label>
                <Input
                  placeholder="e.g. Price, Author Name"
                  value={newFieldLabel}
                  onChange={(e) => setNewFieldLabel(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label className="text-sm font-medium text-foreground">Key</Label>
                  <button
                    type="button"
                    className="text-xs hover:underline"
                    onClick={() => setAutoFieldKey(!autoFieldKey)}
                  >
                    {autoFieldKey ? "Edit manually" : "Auto-generate"}
                  </button>
                </div>
                <Input
                  placeholder="field_key"
                  value={newFieldKey}
                  onChange={(e) => {
                    setAutoFieldKey(false);
                    setNewFieldKey(e.target.value);
                  }}
                  disabled={autoFieldKey}
                  className="font-mono text-sm"
                />
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">Type</Label>
                <FieldTypePicker value={newFieldType} onValueChange={(v) => setNewFieldType(v as NodeTypeField["type"])} />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">&nbsp;</Label>
                <label htmlFor="new-field-required" className="flex items-center gap-2 h-9 cursor-pointer">
                  <Switch
                    id="new-field-required"
                    checked={newFieldRequired}
                    onCheckedChange={setNewFieldRequired}
                  />
                  <span className="text-sm font-medium text-foreground">Required</span>
                </label>
              </div>
            </div>

            {newFieldType === "select" && (
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">Options (comma-separated)</Label>
                <Input
                  placeholder="e.g. Option A, Option B, Option C"
                  value={newFieldOptions}
                  onChange={(e) => setNewFieldOptions(e.target.value)}
                />
              </div>
            )}

            {(newFieldType === "group" || newFieldType === "repeater") && (
              <SubFieldsEditor
                value={newFieldSubFields}
                onChange={setNewFieldSubFields}
                label={newFieldType === "group" ? "Group sub-fields" : "Repeater row fields"}
              />
            )}

            {newFieldType === "node" && (
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Node Type Filter</Label>
                  <Input
                    value={newFieldNodeTypeFilter}
                    onChange={(e) => setNewFieldNodeTypeFilter(e.target.value)}
                    placeholder="e.g. page, product (empty = all)"
                  />
                </div>
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">&nbsp;</Label>
                  <label className="flex items-center gap-2 h-9 cursor-pointer">
                    <Switch checked={newFieldMultiple} onCheckedChange={setNewFieldMultiple} />
                    <span className="text-sm text-foreground">Allow multiple selection</span>
                  </label>
                </div>
              </div>
            )}

            {newFieldType === "term" && (
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Taxonomy Slug</Label>
                  <Input
                    value={newFieldTaxonomy}
                    onChange={(e) => setNewFieldTaxonomy(e.target.value)}
                    placeholder="e.g. trip_tag, category"
                  />
                </div>
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Node Type (for terms)</Label>
                  <Input
                    value={newFieldTermNodeType}
                    onChange={(e) => setNewFieldTermNodeType(e.target.value)}
                    placeholder="e.g. trip, post (must match taxonomy)"
                  />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <label className="flex items-center gap-2 h-9 cursor-pointer">
                    <Switch checked={newFieldMultiple} onCheckedChange={setNewFieldMultiple} />
                    <span className="text-sm text-foreground">Allow multiple selection</span>
                  </label>
                </div>
              </div>
            )}

            {["text", "textarea", "number", "email", "url"].includes(newFieldType) && (
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">Placeholder</Label>
                <Input
                  placeholder="Placeholder text shown when empty"
                  value={newFieldPlaceholder}
                  onChange={(e) => setNewFieldPlaceholder(e.target.value)}
                />
              </div>
            )}

            {!["group", "repeater"].includes(newFieldType) && (
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">Default Value</Label>
                <Input
                  placeholder="Default value for new content"
                  value={newFieldDefaultValue}
                  onChange={(e) => setNewFieldDefaultValue(e.target.value)}
                />
              </div>
            )}

            <div className="space-y-2">
              <Label className="text-sm font-medium text-foreground">Help Text</Label>
              <Input
                placeholder="Instructions shown below the field"
                value={newFieldHelpText}
                onChange={(e) => setNewFieldHelpText(e.target.value)}
              />
            </div>

            {(newFieldType === "radio" || newFieldType === "checkbox") && (
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">Options (comma-separated)</Label>
                <Input
                  placeholder="e.g. Option A, Option B, Option C"
                  value={newFieldOptions}
                  onChange={(e) => setNewFieldOptions(e.target.value)}
                />
              </div>
            )}

            {(newFieldType === "number" || newFieldType === "range") && (
              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Min</Label>
                  <Input type="number" placeholder="0" value={newFieldMin} onChange={(e) => setNewFieldMin(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Max</Label>
                  <Input type="number" placeholder="100" value={newFieldMax} onChange={(e) => setNewFieldMax(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Step</Label>
                  <Input type="number" placeholder="1" value={newFieldStep} onChange={(e) => setNewFieldStep(e.target.value)} />
                </div>
              </div>
            )}

            {(newFieldType === "text" || newFieldType === "textarea") && (
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Min Length</Label>
                  <Input type="number" placeholder="No minimum" value={newFieldMinLength} onChange={(e) => setNewFieldMinLength(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Max Length</Label>
                  <Input type="number" placeholder="No maximum" value={newFieldMaxLength} onChange={(e) => setNewFieldMaxLength(e.target.value)} />
                </div>
              </div>
            )}

            {newFieldType === "textarea" && (
              <div className="space-y-2">
                <Label className="text-sm font-medium text-foreground">Rows</Label>
                <Input type="number" placeholder="4 (default)" value={newFieldRows} onChange={(e) => setNewFieldRows(e.target.value)} />
              </div>
            )}

            {["text", "number", "email", "url"].includes(newFieldType) && (
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Prepend</Label>
                  <Input placeholder="e.g. $, https://" value={newFieldPrepend} onChange={(e) => setNewFieldPrepend(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Append</Label>
                  <Input placeholder="e.g. px, kg, %" value={newFieldAppend} onChange={(e) => setNewFieldAppend(e.target.value)} />
                </div>
              </div>
            )}

            {newFieldType === "file" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label className="text-sm font-medium text-foreground">Allowed File Types</Label>
                  <Input placeholder="e.g. pdf, doc, zip (empty = all)" value={newFieldAllowedTypes} onChange={(e) => setNewFieldAllowedTypes(e.target.value)} />
                </div>
                <label className="flex items-center gap-2 h-9 cursor-pointer">
                  <Switch checked={newFieldMultiple} onCheckedChange={setNewFieldMultiple} />
                  <span className="text-sm text-foreground">Allow multiple files</span>
                </label>
              </div>
            )}

            <div className="flex gap-2">
              <Button
                type="button"
                className="bg-primary hover:bg-primary/90 text-primary-foreground font-medium rounded-lg"
                onClick={handleAddField}
              >
                {addLabel}
              </Button>
              <Button
                type="button"
                variant="outline"
                className="rounded-lg border-border"
                onClick={resetAddFieldForm}
              >
                Cancel
              </Button>
            </div>
          </div>
        </>
      )}

      {!disabled && !showAddField && (
        <Button
          type="button"
          variant="outline"
          className="w-full rounded-lg border-dashed border-border text-muted-foreground hover:bg-muted"
          onClick={() => setShowAddField(true)}
        >
          <Plus className="mr-2 h-4 w-4" />
          {addLabel}
        </Button>
      )}
    </div>
  );
}
