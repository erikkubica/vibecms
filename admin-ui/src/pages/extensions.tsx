import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  Puzzle,
  Upload,
  Check,
  Power,
  PowerOff,
  Loader2,
  Package,
  Trash2,
  FolderOpen,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import {
  getExtensions,
  activateExtension,
  deactivateExtension,
  uploadExtension,
  deleteExtension,
  type Extension,
} from "@/api/client";

export default function ExtensionsPage() {
  const [extensions, setExtensions] = useState<Extension[]>([]);
  const [loading, setLoading] = useState(true);
  const [togglingSlug, setTogglingSlug] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Extension | null>(null);
  const [deleting, setDeleting] = useState(false);
  const navigate = useNavigate();

  // Install dialog
  const [installOpen, setInstallOpen] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchExtensions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getExtensions();
      setExtensions(data);
    } catch {
      toast.error("Failed to load extensions");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchExtensions();
  }, [fetchExtensions]);

  // Upload handlers
  async function handleUpload(file: File) {
    if (!file.name.endsWith(".zip")) {
      toast.error("Please upload a .zip file");
      return;
    }
    setUploading(true);
    try {
      await uploadExtension(file);
      toast.success("Extension uploaded successfully");
      setInstallOpen(false);
      fetchExtensions();
    } catch {
      toast.error("Failed to upload extension");
    } finally {
      setUploading(false);
    }
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) handleUpload(file);
    if (fileInputRef.current) fileInputRef.current.value = "";
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) handleUpload(file);
  }

  // Activate / Deactivate
  async function handleToggle(ext: Extension) {
    setTogglingSlug(ext.slug);
    try {
      if (ext.is_active) {
        await deactivateExtension(ext.slug);
        toast.success(`"${ext.name}" deactivated`);
      } else {
        await activateExtension(ext.slug);
        toast.success(`"${ext.name}" activated`);
      }
      fetchExtensions();
    } catch {
      toast.error("Failed to update extension status");
    } finally {
      setTogglingSlug(null);
    }
  }

  // Delete
  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteExtension(deleteTarget.slug);
      toast.success("Extension deleted");
      setDeleteTarget(null);
      fetchExtensions();
    } catch {
      toast.error("Failed to delete extension");
    } finally {
      setDeleting(false);
    }
  }

  const activeCount = extensions.filter((e) => e.is_active).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Extensions</h1>
          <p className="text-sm text-slate-500 mt-1">
            {extensions.length} installed, {activeCount} active
          </p>
        </div>
        <Button
          onClick={() => setInstallOpen(true)}
          className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
        >
          <Plus className="mr-2 h-4 w-4" />
          Install
        </Button>
      </div>

      {/* Extensions Grid */}
      {loading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
        </div>
      ) : extensions.length === 0 ? (
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardContent className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
            <Puzzle className="h-12 w-12" />
            <p className="text-lg font-medium">No extensions installed</p>
            <p className="text-sm">Click Install to add your first extension</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {extensions.map((ext) => (
            <Card
              key={ext.slug}
              className={`rounded-xl shadow-sm overflow-hidden transition-all ${
                ext.is_active
                  ? "border-2 border-emerald-500 ring-2 ring-emerald-500/20"
                  : "border border-slate-200 hover:border-slate-300"
              }`}
            >
              {/* Card header */}
              <div className="flex items-center justify-between px-4 pt-3 pb-2">
                <div className="flex items-center gap-2 min-w-0">
                  <Puzzle className="h-4 w-4 text-emerald-400 shrink-0" />
                  <span className="text-xs font-medium text-slate-500 truncate">
                    {ext.priority !== 50 ? `Extension \u00b7 Priority ${ext.priority}` : "Extension"}
                  </span>
                </div>
                {ext.is_active ? (
                  <Badge className="bg-emerald-500 text-white hover:bg-emerald-500 border-0 text-xs shadow-sm">
                    <Check className="mr-1 h-3 w-3" />
                    Active
                  </Badge>
                ) : (
                  <Badge className="bg-slate-400 text-white hover:bg-slate-400 border-0 text-xs">
                    Inactive
                  </Badge>
                )}
              </div>

              <div className="relative h-28 bg-gradient-to-br from-slate-100 to-slate-200 flex items-center justify-center mx-3 rounded-lg overflow-hidden">
                <img
                  src={`/admin/api/extensions/${ext.slug}/preview`}
                  alt={ext.name}
                  className="h-full w-full object-cover"
                />
              </div>

              <CardContent className="p-4 space-y-3">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <h3 className="font-semibold text-slate-900 truncate">{ext.name}</h3>
                    {ext.author && (
                      <p className="text-xs text-slate-500 mt-0.5">by {ext.author}</p>
                    )}
                  </div>
                  <Badge variant="outline" className="shrink-0 text-xs font-mono">
                    v{ext.version}
                  </Badge>
                </div>

                {ext.description && (
                  <p className="text-xs text-slate-500 line-clamp-2">{ext.description}</p>
                )}

                <div className="flex items-center gap-2">
                  <Badge className="bg-slate-100 text-slate-600 hover:bg-slate-100 border-0 text-xs font-mono">
                    <Package className="mr-1 h-3 w-3" />
                    {ext.slug}
                  </Badge>
                </div>

                <div className="flex items-center gap-2 pt-1 border-t border-slate-100">
                  <Button
                    size="sm"
                    variant={ext.is_active ? "outline" : "default"}
                    className={
                      ext.is_active
                        ? "text-xs"
                        : "text-xs bg-emerald-600 hover:bg-emerald-700 text-white"
                    }
                    disabled={togglingSlug === ext.slug}
                    onClick={() => handleToggle(ext)}
                  >
                    {togglingSlug === ext.slug ? (
                      <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                    ) : ext.is_active ? (
                      <PowerOff className="mr-1 h-3 w-3" />
                    ) : (
                      <Power className="mr-1 h-3 w-3" />
                    )}
                    {ext.is_active ? "Deactivate" : "Activate"}
                  </Button>

                  <Button
                    size="sm"
                    variant="outline"
                    className="text-xs"
                    onClick={() => navigate(`/admin/extensions/${ext.slug}/files`)}
                  >
                    <FolderOpen className="mr-1 h-3 w-3" />
                    Files
                  </Button>

                  <div className="flex-1" />

                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-xs text-red-500 hover:text-red-600 hover:bg-red-50"
                    disabled={ext.is_active}
                    onClick={() => setDeleteTarget(ext)}
                    title={
                      ext.is_active
                        ? "Deactivate extension before deleting"
                        : "Delete extension"
                    }
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Install dialog */}
      <Dialog open={installOpen} onOpenChange={setInstallOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Install Extension</DialogTitle>
            <DialogDescription>
              Upload a ZIP archive containing an extension with an extension.json manifest.
            </DialogDescription>
          </DialogHeader>
          <div
            className={`relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-8 transition-colors ${
              dragOver
                ? "border-indigo-400 bg-indigo-50"
                : "border-slate-300 bg-slate-50 hover:border-slate-400"
            }`}
            onDragOver={(e) => {
              e.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
          >
            <Puzzle className="mb-3 h-10 w-10 text-slate-400" />
            <p className="mb-1 text-sm font-medium text-slate-700">
              Drag and drop a ZIP file here
            </p>
            <p className="mb-4 text-xs text-slate-500">or click to browse</p>
            <input
              ref={fileInputRef}
              type="file"
              accept=".zip"
              className="hidden"
              onChange={handleFileChange}
            />
            <Button
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
              className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
            >
              {uploading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Uploading...
                </>
              ) : (
                <>
                  <Upload className="mr-2 h-4 w-4" />
                  Choose File
                </>
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation dialog */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Extension</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.name}&quot;?
              This will remove all extension files. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteTarget(null)}
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
