import { useEffect, useState, useCallback, useRef } from "react";
import {
  Upload,
  Search,
  Loader2,
  Image as ImageIcon,
  FileText,
  Film,
  Music,
  File,
  Check,
  ChevronLeft,
  ChevronRight,
} from "@vibecms/icons";
import {
  Button,
  Input,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@vibecms/ui";
import { toast } from "sonner";

// ---------- Types ----------

export interface MediaFile {
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

interface MediaPickerModalProps {
  open: boolean;
  onClose: () => void;
  onSelect: (files: MediaFile[]) => void;
  multiple?: boolean;
  mimeFilter?: string;
}

// ---------- API helpers ----------

async function fetchMedia(params: {
  page: number;
  per_page: number;
  mime_type?: string;
  search?: string;
}): Promise<{ data: MediaFile[]; meta: PaginationMeta }> {
  const qs = new URLSearchParams();
  qs.set("page", String(params.page));
  qs.set("per_page", String(params.per_page));
  if (params.mime_type) qs.set("mime_type", params.mime_type);
  if (params.search) qs.set("search", params.search);
  qs.set("sort_by", "date_desc");

  const res = await fetch(`/admin/api/ext/media-manager/?${qs.toString()}`, {
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to fetch media");
  const body = await res.json();
  return { data: body.data, meta: body.meta };
}

async function uploadMediaFile(file: globalThis.File): Promise<MediaFile> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/admin/api/ext/media-manager/upload");
    xhr.withCredentials = true;
    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText).data);
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

function FileIcon({ mime, className }: { mime: string; className?: string }) {
  if (isImage(mime)) return <ImageIcon className={className} />;
  if (mime.startsWith("video/")) return <Film className={className} />;
  if (mime.startsWith("audio/")) return <Music className={className} />;
  if (mime.includes("pdf") || mime.includes("document") || mime.includes("text"))
    return <FileText className={className} />;
  return <File className={className} />;
}

// ---------- Component ----------

export default function MediaPickerModal({
  open,
  onClose,
  onSelect,
  multiple = false,
  mimeFilter,
}: MediaPickerModalProps) {
  const [files, setFiles] = useState<MediaFile[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>({ total: 0, page: 1, per_page: 24, total_pages: 0 });
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Map<number, MediaFile>>(new Map());
  const [uploading, setUploading] = useState(false);
  const [mimeType, setMimeType] = useState(mimeFilter || "");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const searchTimer = useRef<ReturnType<typeof setTimeout>>();

  const load = useCallback(async (page: number) => {
    setLoading(true);
    try {
      const result = await fetchMedia({
        page,
        per_page: 24,
        mime_type: mimeType || undefined,
        search: search || undefined,
      });
      setFiles(result.data);
      setMeta(result.meta);
    } catch {
      toast.error("Failed to load media");
    } finally {
      setLoading(false);
    }
  }, [search, mimeType]);

  useEffect(() => {
    if (open) {
      setSelected(new Map());
      load(1);
    }
  }, [open, load]);

  function handleSearchChange(val: string) {
    setSearch(val);
    clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => load(1), 300);
  }

  function toggleSelect(file: MediaFile) {
    if (multiple) {
      setSelected((prev) => {
        const next = new Map(prev);
        if (next.has(file.id)) next.delete(file.id);
        else next.add(file.id, file);
        return next;
      });
    } else {
      setSelected(new Map([[file.id, file]]));
    }
  }

  function handleConfirm() {
    if (selected.size === 0) return;
    onSelect(Array.from(selected.values()));
    onClose();
  }

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const fileList = e.target.files;
    if (!fileList?.length) return;
    setUploading(true);
    try {
      for (const f of Array.from(fileList)) {
        await uploadMediaFile(f);
      }
      toast.success(`Uploaded ${fileList.length} file(s)`);
      await load(1);
    } catch {
      toast.error("Upload failed");
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-4xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Select Media</DialogTitle>
        </DialogHeader>

        {/* Toolbar */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <Input
              placeholder="Search files..."
              value={search}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="pl-9 rounded-lg border-slate-300"
            />
          </div>
          <Select value={mimeType} onValueChange={(v) => { setMimeType(v === "all" ? "" : v); }}>
            <SelectTrigger className="w-[150px] rounded-lg border-slate-300">
              <SelectValue placeholder="All types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All types</SelectItem>
              <SelectItem value="image">Images</SelectItem>
              <SelectItem value="application">Documents</SelectItem>
              <SelectItem value="video">Videos</SelectItem>
              <SelectItem value="audio">Audio</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            size="sm"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            className="rounded-lg"
          >
            {uploading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Upload className="mr-2 h-4 w-4" />}
            Upload
          </Button>
          <input ref={fileInputRef} type="file" multiple className="hidden" onChange={handleUpload} />
        </div>

        {/* Grid */}
        <div className="flex-1 overflow-y-auto min-h-0 mt-2">
          {loading ? (
            <div className="flex items-center justify-center h-48">
              <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
            </div>
          ) : files.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-48 text-slate-400">
              <ImageIcon className="h-12 w-12 mb-2" />
              <p className="text-sm">No media files found</p>
            </div>
          ) : (
            <div className="grid grid-cols-4 sm:grid-cols-5 md:grid-cols-6 gap-2">
              {files.map((file) => {
                const isSelected = selected.has(file.id);
                return (
                  <button
                    key={file.id}
                    type="button"
                    onClick={() => toggleSelect(file)}
                    className={`group relative aspect-square rounded-lg border-2 overflow-hidden transition-all ${
                      isSelected
                        ? "border-indigo-500 ring-2 ring-indigo-500/20"
                        : "border-slate-200 hover:border-slate-300"
                    }`}
                  >
                    {isImage(file.mime_type) ? (
                      <img
                        src={file.url}
                        alt={file.alt || file.original_name}
                        className="h-full w-full object-cover"
                        loading="lazy"
                      />
                    ) : (
                      <div className="h-full w-full flex flex-col items-center justify-center bg-slate-50 p-2">
                        <FileIcon mime={file.mime_type} className="h-8 w-8 text-slate-400" />
                        <span className="text-[10px] text-slate-500 mt-1 truncate w-full text-center">
                          {file.original_name}
                        </span>
                      </div>
                    )}
                    {/* Selection indicator */}
                    {isSelected && (
                      <div className="absolute top-1 right-1 h-5 w-5 rounded-full bg-indigo-500 flex items-center justify-center">
                        <Check className="h-3 w-3 text-white" />
                      </div>
                    )}
                    {/* File info overlay on hover */}
                    <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent p-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      <p className="text-[10px] text-white truncate">{file.original_name}</p>
                      <p className="text-[9px] text-white/70">{humanFileSize(file.size)}</p>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Pagination */}
        {meta.total_pages > 1 && (
          <div className="flex items-center justify-center gap-2 pt-2">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={meta.page <= 1}
              onClick={() => load(meta.page - 1)}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm text-slate-500">
              {meta.page} / {meta.total_pages}
            </span>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={meta.page >= meta.total_pages}
              onClick={() => load(meta.page + 1)}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose} className="rounded-lg">
            Cancel
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={selected.size === 0}
            className="rounded-lg bg-indigo-600 hover:bg-indigo-700"
          >
            {selected.size > 0
              ? `Select ${selected.size} file${selected.size > 1 ? "s" : ""}`
              : "Select"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
