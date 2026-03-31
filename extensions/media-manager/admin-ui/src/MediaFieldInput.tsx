import { useState } from "react";
import {
  Image as ImageIcon,
  FileText,
  Film,
  Music,
  File,
  X,
  Plus,
  ImagePlus,
  ChevronUp,
  ChevronDown,
} from "@vibecms/icons";
import { Button, Input } from "@vibecms/ui";
import MediaPickerModal, { type MediaFile } from "./MediaPickerModal";

// ---------- Types ----------

interface MediaValue {
  url: string;
  id?: number;
  filename?: string;
  alt?: string;
  mime_type?: string;
  width?: number | null;
  height?: number | null;
}

interface FieldDef {
  key: string;
  label: string;
  type: string;
  required?: boolean;
  placeholder?: string;
  help_text?: string;
  multiple?: boolean;
  allowed_types?: string;
}

interface MediaFieldInputProps {
  field: FieldDef;
  value: unknown;
  onChange: (val: unknown) => void;
}

// ---------- Helpers ----------

function isImage(mime?: string): boolean {
  return !!mime && mime.startsWith("image/");
}

function FileIcon({ mime, className }: { mime?: string; className?: string }) {
  if (!mime) return <File className={className} />;
  if (isImage(mime)) return <ImageIcon className={className} />;
  if (mime.startsWith("video/")) return <Film className={className} />;
  if (mime.startsWith("audio/")) return <Music className={className} />;
  if (mime.includes("pdf") || mime.includes("document") || mime.includes("text"))
    return <FileText className={className} />;
  return <File className={className} />;
}

function mediaFileToValue(file: MediaFile): MediaValue {
  return {
    url: file.url,
    id: file.id,
    filename: file.original_name,
    alt: file.alt,
    mime_type: file.mime_type,
    width: file.width,
    height: file.height,
  };
}

function normalizeToMediaValue(val: unknown): MediaValue | null {
  if (!val) return null;
  if (typeof val === "string") {
    if (!val) return null;
    return { url: val };
  }
  if (typeof val === "object" && val !== null && "url" in val) {
    return val as MediaValue;
  }
  return null;
}

function normalizeToMediaValues(val: unknown): MediaValue[] {
  if (!val) return [];
  if (Array.isArray(val)) {
    return val
      .map((v) => {
        if (typeof v === "string" && v) return { url: v } as MediaValue;
        if (typeof v === "object" && v !== null && "url" in v) return v as MediaValue;
        return null;
      })
      .filter((v): v is MediaValue => v !== null);
  }
  const single = normalizeToMediaValue(val);
  return single ? [single] : [];
}

function getMimeFilter(field: FieldDef): string | undefined {
  if (field.type === "image") return "image";
  if (field.allowed_types) {
    if (field.allowed_types.startsWith("image")) return "image";
    if (field.allowed_types.startsWith("video")) return "video";
    if (field.allowed_types.startsWith("audio")) return "audio";
  }
  return undefined;
}

function isMultiMode(field: FieldDef): boolean {
  return field.type === "gallery" || !!field.multiple;
}

// ---------- Single media preview ----------

function SingleMediaPreview({
  media,
  onRemove,
  onAltChange,
}: {
  media: MediaValue;
  onRemove: () => void;
  onAltChange?: (alt: string) => void;
}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50 overflow-hidden">
      <div className="flex items-start gap-3 p-3">
        {/* Thumbnail */}
        {isImage(media.mime_type) ? (
          <div className="shrink-0 h-20 w-20 rounded-md overflow-hidden border border-slate-200 bg-white">
            <img
              src={media.url}
              alt={media.alt || ""}
              className="h-full w-full object-cover"
            />
          </div>
        ) : (
          <div className="shrink-0 h-20 w-20 rounded-md border border-slate-200 bg-white flex items-center justify-center">
            <FileIcon mime={media.mime_type} className="h-8 w-8 text-slate-400" />
          </div>
        )}

        {/* Info */}
        <div className="flex-1 min-w-0 space-y-2">
          <p className="text-sm font-medium text-slate-700 truncate">
            {media.filename || media.url.split("/").pop() || "File"}
          </p>
          {media.width && media.height && (
            <p className="text-xs text-slate-400">{media.width} x {media.height}px</p>
          )}
          {onAltChange && isImage(media.mime_type) && (
            <Input
              placeholder="Alt text..."
              value={media.alt || ""}
              onChange={(e) => onAltChange(e.target.value)}
              className="h-8 text-xs rounded-md border-slate-300"
            />
          )}
        </div>

        {/* Remove */}
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0 text-slate-400 hover:text-red-500"
          onClick={onRemove}
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

// ---------- Gallery item ----------

function GalleryItem({
  media,
  index,
  total,
  onRemove,
  onMove,
}: {
  media: MediaValue;
  index: number;
  total: number;
  onRemove: () => void;
  onMove: (dir: "up" | "down") => void;
}) {
  return (
    <div className="group relative aspect-square rounded-lg border border-slate-200 overflow-hidden bg-white">
      {isImage(media.mime_type) ? (
        <img src={media.url} alt={media.alt || ""} className="h-full w-full object-cover" loading="lazy" />
      ) : (
        <div className="h-full w-full flex flex-col items-center justify-center p-2">
          <FileIcon mime={media.mime_type} className="h-8 w-8 text-slate-400" />
          <span className="text-[10px] text-slate-500 mt-1 truncate w-full text-center">
            {media.filename || "File"}
          </span>
        </div>
      )}
      {/* Hover overlay */}
      <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-1">
        {index > 0 && (
          <Button type="button" variant="ghost" size="icon" className="h-7 w-7 text-white hover:bg-white/20" onClick={() => onMove("up")}>
            <ChevronUp className="h-4 w-4" />
          </Button>
        )}
        {index < total - 1 && (
          <Button type="button" variant="ghost" size="icon" className="h-7 w-7 text-white hover:bg-white/20" onClick={() => onMove("down")}>
            <ChevronDown className="h-4 w-4" />
          </Button>
        )}
        <Button type="button" variant="ghost" size="icon" className="h-7 w-7 text-white hover:bg-red-500/80" onClick={onRemove}>
          <X className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

// ---------- Main component ----------

export default function MediaFieldInput({ field, value, onChange }: MediaFieldInputProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const multi = isMultiMode(field);
  const mimeFilter = getMimeFilter(field);

  if (multi) {
    // Gallery / multi-file mode
    const items = normalizeToMediaValues(value);

    function handleSelect(files: MediaFile[]) {
      const newValues = files.map(mediaFileToValue);
      onChange([...items, ...newValues]);
    }

    function handleRemove(index: number) {
      onChange(items.filter((_, i) => i !== index));
    }

    function handleMove(index: number, dir: "up" | "down") {
      const target = dir === "up" ? index - 1 : index + 1;
      if (target < 0 || target >= items.length) return;
      const next = [...items];
      [next[index], next[target]] = [next[target], next[index]];
      onChange(next);
    }

    return (
      <div className="space-y-3">
        {items.length > 0 && (
          <div className="grid grid-cols-4 sm:grid-cols-5 md:grid-cols-6 gap-2">
            {items.map((item, i) => (
              <GalleryItem
                key={item.id || item.url}
                media={item}
                index={i}
                total={items.length}
                onRemove={() => handleRemove(i)}
                onMove={(dir) => handleMove(i, dir)}
              />
            ))}
          </div>
        )}
        <Button
          type="button"
          variant="outline"
          className="w-full rounded-lg border-dashed border-slate-300 text-slate-500 hover:border-indigo-400 hover:text-indigo-600"
          onClick={() => setPickerOpen(true)}
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Media
        </Button>
        <MediaPickerModal
          open={pickerOpen}
          onClose={() => setPickerOpen(false)}
          onSelect={handleSelect}
          multiple
          mimeFilter={mimeFilter}
        />
      </div>
    );
  }

  // Single mode (image / file / media)
  const current = normalizeToMediaValue(value);

  function handleSelect(files: MediaFile[]) {
    if (files.length > 0) {
      onChange(mediaFileToValue(files[0]));
    }
  }

  function handleAltChange(alt: string) {
    if (current) {
      onChange({ ...current, alt });
    }
  }

  return (
    <div className="space-y-2">
      {current ? (
        <SingleMediaPreview
          media={current}
          onRemove={() => onChange(null)}
          onAltChange={handleAltChange}
        />
      ) : (
        <button
          type="button"
          onClick={() => setPickerOpen(true)}
          className="w-full rounded-lg border-2 border-dashed border-slate-300 hover:border-indigo-400 bg-slate-50 hover:bg-indigo-50/50 transition-colors p-6 flex flex-col items-center gap-2"
        >
          <ImagePlus className="h-8 w-8 text-slate-400" />
          <span className="text-sm text-slate-500">
            Click to select {field.type === "file" ? "a file" : "an image"}
          </span>
        </button>
      )}

      {current && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="rounded-lg text-sm"
          onClick={() => setPickerOpen(true)}
        >
          <ImagePlus className="mr-2 h-4 w-4" />
          Replace
        </Button>
      )}

      <MediaPickerModal
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        onSelect={handleSelect}
        multiple={false}
        mimeFilter={mimeFilter}
      />
    </div>
  );
}
