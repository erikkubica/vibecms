import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, useSearchParams, Link } from "react-router-dom";
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
  ExternalLink,
  Code as CodeIcon,
  Tag,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
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
import BlockPicker, { BLOCK_ICON_MAP } from "@/components/ui/block-picker";
import { usePageMeta } from "@/components/layout/page-meta";
import { toast } from "sonner";
import {
  getNode,
  createNode,
  updateNode,
  deleteNode,
  setHomepage,
  getHomepageId,
  getNodeTypes,
  getLanguages,
  getBlockTypes,
  getTemplates,
  getTemplate,
  getLayouts,
  getNodeTranslations,
  createNodeTranslation,
  searchNodes,
  listTerms,
  createTerm,
  getLayoutPartials,
  type ContentNode,
  type NodeType,
  type NodeTypeField,
  type Language,
  type BlockType,
  type Template,
  type Layout,
  type LayoutBlock,
  type TaxonomyTerm,
} from "@/api/client";

interface NodeEditorProps {
  nodeTypeProp: string;
}

interface BlockData {
  type: string;
  fields: Record<string, unknown>;
  [key: string]: unknown;
}

function slugify(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function getFieldWidth(field: NodeTypeField): number {
  const w = field.width;
  if (typeof w === "number" && w > 0 && w <= 100) return Math.round(w);
  return 100;
}

export default function NodeEditorPage({ nodeTypeProp }: NodeEditorProps) {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
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
  const label = nodeTypeDef?.label || nodeTypeProp.charAt(0).toUpperCase() + nodeTypeProp.slice(1);
  const labelPlural = nodeTypeDef?.label_plural || `${label}s`;
  const basePath = `/admin/${nodeTypeProp === "page" ? "pages" : nodeTypeProp === "post" ? "posts" : `content/${nodeTypeProp}`}`;

  // Block types & templates
  const [blockTypes, setBlockTypes] = useState<BlockType[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [layouts, setLayouts] = useState<Layout[]>([]);
  const [showAddBlock, setShowAddBlock] = useState(false);
  const [showInsertTemplate, setShowInsertTemplate] = useState(false);

  // Form state
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [status, setStatus] = useState("draft");
  const [languageCode, setLanguageCode] = useState("en");
  const [parentId, setParentId] = useState("");
  const [layoutId, setLayoutId] = useState<string>("");
  const [blocks, setBlocks] = useState<BlockData[]>([]);
  const [fieldsData, setFieldsData] = useState<Record<string, unknown>>({});
  const [layoutData, setLayoutData] = useState<Record<string, Record<string, unknown>>>({});
  const [layoutPartials, setLayoutPartials] = useState<LayoutBlock[]>([]);
  const [originalNode, setOriginalNode] = useState<ContentNode | null>(null);

  usePageMeta([
    labelPlural,
    isEdit ? (title ? `Edit "${title}"` : "Edit") : `New ${label}`,
  ]);

  // Page templates
  const [showLoadTemplate, setShowLoadTemplate] = useState(false);
  const [showConfirmTemplate, setShowConfirmTemplate] = useState(false);
  const [selectedTemplateId, setSelectedTemplateId] = useState<number | null>(null);
  const [applyingTemplate, setApplyingTemplate] = useState(false);

  // Parent node search
  const [parentNode, setParentNode] = useState<{ id: number; title: string; slug: string } | null>(null);
  const [parentSearch, setParentSearch] = useState("");
  const [parentResults, setParentResults] = useState<{ id: number; title: string; slug: string }[]>([]);
  const [parentSearching, setParentSearching] = useState(false);
  const [showParentResults, setShowParentResults] = useState(false);

  // SEO
  const [seoTitle, setSeoTitle] = useState("");
  const [seoDescription, setSeoDescription] = useState("");

  // Standard Fields
  const [featuredImage, setFeaturedImage] = useState<Record<string, unknown>>({});
  const [excerpt, setExcerpt] = useState("");
  const [taxonomies, setTaxonomies] = useState<Record<string, string[]>>({});
  const [availableTerms, setAvailableTerms] = useState<Record<string, TaxonomyTerm[]>>({});
  const [taxonomySearch, setTaxonomySearch] = useState<Record<string, string>>({});
  const [taxonomyDropdownOpen, setTaxonomyDropdownOpen] = useState<Record<string, boolean>>({});

  // Homepage
  const [homepageId, setHomepageId] = useState<number | null>(null);

  // Translations
  const [translations, setTranslations] = useState<ContentNode[]>([]);
  const [creatingTranslation, setCreatingTranslation] = useState(false);

  // Block editor state — persisted to localStorage per node
  const storageKey = id ? `squilla:collapsed-blocks:${id}` : "";
  const [collapsedBlocks, setCollapsedBlocks] = useState<Set<number>>(() => {
    if (!storageKey) return new Set();
    try {
      const saved = localStorage.getItem(storageKey);
      if (saved) return new Set(JSON.parse(saved) as number[]);
    } catch { /* ignore */ }
    return new Set();
  });
  useEffect(() => {
    if (!storageKey) return;
    if (collapsedBlocks.size === 0) {
      localStorage.removeItem(storageKey);
    } else {
      localStorage.setItem(storageKey, JSON.stringify([...collapsedBlocks]));
    }
  }, [collapsedBlocks, storageKey]);

  const [showRawJson, setShowRawJson] = useState(false);
  const [rawJson, setRawJson] = useState("");

  useEffect(() => {
    getLanguages(true).then(setLanguages).catch(() => {});
    getBlockTypes().then(setBlockTypes).catch(() => {});
    getTemplates().then(setTemplates).catch(() => {});
    getHomepageId().then(setHomepageId).catch(() => {});
  }, []);

  // Fetch layouts (all — filtering by language happens at render time via cascade)
  useEffect(() => {
    getLayouts().then(setLayouts).catch(() => {});
  }, []);

  // Fetch layout partials when layoutId changes
  useEffect(() => {
    if (!layoutId) {
      setLayoutPartials([]);
      return;
    }
    getLayoutPartials(layoutId)
      .then((partials) => {
        // Only keep partials that have field_schema
        setLayoutPartials(partials.filter((p) => p.field_schema && p.field_schema.length > 0));
      })
      .catch(() => setLayoutPartials([]));
  }, [layoutId]);

  // Fetch node type definition
  useEffect(() => {
    getNodeTypes()
      .then((types) => {
        const def = types.find((t) => t.slug === nodeTypeProp);
        setNodeTypeDef(def || null);
      })
      .catch(() => {});
  }, [nodeTypeProp]);

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
        setLayoutId(node.layout_id ? String(node.layout_id) : "");
        setParentId(node.parent_id ? String(node.parent_id) : "");
        // SEO
        const seo = (node.seo_settings || {}) as Record<string, string>;
        setSeoTitle(seo.meta_title || "");
        setSeoDescription(seo.meta_description || "");
        // Standard Fields
        setFeaturedImage((node.featured_image as Record<string, unknown>) || {});
        setExcerpt(node.excerpt || "");
        setTaxonomies(node.taxonomies || {});

        // Parse blocks_data into typed blocks
        const rawBlocks = (node.blocks_data ?? []) as unknown as BlockData[];
        const parsedBlocks: BlockData[] = rawBlocks.map((b) => ({
          type: (b as Record<string, unknown>).type as string || "",
          fields: ((b as Record<string, unknown>).fields as Record<string, unknown>) || {},
        }));
        setBlocks(parsedBlocks);
        setFieldsData((node.fields_data as Record<string, unknown>) ?? {});
        setLayoutData((node.layout_data as Record<string, Record<string, unknown>>) ?? {});
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

  // Load translations when editing
  useEffect(() => {
    if (!isEdit || !id) return;
    getNodeTranslations(id)
      .then(setTranslations)
      .catch(() => setTranslations([]));
  }, [id, isEdit]);

  // Fetch available terms for each taxonomy defined on the node type.
  // Scope to the node's own language — a `de` post must only see `de`
  // terms, otherwise typing "Com" on a German post wouldn't match the
  // existing English "Computers" term and the picker would offer to create
  // a duplicate.
  useEffect(() => {
    if (!nodeTypeDef?.taxonomies) return;
    if (!languageCode) return;
    const taxDefs = nodeTypeDef.taxonomies as Array<{slug: string; label: string; multiple?: boolean}>;
    taxDefs.forEach(tax => {
      listTerms(nodeTypeProp, tax.slug, { language_code: languageCode })
        .then(terms => setAvailableTerms(prev => ({ ...prev, [tax.slug]: terms })))
        .catch(() => {});
    });
  }, [nodeTypeDef, nodeTypeProp, languageCode]);

  async function handleCreateTranslation(langCode: string) {
    if (!id) return;
    setCreatingTranslation(true);
    try {
      const newNode = await createNodeTranslation(id, { language_code: langCode });
      toast.success(`Translation created in ${langCode}`);
      navigate(`${basePath}/${newNode.id}/edit`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create translation";
      toast.error(message);
    } finally {
      setCreatingTranslation(false);
    }
  }

  // Load parent node details when editing (to show slug prefix)
  useEffect(() => {
    if (parentId && !parentNode) {
      getNode(parentId)
        .then((n) => setParentNode({ id: n.id, title: n.title, slug: n.slug }))
        .catch(() => setParentNode(null));
    }
  }, [parentId]);

  // Parent node search with debounce
  useEffect(() => {
    if (!parentSearch.trim()) {
      setParentResults([]);
      return;
    }
    const timer = setTimeout(async () => {
      setParentSearching(true);
      try {
        const res = await searchNodes({
          q: parentSearch,
          node_type: nodeTypeProp,
          limit: 10,
        });
        // Filter out self
        setParentResults(res.filter((r) => String(r.id) !== id));
      } catch {
        setParentResults([]);
      } finally {
        setParentSearching(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [parentSearch, nodeTypeProp, id]);

  // Auto-generate slug from title
  useEffect(() => {
    if (autoSlug) {
      setSlug(slugify(title));
    }
  }, [title, autoSlug]);

  // Auto-load template from query parameter (e.g. ?template=slug)
  useEffect(() => {
    const templateSlug = searchParams.get("template");
    if (templateSlug && !isEdit && !loading) {
      getTemplates()
        .then((templates) => {
          const match = templates.find((t) => t.slug === templateSlug);
          if (!match) throw new Error("Template not found");
          return getTemplate(match.id);
        })
        .then((detail) => {
          const config = detail.block_config as Array<{ block_type_slug: string; default_values: Record<string, unknown> }>;
          const newBlocks: BlockData[] = config.map((b) => ({
            type: b.block_type_slug,
            fields: { ...(b.default_values || {}) },
          }));
          setBlocks(newBlocks);
          toast.success(`Loaded template "${detail.label}" with ${newBlocks.length} block(s)`);
        })
        .catch(() => {
          toast.error("Failed to load template");
        });
    }
  }, [searchParams, isEdit, loading]);

  // Sync rawJson when blocks change and raw view is open
  useEffect(() => {
    if (showRawJson) {
      setRawJson(JSON.stringify(blocks, null, 2));
    }
  }, [blocks, showRawJson]);

  const customFields: NodeTypeField[] = (nodeTypeDef?.field_schema ?? []).map(f => ({ ...f, key: f.key || f.name }));

  // Resolve whether block composition is enabled for this node.
  // Precedence: explicit layout setting > node type setting. Default true.
  const selectedLayout = layouts.find((l) => String(l.id) === String(layoutId));
  const layoutAllowsBlocks = selectedLayout ? (selectedLayout.supports_blocks !== false) : true;
  const nodeTypeAllowsBlocks = nodeTypeDef ? (nodeTypeDef.supports_blocks !== false) : true;
  const blocksEnabled = layoutAllowsBlocks && nodeTypeAllowsBlocks;

  // Resolve language URL slug
  const currentLang = languages.find((l) => l.code === languageCode);
  const langSlug = currentLang?.hide_prefix ? "" : (currentLang?.slug || languageCode);

  // Resolve the URL prefix for the current language
  const urlPrefix = (() => {
    if (nodeTypeProp === "page") return "";
    if (!nodeTypeDef) return nodeTypeProp !== "post" ? nodeTypeProp : "";
    const prefixes = nodeTypeDef.url_prefixes || {};
    const translated = prefixes[languageCode];
    if (translated) return translated;
    if (nodeTypeProp === "post") return prefixes["en"] || "";
    return nodeTypeProp;
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

  function openLoadTemplate() {
    setShowLoadTemplate(true);
  }

  function selectTemplate(id: number) {
    setSelectedTemplateId(id);
    setShowLoadTemplate(false);
    setShowConfirmTemplate(true);
  }

  function applyTemplate() {
    if (!selectedTemplateId) return;
    setApplyingTemplate(true);
    const tpl = templates.find((t) => t.id === selectedTemplateId);
    if (!tpl) {
      toast.error("Template not found");
      setApplyingTemplate(false);
      setShowConfirmTemplate(false);
      setSelectedTemplateId(null);
      return;
    }
    const config = tpl.block_config as Array<{ block_type_slug: string; default_values: Record<string, unknown> }>;
    const newBlocks: BlockData[] = config.map((b) => ({
      type: b.block_type_slug,
      fields: { ...(b.default_values || {}) },
    }));
    setBlocks(newBlocks);
    setCollapsedBlocks(new Set());
    toast.success(`Loaded template "${tpl.label}" with ${newBlocks.length} block(s)`);
    setApplyingTemplate(false);
    setShowConfirmTemplate(false);
    setSelectedTemplateId(null);
  }

  async function handleSave(e: FormEvent, publishStatus?: string) {
    e.preventDefault();

    const nodeData: Partial<ContentNode> = {
      title,
      slug,
      node_type: nodeTypeProp,
      status: publishStatus || status,
      language_code: languageCode,
      parent_id: parentId ? Number(parentId) : null,
      layout_id: layoutId ? Number(layoutId) : null,
      blocks_data: blocks as unknown as Record<string, unknown>[],
      fields_data: fieldsData,
      featured_image: featuredImage,
      excerpt: excerpt,
      taxonomies: taxonomies,
      seo_settings: {
        meta_title: seoTitle,
        meta_description: seoDescription,
      },
      layout_data: layoutData,
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
      setHomepageId(Number(id));
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
    <div className="space-y-4">
      <form onSubmit={(e) => handleSave(e)} className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Main content */}
        <div className="space-y-4 min-w-0">
          {/* Inline Title + Slug + ID + View header row */}
          <div
            className="flex items-center gap-1.5"
            style={{
              padding: 6,
              background: "var(--card-bg)",
              border: "1px solid var(--border)",
              borderRadius: "var(--radius-lg)",
              boxShadow: "var(--shadow-sm)",
            }}
          >
            <Button variant="ghost" size="icon" asChild className="h-7 w-7 shrink-0">
              <Link to={basePath} title={`Back to ${label}s`}>
                <ArrowLeft className="h-3.5 w-3.5" style={{ color: "var(--fg-muted)" }} />
              </Link>
            </Button>
            <div className="flex items-center gap-1.5 flex-[1_1_60%] min-w-0 px-1">
              <span
                className="shrink-0 uppercase"
                style={{
                  fontSize: 10.5,
                  fontWeight: 600,
                  color: "var(--fg-muted)",
                  letterSpacing: "0.06em",
                }}
              >
                Title
              </span>
              <input
                id="title"
                placeholder={`Enter ${label.toLowerCase()} title`}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
                className="flex-1 min-w-0 bg-transparent outline-none"
                style={{
                  border: "none",
                  padding: "6px 4px",
                  fontSize: 14,
                  fontWeight: 500,
                  color: "var(--fg)",
                }}
              />
            </div>
            <div className="w-px h-5 shrink-0" style={{ background: "var(--border)" }} />
            <div className="flex items-center gap-1 flex-[1_1_40%] min-w-0 px-1">
              <span
                className="shrink-0"
                style={{
                  fontSize: 11,
                  color: "var(--fg-subtle)",
                  fontFamily: "var(--font-mono)",
                }}
              >
                /{langSlug ? `${langSlug}/` : ""}
                {urlPrefix ? `${urlPrefix}/` : ""}
                {parentNode ? `${parentNode.slug}/` : ""}
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
                className="flex-1 min-w-0 bg-transparent outline-none disabled:opacity-60"
                style={{
                  border: "none",
                  padding: "6px 0",
                  fontSize: 12.5,
                  color: "var(--fg)",
                  fontFamily: "var(--font-mono)",
                }}
              />
              <button
                type="button"
                className="shrink-0 px-1.5 py-0.5 rounded text-[10.5px] font-medium uppercase"
                style={{
                  color: autoSlug ? "var(--accent)" : "var(--fg-muted)",
                  background: autoSlug ? "color-mix(in oklab, var(--accent) 12%, transparent)" : "var(--sub-bg)",
                  border: "1px solid var(--border)",
                  letterSpacing: "0.04em",
                }}
                onClick={() => setAutoSlug(!autoSlug)}
                title={autoSlug ? "Click to edit slug manually" : "Click to auto-generate slug from title"}
              >
                {autoSlug ? "Auto" : "Edit"}
              </button>
            </div>
            {isEdit && (
              <Badge
                variant="secondary"
                className="shrink-0 font-mono"
                style={{ fontSize: 10.5, background: "var(--sub-bg)", color: "var(--fg-muted)", border: "1px solid var(--border)" }}
              >
                ID {id}
              </Badge>
            )}
            {isEdit && originalNode && status === "published" && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-7 px-2 text-[12px] shrink-0"
                asChild
              >
                <a href={originalNode.full_url} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="mr-1 h-3 w-3" />
                  View
                </a>
              </Button>
            )}
          </div>

          {/* Visual Block Editor */}
          {blocksEnabled && (
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <h2 className="font-semibold" style={{ fontSize: 14, color: "var(--fg)" }}>Blocks</h2>
                <Badge
                  variant="secondary"
                  style={{ fontSize: 10.5, background: "var(--sub-bg)", color: "var(--fg-muted)", border: "1px solid var(--border)" }}
                >
                  {blocks.length}
                </Badge>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-7 text-[12px]"
                onClick={openLoadTemplate}
              >
                <LayoutTemplate className="mr-1.5 h-3.5 w-3.5" />
                Load from Template
              </Button>
            </div>
            <div className="space-y-3">
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

                const typeCategory = block.type.split("-")[0];
                return (
                  <div
                    key={index}
                    className="overflow-hidden"
                    style={{
                      border: "1px solid var(--border)",
                      borderRadius: "var(--radius-lg)",
                      background: "var(--card-bg)",
                    }}
                  >
                    {/* Block header */}
                    <div
                      className="flex items-center gap-2 cursor-pointer select-none"
                      style={{
                        padding: "8px 10px",
                        background: "var(--sub-bg)",
                        borderBottom: isCollapsed ? "none" : "1px solid var(--border)",
                      }}
                      onClick={() => toggleBlockCollapse(index)}
                    >
                      <ChevronDown
                        className="shrink-0 transition-transform"
                        size={12}
                        style={{
                          color: "var(--fg-muted)",
                          transform: isCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                        }}
                      />
                      <BlockIcon size={14} style={{ color: "var(--fg-muted)" }} className="shrink-0" />
                      <span
                        className="font-semibold"
                        style={{ fontSize: 12.5, color: "var(--fg)" }}
                      >
                        {getBlockLabel(block.type)}
                      </span>
                      <span
                        className="font-mono"
                        style={{ fontSize: 11, color: "var(--fg-muted)" }}
                      >
                        {block.type}
                      </span>
                      {typeCategory && typeCategory !== block.type && (
                        <Badge
                          variant="secondary"
                          style={{
                            fontSize: 10,
                            background: "color-mix(in oklab, var(--accent) 10%, transparent)",
                            color: "var(--accent-strong)",
                            border: "1px solid color-mix(in oklab, var(--accent) 20%, transparent)",
                          }}
                        >
                          {typeCategory}
                        </Badge>
                      )}
                      <div className="flex-1" />
                      <div className="flex items-center gap-0.5" onClick={(e) => e.stopPropagation()}>
                        <button
                          type="button"
                          onClick={() => moveBlock(index, "up")}
                          disabled={index === 0}
                          className="p-1 rounded disabled:opacity-30 disabled:cursor-not-allowed hover:bg-black/5"
                          style={{ color: "var(--fg-muted)" }}
                          title="Move up"
                        >
                          <ChevronUp className="h-3.5 w-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => moveBlock(index, "down")}
                          disabled={index === blocks.length - 1}
                          className="p-1 rounded disabled:opacity-30 disabled:cursor-not-allowed hover:bg-black/5"
                          style={{ color: "var(--fg-muted)" }}
                          title="Move down"
                        >
                          <ChevronDown className="h-3.5 w-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => removeBlock(index)}
                          className="p-1 rounded hover:bg-red-50"
                          style={{ color: "var(--danger)" }}
                          title="Delete block"
                        >
                          <X className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>

                    {/* Block fields */}
                    {!isCollapsed && (
                      <div style={{ padding: "12px 14px 14px" }} className="space-y-3">
                        {blockTypeDef?.description && (
                          <div
                            style={{
                              fontSize: 12,
                              color: "var(--fg-muted)",
                              padding: "8px 10px",
                              background: "var(--sub-bg)",
                              border: "1px solid var(--border)",
                              borderLeft: "2px solid var(--accent)",
                              borderRadius: "var(--radius)",
                              lineHeight: 1.5,
                            }}
                          >
                            {blockTypeDef.description}
                          </div>
                        )}
                        {blockFields.length === 0 ? (
                          <p className="text-center py-2" style={{ fontSize: 13, color: "var(--fg-subtle)" }}>
                            This block type has no fields defined.
                          </p>
                        ) : (
                          <div className="flex flex-wrap" style={{ gap: "12px 14px" }}>
                            {blockFields.map((field) => {
                              const w = getFieldWidth(field);
                              return (
                                <div
                                  key={field.key}
                                  className="space-y-1.5 min-w-0"
                                  style={{
                                    flex: `0 0 calc(${w}% - 14px)`,
                                    maxWidth: `calc(${w}% - 14px)`,
                                  }}
                                >
                                  <Label className="font-medium" style={{ fontSize: 12, color: "var(--fg-2)" }}>
                                    {field.label}
                                    {field.required && <span className="ml-1" style={{ color: "var(--danger)" }}>*</span>}
                                  </Label>
                                  <CustomFieldInput
                                    field={field}
                                    value={block.fields[field.key]}
                                    onChange={(val) => updateBlockField(index, field.key, val)}
                                    languageCode={languageCode}
                                  />
                                </div>
                              );
                            })}
                          </div>
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
            </div>
          </div>
          )}

          {/* Excerpt */}
          <Card>
            <SectionHeader title="Excerpt" />
            <CardContent className="space-y-2">
              <Textarea
                placeholder="Enter a short summary or teaser..."
                value={excerpt}
                onChange={(e) => setExcerpt(e.target.value)}
                rows={3}
                className="resize-none"
              />
              <p style={{ fontSize: 11, color: "var(--fg-muted)" }}>Short description used in cards and search results. If empty, it may be auto-generated from content.</p>
            </CardContent>
          </Card>

          {/* Custom Fields */}
          {customFields.length > 0 && (
            <Card>
              <SectionHeader title="Custom Fields" />
              <CardContent>
                <div className="flex flex-wrap" style={{ gap: "16px 14px" }}>
                  {customFields.map((field) => {
                    const w = getFieldWidth(field);
                    return (
                      <div
                        key={field.key}
                        className="space-y-2 min-w-0"
                        style={{ flex: `0 0 calc(${w}% - 14px)`, maxWidth: `calc(${w}% - 14px)` }}
                      >
                        <Label className="text-sm font-medium text-slate-700">
                          {field.label}
                          {field.required && <span className="ml-1 text-red-500">*</span>}
                        </Label>
                        <CustomFieldInput
                          field={field}
                          value={fieldsData[field.key]}
                          onChange={(val) => updateFieldValue(field.key, val)}
                          languageCode={languageCode}
                        />
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Layout Partial Fields */}
          {layoutPartials.map((partial) => {
            const partialCollapsed = collapsedBlocks.has(-partial.id);
            return (
            <div
              key={partial.slug}
              className="overflow-hidden"
              style={{
                border: "1px solid var(--border)",
                borderRadius: "var(--radius-lg)",
                background: "var(--card-bg)",
              }}
            >
              <div
                className="flex items-center gap-2 cursor-pointer select-none"
                style={{
                  padding: "8px 10px",
                  background: "var(--sub-bg)",
                  borderBottom: partialCollapsed ? "none" : "1px solid var(--border)",
                }}
                onClick={() => {
                  setCollapsedBlocks((prev) => {
                    const next = new Set(prev);
                    const key = -partial.id;
                    if (next.has(key)) next.delete(key); else next.add(key);
                    return next;
                  });
                }}
              >
                <ChevronDown
                  size={12}
                  className="shrink-0 transition-transform"
                  style={{
                    color: "var(--fg-muted)",
                    transform: partialCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                  }}
                />
                <LayoutTemplate size={14} style={{ color: "var(--fg-muted)" }} className="shrink-0" />
                <span className="font-semibold" style={{ fontSize: 12.5, color: "var(--fg)" }}>
                  {partial.name}
                </span>
                <Badge
                  variant="secondary"
                  className="ml-auto"
                  style={{
                    fontSize: 10,
                    background: "color-mix(in oklab, #a855f7 14%, transparent)",
                    color: "#7e22ce",
                    border: "1px solid color-mix(in oklab, #a855f7 24%, transparent)",
                  }}
                >
                  Layout Partial
                </Badge>
              </div>
              {!partialCollapsed && (
                <div style={{ padding: "12px 14px 14px" }}>
                  <div className="flex flex-wrap" style={{ gap: "16px 14px" }}>
                    {(partial.field_schema || []).map((field) => {
                      const fieldKey = field.key || field.name;
                      const partialData = layoutData[partial.slug] || {};
                      const w = getFieldWidth(field);
                      return (
                        <div
                          key={fieldKey}
                          className="space-y-2 min-w-0"
                          style={{ flex: `0 0 calc(${w}% - 14px)`, maxWidth: `calc(${w}% - 14px)` }}
                        >
                          <Label className="text-sm font-medium text-slate-700">
                            {field.label}
                            {field.required && <span className="ml-1 text-red-500">*</span>}
                          </Label>
                          {(field as any).default_from && !partialData[fieldKey] && (
                            <p className="text-xs text-slate-400">
                              Falls back to <code className="bg-slate-100 px-1 rounded">{(field as any).default_from}</code>
                            </p>
                          )}
                          <CustomFieldInput
                            field={field}
                            value={partialData[fieldKey]}
                            onChange={(val) => {
                              setLayoutData((prev) => ({
                                ...prev,
                                [partial.slug]: {
                                  ...(prev[partial.slug] || {}),
                                  [fieldKey]: val,
                                },
                              }));
                            }}
                            languageCode={languageCode}
                          />
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
            );
          })}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Publish" />
            <CardContent className="space-y-4">
              {/* Status + Language row */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-slate-500">Status</Label>
                  <Select value={status} onValueChange={setStatus}>
                    <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="draft">Draft</SelectItem>
                      <SelectItem value="published">Published</SelectItem>
                      <SelectItem value="archived">Archived</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-slate-500">Language</Label>
                  <Select value={languageCode} onValueChange={setLanguageCode}>
                    <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {languages.map((lang) => (
                        <SelectItem key={lang.code} value={lang.code}>
                          {lang.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Layout */}
              <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-slate-500">Layout</Label>
                  <Select value={layoutId || "auto"} onValueChange={(v) => setLayoutId(v === "auto" ? "" : v)}>
                    <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
                      <SelectValue placeholder="Auto" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="auto">Auto (cascade)</SelectItem>
                      {layouts.map((layout) => (
                        <SelectItem key={layout.id} value={String(layout.id)}>
                          {layout.name}
                          {layout.source === "theme" ? ` [${layout.theme_name || "theme"}]` : " [custom]"}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
              </div>

              {/* Parent */}
              <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-slate-500">Parent</Label>
                  {parentNode ? (
                    <div className="flex items-center gap-2 h-9 rounded-lg border border-indigo-200 bg-indigo-50 px-3">
                      <span className="flex-1 text-sm font-medium text-slate-800 truncate">{parentNode.title}</span>
                      <span className="text-[10px] text-slate-400 font-mono">/{parentNode.slug}</span>
                      <button
                        type="button"
                        className="text-slate-400 hover:text-red-500 shrink-0"
                        onClick={() => { setParentId(""); setParentNode(null); }}
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  ) : (
                    <div className="relative">
                      <Input
                        placeholder={`Search ${label.toLowerCase()}s...`}
                        value={parentSearch}
                        onChange={(e) => {
                          setParentSearch(e.target.value);
                          setShowParentResults(true);
                        }}
                        onFocus={() => setShowParentResults(true)}
                        onBlur={() => setTimeout(() => setShowParentResults(false), 200)}
                        className="h-9 rounded-lg border-slate-300 text-sm focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                      />
                      {showParentResults && (parentSearch.trim() || parentSearching) && (
                        <div className="absolute z-50 mt-1 w-full rounded-lg border border-slate-200 bg-white shadow-lg max-h-48 overflow-y-auto">
                          {parentSearching ? (
                            <div className="px-3 py-2 text-sm text-slate-400">Searching...</div>
                          ) : parentResults.length === 0 ? (
                            <div className="px-3 py-2 text-sm text-slate-400">
                              {parentSearch.trim() ? "No results found" : "Type to search..."}
                            </div>
                          ) : (
                            parentResults.map((node) => (
                              <button
                                key={node.id}
                                type="button"
                                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-indigo-50 transition-colors"
                                onMouseDown={(e) => e.preventDefault()}
                                onClick={() => {
                                  setParentId(String(node.id));
                                  setParentNode(node);
                                  setParentSearch("");
                                  setParentResults([]);
                                  setShowParentResults(false);
                                }}
                              >
                                <span className="font-medium text-slate-800 truncate">{node.title}</span>
                                <span className="text-[10px] text-slate-400 font-mono ml-auto shrink-0">/{node.slug}</span>
                              </button>
                            ))
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </div>

              {/* Save buttons */}
              <div className="flex gap-2 pt-1">
                <Button
                  type="submit"
                  className="flex-1 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
                  disabled={saving}
                >
                  <Save className="mr-1.5 h-3.5 w-3.5" />
                  {saving ? "Saving..." : "Save"}
                </Button>
                {status !== "published" && (
                  <Button
                    type="button"
                    className="flex-1 bg-emerald-600 hover:bg-emerald-700 text-white font-medium rounded-lg h-9 text-sm"
                    disabled={saving}
                    onClick={(e) => handleSave(e, "published")}
                  >
                    <Globe className="mr-1.5 h-3.5 w-3.5" />
                    Publish
                  </Button>
                )}
              </div>

              {/* Actions (edit mode) */}
              {isEdit && (
                <>
                  <Separator />
                  <div className="flex gap-2">
                    {nodeTypeProp === "page" && (
                      homepageId === Number(id) ? (
                        <Button
                          type="button"
                          variant="outline"
                          className="flex-1 bg-emerald-100 text-emerald-800 border-emerald-300 rounded-lg font-medium h-8 text-xs cursor-default"
                          disabled
                        >
                          <Home className="mr-1.5 h-3.5 w-3.5" />
                          Current Homepage
                        </Button>
                      ) : (
                        <Button
                          type="button"
                          variant="outline"
                          className="flex-1 bg-slate-50 text-slate-700 border-slate-200 hover:bg-emerald-50 hover:text-emerald-700 hover:border-emerald-200 rounded-lg font-medium h-8 text-xs"
                          onClick={handleSetHomepage}
                        >
                          <Home className="mr-1.5 h-3.5 w-3.5" />
                          Set as Homepage
                        </Button>
                      )
                    )}
                    <Button
                      type="button"
                      variant="outline"
                      className="flex-1 bg-red-50 text-red-700 border-red-200 hover:bg-red-100 rounded-lg font-medium h-8 text-xs"
                      onClick={() => setShowDelete(true)}
                    >
                      <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                      Delete
                    </Button>
                  </div>
                </>
              )}

              {/* Metadata (edit mode) */}
              {isEdit && originalNode && (
                <>
                  <Separator />
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-400">
                    <div className="flex justify-between">
                      <span>Version</span>
                      <span className="font-mono text-slate-600">{originalNode.version}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Created</span>
                      <span className="text-slate-600">{new Date(originalNode.created_at).toLocaleDateString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Updated</span>
                      <span className="text-slate-600">{new Date(originalNode.updated_at).toLocaleDateString()}</span>
                    </div>
                    {originalNode.published_at && (
                      <div className="flex justify-between">
                        <span>Published</span>
                        <span className="text-slate-600">{new Date(originalNode.published_at).toLocaleDateString()}</span>
                      </div>
                    )}
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* Featured Image */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Featured Image" />
            <CardContent className="space-y-2">
              <CustomFieldInput
                field={{ name: "featured_image", key: "featured_image", label: "Featured Image", type: "image" }}
                value={featuredImage}
                onChange={(val) => setFeaturedImage(val as Record<string, unknown>)}
              />
              <p className="text-[11px] text-slate-400">Main image used for listings, sliders, and social sharing.</p>
            </CardContent>
          </Card>

          {/* Taxonomies */}
          {nodeTypeDef?.taxonomies && (nodeTypeDef.taxonomies as Array<{slug: string; label: string; multiple?: boolean}>).length > 0 && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <SectionHeader title="Taxonomies" />
              <CardContent className="space-y-4">
                {(nodeTypeDef.taxonomies as Array<{slug: string; label: string; multiple?: boolean}>).map((tax) => {
                  const searchValue = taxonomySearch[tax.slug] || "";
                  const isOpen = taxonomyDropdownOpen[tax.slug] || false;
                  const terms = availableTerms[tax.slug] || [];
                  const selectedTerms = taxonomies[tax.slug] || [];
                  const filtered = searchValue.trim()
                    ? terms.filter(t => t.name.toLowerCase().includes(searchValue.toLowerCase()) && !selectedTerms.includes(t.name))
                    : terms.filter(t => !selectedTerms.includes(t.name));
                  const exactMatch = terms.some(t => t.name.toLowerCase() === searchValue.trim().toLowerCase());

                  return (
                    <div key={tax.slug} className="space-y-2">
                      <Label className="text-xs font-medium text-slate-500">{tax.label}</Label>
                      {/* Selected terms as badges */}
                      {selectedTerms.length > 0 && (
                        <div className="flex flex-wrap gap-1.5">
                          {selectedTerms.map((term, i) => (
                            <Badge key={i} variant="secondary" className="bg-indigo-50 text-indigo-700 hover:bg-indigo-100 border-indigo-100 gap-1 px-2 py-0.5 text-xs">
                              {term}
                              <button
                                type="button"
                                className="hover:text-red-500 ml-0.5"
                                onClick={() => {
                                  const newTerms = [...selectedTerms];
                                  newTerms.splice(i, 1);
                                  setTaxonomies({ ...taxonomies, [tax.slug]: newTerms });
                                }}
                              >
                                <X className="h-3 w-3" />
                              </button>
                            </Badge>
                          ))}
                        </div>
                      )}
                      {/* Search input */}
                      <div className="relative">
                        <Input
                          placeholder={`Search ${tax.label.toLowerCase()}...`}
                          value={searchValue}
                          onChange={(e) => {
                            setTaxonomySearch(prev => ({ ...prev, [tax.slug]: e.target.value }));
                            setTaxonomyDropdownOpen(prev => ({ ...prev, [tax.slug]: true }));
                          }}
                          onFocus={() => setTaxonomyDropdownOpen(prev => ({ ...prev, [tax.slug]: true }))}
                          onBlur={() => setTimeout(() => setTaxonomyDropdownOpen(prev => ({ ...prev, [tax.slug]: false })), 200)}
                          className="h-8 rounded-lg border-slate-300 text-xs focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                        />
                        {isOpen && (searchValue.trim() || filtered.length > 0) && (
                          <div className="absolute z-50 mt-1 w-full rounded-lg border border-slate-200 bg-white shadow-lg max-h-40 overflow-y-auto">
                            {filtered.length === 0 && !searchValue.trim() && (
                              <div className="px-3 py-2 text-xs text-slate-400">No terms available</div>
                            )}
                            {filtered.slice(0, 20).map((term) => (
                              <button
                                key={term.id}
                                type="button"
                                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-indigo-50 transition-colors"
                                onMouseDown={(e) => e.preventDefault()}
                                onClick={() => {
                                  if (!tax.multiple && selectedTerms.length > 0) {
                                    setTaxonomies({ ...taxonomies, [tax.slug]: [term.name] });
                                  } else {
                                    setTaxonomies({ ...taxonomies, [tax.slug]: [...selectedTerms, term.name] });
                                  }
                                  setTaxonomySearch(prev => ({ ...prev, [tax.slug]: "" }));
                                  setTaxonomyDropdownOpen(prev => ({ ...prev, [tax.slug]: false }));
                                }}
                              >
                                <Tag className="h-3 w-3 text-slate-400" />
                                <span className="font-medium text-slate-700">{term.name}</span>
                              </button>
                            ))}
                            {searchValue.trim() && !exactMatch && (
                              <button
                                type="button"
                                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-emerald-50 transition-colors border-t border-slate-100"
                                onMouseDown={(e) => e.preventDefault()}
                                onClick={async () => {
                                  const val = searchValue.trim();
                                  // Persist a real taxonomy_terms row in the node's
                                  // language so the term appears in the term list
                                  // and other nodes can reuse it. Pre-existing rows
                                  // surface SLUG_CONFLICT — fall back to selecting
                                  // the name without creating a duplicate row.
                                  try {
                                    const created = await createTerm(nodeTypeProp, tax.slug, {
                                      name: val,
                                      language_code: languageCode,
                                    });
                                    setAvailableTerms(prev => ({
                                      ...prev,
                                      [tax.slug]: [...(prev[tax.slug] || []), created],
                                    }));
                                  } catch {
                                    // Non-fatal: a row may already exist in another
                                    // language scope. The selection below still
                                    // wires the name onto the node.
                                  }
                                  if (!selectedTerms.includes(val)) {
                                    if (!tax.multiple && selectedTerms.length > 0) {
                                      setTaxonomies({ ...taxonomies, [tax.slug]: [val] });
                                    } else {
                                      setTaxonomies({ ...taxonomies, [tax.slug]: [...selectedTerms, val] });
                                    }
                                  }
                                  setTaxonomySearch(prev => ({ ...prev, [tax.slug]: "" }));
                                  setTaxonomyDropdownOpen(prev => ({ ...prev, [tax.slug]: false }));
                                }}
                              >
                                <Plus className="h-3 w-3 text-emerald-600" />
                                <span className="font-medium text-emerald-700">Create: {searchValue.trim()}</span>
                              </button>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </CardContent>
            </Card>
          )}

          {/* Translations (edit mode) — single dropdown pattern matches the
              term editor for consistency across the admin. */}
          {isEdit && (
            <Card className="rounded-xl border border-slate-200 shadow-sm">
              <SectionHeader title="Translations" />
              <CardContent>
                <div className="space-y-1.5">
                  <div className="flex items-center gap-2 rounded-md bg-indigo-50 border border-indigo-100 px-3 py-2">
                    <span className="text-sm">{languages.find(l => l.code === languageCode)?.flag || "🌐"}</span>
                    <span className="text-xs font-medium text-indigo-700 flex-1 truncate">{languages.find(l => l.code === languageCode)?.name || languageCode}</span>
                    <Badge className="bg-indigo-100 text-indigo-600 border-0 text-[10px] h-5">Current</Badge>
                  </div>
                  {translations.map((t) => {
                    const lang = languages.find(l => l.code === t.language_code);
                    return (
                      <Link
                        key={t.id}
                        to={`${basePath}/${t.id}/edit`}
                        className="flex items-center gap-2 rounded-md border border-slate-200 px-3 py-2 hover:bg-slate-50 transition-colors"
                      >
                        <span className="text-sm">{lang?.flag || "🌐"}</span>
                        <span className="text-xs font-medium text-slate-700 flex-1 truncate">{lang?.name || t.language_code}</span>
                        <Badge className={`border-0 text-[10px] h-5 ${t.status === "published" ? "bg-emerald-100 text-emerald-700" : "bg-slate-100 text-slate-500"}`}>
                          {t.status}
                        </Badge>
                      </Link>
                    );
                  })}
                  {translations.length === 0 && (
                    <p className="text-[11px] text-slate-400 text-center py-1">No translations yet</p>
                  )}
                </div>
                {(() => {
                  const taken = new Set([languageCode, ...translations.map((t) => t.language_code)]);
                  const remaining = languages.filter((l) => !taken.has(l.code));
                  if (remaining.length === 0) return null;
                  return (
                    <div className="mt-2">
                      <Select
                        value=""
                        onValueChange={(v) => v && handleCreateTranslation(v)}
                        disabled={creatingTranslation}
                      >
                        <SelectTrigger className="h-9 rounded-lg border-slate-300 text-sm">
                          <SelectValue
                            placeholder={creatingTranslation ? "Creating…" : "+ Add translation"}
                          />
                        </SelectTrigger>
                        <SelectContent>
                          {remaining.map((lang) => (
                            <SelectItem key={lang.code} value={lang.code}>
                              {lang.flag ? `${lang.flag} ` : ""}{lang.name || lang.code}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  );
                })()}
              </CardContent>
            </Card>
          )}

          {/* SEO Settings */}
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="SEO" />
            <CardContent className="space-y-3">
                <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-slate-500">
                    Meta Title
                  </Label>
                  <Input
                    placeholder={title || "Page title"}
                    value={seoTitle}
                    onChange={(e) => setSeoTitle(e.target.value)}
                    className="h-9 rounded-lg border-slate-300 text-sm focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                  <p className="text-[11px] text-slate-400">
                    {seoTitle.length || 0}/60 — Leave empty to use page title
                  </p>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-slate-500">
                    Meta Description
                  </Label>
                  <Textarea
                    placeholder="Brief description for search engines..."
                    value={seoDescription}
                    onChange={(e) => setSeoDescription(e.target.value)}
                    rows={3}
                    className="rounded-lg border-slate-300 text-sm focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 resize-none"
                  />
                  <p className="text-[11px] text-slate-400">
                    {seoDescription.length || 0}/160 recommended
                  </p>
                </div>
                {/* Preview */}
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
                  <p className="text-[11px] text-slate-400 mb-1">Search preview</p>
                  <p className="text-sm font-medium text-indigo-700 truncate">
                    {seoTitle || title || "Page Title"}
                  </p>
                  <p className="text-xs text-emerald-700 truncate">
                    {typeof window !== "undefined" ? window.location.origin : ""}
                    {originalNode?.full_url || "/"}
                  </p>
                  <p className="text-xs text-slate-500 line-clamp-2 mt-0.5">
                    {seoDescription || "No description set. Search engines will use page content."}
                  </p>
                </div>
              </CardContent>
          </Card>
        </div>
      </form>

      {/* Add Block picker */}
      <BlockPicker
        open={showAddBlock}
        onClose={() => setShowAddBlock(false)}
        onSelect={(item) => addBlock(item.slug)}
        items={blockTypes.map((bt) => ({
          id: bt.id,
          slug: bt.slug,
          label: bt.label,
          description: bt.description,
          icon: bt.icon,
        }))}
        title="Add Block"
        description="Select a block type to add to this page."
        emptyMessage="No block types available. Create block types first."
      />

      {/* Insert Template picker */}
      <BlockPicker
        open={showInsertTemplate}
        onClose={() => setShowInsertTemplate(false)}
        onSelect={(item) => {
          const tpl = templates.find((t) => t.id === item.id);
          if (tpl) insertTemplate(tpl);
        }}
        items={templates.map((tpl) => ({
          id: tpl.id,
          slug: tpl.slug,
          label: tpl.label,
          description: tpl.description,
          icon: "layout-template",
          badge: `${tpl.block_config?.length ?? 0} block(s)`,
        }))}
        title="Insert Template"
        description="Select a template to insert its blocks with default values."
        emptyMessage="No templates available. Create templates first."
      />

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

      {/* Load Page Template dialog */}
      {/* Load from Template picker */}
      <BlockPicker
        open={showLoadTemplate}
        onClose={() => setShowLoadTemplate(false)}
        onSelect={(item) => selectTemplate(item.id as number)}
        items={templates.map((tpl) => ({
          id: tpl.id,
          slug: tpl.slug,
          label: tpl.label,
          description: tpl.description,
          icon: "layout-template",
          badge: `${tpl.block_config?.length ?? 0} block(s)`,
        }))}
        title="Load from Template"
        description="Select a page template to apply. This will replace all existing blocks."
        emptyMessage="No templates available. Create templates first."
      />

      {/* Confirm template replacement dialog */}
      <Dialog open={showConfirmTemplate} onOpenChange={(open) => {
        if (!open) {
          setShowConfirmTemplate(false);
          setSelectedTemplateId(null);
        }
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Replace existing blocks?</DialogTitle>
            <DialogDescription>
              This will replace all existing blocks with the template&apos;s blocks. Any unsaved block content will be lost. Continue?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowConfirmTemplate(false);
                setSelectedTemplateId(null);
              }}
              disabled={applyingTemplate}
            >
              Cancel
            </Button>
            <Button
              className="bg-indigo-600 hover:bg-indigo-700 text-white"
              onClick={applyTemplate}
              disabled={applyingTemplate}
            >
              {applyingTemplate ? "Applying..." : "Apply Template"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

    </div>
  );
}
