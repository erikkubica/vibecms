import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, useSearchParams, Link } from "react-router-dom";
import {
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
  Eye,
  Code as CodeIcon,
  Tag,
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
import { Titlebar } from "@/components/ui/titlebar";
import { TabsCard } from "@/components/ui/tabs-card";
import { PublishActions } from "@/components/ui/publish-actions";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LanguageSelect, LanguageLabel } from "@/components/ui/language-select";
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
import BlockPicker from "@/components/ui/block-picker";
import { usePageMeta } from "@/components/layout/page-meta";
import { useExtensions } from "@/hooks/use-extensions";
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
  listNodeRevisions,
  restoreNodeRevision,
  type NodeRevision,
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
  const [activeTab, setActiveTab] = useState<"blocks" | "excerpt" | "seo" | "custom">("blocks");

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

  // SEO. The per-node SEO panel only renders when an extension declares
  // provides:["seo"] in its manifest (the bundled seo-extension does).
  // Without an active SEO provider the editor doesn't surface the tab —
  // matches the kernel/extensions hard rule that disabling the extension
  // should remove the feature, not leave dead UI behind.
  const { hasProvider } = useExtensions();
  const seoEnabled = hasProvider("seo");
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

  // Revisions — list of historical snapshots, newest first. Loaded on
  // mount in edit mode and refreshed after each save so a fresh save
  // surfaces in the panel without a manual reload.
  const [revisions, setRevisions] = useState<NodeRevision[]>([]);
  const [restoringRevisionID, setRestoringRevisionID] = useState<number | null>(null);
  const [showRestoreConfirm, setShowRestoreConfirm] = useState<NodeRevision | null>(null);

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
  }, []);

  // Refetch the homepage_node_id whenever the editor switches to a
  // different node language — homepage_node_id is per-locale, so the
  // "Current homepage" / "Set as homepage" button has to consult the
  // row for THIS node's language. Without this, switching from EN to
  // DE in the language picker would show the EN homepage's flag on
  // the DE node.
  useEffect(() => {
    if (!languageCode) return;
    getHomepageId(languageCode).then(setHomepageId).catch(() => {});
  }, [languageCode]);

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

  // Load revision history when editing. Refreshes after each save via
  // the saving→idle transition so the just-created snapshot shows up
  // without a manual reload.
  useEffect(() => {
    if (!isEdit || !id) return;
    if (saving) return;
    listNodeRevisions(id)
      .then(setRevisions)
      .catch(() => setRevisions([]));
  }, [id, isEdit, saving]);

  async function handleRestoreRevision(rev: NodeRevision) {
    if (!id) return;
    setRestoringRevisionID(rev.id);
    try {
      await restoreNodeRevision(id, rev.id);
      toast.success("Revision restored. The previous state was saved as a new revision.");
      setShowRestoreConfirm(null);
      // Refresh node + revisions. Easiest is a soft reload of the
      // editor route — the page already pulls fresh data on mount.
      window.location.reload();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to restore revision";
      toast.error(message);
    } finally {
      setRestoringRevisionID(null);
    }
  }

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

  // saveNode performs the actual write to the database. Used by the
  // form-submit Save button and the Publish button. Preview does NOT
  // call this — Preview is a side-effect-free render that POSTs the
  // in-flight form state to the server, gets HTML back, and renders
  // it without touching the DB. Returns true on success.
  async function saveNode(publishStatus?: string): Promise<boolean> {
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
        return true;
      }
      const created = await createNode(nodeData);
      toast.success(`${label} created successfully`);
      navigate(`${basePath}/${created.id}/edit`, { replace: true });
      return true;
    } catch (err) {
      const message =
        err instanceof Error ? err.message : `Failed to save ${label.toLowerCase()}`;
      toast.error(message);
      return false;
    } finally {
      setSaving(false);
    }
  }

  async function handleSave(e: FormEvent, publishStatus?: string) {
    e.preventDefault();
    await saveNode(publishStatus);
  }

  // handlePreview renders the in-flight form state in a new tab WITHOUT
  // saving. POSTs the current editor state to the preview endpoint, gets
  // HTML back, and points a placeholder tab at a blob URL of the
  // response. Nothing touches the database — close the tab and the
  // preview is gone.
  //
  // The placeholder tab is opened synchronously on the click event
  // (popup blockers ignore async window.open). On any failure we close
  // it so the operator never sees stale output.
  function handlePreview() {
    if (!isEdit || !id) return;
    // Open without `noopener` — that feature flag forces window.open to
    // return null even when the popup actually opened, which made the
    // 'popup blocked' toast fire on a tab that DID open. The preview
    // navigates to a blob URL on the same origin, so the lack of opener
    // isolation is acceptable here.
    const previewWindow = window.open("about:blank", "_blank");
    if (!previewWindow) {
      toast.error("Popup blocked — allow popups for this site to use Preview.");
      return;
    }
    previewWindow.document.open();
    previewWindow.document.write(
      `<!doctype html><meta charset="utf-8"><title>Loading preview…</title>` +
      `<style>body{font:14px/1.5 system-ui,sans-serif;color:#475569;margin:48px;text-align:center}</style>` +
      `Rendering preview…`,
    );
    previewWindow.document.close();

    void (async () => {
      try {
        const draft = {
          title,
          slug,
          status,
          language_code: languageCode,
          excerpt,
          layout_id: layoutId ? Number(layoutId) : undefined,
          blocks_data: blocks,
          fields_data: fieldsData,
          seo_settings: { meta_title: seoTitle, meta_description: seoDescription },
          featured_image: featuredImage,
          taxonomies: taxonomies,
        };
        const res = await fetch(`/admin/api/nodes/${id}/preview`, {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(draft),
        });
        const html = await res.text();
        // Use a Blob URL so the rendered document gets a proper origin
        // and absolute paths (/theme/assets/..., /media/...) resolve
        // against this site. document.write into about:blank would set
        // the base URL to about:blank — every absolute path 404s.
        const blob = new Blob([html], { type: "text/html;charset=utf-8" });
        previewWindow.location.href = URL.createObjectURL(blob);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to render preview";
        toast.error(message);
        previewWindow.close();
      }
    })();
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
      // Always write the homepage for this node's locale — never the
      // admin's current locale. Otherwise saving the German node as
      // homepage from an English admin context lands the node id in
      // the EN row and /en serves a German page (the kernel's
      // validateNodeSelect would now reject this anyway, but routing
      // the write to the right locale up front avoids the round-trip).
      await setHomepage(id, languageCode);
      setHomepageId(Number(id));
      toast.success("Homepage updated successfully");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to set homepage";
      toast.error(msg);
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin" style={{ color: "var(--accent-strong)" }} />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <form onSubmit={(e) => handleSave(e)} className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Main content */}
        <div className="space-y-4 min-w-0">
          {/* Titlebar — title segment + slug segment + ID/View */}
          <Titlebar
            title={title}
            onTitleChange={setTitle}
            titlePlaceholder={`Enter ${label.toLowerCase()} title`}
            slug={slug}
            onSlugChange={(v) => { setAutoSlug(false); setSlug(v); }}
            slugPrefix={`/${langSlug ? `${langSlug}/` : ""}${urlPrefix ? `${urlPrefix}/` : ""}${parentNode ? `${parentNode.slug}/` : ""}`}
            autoSlug={autoSlug}
            onAutoSlugToggle={() => setAutoSlug(!autoSlug)}
            id={isEdit ? id : undefined}
            viewHref={isEdit && originalNode && status === "published" ? originalNode.full_url : undefined}
            onBack={() => navigate(basePath)}
          />

          {/* Visual Block Editor — unified TabsCard primitive */}
          <TabsCard
            value={activeTab}
            onValueChange={(v) => setActiveTab(v as typeof activeTab)}
            tabs={[
              ...(blocksEnabled
                ? [
                    {
                      value: "blocks",
                      label: "Blocks",
                      badge: blocks.length,
                      content: (<>
            <div className="flex items-center justify-between mb-2">
              <span style={{ fontSize: 12, color: "var(--fg-muted)", letterSpacing: "-0.005em" }}>
                Drag, expand, or remove blocks. Changes are saved on Save.
              </span>
              <button
                type="button"
                onClick={openLoadTemplate}
                className="inline-flex items-center cursor-pointer"
                style={{
                  height: 26,
                  padding: "0 8px",
                  borderRadius: 6,
                  background: "transparent",
                  color: "var(--fg-muted)",
                  fontSize: 11.5,
                  fontWeight: 500,
                  gap: 5,
                  border: "none",
                  transition: "background 0.12s, color 0.12s",
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = "var(--hover-bg)";
                  e.currentTarget.style.color = "var(--fg)";
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = "transparent";
                  e.currentTarget.style.color = "var(--fg-muted)";
                }}
              >
                <LayoutTemplate size={12} style={{ opacity: 0.65 }} />
                Load template
              </button>
            </div>
            <div>
              {blocks.length === 0 && (
                <div
                  className="flex flex-col items-center justify-center"
                  style={{
                    gap: 8,
                    padding: "44px 0",
                    borderRadius: "var(--radius-lg)",
                    background: "var(--sub-bg)",
                    color: "var(--fg-subtle)",
                  }}
                >
                  <Square style={{ width: 28, height: 28, color: "var(--fg-subtle)" }} />
                  <p style={{ fontSize: 13, fontWeight: 500, color: "var(--fg-muted)", margin: 0 }}>No blocks yet</p>
                  <p style={{ fontSize: 12, color: "var(--fg-subtle)", margin: 0 }}>Add a block or load a template to get started</p>
                </div>
              )}

              {blocks.map((block, index) => {
                const blockTypeDef = getBlockTypeDef(block.type);
                const isCollapsed = collapsedBlocks.has(index);
                const blockFields = blockTypeDef?.field_schema || [];

                const typeCategory = block.type.split("-")[0];
                return (
                  <div
                    key={index}
                    style={{
                      borderTop: index === 0 ? "none" : "1px solid var(--divider)",
                    }}
                  >
                    {/* Block header — flat row, no bg, no border. Active state via type weight. */}
                    <div
                      className="flex items-center cursor-pointer select-none"
                      style={{
                        gap: 8,
                        padding: "11px 12px",
                        margin: "0 -8px",
                        borderRadius: 6,
                        background: !isCollapsed ? "transparent" : undefined,
                        transition: "background 0.1s",
                      }}
                      onClick={() => toggleBlockCollapse(index)}
                      onMouseEnter={(e) => { if (isCollapsed) e.currentTarget.style.background = "var(--hover-bg)"; }}
                      onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                    >
                      <span
                        onClick={(e) => e.stopPropagation()}
                        style={{ color: "var(--fg-hint)", cursor: "grab", flexShrink: 0, opacity: 0.55, display: "inline-flex" }}
                      >
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7">
                          <circle cx="9" cy="6" r="1.2" fill="currentColor" />
                          <circle cx="9" cy="12" r="1.2" fill="currentColor" />
                          <circle cx="9" cy="18" r="1.2" fill="currentColor" />
                          <circle cx="15" cy="6" r="1.2" fill="currentColor" />
                          <circle cx="15" cy="12" r="1.2" fill="currentColor" />
                          <circle cx="15" cy="18" r="1.2" fill="currentColor" />
                        </svg>
                      </span>
                      <ChevronDown
                        size={11}
                        className="shrink-0 transition-transform"
                        style={{
                          color: "var(--fg-subtle)",
                          transform: isCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                        }}
                      />
                      <span
                        style={{
                          fontSize: 13,
                          fontWeight: isCollapsed ? 500 : 600,
                          color: "var(--fg)",
                          letterSpacing: "-0.01em",
                        }}
                      >
                        {getBlockLabel(block.type)}
                      </span>
                      <span style={{ fontFamily: "var(--font-mono)", fontSize: 11, color: "var(--fg-subtle)" }}>
                        {block.type}
                      </span>
                      {typeCategory && typeCategory !== block.type && (
                        <span
                          style={{
                            fontSize: 10,
                            fontWeight: 500,
                            fontFamily: "var(--font-mono)",
                            padding: "1.5px 6px",
                            borderRadius: 3,
                            background: !isCollapsed ? "var(--accent-weak)" : "var(--sub-bg)",
                            color: !isCollapsed ? "var(--accent-strong)" : "var(--fg-muted)",
                            textTransform: "lowercase",
                            letterSpacing: "0.02em",
                          }}
                        >
                          {typeCategory}
                        </span>
                      )}
                      <span style={{ fontFamily: "var(--font-mono)", fontSize: 10.5, color: "var(--fg-subtle)", marginLeft: 4 }}>
                        · {blockFields.length} {blockFields.length === 1 ? "field" : "fields"}
                      </span>
                      <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 1 }} onClick={(e) => e.stopPropagation()}>
                        <button
                          type="button"
                          onClick={() => moveBlock(index, "up")}
                          disabled={index === 0}
                          style={{ width: 26, height: 26, borderRadius: 5, display: "grid", placeItems: "center", color: "var(--fg-subtle)", background: "transparent", border: "none", cursor: index === 0 ? "default" : "pointer", opacity: index === 0 ? 0.3 : 1, transition: "background 0.1s" }}
                          onMouseEnter={(e) => { if (index !== 0) { e.currentTarget.style.background = "rgba(0,0,0,0.05)"; e.currentTarget.style.color = "var(--fg)"; } }}
                          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "var(--fg-subtle)"; }}
                          title="Move up"
                        >
                          <ChevronUp size={13} />
                        </button>
                        <button
                          type="button"
                          onClick={() => moveBlock(index, "down")}
                          disabled={index === blocks.length - 1}
                          style={{ width: 26, height: 26, borderRadius: 5, display: "grid", placeItems: "center", color: "var(--fg-subtle)", background: "transparent", border: "none", cursor: index === blocks.length - 1 ? "default" : "pointer", opacity: index === blocks.length - 1 ? 0.3 : 1, transition: "background 0.1s" }}
                          onMouseEnter={(e) => { if (index !== blocks.length - 1) { e.currentTarget.style.background = "rgba(0,0,0,0.05)"; e.currentTarget.style.color = "var(--fg)"; } }}
                          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "var(--fg-subtle)"; }}
                          title="Move down"
                        >
                          <ChevronDown size={13} />
                        </button>
                        <button
                          type="button"
                          onClick={() => removeBlock(index)}
                          style={{ width: 26, height: 26, borderRadius: 5, display: "grid", placeItems: "center", color: "var(--fg-subtle)", background: "transparent", border: "none", cursor: "pointer", transition: "background 0.1s, color 0.1s" }}
                          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--danger-bg)"; e.currentTarget.style.color = "var(--danger)"; }}
                          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "var(--fg-subtle)"; }}
                          title="Delete block"
                        >
                          <X size={13} />
                        </button>
                      </div>
                    </div>

                    {/* Block fields */}
                    {!isCollapsed && (
                      <div style={{ padding: "4px 12px 16px" }} className="space-y-3">
                        {blockTypeDef?.description && (
                          <div
                            style={{
                              fontSize: 11.5,
                              color: "var(--fg-muted)",
                              padding: "10px 12px",
                              background: "var(--sub-bg)",
                              borderRadius: 7,
                              lineHeight: 1.55,
                              fontStyle: "italic",
                              marginBottom: 12,
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
                          <div className="flex flex-wrap" style={{ gap: "11px 14px" }}>
                            {blockFields.map((field) => {
                              const w = getFieldWidth(field);
                              const fieldKey = field.key;
                              return (
                                <div
                                  key={field.key}
                                  className="min-w-0 flex flex-col"
                                  style={{
                                    flex: `0 0 calc(${w}% - 14px)`,
                                    maxWidth: `calc(${w}% - 14px)`,
                                    gap: 5,
                                  }}
                                >
                                  <Label className="flex items-center" style={{ fontSize: 12, fontWeight: 500, color: "var(--fg)", letterSpacing: "-0.005em", gap: 5 }}>
                                    {field.label}
                                    <span style={{ fontFamily: "var(--font-mono)", fontSize: 10.5, color: "var(--fg-subtle)", fontWeight: 400, marginLeft: 4 }}>{fieldKey}</span>
                                    {field.required && <span style={{ color: "var(--danger)" }}>*</span>}
                                  </Label>
                                  {(field as { hint?: string }).hint && (
                                    <span style={{ fontSize: 11.5, color: "var(--fg-muted)", lineHeight: 1.45 }}>{(field as { hint?: string }).hint}</span>
                                  )}
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

              {/* Add Block / Insert Template — ghosted, type-led, full-width row */}
              <div
                className="flex"
                style={{
                  gap: 4,
                  marginTop: 6,
                  paddingTop: 8,
                  borderTop: "1px solid var(--divider)",
                }}
              >
                <button
                  type="button"
                  onClick={() => setShowAddBlock(true)}
                  className="inline-flex items-center justify-center"
                  style={{
                    flex: 1,
                    height: 34,
                    borderRadius: 7,
                    background: "transparent",
                    color: "var(--fg-muted)",
                    fontSize: 12.5,
                    fontWeight: 500,
                    gap: 6,
                    border: "none",
                    cursor: "pointer",
                    transition: "background 0.12s, color 0.12s",
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = "var(--hover-bg)";
                    e.currentTarget.style.color = "var(--fg)";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = "transparent";
                    e.currentTarget.style.color = "var(--fg-muted)";
                  }}
                >
                  <Plus size={13} style={{ opacity: 0.7 }} />
                  Add block
                </button>
                {templates.length > 0 && (
                  <button
                    type="button"
                    onClick={() => setShowInsertTemplate(true)}
                    className="inline-flex items-center justify-center"
                    style={{
                      flex: 1,
                      height: 34,
                      borderRadius: 7,
                      background: "transparent",
                      color: "var(--fg-muted)",
                      fontSize: 12.5,
                      fontWeight: 500,
                      gap: 6,
                      border: "none",
                      cursor: "pointer",
                      transition: "background 0.12s, color 0.12s",
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = "var(--hover-bg)";
                      e.currentTarget.style.color = "var(--fg)";
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = "transparent";
                      e.currentTarget.style.color = "var(--fg-muted)";
                    }}
                  >
                    <LayoutTemplate size={13} style={{ opacity: 0.7 }} />
                    Insert template
                  </button>
                )}
              </div>

              {/* Raw JSON collapsible — sep-label style: mono uppercase tiny text with flanking dividers */}
              <div>
                <button
                  type="button"
                  onClick={() => {
                    if (!showRawJson) {
                      setRawJson(JSON.stringify(blocks, null, 2));
                    }
                    setShowRawJson(!showRawJson);
                  }}
                  className="flex items-center cursor-pointer w-full"
                  style={{
                    gap: 8,
                    margin: "8px 0 4px",
                    padding: 0,
                    border: "none",
                    background: "transparent",
                    fontSize: 11,
                    fontWeight: 500,
                    fontFamily: "var(--font-mono)",
                    textTransform: "uppercase",
                    letterSpacing: "0.05em",
                    color: "var(--fg-muted)",
                    transition: "color 0.12s",
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.color = "var(--fg-2)")}
                  onMouseLeave={(e) => (e.currentTarget.style.color = "var(--fg-muted)")}
                >
                  <span style={{ flex: 1, height: 1, background: "var(--divider)" }} />
                  <CodeIcon size={11} />
                  <span>Raw JSON</span>
                  <ChevronRight size={9} style={{ transform: showRawJson ? "rotate(90deg)" : "rotate(0deg)", transition: "transform 0.15s" }} />
                  <span style={{ flex: 1, height: 1, background: "var(--divider)" }} />
                </button>
                {showRawJson && (
                  <div className="mt-3 space-y-2">
                    <Textarea
                      value={rawJson}
                      onChange={(e) => setRawJson(e.target.value)}
                      rows={12}
                      className="font-mono text-xs rounded-lg"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="rounded-lg text-xs"
                      onClick={applyRawJson}
                    >
                      Apply JSON
                    </Button>
                  </div>
                )}
              </div>
            </div>
              </>),
                    },
                  ]
                : []),
              {
                value: "excerpt",
                label: "Excerpt",
                content: (<>
                <div className="space-y-2">
                  <Label style={{ fontSize: 12, fontWeight: 500, color: "var(--fg)", letterSpacing: "-0.005em" }}>Excerpt</Label>
                  <p style={{ fontSize: 11.5, color: "var(--fg-muted)", lineHeight: 1.45 }}>
                    Short description used in cards and search results. If empty, may be auto-generated.
                  </p>
                  <Textarea
                    placeholder="Enter a short summary or teaser…"
                    value={excerpt}
                    onChange={(e) => setExcerpt(e.target.value)}
                    rows={4}
                    className="resize-none"
                    style={{ marginTop: 4 }}
                  />
                </div>
              </>),
              },
              ...(seoEnabled ? [{
                value: "seo",
                label: "SEO",
                content: (<>
                <div className="flex flex-col" style={{ gap: 14 }}>
                  <div className="space-y-1.5">
                    <Label style={{ fontSize: 12, fontWeight: 500, color: "var(--fg)", letterSpacing: "-0.005em" }}>SEO title</Label>
                    <p style={{ fontSize: 11.5, color: "var(--fg-muted)", lineHeight: 1.45 }}>
                      Overrides the page title in search results
                    </p>
                    <Input
                      placeholder={title || "Leave blank to use page title"}
                      value={seoTitle}
                      onChange={(e) => setSeoTitle(e.target.value)}
                      style={{ marginTop: 4 }}
                    />
                    <p className="text-[11px]" style={{ color: "var(--fg-subtle)" }}>
                      {seoTitle.length || 0}/60 — Leave empty to use page title
                    </p>
                  </div>
                  <div className="space-y-1.5">
                    <Label style={{ fontSize: 12, fontWeight: 500, color: "var(--fg)", letterSpacing: "-0.005em" }}>Meta description</Label>
                    <p style={{ fontSize: 11.5, color: "var(--fg-muted)", lineHeight: 1.45 }}>
                      Shown in search results and link previews
                    </p>
                    <Textarea
                      placeholder="Enter a meta description…"
                      value={seoDescription}
                      onChange={(e) => setSeoDescription(e.target.value)}
                      rows={3}
                      className="resize-none"
                      style={{ marginTop: 4 }}
                    />
                    <p className="text-[11px]" style={{ color: "var(--fg-subtle)" }}>
                      {seoDescription.length || 0}/160 recommended
                    </p>
                  </div>
                  <div
                    className="rounded-lg p-3"
                    style={{ background: "var(--sub-bg)", border: "1px solid var(--border)" }}
                  >
                    <p className="text-[11px] mb-1" style={{ color: "var(--fg-subtle)" }}>Search preview</p>
                    <p className="text-sm font-medium truncate" style={{ color: "var(--accent-strong)" }}>
                      {seoTitle || title || "Page Title"}
                    </p>
                    <p className="text-xs truncate" style={{ color: "var(--success)" }}>
                      {typeof window !== "undefined" ? window.location.origin : ""}
                      {originalNode?.full_url || "/"}
                    </p>
                    <p className="text-xs text-muted-foreground line-clamp-2 mt-0.5">
                      {seoDescription || "No description set. Search engines will use page content."}
                    </p>
                  </div>
                </div>
              </>),
              }] : []),
              {
                value: "custom",
                label: "Custom Fields",
                badge: customFields.length,
                content: (<>
                {customFields.length === 0 ? (
                  <div style={{ color: "var(--fg-muted)", fontSize: 12.5, padding: "26px 0", textAlign: "center" }}>
                    No custom fields defined for this content type.
                  </div>
                ) : (
                  <div className="flex flex-wrap" style={{ gap: "16px 14px" }}>
                    {customFields.map((field) => {
                      const w = getFieldWidth(field);
                      const fieldKey = field.key || field.name;
                      return (
                        <div
                          key={field.key}
                          className="space-y-1.5 min-w-0"
                          style={{ flex: `0 0 calc(${w}% - 14px)`, maxWidth: `calc(${w}% - 14px)` }}
                        >
                          <Label className="flex items-center" style={{ fontSize: 12, fontWeight: 500, color: "var(--fg)", letterSpacing: "-0.005em", gap: 5 }}>
                            {field.label}
                            <span style={{ fontFamily: "var(--font-mono)", fontSize: 10.5, color: "var(--fg-subtle)", fontWeight: 400, marginLeft: 4 }}>{fieldKey}</span>
                            {field.required && <span style={{ color: "var(--danger)" }}>*</span>}
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
                )}
              </>),
              },
            ]}
          />

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
                          <Label className="text-sm font-medium text-foreground">
                            {field.label}
                            {field.required && <span className="ml-1" style={{ color: "var(--danger)" }}>*</span>}
                          </Label>
                          {(field as any).default_from && !partialData[fieldKey] && (
                            <p className="text-xs" style={{ color: "var(--fg-subtle)" }}>
                              Falls back to <code className="bg-muted px-1 rounded">{(field as any).default_from}</code>
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
          <Card>
            <SectionHeader
              title="Publish"
              actions={
                <span
                  className="inline-flex items-center"
                  style={{
                    gap: 5,
                    padding: "2.5px 8px",
                    borderRadius: 11,
                    fontSize: 11,
                    fontWeight: 500,
                    letterSpacing: "-0.003em",
                    color: status === "published" ? "var(--success)" : status === "draft" ? "var(--fg-muted)" : "var(--warning)",
                    background: status === "published" ? "var(--success-bg)" : status === "draft" ? "var(--sub-bg)" : "var(--warning-bg)",
                  }}
                >
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: "50%",
                      background: status === "published" ? "var(--success)" : status === "draft" ? "var(--fg-subtle)" : "var(--warning)",
                      boxShadow: status === "published" ? "0 0 0 2px color-mix(in oklab, var(--success) 22%, transparent)" : undefined,
                    }}
                  />
                  {status === "published" ? "Published" : status === "draft" ? "Draft" : status[0].toUpperCase() + status.slice(1)}
                </span>
              }
            />
            <CardContent className="space-y-4">
              {/* Status + Language row */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-muted-foreground">Status</Label>
                  <Select value={status} onValueChange={setStatus}>
                    <SelectTrigger className="h-9 rounded-lg text-sm">
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
                  <Label className="text-xs font-medium text-muted-foreground">Language</Label>
                  <LanguageSelect
                    languages={languages}
                    value={languageCode}
                    onChange={setLanguageCode}
                  />
                </div>
              </div>

              {/* Layout */}
              <div className="space-y-1.5">
                  <Label className="text-xs font-medium text-muted-foreground">Layout</Label>
                  <Select value={layoutId || "auto"} onValueChange={(v) => setLayoutId(v === "auto" ? "" : v)}>
                    <SelectTrigger className="h-9 rounded-lg text-sm">
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
                  <Label className="text-xs font-medium text-muted-foreground">Parent</Label>
                  {parentNode ? (
                    <div className="flex items-center gap-2 h-9 rounded-lg border px-3" style={{ borderColor: "var(--accent-mid)", background: "var(--accent-weak)" }}>
                      <span className="flex-1 text-sm font-medium text-foreground truncate">{parentNode.title}</span>
                      <span className="text-[10px] font-mono" style={{ color: "var(--fg-subtle)" }}>/{parentNode.slug}</span>
                      <button
                        type="button"
                        className="hover:text-destructive shrink-0"
                        style={{ color: "var(--fg-subtle)" }}
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
                        className="h-9 rounded-lg text-sm"
                      />
                      {showParentResults && (parentSearch.trim() || parentSearching) && (
                        <div className="absolute z-50 mt-1 w-full rounded-lg border border-border bg-card shadow-lg max-h-48 overflow-y-auto">
                          {parentSearching ? (
                            <div className="px-3 py-2 text-sm" style={{ color: "var(--fg-subtle)" }}>Searching...</div>
                          ) : parentResults.length === 0 ? (
                            <div className="px-3 py-2 text-sm" style={{ color: "var(--fg-subtle)" }}>
                              {parentSearch.trim() ? "No results found" : "Type to search..."}
                            </div>
                          ) : (
                            parentResults.map((node) => (
                              <button
                                key={node.id}
                                type="button"
                                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent transition-colors"
                                onMouseDown={(e) => e.preventDefault()}
                                onClick={() => {
                                  setParentId(String(node.id));
                                  setParentNode(node);
                                  setParentSearch("");
                                  setParentResults([]);
                                  setShowParentResults(false);
                                }}
                              >
                                <span className="font-medium text-foreground truncate">{node.title}</span>
                                <span className="text-[10px] font-mono ml-auto shrink-0" style={{ color: "var(--fg-subtle)" }}>/{node.slug}</span>
                              </button>
                            ))
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </div>


              {/* Publish actions — 2-col grid: Save+Publish/Unpublish, Preview+Delete */}
              <PublishActions>
                <Button type="submit" className="w-full" disabled={saving}>
                  <Save className="mr-1.5 h-3.5 w-3.5" />
                  {saving ? "Saving…" : "Save"}
                </Button>
                {status !== "published" ? (
                  <Button
                    type="button"
                    className="w-full"
                    style={{ background: "var(--success)", color: "#fff" }}
                    disabled={saving}
                    onClick={(e) => handleSave(e, "published")}
                  >
                    <Globe className="mr-1.5 h-3.5 w-3.5" />
                    Publish
                  </Button>
                ) : (
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full"
                    disabled={saving}
                    onClick={(e) => handleSave(e, "draft")}
                  >
                    <Globe className="mr-1.5 h-3.5 w-3.5" />
                    Unpublish
                  </Button>
                )}
                {isEdit && id && (
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full"
                    disabled={saving}
                    onClick={handlePreview}
                    title="Save and open the rendered page in a new tab."
                  >
                    <Eye className="mr-1.5 h-3.5 w-3.5" />
                    Preview
                  </Button>
                )}
                {isEdit && (
                  <Button
                    type="button"
                    variant="ghost"
                    className="w-full"
                    style={{ color: "var(--danger)" }}
                    onClick={() => setShowDelete(true)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                )}
              </PublishActions>

              {/* Set as Homepage — separate row for pages only */}
              {isEdit && nodeTypeProp === "page" && (
                homepageId === Number(id) ? (
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full cursor-default"
                    style={{ background: "var(--success-bg)", color: "var(--success)" }}
                    disabled
                  >
                    <Home className="mr-1.5 h-3.5 w-3.5" />
                    Current homepage
                  </Button>
                ) : (
                  <Button
                    type="button"
                    variant="ghost"
                    className="w-full"
                    onClick={handleSetHomepage}
                  >
                    <Home className="mr-1.5 h-3.5 w-3.5" />
                    Set as homepage
                  </Button>
                )
              )}

              {/* Metadata (edit mode) */}
              {isEdit && originalNode && (
                <>
                  <Separator />
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs" style={{ color: "var(--fg-subtle)" }}>
                    <div className="flex justify-between">
                      <span>Version</span>
                      <span className="font-mono text-muted-foreground">{originalNode.version}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Created</span>
                      <span className="text-muted-foreground">{new Date(originalNode.created_at).toLocaleDateString("en-GB")}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Updated</span>
                      <span className="text-muted-foreground">{new Date(originalNode.updated_at).toLocaleDateString("en-GB")}</span>
                    </div>
                    {originalNode.published_at && (
                      <div className="flex justify-between">
                        <span>Published</span>
                        <span className="text-muted-foreground">{new Date(originalNode.published_at).toLocaleDateString("en-GB")}</span>
                      </div>
                    )}
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* Featured Image */}
          <Card>
            <SectionHeader title="Featured Image" />
            <CardContent className="space-y-2">
              <CustomFieldInput
                field={{ name: "featured_image", key: "featured_image", title: "Featured Image", label: "Featured Image", type: "image" }}
                value={featuredImage}
                onChange={(val) => setFeaturedImage(val as Record<string, unknown>)}
              />
              <p className="text-[11px]" style={{ color: "var(--fg-subtle)" }}>Main image used for listings, sliders, and social sharing.</p>
            </CardContent>
          </Card>

          {/* Taxonomies */}
          {nodeTypeDef?.taxonomies && (nodeTypeDef.taxonomies as Array<{slug: string; label: string; multiple?: boolean}>).length > 0 && (
            <Card>
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
                      <Label className="text-xs font-medium text-muted-foreground">{tax.label}</Label>
                      {/* Selected terms as badges */}
                      {selectedTerms.length > 0 && (
                        <div className="flex flex-wrap gap-1.5">
                          {selectedTerms.map((term, i) => (
                            <Badge key={i} variant="secondary" className="gap-1 px-2 py-0.5 text-xs" style={{ background: "var(--accent-weak)", color: "var(--accent-strong)", borderColor: "var(--accent-mid)" }}>
                              {term}
                              <button
                                type="button"
                                className="hover:text-destructive ml-0.5"
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
                          className="h-8 rounded-lg text-xs"
                        />
                        {isOpen && (searchValue.trim() || filtered.length > 0) && (
                          <div className="absolute z-50 mt-1 w-full rounded-lg border border-border bg-card shadow-lg max-h-40 overflow-y-auto">
                            {filtered.length === 0 && !searchValue.trim() && (
                              <div className="px-3 py-2 text-xs" style={{ color: "var(--fg-subtle)" }}>No terms available</div>
                            )}
                            {filtered.slice(0, 20).map((term) => (
                              <button
                                key={term.id}
                                type="button"
                                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-accent transition-colors"
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
                                <Tag className="h-3 w-3" style={{ color: "var(--fg-subtle)" }} />
                                <span className="font-medium text-foreground">{term.name}</span>
                              </button>
                            ))}
                            {searchValue.trim() && !exactMatch && (
                              <button
                                type="button"
                                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-accent transition-colors border-t border-border"
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
                                <Plus className="h-3 w-3" style={{ color: "var(--success)" }} />
                                <span className="font-medium" style={{ color: "var(--success)" }}>Create: {searchValue.trim()}</span>
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
              term editor for consistency across the admin. No flags;
              language names are the canonical identifier. */}
          {isEdit && (
            <Card>
              <SectionHeader title="Translations" />
              <CardContent>
                <div className="space-y-1.5">
                  <div className="flex items-center gap-2 rounded-md border px-3 py-2" style={{ background: "var(--accent-weak)", borderColor: "var(--accent-mid)" }}>
                    <span className="text-xs font-medium flex-1 truncate" style={{ color: "var(--accent-strong)" }}>
                      <LanguageLabel languages={languages} code={languageCode} />
                    </span>
                    <Badge className="border-0 text-[10px] h-5" style={{ background: "var(--accent-weak)", color: "var(--accent-strong)" }}>Current</Badge>
                  </div>
                  {translations.map((t) => (
                    <Link
                      key={t.id}
                      to={`${basePath}/${t.id}/edit`}
                      className="flex items-center gap-2 rounded-md border border-border px-3 py-2 hover:bg-muted transition-colors"
                    >
                      <span className="text-xs font-medium text-foreground flex-1 truncate">
                        <LanguageLabel languages={languages} code={t.language_code} />
                      </span>
                      <Badge
                        className="border-0 text-[10px] h-5"
                        style={t.status === "published"
                          ? { background: "var(--success-bg)", color: "var(--success)" }
                          : { background: "var(--muted)", color: "var(--muted-foreground)" }}
                      >
                        {t.status}
                      </Badge>
                    </Link>
                  ))}
                  {translations.length === 0 && (
                    <p className="text-[11px] text-center py-1" style={{ color: "var(--fg-subtle)" }}>No translations yet</p>
                  )}
                </div>
                <div className="mt-2">
                  <LanguageSelect
                    mode="add"
                    languages={languages}
                    existing={[languageCode, ...translations.map((t) => t.language_code)]}
                    onAdd={handleCreateTranslation}
                    disabled={creatingTranslation}
                    placeholder={creatingTranslation ? "Creating…" : "+ Add translation"}
                  />
                </div>
              </CardContent>
            </Card>
          )}

          {/* Revisions — every save creates a snapshot. Restoring an older
              revision is itself a save, so the prior state stays
              recoverable. List capped to the most recent 100. */}
          {isEdit && id && (
            <Card>
              <SectionHeader title="Revisions" />
              <CardContent>
                {revisions.length === 0 ? (
                  <p className="text-[11px] text-center py-1" style={{ color: "var(--fg-subtle)" }}>
                    No revisions yet. Save the page to create one.
                  </p>
                ) : (
                  <div className="space-y-1.5 max-h-72 overflow-y-auto pr-1">
                    {revisions.map((r) => {
                      const when = new Date(r.created_at);
                      const author = r.creator_name || r.creator_email || (r.created_by ? `user #${r.created_by}` : "system");
                      return (
                        <div
                          key={r.id}
                          className="flex items-center gap-2 rounded-md border border-border px-3 py-2 hover:bg-muted"
                        >
                          <div className="flex-1 min-w-0">
                            <p className="text-xs font-medium text-foreground truncate">
                              v{r.version_number || "—"} · {when.toLocaleString()}
                            </p>
                            <p className="text-[11px] truncate" style={{ color: "var(--fg-subtle)" }}>
                              {author} · {r.status}
                            </p>
                          </div>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="h-7 text-[11px] px-2 shrink-0"
                            style={{ color: "var(--accent-strong)" }}
                            disabled={restoringRevisionID !== null}
                            onClick={() => setShowRestoreConfirm(r)}
                          >
                            Restore
                          </Button>
                        </div>
                      );
                    })}
                  </div>
                )}
              </CardContent>
            </Card>
          )}

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

      {/* Restore Revision confirm */}
      <Dialog open={showRestoreConfirm !== null} onOpenChange={(open) => !open && setShowRestoreConfirm(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Restore this revision?</DialogTitle>
            <DialogDescription>
              {showRestoreConfirm && (
                <>
                  This will overwrite the live page with the snapshot from{" "}
                  <strong>{new Date(showRestoreConfirm.created_at).toLocaleString()}</strong>.
                  The current state will be saved as a new revision first, so this action is reversible.
                </>
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowRestoreConfirm(null)}
              disabled={restoringRevisionID !== null}
            >
              Cancel
            </Button>
            <Button
              onClick={() => showRestoreConfirm && handleRestoreRevision(showRestoreConfirm)}
              disabled={restoringRevisionID !== null}
              className="bg-primary hover:opacity-90 text-white"
            >
              {restoringRevisionID !== null ? "Restoring..." : "Restore"}
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
              className="bg-primary hover:opacity-90 text-white"
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
