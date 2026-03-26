import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Globe,
  Trash2,
  Home,
  Loader2,
  Plus,
  ChevronUp,
  ChevronDown,
  ChevronRight,
  X,
  LayoutTemplate,
  Square,
  Type,
  Image,
  MousePointerClick,
  Images,
  Play,
  List,
  Quote,
  MapPin,
  Code as CodeIcon,
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import CustomFieldInput from "@/components/ui/custom-field-input";
import { toast } from "sonner";
import {
  getNode,
  createNode,
  updateNode,
  deleteNode,
  setHomepage,
  getNodeTypes,
  getLanguages,
  getBlockTypes,
  getTemplates,
  type ContentNode,
  type NodeType,
  type NodeTypeField,
  type Language,
  type BlockType,
  type Template,
} from "@/api/client";

const BLOCK_ICON_MAP: Record<string, LucideIcon> = {
  "square": Square,
  "layout-template": LayoutTemplate,
  "type": Type,
  "image": Image,
  "mouse-pointer-click": MousePointerClick,
  "images": Images,
  "play": Play,
  "list": List,
  "quote": Quote,
  "map-pin": MapPin,
  "code": CodeIcon,
  "separator": SeparatorIcon,
  "file-text": FileText,
  "newspaper": Newspaper,
  "shopping-bag": ShoppingBag,
  "calendar": Calendar,
  "users": Users,
  "folder": Folder,
  "bookmark": Bookmark,
  "tag": Tag,
  "star": Star,
  "heart": Heart,
};

interface NodeEditorProps {
  nodeType: string;
}

interface BlockData {
  type: string;
  fields: Record<string, unknown>;
  [key: string]: unknown;
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function NodeEditorPage({ nodeType }: NodeEditorProps) {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  // Languages
  const [languages, setLanguages] = useState<Language[]>([]);

  // Node type definition
  const [nodeTypeDef, setNodeTypeDef] = useState<NodeType | null>(null);
  const label = nodeTypeDef?.label || nodeType.charAt(0).toUpperCase() + nodeType.slice(1);
  const basePath = `/admin/${nodeType === "page" ? "pages" : nodeType === "post" ? "posts" : `content/${nodeType}`}`;

  // Block types & templates
  const [blockTypes, setBlockTypes] = useState<BlockType[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [showAddBlock, setShowAddBlock] = useState(false);
  const [showInsertTemplate, setShowInsertTemplate] = useState(false);

  // Form state
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [status, setStatus] = useState("draft");
  const [languageCode, setLanguageCode] = useState("en");
  const [parentId, setParentId] = useState("");
  const [blocks, setBlocks] = useState<BlockData[]>([]);
  const [fieldsData, setFieldsData] = useState<Record<string, unknown>>({});
  const [originalNode, setOriginalNode] = useState<ContentNode | null>(null);

  // Block editor state
  const [collapsedBlocks, setCollapsedBlocks] = useState<Set<number>>(new Set());
  const [showRawJson, setShowRawJson] = useState(false);
  const [rawJson, setRawJson] = useState("");

  useEffect(() => {
    getLanguages(true).then(setLanguages).catch(() => {});
    getBlockTypes().then(setBlockTypes).catch(() => {});
    getTemplates().then(setTemplates).catch(() => {});
  }, []);

  // Fetch node type definition
  useEffect(() => {
    getNodeTypes()
      .then((types) => {
        const def = types.find((t) => t.slug === nodeType);
        setNodeTypeDef(def || null);
      })
      .catch(() => {});
  }, [nodeType]);

  useEffect(() => {
    if (!isEdit) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    getNode(id)
      .then((node) => {
        if (cancelled) return;
        setOriginalNode(node);
        setTitle(node.title);
        setSlug(node.slug);
        setStatus(node.status);
        setLanguageCode(node.language_code || "en");
        setParentId(node.parent_id ? String(node.parent_id) : "");
        // Parse blocks_data into typed blocks
        const rawBlocks = (node.blocks_data ?? []) as unknown as BlockData[];
        const parsedBlocks: BlockData[] = rawBlocks.map((b) => ({
          type: (b as Record<string, unknown>).type as string || "",
          fields: ((b as Record<string, unknown>).fields as Record<string, unknown>) || {},
        }));
        setBlocks(parsedBlocks);
        setFieldsData((node.fields_data as Record<string, unknown>) ?? {});
        setAutoSlug(false);
      })
      .catch(() => {
        toast.error(`Failed to load ${label.toLowerCase()}`);
        navigate(basePath, { replace: true });
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, isEdit, label, basePath, navigate]);

  // Auto-generate slug from title
  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(title));
    }
  }, [title, autoSlug]);

  // Sync rawJson when blocks change and raw view is open
  useEffect(() => {
    if (showRawJson) {
      setRawJson(JSON.stringify(blocks, null, 2));
    }
  }, [blocks, showRawJson]);

  const customFields: NodeTypeField[] = nodeTypeDef?.field_schema ?? [];

  // Resolve language URL slug
  const currentLang = languages.find((l) => l.code === languageCode);
  const langSlug = currentLang?.hide_prefix ? "" : (currentLang?.slug || languageCode);

  // Resolve the URL prefix for the current language
  const urlPrefix = (() => {
    if (nodeType === "page") return "";
    if (!nodeTypeDef) return nodeType !== "post" ? nodeType : "";
    const prefixes = nodeTypeDef.url_prefixes || {};
    const translated = prefixes[languageCode];
    if (translated) return translated;
    if (nodeType === "post") return prefixes["en"] || "";
    return nodeType;
  })();

  function updateFieldValue(key: string, value: unknown) {
    setFieldsData((prev) => ({ ...prev, [key]: value }));
  }

  // Block helpers
  function getBlockTypeDef(blockTypeSlug: string): BlockType | undefined {
    return blockTypes.find((bt) => bt.slug === blockTypeSlug);
  }

  function getBlockIcon(blockTypeSlug: string): LucideIcon {
    const bt = getBlockTypeDef(blockTypeSlug);
    if (bt?.icon && BLOCK_ICON_MAP[bt.icon]) return BLOCK_ICON_MAP[bt.icon];
    return Square;
  }

  function getBlockLabel(blockTypeSlug: string): string {
    const bt = getBlockTypeDef(blockTypeSlug);
    return bt?.label || blockTypeSlug;
  }

  function updateBlockField(blockIndex: number, fieldKey: string, value: unknown) {
    setBlocks((prev) => {
      const newBlocks = [...prev];
      newBlocks[blockIndex] = {
        ...newBlocks[blockIndex],
        fields: { ...newBlocks[blockIndex].fields, [fieldKey]: value },
      };
      return newBlocks;
    });
  }

  function moveBlock(index: number, direction: "up" | "down") {
    const newBlocks = [...blocks];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newBlocks.length) return;
    [newBlocks[index], newBlocks[targetIndex]] = [newBlocks[targetIndex], newBlocks[index]];
    // Update collapsed state
    const newCollapsed = new Set<number>();
    collapsedBlocks.forEach((i) => {
      if (i === index) newCollapsed.add(targetIndex);
      else if (i === targetIndex) newCollapsed.add(index);
      else newCollapsed.add(i);
    });
    setCollapsedBlocks(newCollapsed);
    setBlocks(newBlocks);
  }

  function removeBlock(index: number) {
    setBlocks((prev) => prev.filter((_, i) => i !== index));
    setCollapsedBlocks((prev) => {
      const newSet = new Set<number>();
      prev.forEach((i) => {
        if (i < index) newSet.add(i);
        else if (i > index) newSet.add(i - 1);
      });
      return newSet;
    });
  }

  function addBlock(blockTypeSlug: string) {
    setBlocks((prev) => [...prev, { type: blockTypeSlug, fields: {} }]);
    setShowAddBlock(false);
  }

  function insertTemplate(template: Template) {
    const newBlocks: BlockData[] = (template.block_config || []).map((bc) => ({
      type: bc.block_type_slug,
      fields: { ...bc.default_values },
    }));
    setBlocks((prev) => [...prev, ...newBlocks]);
    setShowInsertTemplate(false);
    toast.success(`Inserted ${newBlocks.length} block(s) from "${template.label}"`);
  }

  function toggleBlockCollapse(index: number) {
    setCollapsedBlocks((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(index)) newSet.delete(index);
      else newSet.add(index);
      return newSet;
    });
  }

  function applyRawJson() {
    try {
      const parsed = JSON.parse(rawJson);
      if (!Array.isArray(parsed)) {
        toast.error("JSON must be an array of blocks");
        return;
      }
      const typedBlocks: BlockData[] = parsed.map((b: Record<string, unknown>) => ({
        type: (b.type as string) || "",
        fields: (b.fields as Record<string, unknown>) || {},
      }));
      setBlocks(typedBlocks);
      toast.success("Blocks updated from JSON");
    } catch {
      toast.error("Invalid JSON");
    }
  }

  async function handleSave(e: FormEvent, publishStatus?: string) {
    e.preventDefault();

    const nodeData: Partial<ContentNode> = {
      title,
      slug,
      node_type: nodeType,
      status: publishStatus || status,
      language_code: languageCode,
      parent_id: parentId ? Number(parentId) : null,
      blocks_data: blocks as unknown as Record<string, unknown>[],
      fields_data: fieldsData,
    };

    setSaving(true);
    try {
      if (isEdit) {
        const updated = await updateNode(id, nodeData);
        setOriginalNode(updated);
        setStatus(updated.status);
        toast.success(`${label} updated successfully`);
      } else {
        const created = await createNode(nodeData);
        toast.success(`${label} created successfully`);
        navigate(`${basePath}/${created.id}/edit`, { replace: true });
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : `Failed to save ${label.toLowerCase()}`;
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!id) return;
    setDeleting(true);
    try {
      await deleteNode(id);
      toast.success(`${label} deleted successfully`);
      navigate(basePath, { replace: true });
    } catch {
      toast.error(`Failed to delete ${label.toLowerCase()}`);
    } finally {
      setDeleting(false);
    }
  }

  async function handleSetHomepage() {
    if (!id) return;
    try {
      await setHomepage(id);
      toast.success("Homepage updated successfully");
    } catch {
      toast.error("Failed to set homepage");
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
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild className="rounded-lg hover:bg-slate-200">
          <Link to={basePath}>
            <ArrowLeft className="h-5 w-5 text-slate-600" />
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">
          {isEdit ? `Edit ${label}` : `New ${label}`}
        </h1>
      </div>

      <form onSubmit={(e) => handleSave(e)} className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">Content</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-6 pt-0">
              <div className="space-y-2">
                <Label htmlFor="title" className="text-sm font-medium text-slate-700">Title</Label>
                <Input
                  id="title"
                  placeholder={`Enter ${label.toLowerCase()} title`}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
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
                <div className="flex items-center rounded-lg border border-slate-300 focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 overflow-hidden">
                  <span className="shrink-0 bg-slate-100 px-3 py-2 text-sm text-slate-500 border-r border-slate-300">
                    /{langSlug ? `${langSlug}/` : ""}{urlPrefix ? `${urlPrefix}/` : ""}
                  </span>
                  <input
                    id="slug"
                    placeholder="url-slug"
                    value={slug}
                    onChange={(e) => {
                      setAutoSlug(false);
                      setSlug(e.target.value);
                    }}
                    disabled={autoSlug}
                    required
                    className="flex-1 bg-transparent px-3 py-2 text-sm outline-none disabled:opacity-50"
                  />
                </div>
                {isEdit && originalNode ? (
                  <p className="text-xs text-slate-400 font-mono">
                    Full URL: {originalNode.full_url}
                  </p>
                ) : slug ? (
                  <p className="text-xs text-slate-400 font-mono">
                    Full URL: /{langSlug ? `${langSlug}/` : ""}{urlPrefix ? `${urlPrefix}/` : ""}{slug}
                  </p>
                ) : null}
              </div>
            </CardContent>
          </Card>

          {/* Visual Block Editor */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">Blocks</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 p-6 pt-0">
              {blocks.length === 0 && (
                <div className="flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-slate-200 py-12 text-slate-400">
                  <Square className="h-10 w-10" />
                  <p className="text-sm font-medium">No blocks yet</p>
                  <p className="text-xs">Add blocks or insert a template to get started</p>
                </div>
              )}

              {blocks.map((block, index) => {
                const blockTypeDef = getBlockTypeDef(block.type);
                const BlockIcon = getBlockIcon(block.type);
                const isCollapsed = collapsedBlocks.has(index);
                const blockFields = blockTypeDef?.field_schema || [];

                return (
                  <div
                    key={index}
                    className="rounded-lg border border-slate-200 bg-white shadow-sm overflow-hidden"
                  >
                    {/* Block header */}
                    <div className="flex items-center gap-2 bg-slate-50 px-4 py-2.5 border-b border-slate-200">
                      {/* Move buttons */}
                      <div className="flex flex-col gap-0">
                        <button
                          type="button"
                          onClick={() => moveBlock(index, "up")}
                          disabled={index === 0}
                          className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed"
                        >
                          <ChevronUp className="h-3.5 w-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => moveBlock(index, "down")}
                          disabled={index === blocks.length - 1}
                          className="text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed"
                        >
                          <ChevronDown className="h-3.5 w-3.5" />
                        </button>
                      </div>

                      {/* Collapse toggle + label */}
                      <button
                        type="button"
                        onClick={() => toggleBlockCollapse(index)}
                        className="flex flex-1 items-center gap-2 text-left"
                      >
                        <ChevronRight
                          className={`h-4 w-4 text-slate-400 transition-transform ${
                            !isCollapsed ? "rotate-90" : ""
                          }`}
                        />
                        <BlockIcon className="h-4 w-4 text-slate-500" />
                        <span className="text-sm font-medium text-slate-700">
                          {getBlockLabel(block.type)}
                        </span>
                        <span className="text-xs text-slate-400 font-mono">{block.type}</span>
                      </button>

                      {/* Delete block */}
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-red-400 hover:text-red-600"
                        onClick={() => removeBlock(index)}
                      >
                        <X className="h-3.5 w-3.5" />
                      </Button>
                    </div>

                    {/* Block fields */}
                    {!isCollapsed && (
                      <div className="p-4 space-y-3">
                        {blockFields.length === 0 ? (
                          <p className="text-sm text-slate-400 text-center py-2">
                            This block type has no fields defined.
                          </p>
                        ) : (
                          blockFields.map((field) => (
                            <div key={field.key} className="space-y-1.5">
                              <Label className="text-sm font-medium text-slate-700">
                                {field.label}
                                {field.required && <span className="ml-1 text-red-500">*</span>}
                              </Label>
                              <CustomFieldInput
                                field={field}
                                value={block.fields[field.key]}
                                onChange={(val) => updateBlockField(index, field.key, val)}
                              />
                            </div>
                          ))
                        )}
                      </div>
                    )}
                  </div>
                );
              })}

              {/* Add Block / Insert Template buttons */}
              <div className="flex gap-2 pt-1">
                <Button
                  type="button"
                  variant="outline"
                  className="flex-1 rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
                  onClick={() => setShowAddBlock(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Block
                </Button>
                {templates.length > 0 && (
                  <Button
                    type="button"
                    variant="outline"
                    className="flex-1 rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
                    onClick={() => setShowInsertTemplate(true)}
                  >
                    <LayoutTemplate className="mr-2 h-4 w-4" />
                    Insert Template
                  </Button>
                )}
              </div>

              {/* Raw JSON collapsible */}
              <Separator />
              <div>
                <button
                  type="button"
                  onClick={() => {
                    if (!showRawJson) {
                      setRawJson(JSON.stringify(blocks, null, 2));
                    }
                    setShowRawJson(!showRawJson);
                  }}
                  className="flex items-center gap-2 text-xs text-slate-400 hover:text-slate-600 transition-colors"
                >
                  <CodeIcon className="h-3.5 w-3.5" />
                  <span>Advanced: Raw JSON</span>
                  <ChevronRight
                    className={`h-3 w-3 transition-transform ${showRawJson ? "rotate-90" : ""}`}
                  />
                </button>
                {showRawJson && (
                  <div className="mt-3 space-y-2">
                    <Textarea
                      value={rawJson}
                      onChange={(e) => setRawJson(e.target.value)}
                      rows={12}
                      className="font-mono text-xs rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="rounded-lg border-slate-300 text-xs"
                      onClick={applyRawJson}
                    >
                      Apply JSON
                    </Button>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Custom Fields */}
          {customFields.length > 0 && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <CardHeader>
                <CardTitle className="text-lg font-semibold text-slate-900">Custom Fields</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4 p-6 pt-0">
                {customFields.map((field) => (
                  <div key={field.key} className="space-y-2">
                    <Label className="text-sm font-medium text-slate-700">
                      {field.label}
                      {field.required && <span className="ml-1 text-red-500">*</span>}
                    </Label>
                    <CustomFieldInput
                      field={field}
                      value={fieldsData[field.key]}
                      onChange={(val) => updateFieldValue(field.key, val)}
                    />
                  </div>
                ))}
              </CardContent>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">Publish</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-6 pt-0">
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Status</Label>
                <Select value={status} onValueChange={setStatus}>
                  <SelectTrigger className="rounded-lg border-slate-300">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="draft">Draft</SelectItem>
                    <SelectItem value="published">Published</SelectItem>
                    <SelectItem value="archived">Archived</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Language</Label>
                <Select value={languageCode} onValueChange={setLanguageCode}>
                  <SelectTrigger className="rounded-lg border-slate-300">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {languages.map((lang) => (
                      <SelectItem key={lang.code} value={lang.code}>
                        {lang.flag} {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="parent-id" className="text-sm font-medium text-slate-700">Parent ID</Label>
                <Input
                  id="parent-id"
                  type="number"
                  placeholder="None"
                  value={parentId}
                  onChange={(e) => setParentId(e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <Separator />

              <div className="flex flex-col gap-2">
                <Button
                  type="submit"
                  className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
                  disabled={saving}
                >
                  <Save className="mr-2 h-4 w-4" />
                  {saving ? "Saving..." : "Save"}
                </Button>
                {status !== "published" && (
                  <Button
                    type="button"
                    className="w-full bg-emerald-600 hover:bg-emerald-700 text-white font-medium rounded-lg"
                    disabled={saving}
                    onClick={(e) => handleSave(e, "published")}
                  >
                    <Globe className="mr-2 h-4 w-4" />
                    Publish
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Actions (edit mode only) */}
          {isEdit && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <CardHeader>
                <CardTitle className="text-lg font-semibold text-slate-900">Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 p-6 pt-0">
                {nodeType === "page" && (
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full bg-emerald-50 text-emerald-700 border-emerald-200 hover:bg-emerald-100 rounded-lg font-medium"
                    onClick={handleSetHomepage}
                  >
                    <Home className="mr-2 h-4 w-4" />
                    Set as Homepage
                  </Button>
                )}
                <Button
                  type="button"
                  variant="outline"
                  className="w-full bg-red-50 text-red-700 border-red-200 hover:bg-red-100 rounded-lg font-medium"
                  onClick={() => setShowDelete(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete {label}
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Node info (edit mode) */}
          {isEdit && originalNode && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <CardContent className="space-y-2 p-6 text-sm text-slate-500">
                <div className="flex justify-between">
                  <span>Version</span>
                  <span className="font-mono">{originalNode.version}</span>
                </div>
                <div className="flex justify-between">
                  <span>Created</span>
                  <span>
                    {new Date(originalNode.created_at).toLocaleDateString()}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>Updated</span>
                  <span>
                    {new Date(originalNode.updated_at).toLocaleDateString()}
                  </span>
                </div>
                {originalNode.published_at && (
                  <div className="flex justify-between">
                    <span>Published</span>
                    <span>
                      {new Date(
                        originalNode.published_at
                      ).toLocaleDateString()}
                    </span>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </form>

      {/* Add Block dialog */}
      <Dialog open={showAddBlock} onOpenChange={setShowAddBlock}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Add Block</DialogTitle>
            <DialogDescription>
              Select a block type to add to this page.
            </DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-2 max-h-80 overflow-y-auto py-2">
            {blockTypes.map((bt) => {
              const IconComp = BLOCK_ICON_MAP[bt.icon] || Square;
              return (
                <button
                  key={bt.id}
                  type="button"
                  onClick={() => addBlock(bt.slug)}
                  className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white p-3 text-left transition-all hover:border-indigo-300 hover:bg-indigo-50"
                >
                  <IconComp className="h-5 w-5 text-slate-500 shrink-0" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-slate-800 truncate">{bt.label}</p>
                    {bt.description && (
                      <p className="text-xs text-slate-400 truncate">{bt.description}</p>
                    )}
                  </div>
                </button>
              );
            })}
            {blockTypes.length === 0 && (
              <p className="col-span-2 text-center text-sm text-slate-400 py-8">
                No block types available. Create block types first.
              </p>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Insert Template dialog */}
      <Dialog open={showInsertTemplate} onOpenChange={setShowInsertTemplate}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Insert Template</DialogTitle>
            <DialogDescription>
              Select a template to insert its blocks with default values.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 max-h-80 overflow-y-auto py-2">
            {templates.map((tpl) => (
              <button
                key={tpl.id}
                type="button"
                onClick={() => insertTemplate(tpl)}
                className="flex w-full items-center gap-3 rounded-lg border border-slate-200 bg-white p-3 text-left transition-all hover:border-indigo-300 hover:bg-indigo-50"
              >
                <LayoutTemplate className="h-5 w-5 text-slate-500 shrink-0" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-slate-800">{tpl.label}</p>
                  {tpl.description && (
                    <p className="text-xs text-slate-400 truncate">{tpl.description}</p>
                  )}
                </div>
                <span className="text-xs text-slate-400 shrink-0">
                  {tpl.block_config?.length ?? 0} block(s)
                </span>
              </button>
            ))}
            {templates.length === 0 && (
              <p className="text-center text-sm text-slate-400 py-8">
                No templates available. Create templates first.
              </p>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {label}</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{originalNode?.title}&quot;?
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
    </div>
  );
}
