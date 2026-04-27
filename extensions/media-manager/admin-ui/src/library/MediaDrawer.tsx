import { useState, useEffect } from "react";
import {
  X,
  Link as LinkIcon,
  Download,
  Copy,
  Check,
  Trash2,
  Zap,
  RotateCcw,
  RefreshCw,
  Loader2,
  ExternalLink,
  Sparkles,
} from "@vibecms/icons";
import {
  MediaFile,
  isImage,
  isVideo,
  isAudio,
  imageSize,
  humanFileSize,
  mimeLabel,
  savedPercent,
  MediaImage,
  MediaVideo,
  FileTypeIcon,
  getFileExtension,
} from "./helpers";

type Tab = "details" | "variants" | "usage";

interface MediaDrawerProps {
  file: MediaFile;
  copyState: number | null;
  saving: boolean;
  restoring: boolean;
  reoptimizing: boolean;
  onClose: () => void;
  onSave: (patch: { alt?: string; original_name?: string }) => void;
  onCopy: (f: MediaFile) => void;
  onDownload: (f: MediaFile) => void;
  onDelete: (f: MediaFile) => void;
  onRestore: () => void;
  onReoptimize: () => void;
}

const VARIANT_SIZES: { label: string; size: string }[] = [
  { label: "thumbnail", size: "thumbnail" },
  { label: "medium", size: "medium" },
  { label: "large", size: "large" },
  { label: "original", size: "" },
];

function Field({
  label,
  children,
  hint,
  required,
}: {
  label: string;
  children: React.ReactNode;
  hint?: string;
  required?: boolean;
}) {
  return (
    <div>
      <label className="text-[11px] font-semibold uppercase tracking-wider text-slate-500 flex items-center gap-1.5">
        {label}
        {required && <span className="text-rose-500">*</span>}
      </label>
      <div className="mt-1">{children}</div>
      {hint && <p className="mt-1 text-[11px] text-slate-400">{hint}</p>}
    </div>
  );
}

export default function MediaDrawer({
  file,
  copyState,
  saving,
  restoring,
  reoptimizing,
  onClose,
  onSave,
  onCopy,
  onDownload,
  onDelete,
  onRestore,
  onReoptimize,
}: MediaDrawerProps) {
  const [name, setName] = useState(file.original_name);
  const [alt, setAlt] = useState(file.alt || "");
  const [tab, setTab] = useState<Tab>("details");

  useEffect(() => {
    setName(file.original_name);
    setAlt(file.alt || "");
    setTab("details");
  }, [file.id, file.original_name, file.alt]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const dirty = name !== file.original_name || alt !== (file.alt || "");
  const ext = getFileExtension(file.original_name);
  const base = ext ? name.replace(/\.[^.]+$/, "") : name;
  const fullUrl = typeof window !== "undefined" ? window.location.origin + file.url : file.url;
  const showOptControls = isImage(file.mime_type) && file.mime_type !== "image/svg+xml";

  return (
    <div className="fixed inset-0 z-40 flex" onMouseDown={onClose}>
      <div className="flex-1 bg-slate-900/30 backdrop-blur-[2px]" />
      <div
        className="w-[480px] max-w-full bg-white h-full shadow-2xl flex flex-col overflow-hidden"
        onMouseDown={(e) => e.stopPropagation()}
      >
        {/* header */}
        <div className="h-14 px-4 flex items-center gap-2 border-b border-slate-200 shrink-0">
          <div className="text-[13px] font-semibold text-slate-900 flex-1 truncate">Edit media</div>
          <button
            type="button"
            onClick={() => onCopy(file)}
            className="h-8 px-2.5 rounded-md hover:bg-slate-100 text-slate-500 text-[12px] flex items-center gap-1.5 cursor-pointer"
          >
            {copyState === file.id ? (
              <>
                <Check className="h-3 w-3 text-emerald-600" /> Copied
              </>
            ) : (
              <>
                <LinkIcon className="h-3 w-3" /> Copy URL
              </>
            )}
          </button>
          <button
            type="button"
            onClick={() => onDownload(file)}
            className="w-8 h-8 rounded-md hover:bg-slate-100 text-slate-500 grid place-items-center cursor-pointer"
            title="Download"
          >
            <Download className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-md hover:bg-slate-100 text-slate-500 grid place-items-center cursor-pointer"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* preview */}
        <div className="p-4 pb-2">
          <div className="relative rounded-lg overflow-hidden bg-slate-100 aspect-[4/3] border border-slate-200">
            {isImage(file.mime_type) ? (
              <MediaImage
                src={imageSize(file.url, "large", file.updated_at)}
                alt={file.alt || file.original_name}
                className="absolute inset-0 w-full h-full object-contain"
              />
            ) : isVideo(file.mime_type) ? (
              <MediaVideo src={file.url} controls className="absolute inset-0 w-full h-full" />
            ) : isAudio(file.mime_type) ? (
              <div className="absolute inset-0 grid place-items-center p-4">
                <audio src={file.url} controls className="w-full" />
              </div>
            ) : (
              <div className="absolute inset-0 grid place-items-center">
                <div className="flex flex-col items-center gap-2">
                  <FileTypeIcon mime={file.mime_type} className="h-10 w-10 text-slate-400" />
                  <span className="rounded bg-slate-200 px-2 py-0.5 text-xs font-semibold text-slate-500">
                    {ext}
                  </span>
                </div>
              </div>
            )}
            <div className="absolute bottom-2 left-2 flex items-center gap-1.5">
              {file.is_optimized && file.optimization_savings > 0 ? (
                <div className="rounded-full bg-emerald-500/95 text-white px-2 py-0.5 text-[10.5px] font-semibold flex items-center gap-1">
                  <Zap className="h-3 w-3" /> Saved {humanFileSize(file.optimization_savings)}
                </div>
              ) : showOptControls ? (
                <div className="rounded-full bg-slate-900/70 text-white px-2 py-0.5 text-[10.5px] font-medium">
                  Original
                </div>
              ) : null}
              {file.width && file.height && (
                <div className="rounded-full bg-slate-900/70 text-white/90 px-2 py-0.5 text-[10.5px] font-mono tabular-nums">
                  {file.width}×{file.height}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* tabs */}
        <div className="px-4 border-b border-slate-200 flex items-center gap-1">
          {(
            [
              { id: "details", label: "Details" },
              { id: "variants", label: "Variants" },
            ] as const
          ).map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => setTab(t.id)}
              className={`px-3 py-2 text-[12.5px] font-medium border-b-2 -mb-px transition-colors cursor-pointer ${
                tab === t.id
                  ? "border-indigo-500 text-slate-900"
                  : "border-transparent text-slate-500 hover:text-slate-800"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {/* content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 text-[13px]">
          {tab === "details" && (
            <>
              <Field label="Filename" required>
                <div className="flex rounded-lg border border-slate-300 focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 transition-colors overflow-hidden bg-white">
                  <input
                    value={base}
                    onChange={(e) => setName(e.target.value + (ext ? "." + ext.toLowerCase() : ""))}
                    className="flex-1 min-w-0 px-3 py-2 text-[13px] outline-none bg-transparent"
                  />
                  {ext && (
                    <div className="px-2.5 py-2 text-[12px] text-slate-400 border-l border-slate-200 bg-slate-50 font-mono">
                      .{ext.toLowerCase()}
                    </div>
                  )}
                </div>
              </Field>

              <Field label="Alt text" hint="Describe the image for screen readers and SEO.">
                <textarea
                  value={alt}
                  onChange={(e) => setAlt(e.target.value)}
                  rows={2}
                  placeholder="A brief description of what's in this image…"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-[13px] outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 transition-colors resize-none"
                />
                <div className="mt-1 flex items-center justify-end">
                  <span className="text-[10.5px] font-mono text-slate-400 tabular-nums">{alt.length}/160</span>
                </div>
              </Field>

              <Field label="Public URL">
                <div className="flex rounded-lg border border-slate-200 bg-slate-50 overflow-hidden">
                  <code className="flex-1 min-w-0 px-3 py-2 text-[11.5px] text-slate-600 truncate font-mono">
                    {fullUrl}
                  </code>
                  <button
                    type="button"
                    onClick={() => onCopy(file)}
                    title="Copy URL"
                    className="px-2.5 border-l border-slate-200 hover:bg-slate-100 text-slate-600 grid place-items-center cursor-pointer"
                  >
                    {copyState === file.id ? (
                      <Check className="h-3.5 w-3.5 text-emerald-600" />
                    ) : (
                      <Copy className="h-3.5 w-3.5" />
                    )}
                  </button>
                  <a
                    href={file.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    title="Open in new tab"
                    className="px-2.5 border-l border-slate-200 hover:bg-slate-100 text-slate-600 grid place-items-center cursor-pointer"
                  >
                    <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                </div>
              </Field>

              <div className="grid grid-cols-2 gap-3">
                <Field label="Type">
                  <div className="text-[12.5px] text-slate-700">{mimeLabel(file.mime_type)}</div>
                </Field>
                <Field label="Size">
                  <div className="font-mono text-[12.5px] text-slate-700 tabular-nums">{humanFileSize(file.size)}</div>
                </Field>
                {file.width && file.height && (
                  <Field label="Dimensions">
                    <div className="font-mono text-[12.5px] text-slate-700 tabular-nums">
                      {file.width} × {file.height} px
                    </div>
                  </Field>
                )}
                <Field label="Uploaded">
                  <div className="text-[12.5px] text-slate-700">{new Date(file.created_at).toLocaleDateString()}</div>
                  <div className="text-[10.5px] text-slate-400 font-mono">
                    {new Date(file.created_at).toLocaleTimeString()}
                  </div>
                </Field>
              </div>

              {showOptControls && (
                <div
                  className={`rounded-lg p-3 border ${
                    file.is_optimized ? "bg-emerald-50/60 border-emerald-200" : "bg-amber-50/60 border-amber-200"
                  }`}
                >
                  <div className="flex items-start gap-2">
                    {file.is_optimized ? (
                      <Zap className="h-4 w-4 text-emerald-600 shrink-0 mt-0.5" />
                    ) : (
                      <Sparkles className="h-4 w-4 text-amber-600 shrink-0 mt-0.5" />
                    )}
                    <div className="flex-1 min-w-0">
                      <p
                        className={`text-[12px] font-semibold ${
                          file.is_optimized ? "text-emerald-800" : "text-amber-800"
                        }`}
                      >
                        {file.is_optimized ? "Optimized" : "Not optimized"}
                      </p>
                      {file.is_optimized && file.original_size > 0 && (
                        <p className="text-[11px] text-emerald-700 mt-0.5 tabular-nums font-mono">
                          {humanFileSize(file.original_size)} → {humanFileSize(file.size)} (−{savedPercent(file)}%)
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="mt-2 flex gap-1.5">
                    {file.is_optimized && file.original_path && (
                      <button
                        type="button"
                        onClick={onRestore}
                        disabled={restoring || reoptimizing}
                        className="flex-1 h-8 rounded-md text-[11.5px] text-amber-700 border border-amber-200 hover:bg-amber-50 font-medium flex items-center justify-center gap-1 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {restoring ? <Loader2 className="h-3 w-3 animate-spin" /> : <RotateCcw className="h-3 w-3" />}
                        Restore original
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={onReoptimize}
                      disabled={restoring || reoptimizing}
                      className="flex-1 h-8 rounded-md text-[11.5px] text-emerald-700 border border-emerald-200 hover:bg-emerald-50 font-medium flex items-center justify-center gap-1 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {reoptimizing ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <RefreshCw className="h-3 w-3" />
                      )}
                      {file.is_optimized ? "Re-optimize" : "Optimize now"}
                    </button>
                  </div>
                </div>
              )}
            </>
          )}

          {tab === "variants" && (
            <div className="space-y-3">
              <div className="text-[11.5px] text-slate-500">
                Auto-generated responsive sizes served from the image cache.
              </div>
              {VARIANT_SIZES.map((v) => {
                const url = v.size ? imageSize(file.url, v.size, file.updated_at) : file.url;
                const isOriginal = !v.size;
                return (
                  <div
                    key={v.label}
                    className="flex items-center gap-3 p-2.5 rounded-lg border border-slate-200 hover:border-slate-300 bg-white"
                  >
                    <div className="w-10 h-10 rounded-md overflow-hidden border border-slate-200 bg-slate-50 grid place-items-center shrink-0">
                      {isImage(file.mime_type) ? (
                        <MediaImage src={url} alt="" className="w-full h-full object-cover" />
                      ) : (
                        <FileTypeIcon mime={file.mime_type} className="h-4 w-4 text-slate-400" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-[12.5px] font-medium text-slate-800 capitalize">{v.label}</div>
                      <div className="text-[10.5px] font-mono text-slate-400 truncate">{url}</div>
                    </div>
                    <a
                      href={url}
                      target="_blank"
                      rel="noopener noreferrer"
                      onClick={(e) => e.stopPropagation()}
                      className="px-2 h-7 rounded-md border border-slate-200 hover:border-slate-300 text-[11px] text-slate-600 flex items-center gap-1 cursor-pointer"
                    >
                      <ExternalLink className="h-3 w-3" /> {isOriginal ? "Open" : "View"}
                    </a>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* footer */}
        <div className="border-t border-slate-200 p-3 flex items-center gap-2 shrink-0 bg-white">
          <button
            type="button"
            onClick={() => onDelete(file)}
            className="px-3 h-9 rounded-lg text-red-600 hover:bg-red-50 text-[12.5px] font-medium flex items-center gap-1.5 cursor-pointer"
          >
            <Trash2 className="h-3.5 w-3.5" /> Delete
          </button>
          <div className="flex-1" />
          <button
            type="button"
            onClick={onClose}
            className="px-3 h-9 rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-[12.5px] font-medium cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => {
              const patch: { alt?: string; original_name?: string } = {};
              if (alt !== (file.alt || "")) patch.alt = alt;
              if (name !== file.original_name && name.trim()) patch.original_name = name;
              onSave(patch);
            }}
            disabled={!dirty || saving}
            className="px-3.5 h-9 rounded-lg bg-indigo-600 hover:bg-indigo-700 disabled:bg-slate-200 disabled:text-slate-400 disabled:cursor-not-allowed text-white text-[12.5px] font-medium shadow-sm transition-colors flex items-center gap-1.5 cursor-pointer"
          >
            {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
            Save changes
          </button>
        </div>
      </div>
    </div>
  );
}
