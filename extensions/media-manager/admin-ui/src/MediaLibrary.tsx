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
  LayoutGrid,
  List,
  Download,
  Pencil,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  ArrowUpDown,
  AlertCircle,
} from "@vibecms/icons";
import {
  Button,
  Input,
  Card,
  CardContent,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Label,
  Badge,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@vibecms/ui";
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
  total_pages: number;
}

// ---------- API helpers ----------

async function fetchMedia(params: {
  page: number;
  per_page: number;
  mime_type?: string;
  search?: string;
  sort_by?: string;
}): Promise<{ data: MediaFile[]; meta: PaginationMeta }> {
  const qs = new URLSearchParams();
  qs.set("page", String(params.page));
  qs.set("per_page", String(params.per_page));
  if (params.mime_type) qs.set("mime_type", params.mime_type);
  if (params.search) qs.set("search", params.search);
  if (params.sort_by) qs.set("sort_by", params.sort_by);

  const res = await fetch(`/admin/api/ext/media-manager/?${qs.toString()}`, {
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to fetch media");
  const body = await res.json();
  return {
    data: body.data,
    meta: {
      total: body.meta.total,
      page: body.meta.page,
      per_page: body.meta.per_page,
      total_pages: body.meta.total_pages,
    },
  };
}

async function uploadMediaFile(
  file: globalThis.File,
  onProgress?: (pct: number) => void
): Promise<MediaFile> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/admin/api/ext/media-manager/upload");
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

async function updateMedia(
  id: number,
  data: { alt?: string; original_name?: string }
): Promise<MediaFile> {
  const res = await fetch(`/admin/api/ext/media-manager/${id}`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to update");
  const body = await res.json();
  return body.data;
}

async function deleteMedia(id: number): Promise<void> {
  const res = await fetch(`/admin/api/ext/media-manager/${id}`, {
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

function BrokenMediaFallback({ className }: { className?: string }) {
  return (
    <div className={`flex flex-col items-center justify-center gap-2 w-full h-full min-h-32 aspect-square bg-slate-50 text-slate-400 ${className || ""}`}>
      <AlertCircle className="h-8 w-8 text-slate-300" />
      <span className="text-xs font-medium text-slate-400">Failed to load</span>
    </div>
  );
}

function MediaImage({ src, alt, className, style, onError }: { src: string; alt: string; className?: string; style?: React.CSSProperties; onError?: () => void }) {
  const [broken, setBroken] = useState(false);
  if (broken) return <BrokenMediaFallback />;
  return (
    <img
      src={src}
      alt={alt}
      className={className}
      style={style}
      loading="lazy"
      onError={() => { setBroken(true); onError?.(); }}
    />
  );
}

function MediaVideo({ src, className, controls }: { src: string; className?: string; controls?: boolean }) {
  const [broken, setBroken] = useState(false);
  if (broken) return <BrokenMediaFallback />;
  return (
    <video
      src={src}
      className={className}
      controls={controls}
      onError={() => setBroken(true)}
    />
  );
}

function mimeLabel(mime: string): string {
  const sub = mime.split("/")[1];
  if (!sub) return mime;
  return sub.replace(/^x-/, "").toUpperCase();
}

const MIME_FILTERS = [
  { value: "all", label: "All files" },
  { value: "image", label: "Images" },
  { value: "application", label: "Documents" },
  { value: "video", label: "Videos" },
  { value: "audio", label: "Audio" },
];

const SORT_OPTIONS = [
  { value: "date_desc", label: "Newest first" },
  { value: "date_asc", label: "Oldest first" },
  { value: "name_asc", label: "Name A–Z" },
  { value: "name_desc", label: "Name Z–A" },
  { value: "size_desc", label: "Largest first" },
  { value: "size_asc", label: "Smallest first" },
];

const PER_PAGE_OPTIONS = [24, 48, 96];

// ---------- Component ----------

export default function MediaLibrary() {
  const [files, setFiles] = useState<MediaFile[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(24);
  const [search, setSearch] = useState("");
  const [searchDebounce, setSearchDebounce] = useState("");
  const [mimeFilter, setMimeFilter] = useState("all");
  const [sortBy, setSortBy] = useState("date_desc");
  const [viewMode, setViewMode] = useState<"grid" | "table">("grid");
  const [selected, setSelected] = useState<MediaFile | null>(null);
  const [mobileDetailOpen, setMobileDetailOpen] = useState(false);

  function selectFile(file: MediaFile | null) {
    setSelected(file);
    if (file && typeof window !== "undefined" && window.innerWidth < 1024) {
      setMobileDetailOpen(true);
    }
  }

  // Bulk selection
  const [bulkSelected, setBulkSelected] = useState<Set<number>>(new Set());

  // Upload
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadQueue, setUploadQueue] = useState(0);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Detail panel
  const [editAlt, setEditAlt] = useState("");
  const [editName, setEditName] = useState("");
  const [savingDetail, setSavingDetail] = useState(false);
  const [copied, setCopied] = useState(false);

  // Delete dialog
  const [deleteTarget, setDeleteTarget] = useState<MediaFile | null>(null);
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

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
        sort_by: sortBy,
      });
      setFiles(res.data || []);
      setMeta(res.meta);
    } catch {
      toast.error("Failed to load media files");
    } finally {
      setLoading(false);
    }
  }, [page, perPage, mimeFilter, searchDebounce, sortBy]);

  useEffect(() => {
    setPage(1);
    setBulkSelected(new Set());
  }, [searchDebounce, mimeFilter, sortBy, perPage]);

  useEffect(() => {
    fetchFiles();
  }, [fetchFiles]);

  // When selected changes, populate edit fields
  useEffect(() => {
    if (selected) {
      setEditAlt(selected.alt || "");
      setEditName(selected.original_name || "");
      setCopied(false);
    }
  }, [selected]);

  // Upload handler — supports multiple files
  async function handleUploadFiles(fileList: FileList | globalThis.File[]) {
    const arr = Array.from(fileList);
    if (arr.length === 0) return;
    setUploading(true);
    setUploadQueue(arr.length);
    let completed = 0;

    for (const file of arr) {
      setUploadProgress(0);
      try {
        await uploadMediaFile(file, setUploadProgress);
        completed++;
      } catch {
        toast.error(`Failed to upload ${file.name}`);
      }
    }

    setUploading(false);
    setUploadProgress(0);
    setUploadQueue(0);
    if (completed > 0) {
      toast.success(
        completed === 1
          ? "File uploaded successfully"
          : `${completed} files uploaded successfully`
      );
      await fetchFiles();
    }
  }

  function handleFileInput(e: React.ChangeEvent<HTMLInputElement>) {
    const fileList = e.target.files;
    if (fileList && fileList.length > 0) handleUploadFiles(fileList);
    if (fileInputRef.current) fileInputRef.current.value = "";
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    const fileList = e.dataTransfer.files;
    if (fileList && fileList.length > 0) handleUploadFiles(fileList);
  }

  // Save detail edits (alt + rename)
  async function handleSaveDetail() {
    if (!selected) return;
    setSavingDetail(true);
    try {
      const updates: { alt?: string; original_name?: string } = {};
      if (editAlt !== (selected.alt || "")) updates.alt = editAlt;
      if (editName !== selected.original_name && editName.trim() !== "")
        updates.original_name = editName;
      if (Object.keys(updates).length === 0) {
        setSavingDetail(false);
        return;
      }
      const updated = await updateMedia(selected.id, updates);
      setSelected(updated);
      setFiles((prev) => prev.map((f) => (f.id === updated.id ? updated : f)));
      toast.success("File updated");
    } catch {
      toast.error("Failed to update file");
    } finally {
      setSavingDetail(false);
    }
  }

  // Copy URL — fallback for HTTP (non-secure contexts)
  function copyToClipboard(text: string) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text);
    }
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand("copy");
    document.body.removeChild(textarea);
    return Promise.resolve();
  }

  function handleCopyUrl() {
    if (!selected) return;
    const fullUrl = window.location.origin + selected.url;
    copyToClipboard(fullUrl).then(() => {
      setCopied(true);
      toast.success("URL copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    });
  }

  // Single delete
  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteMedia(deleteTarget.id);
      toast.success("File deleted");
      if (selected?.id === deleteTarget.id) setSelected(null);
      setBulkSelected((prev) => {
        const next = new Set(prev);
        next.delete(deleteTarget.id);
        return next;
      });
      setDeleteTarget(null);
      await fetchFiles();
    } catch {
      toast.error("Failed to delete file");
    } finally {
      setDeleting(false);
    }
  }

  // Bulk delete
  async function handleBulkDelete() {
    setDeleting(true);
    let deleted = 0;
    for (const id of bulkSelected) {
      try {
        await deleteMedia(id);
        deleted++;
      } catch {
        // continue
      }
    }
    setDeleting(false);
    setBulkDeleteOpen(false);
    if (selected && bulkSelected.has(selected.id)) setSelected(null);
    setBulkSelected(new Set());
    toast.success(`${deleted} file${deleted !== 1 ? "s" : ""} deleted`);
    await fetchFiles();
  }

  // Bulk selection helpers
  function toggleBulkSelect(id: number) {
    setBulkSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleSelectAll() {
    if (bulkSelected.size === files.length) {
      setBulkSelected(new Set());
    } else {
      setBulkSelected(new Set(files.map((f) => f.id)));
    }
  }

  const totalPages = meta ? Math.ceil(meta.total / meta.per_page) : 0;
  const hasDetailChanges =
    selected &&
    (editAlt !== (selected.alt || "") ||
      (editName !== selected.original_name && editName.trim() !== ""));

  // Pagination page numbers
  function getPageNumbers(): (number | "...")[] {
    if (totalPages <= 7) return Array.from({ length: totalPages }, (_, i) => i + 1);
    const pages: (number | "...")[] = [1];
    if (page > 3) pages.push("...");
    for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) {
      pages.push(i);
    }
    if (page < totalPages - 2) pages.push("...");
    if (totalPages > 1) pages.push(totalPages);
    return pages;
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Media Library</h1>
          {meta && (
            <p className="text-sm text-slate-500 mt-0.5">
              {meta.total} file{meta.total !== 1 ? "s" : ""}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            multiple
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
                Uploading{uploadQueue > 1 ? ` (${uploadQueue})` : ""} {uploadProgress}%
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

      {/* Toolbar */}
      <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-3 shadow-sm">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <Input
              placeholder="Search media files..."
              value={search}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearch(e.target.value)}
              className="pl-9 rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
            />
          </div>
          <div className="flex items-center gap-1 border border-slate-200 rounded-lg p-1 shrink-0 bg-slate-50">
            <button
              onClick={() => setViewMode("grid")}
              className={`p-2 rounded-md transition-colors ${
                viewMode === "grid"
                  ? "bg-white text-indigo-700 shadow-sm border border-slate-200"
                  : "text-slate-400 hover:text-slate-600 border border-transparent"
              }`}
              title="Grid view"
            >
              <LayoutGrid className="h-4 w-4" />
            </button>
            <button
              onClick={() => setViewMode("table")}
              className={`p-2 rounded-md transition-colors ${
                viewMode === "table"
                  ? "bg-white text-indigo-700 shadow-sm border border-slate-200"
                  : "text-slate-400 hover:text-slate-600 border border-transparent"
              }`}
              title="Table view"
            >
              <List className="h-4 w-4" />
            </button>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Select value={mimeFilter} onValueChange={setMimeFilter}>
            <SelectTrigger className="w-32 rounded-lg border-slate-300 text-xs h-8">
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
          <Select value={sortBy} onValueChange={setSortBy}>
            <SelectTrigger className="w-36 rounded-lg border-slate-300 text-xs h-8">
              <ArrowUpDown className="mr-1.5 h-3 w-3 text-slate-400" />
              <SelectValue placeholder="Sort by" />
            </SelectTrigger>
            <SelectContent>
              {SORT_OPTIONS.map((s) => (
                <SelectItem key={s.value} value={s.value}>
                  {s.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Bulk actions bar */}
      {bulkSelected.size > 0 && (
        <div className="flex items-center gap-3 rounded-lg border border-indigo-200 bg-indigo-50 px-4 py-2">
          <span className="text-sm font-medium text-indigo-700">
            {bulkSelected.size} selected
          </span>
          <Button
            variant="outline"
            size="sm"
            className="text-red-600 border-red-200 hover:bg-red-50 rounded-lg text-xs"
            onClick={() => setBulkDeleteOpen(true)}
          >
            <Trash2 className="mr-1.5 h-3.5 w-3.5" />
            Delete Selected
          </Button>
          <button
            className="ml-auto text-xs text-indigo-600 hover:underline"
            onClick={() => setBulkSelected(new Set())}
          >
            Clear selection
          </button>
        </div>
      )}

      {/* Upload progress */}
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

      {/* Main area */}
      <div className="flex gap-6">
        {/* Content area */}
        <div className="flex-1 min-w-0">
          <Card
            className={`relative rounded-xl border shadow-sm overflow-hidden py-0 gap-0 transition-colors ${
              dragOver ? "border-indigo-400 bg-indigo-50/50" : "border-slate-200"
            }`}
            onDragOver={(e: React.DragEvent) => {
              e.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
          >
            <CardContent className="p-0">
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
              ) : viewMode === "grid" ? (
                /* ---- GRID VIEW ---- */
                <div className="grid grid-cols-2 gap-2 p-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
                  {files.map((file) => {
                    const isBulk = bulkSelected.has(file.id);
                    const isSelected = selected?.id === file.id;
                    return (
                      <div
                        key={file.id}
                        onClick={(e) => {
                          if (e.shiftKey || e.ctrlKey || e.metaKey) {
                            toggleBulkSelect(file.id);
                          } else {
                            selectFile(isSelected ? null : file);
                          }
                        }}
                        className={`group relative flex flex-col overflow-hidden rounded-lg cursor-pointer transition-all duration-150 bg-white ${
                          isSelected
                            ? "ring-2 ring-indigo-500"
                            : isBulk
                            ? "ring-2 ring-indigo-300"
                            : "ring-1 ring-slate-200 hover:ring-slate-300 hover:shadow-md"
                        }`}
                      >
                        {/* Thumbnail — clean, no overlays */}
                        <div className="relative w-full bg-slate-100 flex items-center justify-center overflow-hidden" style={{ aspectRatio: "1 / 1" }}>
                          {isImage(file.mime_type) ? (
                            <MediaImage
                              src={file.url}
                              alt={file.alt || file.original_name}
                              style={{ width: "100%", height: "100%", objectFit: "cover" }}
                            />
                          ) : (
                            <div className="flex flex-col items-center gap-2">
                              <FileIcon mime={file.mime_type} className="h-8 w-8 text-slate-400" />
                              <span className="rounded-md bg-slate-200/80 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-slate-500">
                                {getFileExtension(file.original_name)}
                              </span>
                            </div>
                          )}
                          {/* Selection check — only when selected */}
                          {isSelected && (
                            <div className="absolute top-1.5 right-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-indigo-500">
                              <Check className="h-3 w-3 text-white" />
                            </div>
                          )}
                          {/* Bulk check — only when bulk selected */}
                          {isBulk && !isSelected && (
                            <div className="absolute top-1.5 right-1.5 flex h-5 w-5 items-center justify-center rounded bg-indigo-500">
                              <Check className="h-3 w-3 text-white" />
                            </div>
                          )}
                        </div>

                        {/* Footer — file info + actions on hover */}
                        <div className="relative border-t border-slate-100">
                          {/* Default: file info */}
                          <div className="px-2.5 py-2 group-hover:opacity-0 transition-opacity duration-100">
                            <p className="truncate text-[12px] font-medium text-slate-800 leading-tight">
                              {file.original_name}
                            </p>
                            <p className="mt-0.5 text-[10px] text-slate-400 tabular-nums">
                              {humanFileSize(file.size)}
                            </p>
                          </div>
                          {/* Hover: action buttons */}
                          <div className="absolute inset-0 flex items-center justify-center gap-1 px-2 opacity-0 group-hover:opacity-100 transition-opacity duration-100 bg-white">
                            <button
                              className="flex h-7 flex-1 items-center justify-center gap-1 rounded-md text-slate-600 hover:bg-slate-100 text-[11px] font-medium transition-colors"
                              title="Copy URL"
                              onClick={(e) => {
                                e.stopPropagation();
                                copyToClipboard(window.location.origin + file.url).then(() => toast.success("URL copied"));
                              }}
                            >
                              <Copy className="h-3 w-3" />
                              Copy
                            </button>
                            <button
                              className="flex h-7 flex-1 items-center justify-center gap-1 rounded-md text-slate-600 hover:bg-slate-100 text-[11px] font-medium transition-colors"
                              title="Download"
                              onClick={(e) => {
                                e.stopPropagation();
                                window.open(file.url, "_blank");
                              }}
                            >
                              <Download className="h-3 w-3" />
                              Save
                            </button>
                            <button
                              className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-red-400 hover:text-red-600 hover:bg-red-50 transition-colors"
                              title="Delete"
                              onClick={(e) => { e.stopPropagation(); setDeleteTarget(file); }}
                            >
                              <Trash2 className="h-3 w-3" />
                            </button>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                /* ---- TABLE VIEW ---- */
                <Table>
                  <TableHeader>
                    <TableRow className="bg-slate-50 hover:bg-slate-50">
                      <TableHead className="w-10 pl-4">
                        <input
                          type="checkbox"
                          checked={bulkSelected.size === files.length && files.length > 0}
                          onChange={toggleSelectAll}
                          className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                        />
                      </TableHead>
                      <TableHead className="w-12"></TableHead>
                      <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                        Name
                      </TableHead>
                      <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                        Type
                      </TableHead>
                      <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                        Size
                      </TableHead>
                      <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                        Dimensions
                      </TableHead>
                      <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                        Date
                      </TableHead>
                      <TableHead className="w-20"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {files.map((file) => {
                      const isBulk = bulkSelected.has(file.id);
                      return (
                        <TableRow
                          key={file.id}
                          className={`cursor-pointer transition-colors ${
                            selected?.id === file.id
                              ? "bg-indigo-50"
                              : isBulk
                              ? "bg-indigo-50/50"
                              : "hover:bg-slate-50"
                          }`}
                          onClick={() =>
                            selectFile(selected?.id === file.id ? null : file)
                          }
                        >
                          <TableCell className="pl-4">
                            <input
                              type="checkbox"
                              checked={isBulk}
                              onChange={(e) => {
                                e.stopPropagation();
                                toggleBulkSelect(file.id);
                              }}
                              onClick={(e) => e.stopPropagation()}
                              className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                            />
                          </TableCell>
                          <TableCell>
                            <div className="h-10 w-10 rounded-md bg-slate-100 overflow-hidden flex items-center justify-center">
                              {isImage(file.mime_type) ? (
                                <MediaImage
                                  src={file.url}
                                  alt={file.original_name}
                                  className="h-full w-full object-cover"
                                />
                              ) : (
                                <FileIcon
                                  mime={file.mime_type}
                                  className="h-5 w-5 text-slate-400"
                                />
                              )}
                            </div>
                          </TableCell>
                          <TableCell>
                            <p className="text-sm font-medium text-slate-800 truncate max-w-[200px]">
                              {file.original_name}
                            </p>
                            {file.alt && (
                              <p className="text-xs text-slate-400 truncate max-w-[200px]">
                                {file.alt}
                              </p>
                            )}
                          </TableCell>
                          <TableCell>
                            <Badge
                              className="text-[10px] font-medium border-0"
                              variant="secondary"
                            >
                              {mimeLabel(file.mime_type)}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-sm text-slate-600">
                            {humanFileSize(file.size)}
                          </TableCell>
                          <TableCell className="text-sm text-slate-600">
                            {file.width && file.height
                              ? `${file.width}×${file.height}`
                              : "—"}
                          </TableCell>
                          <TableCell className="text-sm text-slate-500">
                            {new Date(file.created_at).toLocaleDateString()}
                          </TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 w-7 p-0 text-red-400 hover:text-red-600 hover:bg-red-50"
                              onClick={(e: React.MouseEvent) => {
                                e.stopPropagation();
                                setDeleteTarget(file);
                              }}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              )}

              {/* Drag overlay */}
              {dragOver && (
                <div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-xl bg-indigo-50/80 border-2 border-dashed border-indigo-400">
                  <div className="text-center">
                    <Upload className="mx-auto h-10 w-10 text-indigo-500" />
                    <p className="mt-2 text-sm font-medium text-indigo-700">
                      Drop files to upload
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Pagination */}
          {meta && totalPages > 1 && (
            <div className="mt-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <p className="text-sm text-slate-500">
                  {(meta.page - 1) * meta.per_page + 1}–
                  {Math.min(meta.page * meta.per_page, meta.total)} of {meta.total}
                </p>
                <Select
                  value={String(perPage)}
                  onValueChange={(v: string) => setPerPage(Number(v))}
                >
                  <SelectTrigger className="w-20 h-8 rounded-lg border-slate-300 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PER_PAGE_OPTIONS.map((n) => (
                      <SelectItem key={n} value={String(n)}>
                        {n}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <span className="text-xs text-slate-400">per page</span>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage(1)}
                  className="h-8 w-8 p-0 rounded-lg border-slate-300"
                >
                  <ChevronsLeft className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                  className="h-8 w-8 p-0 rounded-lg border-slate-300"
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                {getPageNumbers().map((p, i) =>
                  p === "..." ? (
                    <span key={`dots-${i}`} className="px-1 text-slate-400 text-sm">
                      ...
                    </span>
                  ) : (
                    <Button
                      key={p}
                      variant={page === p ? "default" : "outline"}
                      size="sm"
                      onClick={() => setPage(p as number)}
                      className={`h-8 w-8 p-0 rounded-lg text-xs ${
                        page === p
                          ? "bg-indigo-600 text-white hover:bg-indigo-700"
                          : "border-slate-300"
                      }`}
                    >
                      {p}
                    </Button>
                  )
                )}
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                  className="h-8 w-8 p-0 rounded-lg border-slate-300"
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage(totalPages)}
                  className="h-8 w-8 p-0 rounded-lg border-slate-300"
                >
                  <ChevronsRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* Detail panel — sidebar on lg, dialog on smaller */}
        {selected && (
          <>
          {/* Desktop sidebar */}
          <div className="hidden w-80 shrink-0 lg:block">
            <Card className="rounded-xl border border-slate-200 shadow-sm sticky top-0">
              <CardContent className="p-4 space-y-4">
                {/* Close */}
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-slate-900">File Details</h3>
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
                    <div className="aspect-square overflow-hidden">
                      <MediaImage
                        src={selected.url}
                        alt={selected.alt || selected.original_name}
                        className="w-full h-full object-cover"
                      />
                    </div>
                  ) : isVideo(selected.mime_type) ? (
                    <MediaVideo
                      src={selected.url}
                      controls
                      className="w-full max-h-48"
                    />
                  ) : isAudio(selected.mime_type) ? (
                    <div className="p-4">
                      <audio src={selected.url} controls className="w-full" />
                    </div>
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
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <span className="text-slate-500">Type</span>
                      <p className="font-medium text-slate-800">
                        {mimeLabel(selected.mime_type)}
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
                        {selected.width} × {selected.height}
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

                {/* Rename */}
                <div className="space-y-1.5">
                  <Label className="text-xs text-slate-500">File Name</Label>
                  <Input
                    value={editName}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      setEditName(e.target.value)
                    }
                    className="rounded-lg border-slate-300 text-sm"
                  />
                </div>

                {/* Alt text */}
                <div className="space-y-1.5">
                  <Label className="text-xs text-slate-500">Alt Text</Label>
                  <Input
                    placeholder="Describe this file for accessibility..."
                    value={editAlt}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      setEditAlt(e.target.value)
                    }
                    className="rounded-lg border-slate-300 text-sm"
                  />
                </div>

                {/* Save button */}
                {hasDetailChanges && (
                  <Button
                    size="sm"
                    onClick={handleSaveDetail}
                    disabled={savingDetail}
                    className="w-full bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg"
                  >
                    {savingDetail ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Pencil className="mr-2 h-4 w-4" />
                    )}
                    {savingDetail ? "Saving..." : "Save Changes"}
                  </Button>
                )}

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
                  <a
                    href={selected.url}
                    download={selected.original_name}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center justify-center h-8 px-3 rounded-lg border border-slate-200 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                  >
                    <Download className="mr-1.5 h-3.5 w-3.5" />
                    Download
                  </a>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full rounded-lg text-xs text-red-500 hover:text-red-600 hover:bg-red-50 border-red-200"
                  onClick={() => setDeleteTarget(selected)}
                >
                  <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                  Delete File
                </Button>
              </CardContent>
            </Card>
          </div>
          {/* Mobile/tablet dialog — only renders below lg breakpoint */}
          <Dialog open={mobileDetailOpen} onOpenChange={(open: boolean) => { if (!open) { setMobileDetailOpen(false); setSelected(null); } }}>
            <DialogContent className="max-w-md max-h-[85vh] overflow-y-auto lg:hidden">
              <DialogHeader>
                <DialogTitle className="text-sm">File Details</DialogTitle>
              </DialogHeader>
              {/* Preview */}
              <div className="overflow-hidden rounded-lg border border-slate-200 bg-slate-50">
                {isImage(selected.mime_type) ? (
                  <div className="aspect-square overflow-hidden"><MediaImage src={selected.url} alt={selected.alt || selected.original_name} className="w-full h-full object-cover" /></div>
                ) : isVideo(selected.mime_type) ? (
                  <MediaVideo src={selected.url} controls className="w-full max-h-48" />
                ) : isAudio(selected.mime_type) ? (
                  <div className="p-4"><audio src={selected.url} controls className="w-full" /></div>
                ) : (
                  <div className="flex h-24 flex-col items-center justify-center gap-2">
                    <FileIcon mime={selected.mime_type} className="h-10 w-10 text-slate-400" />
                    <span className="rounded bg-slate-200 px-2 py-0.5 text-xs font-semibold text-slate-500">{getFileExtension(selected.original_name)}</span>
                  </div>
                )}
              </div>
              <div className="space-y-3 text-sm">
                <div className="grid grid-cols-2 gap-2">
                  <div><span className="text-slate-500">Type</span><p className="font-medium text-slate-800">{mimeLabel(selected.mime_type)}</p></div>
                  <div><span className="text-slate-500">Size</span><p className="font-medium text-slate-800">{humanFileSize(selected.size)}</p></div>
                </div>
                {selected.width && selected.height && (
                  <div><span className="text-slate-500">Dimensions</span><p className="font-medium text-slate-800">{selected.width} × {selected.height}</p></div>
                )}
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs text-slate-500">File Name</Label>
                <Input value={editName} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditName(e.target.value)} className="rounded-lg border-slate-300 text-sm" />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs text-slate-500">Alt Text</Label>
                <Input placeholder="Describe this file..." value={editAlt} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditAlt(e.target.value)} className="rounded-lg border-slate-300 text-sm" />
              </div>
              {hasDetailChanges && (
                <Button size="sm" onClick={handleSaveDetail} disabled={savingDetail} className="w-full bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg">
                  {savingDetail ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Pencil className="mr-2 h-4 w-4" />}
                  {savingDetail ? "Saving..." : "Save Changes"}
                </Button>
              )}
              <div className="flex gap-2">
                <Button variant="outline" size="sm" className="flex-1 rounded-lg text-xs" onClick={handleCopyUrl}>
                  {copied ? <Check className="mr-1.5 h-3.5 w-3.5 text-emerald-500" /> : <Copy className="mr-1.5 h-3.5 w-3.5" />}
                  {copied ? "Copied" : "Copy URL"}
                </Button>
                <a href={selected.url} download={selected.original_name} target="_blank" rel="noopener noreferrer"
                  className="inline-flex items-center justify-center h-8 px-3 rounded-lg border border-slate-200 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">
                  <Download className="mr-1.5 h-3.5 w-3.5" />Download
                </a>
              </div>
              <Button variant="outline" size="sm" className="w-full rounded-lg text-xs text-red-500 hover:text-red-600 hover:bg-red-50 border-red-200" onClick={() => setDeleteTarget(selected)}>
                <Trash2 className="mr-1.5 h-3.5 w-3.5" />Delete File
              </Button>
            </DialogContent>
          </Dialog>
          </>
        )}
      </div>

      {/* Delete single dialog */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(open: boolean) => !open && setDeleteTarget(null)}
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

      {/* Bulk delete dialog */}
      <Dialog
        open={bulkDeleteOpen}
        onOpenChange={(open: boolean) => !open && setBulkDeleteOpen(false)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {bulkSelected.size} Files</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete {bulkSelected.size} selected file
              {bulkSelected.size !== 1 ? "s" : ""}? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setBulkDeleteOpen(false)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleBulkDelete}
              disabled={deleting}
            >
              {deleting ? "Deleting..." : `Delete ${bulkSelected.size} Files`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
