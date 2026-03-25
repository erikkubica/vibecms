import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowLeft,
  Save,
  Globe,
  Trash2,
  Home,
  Loader2,
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
import { toast } from "sonner";
import {
  getNode,
  createNode,
  updateNode,
  deleteNode,
  setHomepage,
  type ContentNode,
} from "@/api/client";

interface NodeEditorProps {
  nodeType: "page" | "post";
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
  const label = nodeType === "page" ? "Page" : "Post";
  const basePath = nodeType === "page" ? "/admin/pages" : "/admin/posts";

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  // Form state
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [status, setStatus] = useState("draft");
  const [languageCode, setLanguageCode] = useState("en");
  const [parentId, setParentId] = useState("");
  const [blocksJson, setBlocksJson] = useState("[]");
  const [originalNode, setOriginalNode] = useState<ContentNode | null>(null);

  useEffect(() => {
    if (!isEdit) return;
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
        setBlocksJson(JSON.stringify(node.blocks_data ?? [], null, 2));
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

  async function handleSave(e: FormEvent, publishStatus?: string) {
    e.preventDefault();

    // Validate blocks JSON
    let parsedBlocks: Record<string, unknown>[];
    try {
      parsedBlocks = JSON.parse(blocksJson);
      if (!Array.isArray(parsedBlocks)) {
        toast.error("Blocks data must be a JSON array");
        return;
      }
    } catch {
      toast.error("Invalid JSON in blocks data");
      return;
    }

    const nodeData: Partial<ContentNode> = {
      title,
      slug,
      node_type: nodeType,
      status: publishStatus || status,
      language_code: languageCode,
      parent_id: parentId ? Number(parentId) : null,
      blocks_data: parsedBlocks,
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
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link to={basePath}>
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-800">
          {isEdit ? `Edit ${label}` : `New ${label}`}
        </h1>
      </div>

      <form onSubmit={(e) => handleSave(e)} className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="space-y-6 lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle>Content</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  placeholder={`Enter ${label.toLowerCase()} title`}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  required
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="slug">Slug</Label>
                  <button
                    type="button"
                    className="text-xs text-primary-600 hover:underline"
                    onClick={() => setAutoSlug(!autoSlug)}
                  >
                    {autoSlug ? "Edit manually" : "Auto-generate"}
                  </button>
                </div>
                <Input
                  id="slug"
                  placeholder="url-slug"
                  value={slug}
                  onChange={(e) => {
                    setAutoSlug(false);
                    setSlug(e.target.value);
                  }}
                  disabled={autoSlug}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="blocks">Blocks Data (JSON)</Label>
                <Textarea
                  id="blocks"
                  placeholder="[]"
                  value={blocksJson}
                  onChange={(e) => setBlocksJson(e.target.value)}
                  rows={16}
                  className="font-mono text-sm"
                />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Status</Label>
                <Select value={status} onValueChange={setStatus}>
                  <SelectTrigger>
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
                <Label>Language</Label>
                <Select value={languageCode} onValueChange={setLanguageCode}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="en">English</SelectItem>
                    <SelectItem value="es">Spanish</SelectItem>
                    <SelectItem value="fr">French</SelectItem>
                    <SelectItem value="de">German</SelectItem>
                    <SelectItem value="pt">Portuguese</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="parent-id">Parent ID</Label>
                <Input
                  id="parent-id"
                  type="number"
                  placeholder="None"
                  value={parentId}
                  onChange={(e) => setParentId(e.target.value)}
                />
              </div>

              <Separator />

              <div className="flex flex-col gap-2">
                <Button
                  type="submit"
                  className="w-full bg-primary-600 hover:bg-primary-700"
                  disabled={saving}
                >
                  <Save className="mr-2 h-4 w-4" />
                  {saving ? "Saving..." : "Save"}
                </Button>
                {status !== "published" && (
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full"
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
            <Card>
              <CardHeader>
                <CardTitle>Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                {nodeType === "page" && (
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full"
                    onClick={handleSetHomepage}
                  >
                    <Home className="mr-2 h-4 w-4" />
                    Set as Homepage
                  </Button>
                )}
                <Button
                  type="button"
                  variant="outline"
                  className="w-full text-red-600 hover:bg-red-50 hover:text-red-700"
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
            <Card>
              <CardContent className="space-y-2 pt-6 text-sm text-slate-500">
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
