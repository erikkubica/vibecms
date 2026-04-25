import { Copy, Download, Trash2, AlertTriangle } from "@vibecms/icons";
import {
  MediaFile,
  isImage,
  humanFileSize,
  imageSize,
  fmtDate,
  MediaImage,
  FileTypeIcon,
} from "./helpers";
import SelectCheck from "./SelectCheck";

interface MediaListViewProps {
  files: MediaFile[];
  selected: Set<number>;
  onOpen: (f: MediaFile) => void;
  onToggle: (id: number, e: React.MouseEvent) => void;
  onToggleAll: () => void;
  onCopy: (f: MediaFile) => void;
  onDownload: (f: MediaFile) => void;
  onDelete: (f: MediaFile) => void;
}

const COLS = "grid grid-cols-[40px_minmax(0,3fr)_minmax(0,2fr)_90px_110px_110px_100px] gap-3 px-3 py-2 items-center";

export default function MediaListView({
  files,
  selected,
  onOpen,
  onToggle,
  onToggleAll,
  onCopy,
  onDownload,
  onDelete,
}: MediaListViewProps) {
  const allSelected = files.length > 0 && files.every((f) => selected.has(f.id));
  return (
    <div className="rounded-xl border border-slate-200 bg-white overflow-hidden">
      <div className={`${COLS} text-[10.5px] font-semibold uppercase tracking-wide text-slate-500 border-b border-slate-200 bg-slate-50`}>
        <div className="flex items-center">
          <SelectCheck checked={allSelected} onClick={(e) => { e.stopPropagation(); onToggleAll(); }} size={16} />
        </div>
        <div>File</div>
        <div>Alt text</div>
        <div className="tabular-nums">Size</div>
        <div>Dimensions</div>
        <div>Uploaded</div>
        <div className="text-right pr-1">Actions</div>
      </div>
      <div className="divide-y divide-slate-100">
        {files.map((f) => (
          <div
            key={f.id}
            onClick={() => onOpen(f)}
            className={`${COLS} text-[12.5px] cursor-pointer transition-colors ${
              selected.has(f.id) ? "bg-indigo-50/60" : "hover:bg-slate-50"
            }`}
          >
            <div onClick={(e) => e.stopPropagation()}>
              <SelectCheck checked={selected.has(f.id)} onClick={(e) => onToggle(f.id, e)} size={16} />
            </div>
            <div className="flex items-center gap-2.5 min-w-0">
              <div className="w-9 h-9 rounded-md overflow-hidden shrink-0 border border-slate-200 bg-slate-50 grid place-items-center">
                {isImage(f.mime_type) ? (
                  <MediaImage
                    src={imageSize(f.url, "thumbnail", f.updated_at)}
                    alt={f.original_name}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <FileTypeIcon mime={f.mime_type} className="h-4 w-4 text-slate-400" />
                )}
              </div>
              <div className="min-w-0">
                <div className="truncate font-medium text-slate-800">{f.original_name}</div>
                <div className="text-[10.5px] text-slate-400 font-mono truncate">{f.url}</div>
              </div>
            </div>
            <div className="truncate text-slate-600">
              {f.alt || (
                <span className="text-amber-600 italic flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3" /> missing
                </span>
              )}
            </div>
            <div className="tabular-nums font-mono text-[11.5px] text-slate-600">{humanFileSize(f.size)}</div>
            <div className="tabular-nums font-mono text-[11.5px] text-slate-600">
              {f.width && f.height ? `${f.width}×${f.height}` : "—"}
            </div>
            <div className="text-slate-500">{fmtDate(f.created_at)}</div>
            <div className="flex items-center justify-end gap-0.5" onClick={(e) => e.stopPropagation()}>
              <button
                type="button"
                onClick={() => onCopy(f)}
                className="w-7 h-7 grid place-items-center rounded-md text-slate-500 hover:text-slate-900 hover:bg-slate-100 cursor-pointer"
                title="Copy URL"
              >
                <Copy className="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                onClick={() => onDownload(f)}
                className="w-7 h-7 grid place-items-center rounded-md text-slate-500 hover:text-slate-900 hover:bg-slate-100 cursor-pointer"
                title="Download"
              >
                <Download className="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                onClick={() => onDelete(f)}
                className="w-7 h-7 grid place-items-center rounded-md text-red-400 hover:text-red-600 hover:bg-red-50 cursor-pointer"
                title="Delete"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
