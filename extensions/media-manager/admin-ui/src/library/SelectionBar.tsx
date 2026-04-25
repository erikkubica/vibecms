import { Zap, Download, Trash2, X, Loader2 } from "@vibecms/icons";

interface SelectionBarProps {
  count: number;
  onClear: () => void;
  onDelete: () => void;
  onOptimize?: () => void;
  onDownload?: () => void;
  optimizing?: boolean;
}

export default function SelectionBar({ count, onClear, onDelete, onOptimize, onDownload, optimizing }: SelectionBarProps) {
  return (
    <div className="rounded-xl border border-indigo-200 bg-indigo-50/60 px-3 py-2 flex items-center gap-2">
      <div className="w-7 h-7 rounded-md bg-indigo-600 text-white grid place-items-center text-[11px] font-bold tabular-nums">
        {count}
      </div>
      <div className="text-[12.5px] text-indigo-900 font-medium">selected</div>
      <div className="flex-1" />
      {onOptimize && (
        <button
          type="button"
          onClick={onOptimize}
          disabled={optimizing}
          className="h-8 px-2.5 rounded-md hover:bg-white/70 text-indigo-800 text-[12px] font-medium flex items-center gap-1.5 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {optimizing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Zap className="h-3 w-3" />}
          Optimize
        </button>
      )}
      {onDownload && (
        <button
          type="button"
          onClick={onDownload}
          className="h-8 px-2.5 rounded-md hover:bg-white/70 text-indigo-800 text-[12px] font-medium flex items-center gap-1.5 cursor-pointer"
        >
          <Download className="h-3 w-3" /> Download
        </button>
      )}
      <button
        type="button"
        onClick={onDelete}
        className="h-8 px-2.5 rounded-md hover:bg-rose-100 text-rose-700 text-[12px] font-medium flex items-center gap-1.5 cursor-pointer"
      >
        <Trash2 className="h-3 w-3" /> Delete
      </button>
      <div className="w-px h-5 bg-indigo-200 mx-1" />
      <button
        type="button"
        onClick={onClear}
        className="h-8 w-8 grid place-items-center rounded-md hover:bg-white/70 text-indigo-800 cursor-pointer"
        title="Clear selection"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
