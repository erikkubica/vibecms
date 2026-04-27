import React, { useRef, useState } from "react";
import { GripVertical, Copy, Trash2 } from "@vibecms/icons";
import { typeLabelMap } from "./types";
import { normalizeOptions } from "./key-utils";
import FieldEditor from "./FieldEditor";

const { AccordionRow, Chip, Button } = (window as any).__VIBECMS_SHARED__.ui;

interface FieldRowProps {
  field: any;
  index: number;
  isEditing: boolean;
  isFirst: boolean;
  isLast: boolean;
  onToggleEdit: () => void;
  onRemove: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onReorder: (from: number, to: number) => void;
  onDuplicate: () => void;
  updateField: (updates: Record<string, any>) => void;
  allFields?: any[];
}

export default function FieldRow({
  field,
  index,
  isEditing,
  isFirst,
  isLast,
  onToggleEdit,
  onRemove,
  onMoveUp,
  onMoveDown,
  onReorder,
  onDuplicate,
  updateField,
  allFields = [],
}: FieldRowProps) {
  const dragRef = useRef<HTMLDivElement>(null);
  const [isDragOver, setIsDragOver] = useState(false);

  const normalizedOpts = normalizeOptions(field.options);
  const optCount = normalizedOpts.length;
  const hasOptions =
    (field.type === "select" || field.type === "radio") && optCount > 0;

  const handleDragStart = (e: React.DragEvent) => {
    e.dataTransfer.setData("text/plain", String(index));
    e.dataTransfer.effectAllowed = "move";
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setIsDragOver(true);
  };

  const handleDragLeave = () => setIsDragOver(false);

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    const from = parseInt(e.dataTransfer.getData("text/plain"), 10);
    if (!isNaN(from) && from !== index) onReorder(from, index);
  };

  const handleDragEnd = () => setIsDragOver(false);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowUp") { e.preventDefault(); onMoveUp(); }
    if (e.key === "ArrowDown") { e.preventDefault(); onMoveDown(); }
  };

  const headerLeft = (
    <div
      ref={dragRef}
      draggable
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      onDragEnd={handleDragEnd}
      className={`flex items-center gap-2 flex-1 min-w-0 ${isDragOver ? "opacity-50" : ""}`}
    >
      <button
        type="button"
        className="cursor-grab active:cursor-grabbing text-slate-400 hover:text-slate-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400 rounded flex-shrink-0 border-0 bg-transparent p-0"
        aria-label="Drag to reorder"
        tabIndex={0}
        onKeyDown={handleKeyDown}
        disabled={isFirst && isLast}
        onClick={(e) => e.stopPropagation()}
      >
        <GripVertical className="h-4 w-4" />
      </button>
      <span className="text-[13px] font-medium text-slate-800 truncate">
        {field.label || "Untitled Field"}
      </span>
      <span className="text-[11px] text-slate-400 font-mono shrink-0">{field.id}</span>
      <Chip>{typeLabelMap[field.type] || field.type}</Chip>
      {field.required && (
        <span className="text-[9px] text-rose-500 font-bold shrink-0">REQ</span>
      )}
      {field.display_when &&
        (field.display_when.all?.length || field.display_when.any?.length) ? (
        <span className="text-[9px] font-semibold px-1 py-0.5 rounded bg-amber-100 text-amber-600 shrink-0">
          COND
        </span>
      ) : null}
    </div>
  );

  const headerRight = (
    <>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-slate-400 hover:text-slate-600"
        onClick={onDuplicate}
        title="Duplicate field"
      >
        <Copy className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-rose-400 hover:text-rose-600"
        onClick={onRemove}
        title="Remove field"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </>
  );

  return (
    <AccordionRow
      headerLeft={headerLeft}
      headerRight={headerRight}
      open={isEditing}
      onToggle={onToggleEdit}
    >
      <FieldEditor field={field} updateField={updateField} allFields={allFields} />
    </AccordionRow>
  );
}
