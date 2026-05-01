import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Save,
  Trash2,
  Loader2,
  Code,
  Eye,
  FileCode,
  Boxes,
  RefreshCw,
  Unplug,
  Info,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Titlebar } from "@/components/ui/titlebar";
import { MetaRow, MetaList } from "@/components/ui/meta-row";
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
  detachBlockType,
  previewBlockTemplate,
  type BlockType,
  type NodeTypeField,
} from "@/api/client";
import { toast } from "sonner";
import { usePageMeta } from "@/components/layout/page-meta";
import FieldSchemaEditor from "@/components/ui/field-schema-editor";
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
  const [detaching, setDetaching] = useState(false);
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
  const [source, setSource] = useState("custom");
  const [themeName, setThemeName] = useState<string | null>(null);
  const [createdAt, setCreatedAt] = useState<string | null>(null);
  const [updatedAt, setUpdatedAt] = useState<string | null>(null);

  const isManaged = source !== "custom";

  usePageMeta([
    "Block Types",
    isEdit ? (label ? `Edit "${label}"` : "Edit") : "New Block Type",
  ]);

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
          setSource(bt.source || "custom");
          setThemeName(bt.theme_name || null);
          setCreatedAt(bt.created_at || null);
          setUpdatedAt(bt.updated_at || null);
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

  const handleSave = async (e?: FormEvent) => {
    e?.preventDefault();
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

  const handleDetach = async () => {
    if (!id) return;
    setDetaching(true);
    try {
      const detached = await detachBlockType(id);
      setSource(detached.source);
      setThemeName(detached.theme_name || null);
      toast.success(`Block type detached from ${source} — now editable`);
    } catch (err: any) {
      toast.error(err.message || "Failed to detach block type");
    } finally {
      setDetaching(false);
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
        <Loader2 className="h-8 w-8 animate-spin" style={{color: "var(--accent-strong)"}} />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {isManaged && (
        <div className="rounded-lg border p-3 text-xs flex items-start gap-2" style={{background: "var(--warning-bg)", borderColor: "var(--border)", color: "var(--warning)"}}>
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            This block type is managed by the active {source} and is read-only. Click
            &quot;Detach&quot; in the sidebar to create an editable copy.
          </p>
        </div>
      )}

      <form onSubmit={handleSave} className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Main content */}
        <div className="space-y-4 min-w-0">
          <Titlebar
            title={label}
            onTitleChange={handleLabelChange}
            titleLabel="Label"
            titlePlaceholder="e.g. Hero Section"
            slug={slug}
            onSlugChange={isEdit || isManaged ? undefined : (v) => { setAutoSlug(false); setSlug(keyify(v)); }}
            slugPrefix=""
            autoSlug={autoSlug}
            onAutoSlugToggle={isEdit || isManaged ? undefined : () => setAutoSlug(!autoSlug)}
            id={isEdit && id ? Number(id) : undefined}
            onBack={() => navigate("/admin/block-types")}
            actions={isManaged ? (
              <Badge
                className="shrink-0"
                style={{
                  fontSize: 10.5,
                  background: "color-mix(in oklab, #f59e0b 14%, transparent)",
                  color: "#a16207",
                  border: "1px solid color-mix(in oklab, #f59e0b 30%, transparent)",
                }}
              >
                {source === "theme" ? (themeName || "Theme") : source.charAt(0).toUpperCase() + source.slice(1)}
              </Badge>
            ) : undefined}
          />

          {/* Tabs */}
          <Tabs defaultValue="fields" className="w-full">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="fields" className="">
                <Boxes className="mr-2 h-4 w-4" /> Fields
              </TabsTrigger>
              <TabsTrigger value="template" className="">
                <FileCode className="mr-2 h-4 w-4" /> Template
              </TabsTrigger>
              <TabsTrigger value="test-data" className="">
                <Code className="mr-2 h-4 w-4" /> Test Data
              </TabsTrigger>
              <TabsTrigger value="preview" className="" onClick={handlePreview}>
                <Eye className="mr-2 h-4 w-4" /> Preview
              </TabsTrigger>
            </TabsList>

            <TabsContent value="template" className="mt-4 ring-offset-white focus-visible:outline-none">
              <CodeWindow
                title="HTML / Go Template"
                value={htmlTemplate}
                onChange={setHtmlTemplate}
                height="500px"
                disabled={isManaged}
              />
            </TabsContent>

            <TabsContent value="fields" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-border shadow-sm">
                <SectionHeader title="Fields Definition" />
                <CardContent>
                  <p className="text-xs text-muted-foreground mb-4">Configure the data structure for this block.</p>
                  <FieldSchemaEditor
                    fields={fields}
                    onChange={setFields}
                    disabled={isManaged}
                  />
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="test-data" className="mt-4 ring-offset-white focus-visible:outline-none">
              <CodeWindow
                title="Mock Content (JSON)"
                value={JSON.stringify(testData, null, 2)}
                onChange={(v) => {
                  try { setTestData(JSON.parse(v)); } catch {}
                }}
                height="500px"
                disabled={isManaged}
              />
            </TabsContent>

            <TabsContent value="preview" className="mt-4 ring-offset-white focus-visible:outline-none">
              <Card className="rounded-xl border border-border shadow-sm h-[500px] flex flex-col">
                <SectionHeader
                  title="Rendered Preview"
                  actions={
                    <Button variant="ghost" type="button" size="sm" className="h-7 text-xs" onClick={handlePreview} disabled={previewLoading}>
                      {previewLoading ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : <RefreshCw className="mr-1 h-3 w-3" />}
                      Refresh
                    </Button>
                  }
                />
                <div className="flex-1 overflow-hidden bg-card">
                  {previewHtml ? (
                    <iframe
                      title="Block preview"
                      className="h-full w-full border-0 bg-card"
                      sandbox="allow-same-origin allow-scripts"
                      srcDoc={`<!doctype html><html><head><meta charset="utf-8">${previewHead || '<script src="https://cdn.tailwindcss.com"></script>'}<style>body{margin:0;padding:1rem;}</style></head><body class="${previewBodyClass}">${previewHtml}</body></html>`}
                    />
                  ) : (
                    <div className="h-full flex flex-col items-center justify-center space-y-3" style={{color: "var(--fg-subtle)"}}>
                      <Eye className="h-10 w-10 opacity-20" />
                      <p className="text-sm">Click refresh to render template with test data</p>
                    </div>
                  )}
                </div>
              </Card>
            </TabsContent>
          </Tabs>
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {/* Publish card */}
          <Card className="rounded-xl border border-border shadow-sm">
            <SectionHeader title="Publish" />
            <CardContent className="space-y-4">
              {isManaged ? (
                <Button
                  type="button"
                  className="w-full font-medium rounded-lg shadow-sm h-9 text-sm" style={{background: "var(--warning)", color: "#fff"}}
                  onClick={handleDetach}
                  disabled={detaching}
                >
                  <Unplug className="mr-1.5 h-3.5 w-3.5" />
                  {detaching ? "Detaching..." : "Detach"}
                </Button>
              ) : (
                <Button
                  type="submit"
                  className="w-full"
                  disabled={saving}
                >
                  <Save className="mr-1.5 h-3.5 w-3.5" />
                  {saving ? "Saving..." : "Save"}
                </Button>
              )}

              {isEdit && !isManaged && (
                <>
                  <Separator />
                  <Button
                    type="button"
                    variant="ghost"
                    className="w-full"
                    style={{ color: "var(--danger)" }}
                    onClick={() => setShowDeleteDialog(true)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                </>
              )}

              {isEdit && (
                <>
                  <div style={{ height: 1, background: "var(--divider)", margin: "4px 0" }} />
                  <MetaList>
                    <MetaRow label="Source" value={<span className="capitalize">{source}</span>} />
                    {createdAt && <MetaRow label="Created" value={new Date(createdAt).toLocaleDateString("en-GB")} />}
                    {updatedAt && <MetaRow label="Updated" value={new Date(updatedAt).toLocaleDateString("en-GB")} />}
                  </MetaList>
                </>
              )}
            </CardContent>
          </Card>

          {/* Settings card */}
          <Card className="rounded-xl border border-border shadow-sm">
            <SectionHeader title="Settings" />
            <CardContent className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="description" className="text-xs font-medium text-muted-foreground">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Brief summary of what this block does"
                  rows={2}
                  disabled={isManaged}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="icon" className="text-xs font-medium text-muted-foreground">Icon Slug</Label>
                <Input
                  id="icon"
                  value={icon}
                  onChange={(e) => setIcon(e.target.value)}
                  placeholder="boxes, image, text..."
                  className="font-mono text-sm"
                  disabled={isManaged}
                />
              </div>
              <label htmlFor="cache" className={`flex items-center justify-between gap-3 pt-1 ${isManaged ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                <div className="space-y-0.5 min-w-0">
                  <span className="block text-xs font-medium text-foreground">Cache Output</span>
                  <span className="block text-[11px]" style={{color: "var(--fg-subtle)"}}>Cache rendered HTML</span>
                </div>
                <Switch
                  id="cache"
                  checked={cacheOutput}
                  onCheckedChange={setCacheOutput}
                  disabled={isManaged}
                />
              </label>
            </CardContent>
          </Card>
        </div>
      </form>

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
