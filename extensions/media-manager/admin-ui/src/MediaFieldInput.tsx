import { useState, useRef, useCallback } from "react";
import {
  Image as ImageIcon,
  FileText,
  Film,
  Music,
  File,
  X,
  Plus,
  ImagePlus,
  Pencil,
  GripVertical,
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

// ---------- Gallery item with edit popup ----------

function GalleryItem({
  media,
  index,
  onRemove,
  onUpdate,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
  isDragTarget,
}: {
  media: MediaValue;
  index: number;
  onRemove: () => void;
  onUpdate: (updated: MediaValue) => void;
  onDragStart: (e: React.DragEvent, index: number) => void;
  onDragOver: (e: React.DragEvent, index: number) => void;
  onDrop: (e: React.DragEvent, index: number) => void;
  onDragEnd: () => void;
  isDragTarget: boolean;
}) {
  const [editing, setEditing] = useState(false);
  const [editAlt, setEditAlt] = useState(media.alt || "");
  const popupRef = useRef<HTMLDivElement>(null);

  function handleSaveEdit() {
    onUpdate({ ...media, alt: editAlt });
    setEditing(false);
  }

  function handleOpenEdit(e: React.MouseEvent) {
    e.stopPropagation();
    setEditAlt(media.alt || "");
    setEditing(true);
  }

  return (
    <div
      className={`group relative aspect-square rounded-lg border-2 overflow-hidden bg-white cursor-grab active:cursor-grabbing transition-all ${
        isDragTarget
          ? "border-indigo-400 scale-[1.02] shadow-md"
          : "border-slate-200 hover:border-slate-300"
      }`}
      draggable
      onDragStart={(e) => onDragStart(e, index)}
      onDragOver={(e) => onDragOver(e, index)}
      onDrop={(e) => onDrop(e, index)}
      onDragEnd={onDragEnd}
    >
      {isImage(media.mime_type) ? (
        <img
          src={media.url}
          alt={media.alt || ""}
          className="h-full w-full object-cover"
          loading="lazy"
          draggable={false}
        />
      ) : (
        <div className="h-full w-full flex flex-col items-center justify-center p-2">
          <FileIcon mime={media.mime_type} className="h-8 w-8 text-slate-400" />
          <span className="text-[10px] text-slate-500 mt-1 truncate w-full text-center">
            {media.filename || "File"}
          </span>
        </div>
      )}

      {/* Drag handle indicator */}
      <div className="absolute top-1 left-1 opacity-0 group-hover:opacity-100 transition-opacity">
        <div className="bg-black/50 rounded p-0.5">
          <GripVertical className="h-3 w-3 text-white" />
        </div>
      </div>

      {/* Action buttons overlay */}
      <div className="absolute top-1 right-1 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          type="button"
          className="bg-black/60 hover:bg-indigo-600 text-white rounded-full p-1 transition-colors"
          onClick={handleOpenEdit}
          title="Edit details"
        >
          <Pencil className="h-3 w-3" />
        </button>
        <button
          type="button"
          className="bg-black/60 hover:bg-red-600 text-white rounded-full p-1 transition-colors"
          onClick={(e) => { e.stopPropagation(); onRemove(); }}
          title="Remove"
        >
          <X className="h-3 w-3" />
        </button>
      </div>

      {/* Alt text indicator */}
      {media.alt && (
        <div className="absolute bottom-0 inset-x-0 bg-black/50 px-1.5 py-0.5">
          <p className="text-[9px] text-white truncate">{media.alt}</p>
        </div>
      )}

      {/* Edit popup */}
      {editing && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setEditing(false)} />
          <div
            ref={popupRef}
            className="absolute z-50 bottom-full left-0 mb-1 w-56 bg-white rounded-lg border border-slate-200 shadow-xl p-3 space-y-2"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="text-xs font-semibold text-slate-700">Image Settings</p>
            <div className="space-y-1">
              <label className="text-[11px] font-medium text-slate-500">Alt Text</label>
              <Input
                placeholder="Describe this image..."
                value={editAlt}
                onChange={(e) => setEditAlt(e.target.value)}
                className="h-7 text-xs rounded-md border-slate-300"
                autoFocus
                onKeyDown={(e) => { if (e.key === "Enter") handleSaveEdit(); }}
              />
            </div>
            {media.filename && (
              <p className="text-[10px] text-slate-400 truncate">
                {media.filename}
              </p>
            )}
            {media.width && media.height && (
              <p className="text-[10px] text-slate-400">
                {media.width} × {media.height}px
              </p>
            )}
            <div className="flex gap-1.5 pt-1">
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="flex-1 h-6 text-[11px] rounded-md"
                onClick={() => setEditing(false)}
              >
                Cancel
              </Button>
              <Button
                type="button"
                size="sm"
                className="flex-1 h-6 text-[11px] rounded-md bg-indigo-600 hover:bg-indigo-700 text-white"
                onClick={handleSaveEdit}
              >
                Done
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

// ---------- Main component ----------

export default function MediaFieldInput({ field, value, onChange }: MediaFieldInputProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const multi = isMultiMode(field);
  const mimeFilter = getMimeFilter(field);

  // Drag state for gallery reordering
  const dragIndexRef = useRef<number | null>(null);
  const [dragTargetIndex, setDragTargetIndex] = useState<number | null>(null);

  const handleDragStart = useCallback((_e: React.DragEvent, index: number) => {
    dragIndexRef.current = index;
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent, index: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDragTargetIndex(index);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent, dropIndex: number) => {
      e.preventDefault();
      const dragIndex = dragIndexRef.current;
      if (dragIndex === null || dragIndex === dropIndex) {
        setDragTargetIndex(null);
        return;
      }
      const items = normalizeToMediaValues(value);
      const next = [...items];
      const [dragged] = next.splice(dragIndex, 1);
      next.splice(dropIndex, 0, dragged);
      onChange(next);
      setDragTargetIndex(null);
      dragIndexRef.current = null;
    },
    [value, onChange],
  );

  const handleDragEnd = useCallback(() => {
    setDragTargetIndex(null);
    dragIndexRef.current = null;
  }, []);

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

    function handleUpdateItem(index: number, updated: MediaValue) {
      const next = [...items];
      next[index] = updated;
      onChange(next);
    }

    return (
      <div className="space-y-3">
        {items.length > 0 && (
          <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-2">
            {items.map((item, i) => (
              <GalleryItem
                key={item.id || item.url}
                media={item}
                index={i}
                onRemove={() => handleRemove(i)}
                onUpdate={(updated) => handleUpdateItem(i, updated)}
                onDragStart={handleDragStart}
                onDragOver={handleDragOver}
                onDrop={handleDrop}
                onDragEnd={handleDragEnd}
                isDragTarget={dragTargetIndex === i}
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
