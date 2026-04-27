import React from "react";
import FieldRow from "./FieldRow";

interface FieldListProps {
  fields: any[];
  editingFieldIndex: number | null;
  onToggleEdit: (index: number) => void;
  onRemove: (index: number) => void;
  onMoveUp: (index: number) => void;
  onMoveDown: (index: number) => void;
  onReorder: (from: number, to: number) => void;
  onDuplicate: (index: number) => void;
  updateField: (index: number, updates: Record<string, any>) => void;
}

export default function FieldList({
  fields,
  editingFieldIndex,
  onToggleEdit,
  onRemove,
  onMoveUp,
  onMoveDown,
  onReorder,
  onDuplicate,
  updateField,
}: FieldListProps) {
  if (fields.length === 0) {
    return null;
  }

  return (
    <>
      {fields.map((field: any, index: number) => (
        <FieldRow
          key={index}
          field={field}
          index={index}
          isEditing={editingFieldIndex === index}
          isFirst={index === 0}
          isLast={index === fields.length - 1}
          onToggleEdit={() => onToggleEdit(index)}
          onRemove={() => onRemove(index)}
          onMoveUp={() => onMoveUp(index)}
          onMoveDown={() => onMoveDown(index)}
          onReorder={onReorder}
          onDuplicate={() => onDuplicate(index)}
          updateField={(updates) => updateField(index, updates)}
          allFields={fields}
        />
      ))}
    </>
  );
}
