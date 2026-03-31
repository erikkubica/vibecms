import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Plus,
  ChevronUp,
  ChevronDown,
  X,
  Pencil,
  Square,
  LayoutTemplate,
  Type,
  Image,
  MousePointerClick,
  Images,
  Play,
  List,
  Quote,
  MapPin,
  Code,
  SeparatorHorizontal as SeparatorIcon,
  FileText,
  Newspaper,
  ShoppingBag,
  Calendar,
  Users,
  Folder,
  Bookmark,
  Tag,
  Star,
  Heart,
  Unlink,
  Info,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import SubFieldsEditor from "@/components/ui/sub-fields-editor";
import FieldTypePicker from "@/components/ui/field-type-picker";
import CodeEditor from "@/components/ui/code-editor";
import { toast } from "sonner";
import {
  getBlockType,
  createBlockType,
  updateBlockType,
  deleteBlockType,
  detachBlockType,
  type BlockType,
  type NodeTypeField,
  previewBlockTemplate,
} from "@/api/client";
import CustomFieldInput from "@/components/ui/custom-field-input";

const ICON_OPTIONS: { value: string; label: string; icon: LucideIcon }[] = [
  { value: "square", label: "Square", icon: Square },
  { value: "layout-template", label: "Layout", icon: LayoutTemplate },
  { value: "type", label: "Type", icon: Type },
  { value: "image", label: "Image", icon: Image },
  { value: "mouse-pointer-click", label: "Button", icon: MousePointerClick },
  { value: "images", label: "Gallery", icon: Images },
  { value: "play", label: "Play", icon: Play },
  { value: "list", label: "List", icon: List },
  { value: "quote", label: "Quote", icon: Quote },
  { value: "map-pin", label: "Map", icon: MapPin },
  { value: "code", label: "Code", icon: Code },
  { value: "separator", label: "Divider", icon: SeparatorIcon },
  { value: "file-text", label: "File", icon: FileText },
  { value: "newspaper", label: "News", icon: Newspaper },
  { value: "shopping-bag", label: "Shop", icon: ShoppingBag },
  { value: "calendar", label: "Calendar", icon: Calendar },
  { value: "users", label: "Users", icon: Users },
  { value: "folder", label: "Folder", icon: Folder },
  { value: "bookmark", label: "Bookmark", icon: Bookmark },
  { value: "tag", label: "Tag", icon: Tag },
  { value: "star", label: "Star", icon: Star },
  { value: "heart", label: "Heart", icon: Heart },
];


function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function keyify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s]/g, "")
    .replace(/[\s]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

function fieldTypeBadgeClass(type: string): string {
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
      return "bg-indigo-100 text-indigo-700 hover:bg-indigo-100";
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
      return "bg-indigo-100 text-indigo-700 hover:bg-indigo-100";
    case "checkbox":
      return "bg-indigo-100 text-indigo-700 hover:bg-indigo-100";
    default:
      return "bg-slate-100 text-slate-600 hover:bg-slate-100";
  }
}

export default function BlockTypeEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDetach, setShowDetach] = useState(false);
  const [detaching, setDetaching] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  // Form state
  const [label, setLabel] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [icon, setIcon] = useState("square");
  const [fields, setFields] = useState<NodeTypeField[]>([]);
  const [htmlTemplate, setHtmlTemplate] = useState("");
  const [source, setSource] = useState("custom");
  const isManaged = source !== "custom";
  const [originalBlockType, setOriginalBlockType] = useState<BlockType | null>(null);

  // Template tabs state
  const [activeTemplateTab, setActiveTemplateTab] = useState<"template" | "testdata" | "preview">("template");
  const [testData, setTestData] = useState<Record<string, unknown>>({});
  const [previewHtml, setPreviewHtml] = useState<string | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  // Add field form state
  const [showAddField, setShowAddField] = useState(false);
  const [newFieldLabel, setNewFieldLabel] = useState("");
  const [newFieldKey, setNewFieldKey] = useState("");
  const [newFieldType, setNewFieldType] = useState<NodeTypeField["type"]>("text");
  const [newFieldRequired, setNewFieldRequired] = useState(false);
  const [newFieldOptions, setNewFieldOptions] = useState("");
  const [newFieldSubFields, setNewFieldSubFields] = useState<NodeTypeField[]>([]);
  const [newFieldNodeTypeFilter, setNewFieldNodeTypeFilter] = useState("");
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
  const [cacheOutput, setCacheOutput] = useState(true);
  const [autoFieldKey, setAutoFieldKey] = useState(true);

  useEffect(() => {
    if (!isEdit) return;
    let cancelled = false;
    setLoading(true);
    getBlockType(id)
      .then((bt) => {
        if (cancelled) return;
        setOriginalBlockType(bt);
        setLabel(bt.label);
        setSlug(bt.slug);
        setDescription(bt.description || "");
        setIcon(bt.icon || "square");
        setFields(bt.field_schema || []);
        setHtmlTemplate(bt.html_template || "");
        setTestData(bt.test_data || {});
        setSource(bt.source || "custom");
        setCacheOutput(bt.cache_output !== false);
        setAutoSlug(false);
      })
      .catch(() => {
        toast.error("Failed to load block type");
        navigate("/admin/block-types", { replace: true });
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, isEdit, navigate]);

  // Auto-generate slug from label
  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(label));
    }
  }, [label, autoSlug]);

  // Auto-generate field key from field label
  useEffect(() => {
    if (autoFieldKey) {
      setNewFieldKey(keyify(newFieldLabel));
    }
  }, [newFieldLabel, autoFieldKey]);

  // Auto-render preview when switching to preview tab
  useEffect(() => {
    if (activeTemplateTab === "preview" && htmlTemplate.trim()) {
      setPreviewLoading(true);
      previewBlockTemplate(htmlTemplate, testData)
        .then(setPreviewHtml)
        .catch(() => setPreviewHtml('<div class="text-red-500 text-sm">Failed to render preview</div>'))
        .finally(() => setPreviewLoading(false));
    }
  }, [activeTemplateTab]);

  function resetAddFieldForm() {
    setNewFieldLabel("");
    setNewFieldKey("");
    setNewFieldType("text");
    setNewFieldRequired(false);
    setNewFieldOptions("");
    setNewFieldSubFields([]);
    setNewFieldNodeTypeFilter("");
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

    const newField: NodeTypeField = {
      key: newFieldKey,
      label: newFieldLabel,
      type: newFieldType,
      required: newFieldRequired || undefined,
    };

    if (newFieldPlaceholder.trim()) newField.placeholder = newFieldPlaceholder.trim();
    if (newFieldDefaultValue.trim()) newField.default_value = newFieldDefaultValue.trim();
    if (newFieldHelpText.trim()) newField.help_text = newFieldHelpText.trim();

    if ((newFieldType === "select" || newFieldType === "radio" || newFieldType === "checkbox") && newFieldOptions.trim()) {
      newField.options = newFieldOptions.split(",").map((o) => o.trim()).filter(Boolean);
    }
    if ((newFieldType === "group" || newFieldType === "repeater") && newFieldSubFields.length > 0) {
      newField.sub_fields = newFieldSubFields;
    }
    if (newFieldType === "node") {
      if (newFieldNodeTypeFilter.trim()) newField.node_type_filter = newFieldNodeTypeFilter.trim();
      if (newFieldMultiple) newField.multiple = true;
    }
    if (newFieldType === "number" || newFieldType === "range") {
      if (newFieldMin.trim()) newField.min = Number(newFieldMin);
      if (newFieldMax.trim()) newField.max = Number(newFieldMax);
      if (newFieldStep.trim()) newField.step = Number(newFieldStep);
    }
    if (newFieldType === "text" || newFieldType === "textarea") {
      if (newFieldMinLength.trim()) newField.min_length = Number(newFieldMinLength);
      if (newFieldMaxLength.trim()) newField.max_length = Number(newFieldMaxLength);
    }
    if (newFieldType === "textarea" && newFieldRows.trim()) {
      newField.rows = Number(newFieldRows);
    }
    if (["text", "number", "email", "url"].includes(newFieldType)) {
      if (newFieldPrepend.trim()) newField.prepend = newFieldPrepend.trim();
      if (newFieldAppend.trim()) newField.append = newFieldAppend.trim();
    }
    if (newFieldType === "file") {
      if (newFieldAllowedTypes.trim()) newField.allowed_types = newFieldAllowedTypes.trim();
      if (newFieldMultiple) newField.multiple = true;
    }

    setFields([...fields, newField]);
    resetAddFieldForm();
  }

  const [editingFieldIndex, setEditingFieldIndex] = useState<number | null>(null);

  function handleRemoveField(index: number) {
    setFields(fields.filter((_, i) => i !== index));
    if (editingFieldIndex === index) setEditingFieldIndex(null);
  }

  function updateField(index: number, updates: Partial<NodeTypeField>) {
    setFields(fields.map((f, i) => i === index ? { ...f, ...updates } : f));
  }

  function handleMoveField(index: number, direction: "up" | "down") {
    const newFields = [...fields];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newFields.length) return;
    [newFields[index], newFields[targetIndex]] = [newFields[targetIndex], newFields[index]];
    setFields(newFields);
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!label.trim() || !slug.trim()) {
      toast.error("Label and slug are required");
      return;
    }

    const data: Partial<BlockType> = {
      label,
      slug,
      description,
      icon,
      field_schema: fields,
      html_template: htmlTemplate,
      test_data: testData,
      source,
      cache_output: cacheOutput,
    };

    setSaving(true);
    try {
      if (isEdit) {
        const updated = await updateBlockType(id, data);
        setOriginalBlockType(updated);
        toast.success("Block type updated successfully");
      } else {
        const created = await createBlockType(data);
        toast.success("Block type created successfully");
        navigate(`/admin/block-types/${created.id}/edit`, { replace: true });
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save block type";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteBlockType(id);
      toast.success("Block type deleted successfully");
      navigate("/admin/block-types", { replace: true });
    } catch {
      toast.error("Failed to delete block type");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach() {
    if (!id) return;
    setDetaching(true);
    try {
      const detached = await detachBlockType(id);
      setOriginalBlockType(detached);
      setSource(detached.source);
      toast.success("Block type detached — now editable");
      setShowDetach(false);
    } catch {
      toast.error("Failed to detach block type");
    } finally {
      setDetaching(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild className="rounded-lg hover:bg-slate-200">
            <Link to="/admin/block-types">
              <ArrowLeft className="h-5 w-5 text-slate-600" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">
            {isEdit ? "Edit Block Type" : "New Block Type"}
          </h1>
          {isEdit && isManaged && (
            <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-0 text-xs">
              {source === "theme" ? "Theme" : "Extension"}
            </Badge>
          )}
        </div>
        {isEdit && isManaged && (
          <Button
            variant="outline"
            onClick={() => setShowDetach(true)}
            className="text-amber-600 border-amber-300 hover:bg-amber-50"
          >
            <Unlink className="mr-2 h-4 w-4" />
            Detach
          </Button>
        )}
      </div>

      {isEdit && isManaged && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 flex items-start gap-2">
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            This block type is managed by the active {source} and is read-only. To customize it, click
            &quot;Detach&quot; to create an editable copy.
          </p>
        </div>
      )}

      <form onSubmit={handleSave} className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          {/* Basic info */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">Basic Info</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-6 pt-0">
              <div className="space-y-2">
                <Label htmlFor="label" className="text-sm font-medium text-slate-700">Label</Label>
                <Input
                  id="label"
                  placeholder="e.g. Hero, Text Block, Image Gallery"
                  value={label}
                  onChange={(e) => setLabel(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="slug" className="text-sm font-medium text-slate-700">Slug</Label>
                  <button
                    type="button"
                    className="text-xs text-indigo-600 hover:underline"
                    onClick={() => setAutoSlug(!autoSlug)}
                  >
                    {autoSlug ? "Edit manually" : "Auto-generate"}
                  </button>
                </div>
                <Input
                  id="slug"
                  placeholder="block-slug"
                  value={slug}
                  onChange={(e) => {
                    setAutoSlug(false);
                    setSlug(e.target.value);
                  }}
                  disabled={autoSlug}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description" className="text-sm font-medium text-slate-700">Description</Label>
                <Textarea
                  id="description"
                  placeholder="A brief description of this block type"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Icon</Label>
                <div className="grid grid-cols-6 gap-2 sm:grid-cols-8 lg:grid-cols-6 xl:grid-cols-8">
                  {ICON_OPTIONS.map((opt) => {
                    const IconComp = opt.icon;
                    const isSelected = icon === opt.value;
                    return (
                      <button
                        key={opt.value}
                        type="button"
                        onClick={() => setIcon(opt.value)}
                        title={opt.label}
                        className={`flex flex-col items-center gap-1 rounded-lg border-2 p-2.5 transition-all ${
                          isSelected
                            ? "border-indigo-500 bg-indigo-50 text-indigo-700 shadow-sm"
                            : "border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:bg-slate-50"
                        }`}
                      >
                        <IconComp className="h-5 w-5" />
                        <span className="text-[10px] font-medium leading-none">{opt.label}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Fields */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">Fields</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-6 pt-0">
              {fields.length === 0 && !showAddField && (
                <p className="text-sm text-slate-400 text-center py-4">
                  No fields defined yet. Add fields to define the structure of this block type.
                </p>
              )}

              {fields.length > 0 && (
                <div className="space-y-2">
                  {fields.map((field, index) => (
                    <div
                      key={field.key + index}
                      className={`rounded-lg border ${editingFieldIndex === index ? "border-indigo-300 bg-indigo-50/30" : "border-slate-200 bg-slate-50"}`}
                    >
                      <div className="flex items-center gap-3 px-4 py-3">
                        <div className="flex flex-col gap-0.5">
                          <button type="button" onClick={() => handleMoveField(index, "up")} disabled={index === 0} className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed">
                            <ChevronUp className="h-4 w-4" />
                          </button>
                          <button type="button" onClick={() => handleMoveField(index, "down")} disabled={index === fields.length - 1} className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed">
                            <ChevronDown className="h-4 w-4" />
                          </button>
                        </div>
                        <button type="button" className="flex-1 min-w-0 text-left" onClick={() => setEditingFieldIndex(editingFieldIndex === index ? null : index)}>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-slate-800">{field.label}</span>
                            <span className="text-xs text-slate-400 font-mono">{field.key}</span>
                          </div>
                        </button>
                        <Badge className={`${fieldTypeBadgeClass(field.type)} border-0 text-xs`}>{field.type}</Badge>
                        {field.required && <Badge className="bg-red-100 text-red-600 hover:bg-red-100 border-0 text-xs">Required</Badge>}
                        {field.help_text && <Badge className="bg-slate-100 text-slate-500 hover:bg-slate-100 border-0 text-xs" title={field.help_text}>?</Badge>}
                        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 text-slate-400 hover:text-indigo-600 shrink-0" onClick={() => setEditingFieldIndex(editingFieldIndex === index ? null : index)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 text-red-500 hover:text-red-600 shrink-0" onClick={() => handleRemoveField(index)}>
                          <X className="h-4 w-4" />
                        </Button>
                      </div>
                      {editingFieldIndex === index && (
                        <div className="border-t border-indigo-200 px-4 py-3 space-y-3">
                          <div className="grid gap-3 sm:grid-cols-2">
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Label</Label>
                              <Input value={field.label} onChange={(e) => updateField(index, { label: e.target.value })} className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                            </div>
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Key</Label>
                              <Input value={field.key} onChange={(e) => updateField(index, { key: e.target.value })} className="h-8 text-sm font-mono rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                            </div>
                          </div>
                          <div className="grid gap-3 sm:grid-cols-2">
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Type</Label>
                              <FieldTypePicker value={field.type} onValueChange={(v) => updateField(index, { type: v as NodeTypeField["type"] })} compact />
                            </div>
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">&nbsp;</Label>
                              <div className="flex items-center gap-2 h-8">
                                <input type="checkbox" checked={!!field.required} onChange={(e) => updateField(index, { required: e.target.checked || undefined })} className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500" />
                                <span className="text-sm text-slate-700">Required</span>
                              </div>
                            </div>
                          </div>
                          {field.type === "select" && (
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Options (comma-separated)</Label>
                              <Input value={(field.options || []).join(", ")} onChange={(e) => updateField(index, { options: e.target.value.split(",").map((o) => o.trim()).filter(Boolean) })} className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
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
                                <Label className="text-xs font-medium text-slate-600">Node Type Filter</Label>
                                <Input
                                  value={field.node_type_filter || ""}
                                  onChange={(e) => updateField(index, { node_type_filter: e.target.value })}
                                  placeholder="e.g. page, post, product (empty = all)"
                                  className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                                />
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">&nbsp;</Label>
                                <div className="flex items-center gap-2 h-8">
                                  <input type="checkbox" checked={!!field.multiple} onChange={(e) => updateField(index, { multiple: e.target.checked })} className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500" />
                                  <span className="text-sm text-slate-700">Allow multiple</span>
                                </div>
                              </div>
                            </div>
                          )}
                          {/* Placeholder */}
                          {["text", "textarea", "number", "email", "url"].includes(field.type) && (
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Placeholder</Label>
                              <Input value={field.placeholder || ""} onChange={(e) => updateField(index, { placeholder: e.target.value || undefined })} placeholder="Placeholder text" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                            </div>
                          )}
                          {/* Default Value */}
                          {!["group", "repeater"].includes(field.type) && (
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Default Value</Label>
                              <Input value={field.default_value || ""} onChange={(e) => updateField(index, { default_value: e.target.value || undefined })} placeholder="Default value" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                            </div>
                          )}
                          {/* Help Text */}
                          <div className="space-y-1">
                            <Label className="text-xs font-medium text-slate-600">Help Text</Label>
                            <Input value={field.help_text || ""} onChange={(e) => updateField(index, { help_text: e.target.value || undefined })} placeholder="Instructions for content editors" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                          </div>
                          {/* Options for radio/checkbox */}
                          {(field.type === "radio" || field.type === "checkbox") && (
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Options (comma-separated)</Label>
                              <Input value={(field.options || []).join(", ")} onChange={(e) => updateField(index, { options: e.target.value.split(",").map((o) => o.trim()).filter(Boolean) })} className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                            </div>
                          )}
                          {/* Number/Range constraints */}
                          {(field.type === "number" || field.type === "range") && (
                            <div className="grid gap-3 sm:grid-cols-3">
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Min</Label>
                                <Input type="number" value={field.min ?? ""} onChange={(e) => updateField(index, { min: e.target.value ? Number(e.target.value) : undefined })} placeholder="Min" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Max</Label>
                                <Input type="number" value={field.max ?? ""} onChange={(e) => updateField(index, { max: e.target.value ? Number(e.target.value) : undefined })} placeholder="Max" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Step</Label>
                                <Input type="number" value={field.step ?? ""} onChange={(e) => updateField(index, { step: e.target.value ? Number(e.target.value) : undefined })} placeholder="Step" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                            </div>
                          )}
                          {/* Text length constraints */}
                          {(field.type === "text" || field.type === "textarea") && (
                            <div className="grid gap-3 sm:grid-cols-2">
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Min Length</Label>
                                <Input type="number" value={field.min_length ?? ""} onChange={(e) => updateField(index, { min_length: e.target.value ? Number(e.target.value) : undefined })} placeholder="No min" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Max Length</Label>
                                <Input type="number" value={field.max_length ?? ""} onChange={(e) => updateField(index, { max_length: e.target.value ? Number(e.target.value) : undefined })} placeholder="No max" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                            </div>
                          )}
                          {/* Textarea rows */}
                          {field.type === "textarea" && (
                            <div className="space-y-1">
                              <Label className="text-xs font-medium text-slate-600">Rows</Label>
                              <Input type="number" value={field.rows ?? ""} onChange={(e) => updateField(index, { rows: e.target.value ? Number(e.target.value) : undefined })} placeholder="4 (default)" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                            </div>
                          )}
                          {/* Prepend / Append */}
                          {["text", "number", "email", "url"].includes(field.type) && (
                            <div className="grid gap-3 sm:grid-cols-2">
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Prepend</Label>
                                <Input value={field.prepend || ""} onChange={(e) => updateField(index, { prepend: e.target.value || undefined })} placeholder="e.g. $" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Append</Label>
                                <Input value={field.append || ""} onChange={(e) => updateField(index, { append: e.target.value || undefined })} placeholder="e.g. px" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                            </div>
                          )}
                          {/* File options */}
                          {field.type === "file" && (
                            <div className="grid gap-3 sm:grid-cols-2">
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">Allowed Types</Label>
                                <Input value={field.allowed_types || ""} onChange={(e) => updateField(index, { allowed_types: e.target.value || undefined })} placeholder="pdf, doc, zip" className="h-8 text-sm rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20" />
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs font-medium text-slate-600">&nbsp;</Label>
                                <div className="flex items-center gap-2 h-8">
                                  <input type="checkbox" checked={!!field.multiple} onChange={(e) => updateField(index, { multiple: e.target.checked })} className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500" />
                                  <span className="text-sm text-slate-700">Multiple files</span>
                                </div>
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {/* Add field form */}
              {showAddField && (
                <>
                  <Separator />
                  <div className="space-y-4 rounded-lg border border-indigo-200 bg-indigo-50/50 p-4">
                    <p className="text-sm font-semibold text-slate-700">New Field</p>
                    <div className="grid gap-4 sm:grid-cols-2">
                      <div className="space-y-2">
                        <Label className="text-sm font-medium text-slate-700">Label</Label>
                        <Input
                          placeholder="e.g. Title, Content, Image URL"
                          value={newFieldLabel}
                          onChange={(e) => setNewFieldLabel(e.target.value)}
                          className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                        />
                      </div>
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <Label className="text-sm font-medium text-slate-700">Key</Label>
                          <button
                            type="button"
                            className="text-xs text-indigo-600 hover:underline"
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
                        <div className="flex items-center gap-2 h-9">
                          <input
                            type="checkbox"
                            id="new-field-required"
                            checked={newFieldRequired}
                            onChange={(e) => setNewFieldRequired(e.target.checked)}
                            className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                          />
                          <Label htmlFor="new-field-required" className="text-sm font-medium text-slate-700 cursor-pointer">
                            Required
                          </Label>
                        </div>
                      </div>
                    </div>

                    {newFieldType === "select" && (
                      <div className="space-y-2">
                        <Label className="text-sm font-medium text-slate-700">Options (comma-separated)</Label>
                        <Input
                          placeholder="e.g. Option A, Option B, Option C"
                          value={newFieldOptions}
                          onChange={(e) => setNewFieldOptions(e.target.value)}
                          className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
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
                          <Label className="text-sm font-medium text-slate-700">Node Type Filter</Label>
                          <Input
                            value={newFieldNodeTypeFilter}
                            onChange={(e) => setNewFieldNodeTypeFilter(e.target.value)}
                            placeholder="e.g. page, product (empty = all)"
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">&nbsp;</Label>
                          <div className="flex items-center gap-2 h-9">
                            <input type="checkbox" checked={newFieldMultiple} onChange={(e) => setNewFieldMultiple(e.target.checked)} className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500" />
                            <span className="text-sm text-slate-700">Allow multiple selection</span>
                          </div>
                        </div>
                      </div>
                    )}

                    {/* Placeholder - for text, textarea, number, email, url */}
                    {["text", "textarea", "number", "email", "url"].includes(newFieldType) && (
                      <div className="space-y-2">
                        <Label className="text-sm font-medium text-slate-700">Placeholder</Label>
                        <Input
                          placeholder="Placeholder text shown when empty"
                          value={newFieldPlaceholder}
                          onChange={(e) => setNewFieldPlaceholder(e.target.value)}
                          className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                        />
                      </div>
                    )}

                    {/* Default Value */}
                    {!["group", "repeater"].includes(newFieldType) && (
                      <div className="space-y-2">
                        <Label className="text-sm font-medium text-slate-700">Default Value</Label>
                        <Input
                          placeholder="Default value for new content"
                          value={newFieldDefaultValue}
                          onChange={(e) => setNewFieldDefaultValue(e.target.value)}
                          className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                        />
                      </div>
                    )}

                    {/* Help Text */}
                    <div className="space-y-2">
                      <Label className="text-sm font-medium text-slate-700">Help Text</Label>
                      <Input
                        placeholder="Instructions shown below the field"
                        value={newFieldHelpText}
                        onChange={(e) => setNewFieldHelpText(e.target.value)}
                        className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                      />
                    </div>

                    {/* Options for radio and checkbox */}
                    {(newFieldType === "radio" || newFieldType === "checkbox") && (
                      <div className="space-y-2">
                        <Label className="text-sm font-medium text-slate-700">Options (comma-separated)</Label>
                        <Input
                          placeholder="e.g. Option A, Option B, Option C"
                          value={newFieldOptions}
                          onChange={(e) => setNewFieldOptions(e.target.value)}
                          className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                        />
                      </div>
                    )}

                    {/* Number / Range options */}
                    {(newFieldType === "number" || newFieldType === "range") && (
                      <div className="grid gap-4 sm:grid-cols-3">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Min</Label>
                          <Input
                            type="number"
                            placeholder="0"
                            value={newFieldMin}
                            onChange={(e) => setNewFieldMin(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Max</Label>
                          <Input
                            type="number"
                            placeholder="100"
                            value={newFieldMax}
                            onChange={(e) => setNewFieldMax(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Step</Label>
                          <Input
                            type="number"
                            placeholder="1"
                            value={newFieldStep}
                            onChange={(e) => setNewFieldStep(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                      </div>
                    )}

                    {/* Text length constraints */}
                    {(newFieldType === "text" || newFieldType === "textarea") && (
                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Min Length</Label>
                          <Input
                            type="number"
                            placeholder="No minimum"
                            value={newFieldMinLength}
                            onChange={(e) => setNewFieldMinLength(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Max Length</Label>
                          <Input
                            type="number"
                            placeholder="No maximum"
                            value={newFieldMaxLength}
                            onChange={(e) => setNewFieldMaxLength(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                      </div>
                    )}

                    {/* Textarea rows */}
                    {newFieldType === "textarea" && (
                      <div className="space-y-2">
                        <Label className="text-sm font-medium text-slate-700">Rows</Label>
                        <Input
                          type="number"
                          placeholder="4 (default)"
                          value={newFieldRows}
                          onChange={(e) => setNewFieldRows(e.target.value)}
                          className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                        />
                      </div>
                    )}

                    {/* Prepend / Append */}
                    {["text", "number", "email", "url"].includes(newFieldType) && (
                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Prepend</Label>
                          <Input
                            placeholder="e.g. $, https://"
                            value={newFieldPrepend}
                            onChange={(e) => setNewFieldPrepend(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Append</Label>
                          <Input
                            placeholder="e.g. px, kg, %"
                            value={newFieldAppend}
                            onChange={(e) => setNewFieldAppend(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                      </div>
                    )}

                    {/* File allowed types */}
                    {newFieldType === "file" && (
                      <div className="space-y-4">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium text-slate-700">Allowed File Types</Label>
                          <Input
                            placeholder="e.g. pdf, doc, zip (empty = all)"
                            value={newFieldAllowedTypes}
                            onChange={(e) => setNewFieldAllowedTypes(e.target.value)}
                            className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                          />
                        </div>
                        <div className="flex items-center gap-2 h-9">
                          <input type="checkbox" checked={newFieldMultiple} onChange={(e) => setNewFieldMultiple(e.target.checked)} className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500" />
                          <span className="text-sm text-slate-700">Allow multiple files</span>
                        </div>
                      </div>
                    )}

                    <div className="flex gap-2">
                      <Button
                        type="button"
                        className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg"
                        onClick={handleAddField}
                      >
                        Add Field
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        className="rounded-lg border-slate-300"
                        onClick={resetAddFieldForm}
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>
                </>
              )}

              {!showAddField && (
                <Button
                  type="button"
                  variant="outline"
                  className="w-full rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
                  onClick={() => setShowAddField(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Field
                </Button>
              )}
            </CardContent>
          </Card>

          {/* HTML Template + Test Data + Preview */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader className="pb-0">
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg font-semibold text-slate-900">Template &amp; Preview</CardTitle>
              </div>
              {/* Tabs */}
              <div className="flex gap-0 border-b border-slate-200 -mx-6 px-6 mt-3">
                {(["template", "testdata", "preview"] as const).map((tab) => (
                  <button
                    key={tab}
                    type="button"
                    onClick={() => setActiveTemplateTab(tab)}
                    className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                      activeTemplateTab === tab
                        ? "border-indigo-600 text-indigo-700"
                        : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300"
                    }`}
                  >
                    {tab === "template" ? "Template" : tab === "testdata" ? "Test Data" : "Preview"}
                  </button>
                ))}
              </div>
            </CardHeader>
            <CardContent className="space-y-4 p-6 pt-4">
              {/* Template tab */}
              {activeTemplateTab === "template" && (
                <>
                  <p className="text-sm text-slate-500">
                    Define how this block renders on the public site. Use Go template syntax with field keys as variables.
                  </p>
                  {fields.length > 0 && (
                    <div className="flex flex-wrap gap-1.5">
                      {fields.map((f) => (
                        <button
                          key={f.key}
                          type="button"
                          className="rounded bg-slate-100 px-2 py-0.5 text-xs font-mono text-indigo-600 hover:bg-indigo-100 transition-colors cursor-pointer"
                          onClick={() => {
                            const tag = "{{." + f.key + "}}";
                            setHtmlTemplate((prev) => prev + tag);
                          }}
                          title={`Click to insert {{.${f.key}}}`}
                        >
                          {"{{." + f.key + "}}"}
                        </button>
                      ))}
                    </div>
                  )}
                  {source === "theme" && (
                    <div className="rounded-lg bg-amber-50 border border-amber-200 px-3 py-2 text-xs text-amber-700">
                      Theme-defined block — HTML template is managed by the theme and cannot be edited here.
                    </div>
                  )}
                  <CodeEditor
                    value={htmlTemplate}
                    onChange={setHtmlTemplate}
                    disabled={source === "theme"}
                    height="350px"
                    variables={fields.map((f) => f.key)}
                    placeholder={"<div class='my-block'>\n  <h2>{{.title}}</h2>\n  <p>{{.content}}</p>\n</div>"}
                  />
                </>
              )}

              {/* Test Data tab */}
              {activeTemplateTab === "testdata" && (
                <>
                  <p className="text-sm text-slate-500">
                    Fill in sample values for each field to preview how the block will render.
                  </p>
                  {fields.length === 0 ? (
                    <p className="text-sm text-slate-400 text-center py-6">Add fields first to enter test data.</p>
                  ) : (
                    <div className="space-y-3">
                      {fields.map((f) => (
                        <div key={f.key} className="space-y-1.5">
                          <Label className="text-sm font-medium text-slate-700">
                            {f.label}
                            <span className="ml-2 text-xs text-slate-400 font-mono">{f.key}</span>
                          </Label>
                          <CustomFieldInput
                            field={f}
                            value={testData[f.key]}
                            onChange={(val) => setTestData((prev) => ({ ...prev, [f.key]: val }))}
                          />
                        </div>
                      ))}
                      <div className="flex gap-2 pt-2">
                        <Button
                          type="button"
                          variant="outline"
                          className="text-sm rounded-lg border-slate-300 opacity-50 cursor-not-allowed"
                          disabled
                          title="AI-powered auto-fill coming soon"
                        >
                          Auto-fill with AI (coming soon)
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          className="text-sm text-slate-500"
                          onClick={() => setTestData({})}
                        >
                          Clear all
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              )}

              {/* Preview tab */}
              {activeTemplateTab === "preview" && (
                <>
                  <div className="flex items-center justify-between">
                    <p className="text-sm text-slate-500">
                      Live preview of the block with your test data.
                    </p>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="text-xs rounded-lg border-slate-300"
                      onClick={async () => {
                        setPreviewLoading(true);
                        try {
                          const html = await previewBlockTemplate(htmlTemplate, testData);
                          setPreviewHtml(html);
                        } catch {
                          setPreviewHtml('<div class="text-red-500 text-sm">Failed to render preview</div>');
                        } finally {
                          setPreviewLoading(false);
                        }
                      }}
                    >
                      {previewLoading ? "Rendering..." : "Refresh Preview"}
                    </Button>
                  </div>
                  {!htmlTemplate.trim() ? (
                    <div className="text-center py-12 text-sm text-slate-400">
                      No template defined. Write a template first and add test data.
                    </div>
                  ) : previewHtml === null ? (
                    <div className="text-center py-12">
                      <Button
                        type="button"
                        className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg"
                        onClick={async () => {
                          setPreviewLoading(true);
                          try {
                            const html = await previewBlockTemplate(htmlTemplate, testData);
                            setPreviewHtml(html);
                          } catch {
                            setPreviewHtml('<div class="text-red-500 text-sm">Failed to render preview</div>');
                          } finally {
                            setPreviewLoading(false);
                          }
                        }}
                      >
                        {previewLoading ? "Rendering..." : "Generate Preview"}
                      </Button>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {/* Note: Preview HTML is rendered server-side from admin-authored templates and test data - safe for admin use */}
                      <div
                        className="rounded-lg border border-slate-200 bg-white p-6 min-h-[120px]"
                        dangerouslySetInnerHTML={{ __html: previewHtml }}
                      />
                      <details className="text-xs">
                        <summary className="cursor-pointer text-slate-400 hover:text-slate-600">View raw HTML</summary>
                        <pre className="mt-2 rounded-lg bg-slate-900 text-slate-300 p-4 overflow-x-auto text-xs font-mono">{previewHtml}</pre>
                      </details>
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">Save</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-6 pt-0">
              <Button
                type="submit"
                className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
                disabled={saving || isManaged}
              >
                <Save className="mr-2 h-4 w-4" />
                {saving ? "Saving..." : isEdit ? "Update Block Type" : "Create Block Type"}
              </Button>
            </CardContent>
          </Card>

          {/* Output Caching */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">Performance</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 p-6 pt-0">
              <label className="flex items-center justify-between cursor-pointer">
                <div>
                  <p className="text-sm font-medium text-slate-700">Cache rendered output</p>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Disable for blocks showing real-time data
                  </p>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-checked={cacheOutput}
                  onClick={() => setCacheOutput(!cacheOutput)}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    cacheOutput ? "bg-indigo-600" : "bg-slate-300"
                  }`}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                      cacheOutput ? "translate-x-6" : "translate-x-1"
                    }`}
                  />
                </button>
              </label>
              {!cacheOutput && (
                <p className="text-xs text-amber-600 bg-amber-50 border border-amber-200 rounded-lg p-2">
                  This block will be re-rendered on every page load. Use for dynamic content like latest posts, live counters, or user-specific data.
                </p>
              )}
            </CardContent>
          </Card>

          {/* Actions (edit mode only) */}
          {isEdit && !isManaged && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <CardHeader>
                <CardTitle className="text-lg font-semibold text-slate-900">Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 p-6 pt-0">
                <Button
                  type="button"
                  variant="outline"
                  className="w-full bg-red-50 text-red-700 border-red-200 hover:bg-red-100 rounded-lg font-medium"
                  onClick={() => setShowDelete(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete Block Type
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Info (edit mode) */}
          {isEdit && originalBlockType && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <CardContent className="space-y-2 p-6 text-sm text-slate-500">
                <div className="flex justify-between">
                  <span>Created</span>
                  <span>
                    {new Date(originalBlockType.created_at).toLocaleDateString()}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>Updated</span>
                  <span>
                    {new Date(originalBlockType.updated_at).toLocaleDateString()}
                  </span>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </form>

      {/* Delete dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Block Type</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{originalBlockType?.label}&quot;?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDelete(false)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Detach dialog */}
      <Dialog open={showDetach} onOpenChange={setShowDetach}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Detach from {source === "theme" ? "Theme" : "Extension"}</DialogTitle>
            <DialogDescription>
              This will create an editable copy of this block type. The {source} version will no longer
              be used. You can always re-sync from the {source} later.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDetach(false)} disabled={detaching}>
              Cancel
            </Button>
            <Button
              onClick={handleDetach}
              disabled={detaching}
              className="bg-amber-600 hover:bg-amber-700 text-white"
            >
              {detaching ? "Detaching..." : "Detach"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
