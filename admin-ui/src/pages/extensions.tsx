import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  Puzzle,
  Upload,
  Check,
  Power,
  PowerOff,
  Loader2,
  Trash2,
  FolderOpen,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
              className={`group rounded-xl overflow-hidden transition-all duration-200 ${
                ext.is_active
                  ? "border-2 border-emerald-500/70 shadow-md shadow-emerald-500/5"
                  : "border border-slate-200/80 shadow-sm hover:shadow-md hover:border-slate-300"
              }`}
            >
              {/* Preview area — fixed aspect with graceful fallback */}
              <div className="relative bg-slate-100 overflow-hidden">
                <img
                  src={`/admin/api/extensions/${ext.slug}/preview`}
                  alt={ext.name}
                  className="w-full h-auto block"
                  onError={(e) => { e.currentTarget.style.display = "none"; }}
                />
                {/* Status overlay */}
                <div className="absolute top-2.5 right-2.5">
                  {ext.is_active ? (
                    <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500 px-2.5 py-1 text-[11px] font-semibold text-white shadow-sm backdrop-blur-sm">
                      <Check className="h-3 w-3" />
                      Active
                    </span>
                  ) : (
                    <span className="inline-flex items-center rounded-full bg-slate-900/50 px-2.5 py-1 text-[11px] font-medium text-white/80 backdrop-blur-sm">
                      Inactive
                    </span>
                  )}
                </div>
              </div>

              {/* Content */}
              <div className="p-4 space-y-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h3 className="font-semibold text-[15px] text-slate-900 truncate leading-tight">{ext.name}</h3>
                    <p className="text-xs text-slate-400 mt-1">
                      {ext.author ? `by ${ext.author}` : ext.slug}
                      {ext.priority !== 50 && ` · Priority ${ext.priority}`}
                    </p>
                  </div>
                  <span className="shrink-0 rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-mono font-medium text-slate-500 tracking-wide">
                    {ext.version}
                  </span>
                </div>

                {ext.description && (
                  <p className="text-[13px] text-slate-500 leading-relaxed line-clamp-2">{ext.description}</p>
                )}

                {/* Actions */}
                <div className="flex items-center gap-1.5 pt-2">
                  <Button
                    size="sm"
                    className={`text-xs h-8 rounded-lg flex-1 ${
                      ext.is_active
                        ? "bg-white border border-slate-200 text-slate-600 hover:bg-slate-50 hover:border-slate-300 shadow-none"
                        : "bg-emerald-600 hover:bg-emerald-700 text-white shadow-sm"
                    }`}
                    disabled={togglingSlug === ext.slug}
                    onClick={() => handleToggle(ext)}
                  >
                    {togglingSlug === ext.slug ? (
                      <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                    ) : ext.is_active ? (
                      <PowerOff className="mr-1.5 h-3 w-3" />
                    ) : (
                      <Power className="mr-1.5 h-3 w-3" />
                    )}
                    {ext.is_active ? "Deactivate" : "Activate"}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    className="text-xs h-8 rounded-lg border-slate-200"
                    onClick={() => navigate(`/admin/extensions/${ext.slug}/files`)}
                  >
                    <FolderOpen className="mr-1.5 h-3 w-3" />
                    Files
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-8 w-8 p-0 text-slate-400 hover:text-red-500 hover:bg-red-50 rounded-lg"
                    disabled={ext.is_active}
                    onClick={() => setDeleteTarget(ext)}
                    title={ext.is_active ? "Deactivate extension before deleting" : "Delete extension"}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
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
