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
  Code,
  Eye,
  FileCode,
  RefreshCw,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  getBlockType,
  createBlockType,
  updateBlockType,
  deleteBlockType,
  previewBlockTemplate,
  type BlockType,
  type NodeTypeField,
} from "@/api/client";
import { toast } from "sonner";
import { usePageMeta } from "@/components/layout/page-meta";
import FieldTypePicker from "@/components/ui/field-type-picker";
import SubFieldsEditor from "@/components/ui/sub-fields-editor";
import { CodeWindow } from "@/components/ui/code-window";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

function keyify(text: string) {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function BlockTypeEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const [label, setLabel] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [icon, setIcon] = useState("boxes");
  const [fields, setFields] = useState<NodeTypeField[]>([]);
  const [htmlTemplate, setHtmlTemplate] = useState("");
  const [testData, setTestData] = useState<Record<string, unknown>>({});
  const [cacheOutput, setCacheOutput] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  usePageMeta([
    "Block Types",
    isEdit ? (label ? `Edit "${label}"` : "Edit") : "New Block Type",
  ]);

  // New field form state
  const [addingField, setAddingField] = useState(false);
  const [newFieldLabel, setNewFieldLabel] = useState("");
  const [newFieldKey, setNewFieldKey] = useState("");
  const [newFieldType, setNewFieldType] = useState<NodeTypeField["type"]>("text");
  const [newFieldRequired, setNewFieldRequired] = useState(false);
  const [newFieldOptions, setNewFieldOptions] = useState("");
  const [newFieldPlaceholder, setNewFieldPlaceholder] = useState("");
  const [newFieldDefaultValue, setNewFieldDefaultValue] = useState("");
  const [newFieldHelpText, setNewFieldHelpText] = useState("");
  const [newFieldSubFields, setNewFieldSubFields] = useState<NodeTypeField[]>([]);
  const [newFieldNodeTypeFilter, setNewFieldNodeTypeFilter] = useState("");
  const [newFieldMultiple, setNewFieldMultiple] = useState(false);
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

  // Preview state
  const [previewHtml, setPreviewHtml] = useState("");
  const [previewHead, setPreviewHead] = useState("");
  const [previewBodyClass, setPreviewBodyClass] = useState("");
  const [previewLoading, setPreviewLoading] = useState(false);

  useEffect(() => {
    if (isEdit && id) {
      getBlockType(id)
        .then((bt) => {
          setLabel(bt.label);
          setSlug(bt.slug);
          setDescription(bt.description || "");
          setIcon(bt.icon || "boxes");
          setFields(bt.field_schema || []);
          setHtmlTemplate(bt.html_template || "");
          setTestData(bt.test_data || {});
          setCacheOutput(bt.cache_output);
          setAutoSlug(false);
        })
        .catch(() => {
          toast.error("Failed to load block type");
          navigate("/admin/block-types");
        })
        .finally(() => setLoading(false));
    }
  }, [isEdit, id, navigate]);

  const handleLabelChange = (val: string) => {
    setLabel(val);
    if (autoSlug) {
      setSlug(keyify(val));
    }
  };

  const handleNewFieldLabelChange = (val: string) => {
    setNewFieldLabel(val);
    if (autoFieldKey) {
      setNewFieldKey(keyify(val).replace(/-/g, "_"));
    }
  };

  function resetAddFieldForm() {
    setNewFieldLabel("");
    setNewFieldKey("");
    setNewFieldType("text");
    setNewFieldRequired(false);
    setNewFieldOptions("");
    setNewFieldPlaceholder("");
    setNewFieldDefaultValue("");
    setNewFieldHelpText("");
    setNewFieldSubFields([]);
    setNewFieldNodeTypeFilter("");
    setNewFieldMultiple(false);
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
    setAddingField(false);
  }

  function handleAddField() {
    if (!newFieldLabel.trim() || !newFieldKey.trim()) {
      toast.error("Label and key are required");
      return;
    }
    if (fields.some((f) => f.key === newFieldKey)) {
      toast.error("A field with this key already exists");
      return;
    }

    const sf: NodeTypeField = {
      name: newFieldKey,
      key: newFieldKey,
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

    setFields([...fields, sf]);
    resetAddFieldForm();
  }

  const [editingFieldIndex, setEditingFieldIndex] = useState<number | null>(null);

  function handleRemoveField(index: number) {
    setFields(fields.filter((_, i) => i !== index));
    if (editingFieldIndex === index) setEditingFieldIndex(null);
  }

  function handleMoveField(index: number, direction: "up" | "down") {
    const newFields = [...fields];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newFields.length) return;
    [newFields[index], newFields[targetIndex]] = [newFields[targetIndex], newFields[index]];
    setFields(newFields);
    if (editingFieldIndex === index) setEditingFieldIndex(targetIndex);
    else if (editingFieldIndex === targetIndex) setEditingFieldIndex(index);
  }

  function updateField(index: number, updates: Partial<NodeTypeField>) {
    setFields(fields.map((f, i) => i === index ? { ...f, ...updates } : f));
  }

  const handleSave = async (e: FormEvent) => {
    e.preventDefault();
    if (!label || !slug) {
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
      cache_output: cacheOutput,
    };

    setSaving(true);
    try {
      if (isEdit && id) {
        await updateBlockType(id, data);
        toast.success("Block type updated");
      } else {
        await createBlockType(data);
        toast.success("Block type created");
        navigate("/admin/block-types");
      }
    } catch (err: any) {
      toast.error(err.message || "Failed to save block type");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteBlockType(id);
      toast.success("Block type deleted");
      navigate("/admin/block-types");
    } catch (err: any) {
      toast.error(err.message || "Failed to delete block type");
    } finally {
      setDeleting(false);
    }
  };

  const handlePreview = async () => {
    setPreviewLoading(true);
    try {
      const res = await previewBlockTemplate(htmlTemplate, testData);
      setPreviewHtml(res.html);
      setPreviewHead(res.head);
      setPreviewBodyClass(res.body_class);
    } catch (err: any) {
      toast.error("Preview failed: " + (err.message || "unknown error"));
    } finally {
      setPreviewLoading(false);
    }
  };

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
          <Button variant="ghost" size="icon" asChild className="rounded-lg">
            <Link to="/admin/block-types">
              <ArrowLeft className="h-5 w-5" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">
              {isEdit ? `Edit Block: ${label}` : "Create Block Type"}
            </h1>
            <p className="text-sm text-slate-500">
              Build custom content components using liquid-like templates.
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          {isEdit && (
            <Button
              variant="outline"
              className="text-red-600 hover:bg-red-50 hover:text-red-700"
              onClick={() => setShowDeleteDialog(true)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </Button>
          )}
          <Button onClick={handleSave} disabled={saving} className="bg-indigo-600 hover:bg-indigo-700">
            {saving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Save Block Type
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Left Column: Config */}
        <div className="lg:col-span-1 space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="General Configuration" />
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="label">Display Label</Label>
                <Input
                  id="label"
                  value={label}
                  onChange={(e) => handleLabelChange(e.target.value)}
                  placeholder="e.g. Hero Section"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="slug">Slug (Unique Key)</Label>
                <Input
                  id="slug"
                  value={slug}
                  onChange={(e) => {
                    setSlug(keyify(e.target.value));
                    setAutoSlug(false);
                  }}
                  disabled={isEdit}
                  placeholder="hero-section"
                  className="font-mono text-sm"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Brief summary of what this block does"
                  rows={2}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="icon">Icon Slug</Label>
                <Input
                  id="icon"
                  value={icon}
                  onChange={(e) => setIcon(e.target.value)}
                  placeholder="boxes, image, text..."
                  className="font-mono text-sm"
                />
              </div>
              <div className="flex items-center justify-between pt-2">
                <div className="space-y-0.5">
                  <Label htmlFor="cache">Cache Output</Label>
                  <p className="text-[11px] text-slate-400">Improve performance by caching rendered HTML</p>
                </div>
                <input
                  type="checkbox"
                  id="cache"
                  checked={cacheOutput}
                  onChange={(e) => setCacheOutput(e.target.checked)}
                  className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                />
              </div>
            </CardContent>
          </Card>

          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Fields Definition" />
            <CardContent className="space-y-6">
              <p className="text-xs text-slate-500">Configure the data structure for this block.</p>
              {/* Field list */}
              <div className="space-y-3">
                {fields.length === 0 ? (
                  <div className="text-center py-8 rounded-lg border border-dashed border-slate-200 text-slate-400 text-sm italic">
                    No fields defined yet.
                  </div>
                ) : (
                  fields.map((field, index) => (
                    <div key={index} className={`rounded-lg border ${editingFieldIndex === index ? "border-indigo-300 bg-indigo-50/30" : "border-slate-200 bg-white"}`}>
                      <div className="flex items-center gap-2 p-2 px-3">
                        <div className="flex flex-col gap-0.5">
                          <button type="button" onClick={() => handleMoveField(index, "up")} disabled={index === 0} className="text-slate-400 hover:text-slate-600 disabled:opacity-30">
                            <ChevronUp className="h-3.5 w-3.5" />
                          </button>
                          <button type="button" onClick={() => handleMoveField(index, "down")} disabled={index === fields.length - 1} className="text-slate-400 hover:text-slate-600 disabled:opacity-30">
                            <ChevronDown className="h-3.5 w-3.5" />
                          </button>
                        </div>
                        <div className="flex-1 min-w-0" onClick={() => setEditingFieldIndex(editingFieldIndex === index ? null : index)}>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-slate-800">{field.label}</span>
                            <span className="text-[10px] text-slate-400 font-mono">{field.key}</span>
                          </div>
                          <div className="text-[10px] text-slate-500 font-medium uppercase">{field.type}</div>
                        </div>
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-slate-400 hover:text-indigo-600" onClick={() => setEditingFieldIndex(editingFieldIndex === index ? null : index)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-red-400 hover:text-red-600" onClick={() => handleRemoveField(index)}>
                          <X className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                      {/* Field Editor (inline) */}
                      {editingFieldIndex === index && (
                        <div className="border-t border-indigo-100 p-3 space-y-3 bg-white rounded-b-lg">
                          <div className="grid gap-2 sm:grid-cols-2">
                            <div className="space-y-1">
                              <Label className="text-[10px] text-slate-500 uppercase">Label</Label>
                              <Input value={field.label} onChange={e => updateField(index, { label: e.target.value })} className="h-8 text-sm" />
                            </div>
                            <div className="space-y-1">
                              <Label className="text-[10px] text-slate-500 uppercase">Key</Label>
                              <Input value={field.key} onChange={e => updateField(index, { key: e.target.value })} className="h-8 text-sm font-mono" />
                            </div>
                          </div>
                          <div className="space-y-1">
                            <Label className="text-[10px] text-slate-500 uppercase">Help Text</Label>
                            <Input value={field.help || ""} onChange={e => updateField(index, { help: e.target.value })} className="h-8 text-sm" />
                          </div>
                          {(field.type === "group" || field.type === "repeater") && (
                            <SubFieldsEditor
                              value={field.sub_fields || []}
                              onChange={(sub) => updateField(index, { sub_fields: sub })}
                              label={field.type === "group" ? "Group Fields" : "Row Fields"}
                            />
                          )}
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>

              {/* Add field form */}
              {addingField ? (
                <div className="p-4 rounded-xl border border-indigo-200 bg-indigo-50/50 space-y-4">
                  <div className="space-y-2">
                    <Label className="text-xs font-semibold">Field Label</Label>
                    <Input
                      placeholder="e.g. Header Text"
                      value={newFieldLabel}
                      onChange={(e) => handleNewFieldLabelChange(e.target.value)}
                      className="h-9"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label className="text-xs font-semibold">Field Key</Label>
                    <Input
                      placeholder="e.g. header_text"
                      value={newFieldKey}
                      onChange={(e) => {
                        setNewFieldKey(e.target.value.replace(/[^a-z0-9_]/g, ""));
                        setAutoFieldKey(false);
                      }}
                      className="h-9 font-mono"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label className="text-xs font-semibold">Field Type</Label>
                    <FieldTypePicker value={newFieldType} onValueChange={(v) => setNewFieldType(v as NodeTypeField["type"])} />
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" className="flex-1 bg-indigo-600" onClick={handleAddField}>Add</Button>
                    <Button size="sm" variant="ghost" className="flex-1" onClick={resetAddFieldForm}>Cancel</Button>
                  </div>
                </div>
              ) : (
                <Button variant="outline" className="w-full rounded-lg border-dashed border-slate-300 text-slate-500" onClick={() => setAddingField(true)}>
                  <Plus className="mr-2 h-4 w-4" /> Add Field
                </Button>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Right Column: Code & Preview */}
        <div className="lg:col-span-2 space-y-6">
          <Tabs defaultValue="template" className="w-full">
            <TabsList className="grid w-full grid-cols-3 rounded-xl bg-slate-100 p-1">
              <TabsTrigger value="template" className="rounded-lg data-[state=active]:bg-white data-[state=active]:shadow-sm">
                <FileCode className="mr-2 h-4 w-4" /> Template
              </TabsTrigger>
              <TabsTrigger value="test-data" className="rounded-lg data-[state=active]:bg-white data-[state=active]:shadow-sm">
                <Code className="mr-2 h-4 w-4" /> Test Data
              </TabsTrigger>
              <TabsTrigger value="preview" className="rounded-lg data-[state=active]:bg-white data-[state=active]:shadow-sm" onClick={handlePreview}>
                <Eye className="mr-2 h-4 w-4" /> Preview
              </TabsTrigger>
            </TabsList>

            <TabsContent value="template" className="mt-4 ring-offset-white focus-visible:outline-none">
              <CodeWindow
                title="HTML / Go Template"
                value={htmlTemplate}
                onChange={setHtmlTemplate}
                height="500px"
              />
            </TabsContent>

            <TabsContent value="test-data" className="mt-4 ring-offset-white focus-visible:outline-none">
              <CodeWindow
                title="Mock Content (JSON)"
                value={JSON.stringify(testData, null, 2)}
                onChange={(v) => {
                  try { setTestData(JSON.parse(v)); } catch {}
                }}
                height="500px"
              />
            </TabsContent>

            <TabsContent value="preview" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-slate-200 shadow-sm h-[500px] flex flex-col">
                <SectionHeader
                  title="Rendered Preview"
                  actions={
                    <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={handlePreview} disabled={previewLoading}>
                      {previewLoading ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : <RefreshCw className="mr-1 h-3 w-3" />}
                      Refresh
                    </Button>
                  }
                />
                <div className="flex-1 overflow-hidden bg-white">
                  {previewHtml ? (
                    <iframe
                      title="Block preview"
                      className="h-full w-full border-0 bg-white"
                      sandbox="allow-same-origin allow-scripts"
                      srcDoc={`<!doctype html><html><head><meta charset="utf-8">${previewHead || '<script src="https://cdn.tailwindcss.com"></script>'}<style>body{margin:0;padding:1rem;}</style></head><body class="${previewBodyClass}">${previewHtml}</body></html>`}
                    />
                  ) : (
                    <div className="h-full flex flex-col items-center justify-center text-slate-400 space-y-3">
                      <Eye className="h-10 w-10 opacity-20" />
                      <p className="text-sm">Click refresh to render template with test data</p>
                    </div>
                  )}
                </div>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </div>

      {/* Delete dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Block Type?</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete <strong>{label}</strong>? This will break any existing nodes using this block type. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setShowDeleteDialog(false)} disabled={deleting}>Cancel</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "Deleting..." : "Delete Permanently"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
