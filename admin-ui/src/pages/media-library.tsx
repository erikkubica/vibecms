import { useEffect, useState, useCallback, useRef } from "react";
import {
  Upload,
  Search,
  Loader2,
  Trash2,
  Image as ImageIcon,
  FileText,
  Film,
  Music,
  File,
  Copy,
  X,
  Check,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
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
import { Label } from "@/components/ui/label";
import { toast } from "sonner";

// ---------- Types ----------

interface MediaFile {
  id: number;
  filename: string;
  original_name: string;
  mime_type: string;
  size: number;
  path: string;
  url: string;
  width: number | null;
  height: number | null;
  alt: string;
  created_at: string;
  updated_at: string;
}

interface PaginationMeta {
  total: number;
  page: number;
  per_page: number;
}

// ---------- API helpers ----------

async function fetchMedia(params: {
  page: number;
  per_page: number;
  mime_type?: string;
  search?: string;
}): Promise<{ data: MediaFile[]; total: number; page: number; per_page: number }> {
  const qs = new URLSearchParams();
  qs.set("page", String(params.page));
  qs.set("per_page", String(params.per_page));
  if (params.mime_type) qs.set("mime_type", params.mime_type);
  if (params.search) qs.set("search", params.search);

  const res = await fetch(`/admin/api/media?${qs.toString()}`, {
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to fetch media");
  return res.json();
}

async function uploadMediaFile(
  file: globalThis.File,
  onProgress?: (pct: number) => void
): Promise<MediaFile> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/admin/api/media/upload");
    xhr.withCredentials = true;

    xhr.upload.addEventListener("progress", (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100));
      }
    });

    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const body = JSON.parse(xhr.responseText);
          resolve(body.data);
        } catch {
          reject(new Error("Invalid response"));
        }
      } else {
        reject(new Error("Upload failed"));
      }
    });

    xhr.addEventListener("error", () => reject(new Error("Upload failed")));

    const fd = new FormData();
    fd.append("file", file);
    xhr.send(fd);
  });
}

async function updateMediaAlt(id: number, alt: string): Promise<MediaFile> {
  const res = await fetch(`/admin/api/media/${id}`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ alt }),
  });
  if (!res.ok) throw new Error("Failed to update");
  const body = await res.json();
  return body.data;
}

async function deleteMedia(id: number): Promise<void> {
  const res = await fetch(`/admin/api/media/${id}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to delete");
}

// ---------- Helpers ----------

function humanFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function isImage(mime: string): boolean {
  return mime.startsWith("image/");
}

function isVideo(mime: string): boolean {
  return mime.startsWith("video/");
}

function isAudio(mime: string): boolean {
  return mime.startsWith("audio/");
}

function getFileExtension(name: string): string {
  const parts = name.split(".");
  return parts.length > 1 ? parts[parts.length - 1].toUpperCase() : "";
}

function FileIcon({ mime, className }: { mime: string; className?: string }) {
  if (isImage(mime)) return <ImageIcon className={className} />;
  if (isVideo(mime)) return <Film className={className} />;
  if (isAudio(mime)) return <Music className={className} />;
  if (mime.includes("pdf") || mime.includes("document") || mime.includes("text"))
    return <FileText className={className} />;
  return <File className={className} />;
}

const MIME_FILTERS: { value: string; label: string }[] = [
  { value: "all", label: "All files" },
  { value: "image", label: "Images" },
  { value: "application", label: "Documents" },
  { value: "video", label: "Videos" },
  { value: "audio", label: "Audio" },
];

// ---------- Component ----------

export default function MediaLibraryPage() {
  const [files, setFiles] = useState<MediaFile[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchDebounce, setSearchDebounce] = useState("");
  const [mimeFilter, setMimeFilter] = useState("all");
  const [selected, setSelected] = useState<MediaFile | null>(null);

  // Upload
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Detail panel
  const [editAlt, setEditAlt] = useState("");
  const [savingAlt, setSavingAlt] = useState(false);
  const [copied, setCopied] = useState(false);

  // Delete dialog
  const [deleteTarget, setDeleteTarget] = useState<MediaFile | null>(null);
  const [deleting, setDeleting] = useState(false);

  const perPage = 24;

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setSearchDebounce(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const fetchFiles = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetchMedia({
        page,
        per_page: perPage,
        mime_type: mimeFilter === "all" ? undefined : mimeFilter,
        search: searchDebounce || undefined,
      });
      setFiles(res.data || []);
      setMeta({ total: res.total, page: res.page, per_page: res.per_page });
    } catch {
      toast.error("Failed to load media files");
    } finally {
      setLoading(false);
    }
  }, [page, mimeFilter, searchDebounce]);

  useEffect(() => {
    setPage(1);
  }, [searchDebounce, mimeFilter]);

  useEffect(() => {
    fetchFiles();
  }, [fetchFiles]);

  // When selected changes, update alt field
  useEffect(() => {
    if (selected) {
      setEditAlt(selected.alt || "");
      setCopied(false);
    }
  }, [selected]);

  // Upload handler
  async function handleUpload(file: globalThis.File) {
    setUploading(true);
    setUploadProgress(0);
    try {
      await uploadMediaFile(file, setUploadProgress);
      toast.success("File uploaded successfully");
      fetchFiles();
    } catch {
      toast.error("Failed to upload file");
    } finally {
      setUploading(false);
      setUploadProgress(0);
    }
  }

  function handleFileInput(e: React.ChangeEvent<HTMLInputElement>) {
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

  // Save alt text
  async function handleSaveAlt() {
    if (!selected) return;
    setSavingAlt(true);
    try {
      const updated = await updateMediaAlt(selected.id, editAlt);
      setSelected(updated);
      setFiles((prev) => prev.map((f) => (f.id === updated.id ? updated : f)));
      toast.success("Alt text updated");
    } catch {
      toast.error("Failed to update alt text");
    } finally {
      setSavingAlt(false);
    }
  }

  // Copy URL
  function handleCopyUrl() {
    if (!selected) return;
    const fullUrl = window.location.origin + selected.url;
    navigator.clipboard.writeText(fullUrl).then(() => {
      setCopied(true);
      toast.success("URL copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    });
  }

  // Delete
  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteMedia(deleteTarget.id);
      toast.success("File deleted");
      if (selected?.id === deleteTarget.id) setSelected(null);
      setDeleteTarget(null);
      fetchFiles();
    } catch {
      toast.error("Failed to delete file");
    } finally {
      setDeleting(false);
    }
  }

  const totalPages = meta ? Math.ceil(meta.total / meta.per_page) : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900">Media Library</h1>
        <div className="flex items-center gap-2">
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            onChange={handleFileInput}
          />
          <Button
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
          >
            {uploading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Uploading {uploadProgress}%
              </>
            ) : (
              <>
                <Upload className="mr-2 h-4 w-4" />
                Upload
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm sm:flex-row">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <Input
            placeholder="Search media files..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
          />
        </div>
        <Select value={mimeFilter} onValueChange={setMimeFilter}>
          <SelectTrigger className="w-full rounded-lg border-slate-300 sm:w-44">
            <SelectValue placeholder="File type" />
          </SelectTrigger>
          <SelectContent>
            {MIME_FILTERS.map((f) => (
              <SelectItem key={f.value} value={f.value}>
                {f.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Upload progress bar */}
      {uploading && (
        <div className="rounded-lg border border-indigo-200 bg-indigo-50 p-3">
          <div className="flex items-center gap-3">
            <Loader2 className="h-4 w-4 animate-spin text-indigo-600" />
            <div className="flex-1">
              <div className="h-2 rounded-full bg-indigo-100">
                <div
                  className="h-2 rounded-full bg-indigo-600 transition-all duration-300"
                  style={{ width: `${uploadProgress}%` }}
                />
              </div>
            </div>
            <span className="text-sm font-medium text-indigo-700">{uploadProgress}%</span>
          </div>
        </div>
      )}

      {/* Main area: grid + detail panel */}
      <div className="flex gap-6">
        {/* Grid */}
        <div className="flex-1 min-w-0">
          <Card
            className={`rounded-xl border shadow-sm overflow-hidden py-0 gap-0 transition-colors ${
              dragOver
                ? "border-indigo-400 bg-indigo-50/50"
                : "border-slate-200"
            }`}
            onDragOver={(e) => {
              e.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
          >
            <CardContent className="p-4">
              {loading ? (
                <div className="flex h-64 items-center justify-center">
                  <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
                </div>
              ) : files.length === 0 ? (
                <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
                  <ImageIcon className="h-12 w-12" />
                  <p className="text-lg font-medium">No media files found</p>
                  <p className="text-sm">
                    {searchDebounce || mimeFilter !== "all"
                      ? "Try adjusting your filters"
                      : "Upload your first file to get started"}
                  </p>
                  {!searchDebounce && mimeFilter === "all" && (
                    <Button
                      onClick={() => fileInputRef.current?.click()}
                      className="mt-2 bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
                    >
                      <Upload className="mr-2 h-4 w-4" />
                      Upload File
                    </Button>
                  )}
                </div>
              ) : (
                <div className="grid grid-cols-2 gap-3 md:grid-cols-4 lg:grid-cols-6">
                  {files.map((file) => (
                    <button
                      key={file.id}
                      onClick={() =>
                        setSelected(selected?.id === file.id ? null : file)
                      }
                      className={`group relative flex flex-col overflow-hidden rounded-lg border-2 bg-white text-left transition-all hover:shadow-md ${
                        selected?.id === file.id
                          ? "border-indigo-500 ring-2 ring-indigo-500/20"
                          : "border-slate-200 hover:border-slate-300"
                      }`}
                    >
                      {/* Thumbnail */}
                      <div className="relative aspect-square bg-slate-100 flex items-center justify-center overflow-hidden">
                        {isImage(file.mime_type) ? (
                          <img
                            src={file.url}
                            alt={file.alt || file.original_name}
                            className="h-full w-full object-cover"
                            loading="lazy"
                          />
                        ) : (
                          <div className="flex flex-col items-center gap-1.5">
                            <FileIcon
                              mime={file.mime_type}
                              className="h-8 w-8 text-slate-400"
                            />
                            <span className="rounded bg-slate-200 px-1.5 py-0.5 text-[10px] font-semibold text-slate-500">
                              {getFileExtension(file.original_name)}
                            </span>
                          </div>
                        )}
                        {selected?.id === file.id && (
                          <div className="absolute top-1.5 right-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-indigo-500">
                            <Check className="h-3 w-3 text-white" />
                          </div>
                        )}
                      </div>

                      {/* Info */}
                      <div className="p-2">
                        <p className="truncate text-xs font-medium text-slate-700">
                          {file.original_name}
                        </p>
                        <div className="mt-0.5 flex items-center justify-between">
                          <span className="text-[10px] text-slate-400">
                            {humanFileSize(file.size)}
                          </span>
                          <span className="text-[10px] text-slate-400">
                            {new Date(file.created_at).toLocaleDateString()}
                          </span>
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              )}

              {/* Drag overlay */}
              {dragOver && (
                <div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-xl bg-indigo-50/80 border-2 border-dashed border-indigo-400">
                  <div className="text-center">
                    <Upload className="mx-auto h-10 w-10 text-indigo-500" />
                    <p className="mt-2 text-sm font-medium text-indigo-700">
                      Drop file to upload
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Pagination */}
          {meta && totalPages > 1 && (
            <div className="mt-4 flex items-center justify-between">
              <p className="text-sm text-slate-500">
                Showing {(meta.page - 1) * meta.per_page + 1} to{" "}
                {Math.min(meta.page * meta.per_page, meta.total)} of {meta.total}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                  className="rounded-lg border-slate-300"
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                  className="rounded-lg border-slate-300"
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* Detail panel */}
        {selected && (
          <div className="hidden w-80 shrink-0 lg:block">
            <Card className="rounded-xl border border-slate-200 shadow-sm sticky top-0">
              <CardContent className="p-4 space-y-4">
                {/* Close button */}
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-slate-900">
                    File Details
                  </h3>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => setSelected(null)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>

                {/* Preview */}
                <div className="overflow-hidden rounded-lg border border-slate-200 bg-slate-50">
                  {isImage(selected.mime_type) ? (
                    <img
                      src={selected.url}
                      alt={selected.alt || selected.original_name}
                      className="w-full object-contain max-h-48"
                    />
                  ) : (
                    <div className="flex h-32 flex-col items-center justify-center gap-2">
                      <FileIcon
                        mime={selected.mime_type}
                        className="h-10 w-10 text-slate-400"
                      />
                      <span className="rounded bg-slate-200 px-2 py-0.5 text-xs font-semibold text-slate-500">
                        {getFileExtension(selected.original_name)}
                      </span>
                    </div>
                  )}
                </div>

                {/* File info */}
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="text-slate-500">Name</span>
                    <p className="font-medium text-slate-800 break-all">
                      {selected.original_name}
                    </p>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <span className="text-slate-500">Type</span>
                      <p className="font-medium text-slate-800">
                        {selected.mime_type.split("/")[1]?.toUpperCase() || selected.mime_type}
                      </p>
                    </div>
                    <div>
                      <span className="text-slate-500">Size</span>
                      <p className="font-medium text-slate-800">
                        {humanFileSize(selected.size)}
                      </p>
                    </div>
                  </div>
                  {selected.width && selected.height && (
                    <div>
                      <span className="text-slate-500">Dimensions</span>
                      <p className="font-medium text-slate-800">
                        {selected.width} x {selected.height}
                      </p>
                    </div>
                  )}
                  <div>
                    <span className="text-slate-500">Uploaded</span>
                    <p className="font-medium text-slate-800">
                      {new Date(selected.created_at).toLocaleString()}
                    </p>
                  </div>
                </div>

                {/* Alt text */}
                <div className="space-y-2">
                  <Label htmlFor="alt-text" className="text-sm text-slate-500">
                    Alt Text
                  </Label>
                  <div className="flex gap-2">
                    <Input
                      id="alt-text"
                      placeholder="Describe this file..."
                      value={editAlt}
                      onChange={(e) => setEditAlt(e.target.value)}
                      className="rounded-lg border-slate-300 text-sm"
                    />
                    <Button
                      size="sm"
                      onClick={handleSaveAlt}
                      disabled={savingAlt || editAlt === (selected.alt || "")}
                      className="bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg shrink-0"
                    >
                      {savingAlt ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        "Save"
                      )}
                    </Button>
                  </div>
                </div>

                {/* Actions */}
                <div className="flex gap-2 pt-2 border-t border-slate-100">
                  <Button
                    variant="outline"
                    size="sm"
                    className="flex-1 rounded-lg text-xs"
                    onClick={handleCopyUrl}
                  >
                    {copied ? (
                      <Check className="mr-1.5 h-3.5 w-3.5 text-emerald-500" />
                    ) : (
                      <Copy className="mr-1.5 h-3.5 w-3.5" />
                    )}
                    {copied ? "Copied" : "Copy URL"}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="rounded-lg text-xs text-red-500 hover:text-red-600 hover:bg-red-50"
                    onClick={() => setDeleteTarget(selected)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </div>

      {/* Delete dialog */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete File</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.original_name}
              &quot;? This action cannot be undone.
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
