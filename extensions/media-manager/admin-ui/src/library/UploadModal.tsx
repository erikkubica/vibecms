import { useState, useRef, useEffect, useCallback } from "react";
import {
  UploadCloud,
  X,
  Folder,
  Image as ImageIcon,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  RotateCw,
  Check,
} from "@vibecms/icons";
import { humanFileSize } from "./helpers";

type Status = "uploading" | "processing" | "done" | "error";

interface QueueItem {
  id: string;
  name: string;
  size: number;
  progress: number;
  status: Status;
  error: string | null;
  file?: globalThis.File;
}

interface UploadModalProps {
  initialFiles?: globalThis.File[];
  onClose: () => void;
  onUploadFile: (file: globalThis.File, onProgress: (pct: number) => void) => Promise<void>;
  onComplete: (uploadedCount: number) => void;
}

export default function UploadModal({ initialFiles, onClose, onUploadFile, onComplete }: UploadModalProps) {
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [dragHot, setDragHot] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const activeRef = useRef<Set<string>>(new Set());

  const updateItem = useCallback((id: string, patch: Partial<QueueItem>) => {
    setQueue((q) => q.map((it) => (it.id === id ? { ...it, ...patch } : it)));
  }, []);

  const startUpload = useCallback(
    async (item: QueueItem) => {
      if (!item.file || activeRef.current.has(item.id)) return;
      activeRef.current.add(item.id);
      try {
        await onUploadFile(item.file, (pct) => updateItem(item.id, { progress: pct, status: "uploading" }));
        updateItem(item.id, { progress: 100, status: "done" });
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Upload failed";
        updateItem(item.id, { status: "error", error: msg, progress: 100 });
      } finally {
        activeRef.current.delete(item.id);
      }
    },
    [onUploadFile, updateItem]
  );

  const addFiles = useCallback(
    (fs: FileList | globalThis.File[]) => {
      const arr = Array.from(fs);
      const items: QueueItem[] = arr.map((f) => ({
        id: Math.random().toString(36).slice(2),
        name: f.name,
        size: f.size,
        progress: 0,
        status: "uploading",
        error: null,
        file: f,
      }));
      setQueue((q) => [...items, ...q]);
      items.forEach(startUpload);
    },
    [startUpload]
  );

  // Kick off any seeded files
  useEffect(() => {
    if (initialFiles?.length) addFiles(initialFiles);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function retry(id: string) {
    const item = queue.find((q) => q.id === id);
    if (!item) return;
    updateItem(id, { status: "uploading", progress: 0, error: null });
    if (item.file) startUpload({ ...item, status: "uploading", progress: 0, error: null });
  }
  function retryAll() {
    queue.filter((q) => q.status === "error").forEach((q) => retry(q.id));
  }
  function remove(id: string) {
    setQueue((q) => q.filter((it) => it.id !== id));
  }

  const total = queue.length;
  const done = queue.filter((q) => q.status === "done").length;
  const errors = queue.filter((q) => q.status === "error").length;
  const uploading = queue.filter((q) => q.status === "uploading" || q.status === "processing").length;
  const overall = total
    ? Math.round(
        queue.reduce((s, q) => s + (q.status === "done" || q.status === "error" ? 100 : q.progress), 0) / total
      )
    : 0;

  function close() {
    onComplete(done);
    onClose();
  }

  // Esc to close
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [done]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6" onMouseDown={close}>
      <div className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" />
      <div
        className="relative w-[720px] max-w-full max-h-[86vh] rounded-2xl bg-white shadow-2xl overflow-hidden flex flex-col"
        onMouseDown={(e) => e.stopPropagation()}
      >
        {/* header */}
        <div className="h-14 px-5 flex items-center gap-2 border-b border-slate-200 shrink-0">
          <UploadCloud className="h-5 w-5 text-indigo-600" />
          <div className="text-[14px] font-semibold text-slate-900">Upload media</div>
          {total > 0 && (
            <div className="ml-3 flex items-center gap-1.5 text-[11.5px]">
              <span className="px-1.5 py-0.5 rounded-md bg-slate-100 text-slate-600 tabular-nums font-mono">
                {done}/{total} done
              </span>
              {errors > 0 && (
                <span className="px-1.5 py-0.5 rounded-md bg-rose-50 text-rose-600 tabular-nums font-mono">
                  {errors} error{errors > 1 ? "s" : ""}
                </span>
              )}
            </div>
          )}
          <div className="flex-1" />
          <button
            type="button"
            onClick={close}
            className="w-8 h-8 rounded-md hover:bg-slate-100 text-slate-500 grid place-items-center cursor-pointer"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* body */}
        <div className="flex-1 overflow-y-auto p-5">
          <label
            onDragOver={(e) => {
              e.preventDefault();
              setDragHot(true);
            }}
            onDragLeave={() => setDragHot(false)}
            onDrop={(e) => {
              e.preventDefault();
              setDragHot(false);
              if (e.dataTransfer.files?.length) addFiles(e.dataTransfer.files);
            }}
            className={`block rounded-xl border-2 border-dashed transition-all cursor-pointer ${
              dragHot
                ? "border-indigo-500 bg-indigo-50/60"
                : "border-slate-300 hover:border-indigo-400 hover:bg-slate-50/60 bg-slate-50/30"
            }`}
          >
            <input
              ref={inputRef}
              type="file"
              multiple
              className="hidden"
              onChange={(e) => {
                if (e.target.files?.length) addFiles(e.target.files);
                e.target.value = "";
              }}
            />
            <div className="px-6 py-10 text-center">
              <div className="mx-auto w-14 h-14 rounded-2xl bg-indigo-100 grid place-items-center mb-3">
                <UploadCloud className="h-7 w-7 text-indigo-600" />
              </div>
              <div className="text-[14px] font-semibold text-slate-900">Drop files here, or click to browse</div>
              <div className="mt-1 text-[12px] text-slate-500">
                JPG, PNG, WebP, AVIF, SVG, MP4 — multiple files supported
              </div>
              <div className="mt-3 flex items-center justify-center gap-2">
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault();
                    inputRef.current?.click();
                  }}
                  className="px-3 h-8 rounded-md bg-indigo-600 hover:bg-indigo-700 text-white text-[12px] font-medium flex items-center gap-1.5 cursor-pointer"
                >
                  <Folder className="h-3 w-3" /> Browse files
                </button>
              </div>
            </div>
          </label>

          {queue.length > 0 && (
            <div className="mt-5">
              <div className="flex items-center gap-2 mb-2">
                <div className="text-[11px] font-semibold uppercase tracking-wider text-slate-500">
                  Queue · {total}
                </div>
                <div className="flex-1 h-px bg-slate-200" />
                {errors > 0 && (
                  <button
                    type="button"
                    onClick={retryAll}
                    className="text-[11px] text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1 cursor-pointer"
                  >
                    <RotateCw className="h-3 w-3" /> Retry failed
                  </button>
                )}
                <div className="text-[11px] font-mono tabular-nums text-slate-500">{overall}%</div>
              </div>
              <div className="space-y-2">
                {queue.map((item) => (
                  <div
                    key={item.id}
                    className={`rounded-lg border bg-white flex items-center gap-3 p-2.5 ${
                      item.status === "error" ? "border-rose-200 bg-rose-50/40" : "border-slate-200"
                    }`}
                  >
                    <div className="w-10 h-10 rounded-md overflow-hidden border border-slate-200 shrink-0 relative bg-slate-100 grid place-items-center">
                      <ImageIcon className="h-4 w-4 text-slate-400" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <div className="text-[12.5px] font-medium text-slate-800 truncate flex-1">{item.name}</div>
                        <div className="text-[10.5px] font-mono tabular-nums text-slate-400">
                          {humanFileSize(item.size)}
                        </div>
                      </div>
                      {item.status === "uploading" && (
                        <div className="mt-1.5 h-1.5 rounded-full bg-slate-100 overflow-hidden">
                          <div
                            className="h-full bg-indigo-500 transition-[width] duration-200"
                            style={{ width: item.progress + "%" }}
                          />
                        </div>
                      )}
                      {item.status === "processing" && (
                        <div className="mt-1.5 flex items-center gap-1.5 text-[11px] text-indigo-600">
                          <Loader2 className="h-3 w-3 animate-spin" /> Optimizing…
                        </div>
                      )}
                      {item.status === "done" && (
                        <div className="mt-1 flex items-center gap-1.5 text-[11px] text-emerald-700">
                          <CheckCircle2 className="h-3 w-3" /> Uploaded &amp; optimized
                        </div>
                      )}
                      {item.status === "error" && (
                        <div className="mt-1 flex items-center gap-1.5 text-[11px] text-rose-600">
                          <AlertTriangle className="h-3 w-3" /> {item.error}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-0.5 shrink-0">
                      {item.status === "uploading" && (
                        <div className="text-[11px] font-mono tabular-nums text-slate-500 w-10 text-right">
                          {Math.round(item.progress)}%
                        </div>
                      )}
                      {item.status === "error" && (
                        <button
                          type="button"
                          onClick={() => retry(item.id)}
                          className="h-7 px-2 rounded-md hover:bg-rose-100 text-rose-600 text-[11px] font-medium flex items-center gap-1 cursor-pointer"
                        >
                          <RotateCw className="h-3 w-3" /> Retry
                        </button>
                      )}
                      {item.status === "done" && (
                        <div className="w-6 h-6 rounded-full bg-emerald-500 text-white grid place-items-center">
                          <Check className="h-3 w-3" strokeWidth={3} />
                        </div>
                      )}
                      <button
                        type="button"
                        onClick={() => remove(item.id)}
                        className="w-7 h-7 rounded-md hover:bg-slate-100 text-slate-400 hover:text-slate-700 grid place-items-center cursor-pointer"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <div className="border-t border-slate-200 p-3 flex items-center gap-2 shrink-0">
          <div className="text-[11.5px] text-slate-500">
            {uploading > 0
              ? `Uploading ${uploading}…`
              : total === 0
              ? "No files yet"
              : `${done} uploaded · ${errors} failed`}
          </div>
          <div className="flex-1" />
          <button
            type="button"
            onClick={close}
            className="px-3 h-9 rounded-lg border border-slate-300 hover:bg-slate-50 text-slate-700 text-[12.5px] font-medium cursor-pointer"
          >
            {done > 0 ? "Done" : "Close"}
          </button>
        </div>
      </div>
    </div>
  );
}
