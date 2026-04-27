import React, { useState } from "react";
import { Plus, ListPlus } from "@vibecms/icons";
import { typeLabelMap } from "./builder/types";
import FieldList from "./builder/FieldList";
import AddFieldForm from "./builder/AddFieldForm";

const { Button, Card, CardContent, SectionHeader, Chip, EmptyState } =
  (window as any).__VIBECMS_SHARED__.ui;

export default function BuilderTab({ form, setForm }: any) {
  const [editingFieldIndex, setEditingFieldIndex] = useState<number | null>(null);
  const [addingField, setAddingField] = useState(false);

  const fields = form.fields || [];

  const updateField = (index: number, updates: Record<string, any>) => {
    setForm((prev: any) => {
      const newFields = [...prev.fields];
      newFields[index] = { ...newFields[index], ...updates };
      return { ...prev, fields: newFields };
    });
  };

  const handleRemoveField = (index: number) => {
    setForm((prev: any) => ({
      ...prev,
      fields: prev.fields.filter((_: any, i: number) => i !== index),
    }));
    if (editingFieldIndex === index) setEditingFieldIndex(null);
    else if (editingFieldIndex !== null && editingFieldIndex > index) {
      setEditingFieldIndex(editingFieldIndex - 1);
    }
  };

  const handleMoveField = (index: number, direction: "up" | "down") => {
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= fields.length) return;
    handleReorderField(index, targetIndex);
  };

  const handleReorderField = (from: number, to: number) => {
    if (from < 0 || to < 0 || from >= fields.length || to >= fields.length) return;
    setForm((prev: any) => {
      const newFields = [...prev.fields];
      const [moved] = newFields.splice(from, 1);
      newFields.splice(to, 0, moved);
      return { ...prev, fields: newFields };
    });
    if (editingFieldIndex === from) setEditingFieldIndex(to);
    else if (editingFieldIndex !== null) {
      const lo = Math.min(from, to);
      const hi = Math.max(from, to);
      if (editingFieldIndex >= lo && editingFieldIndex <= hi) {
        const shift = from < to ? -1 : 1;
        setEditingFieldIndex(editingFieldIndex + shift);
      }
    }
  };

  const handleDuplicateField = (index: number) => {
    setForm((prev: any) => {
      const original = prev.fields[index];
      let newId = original.id + "_copy";
      let n = 2;
      while (prev.fields.some((f: any) => f.id === newId)) {
        newId = original.id + `_copy_${n++}`;
      }
      const copy = { ...original, id: newId, label: original.label + " (Copy)" };
      const newFields = [
        ...prev.fields.slice(0, index + 1),
        copy,
        ...prev.fields.slice(index + 1),
      ];
      return { ...prev, fields: newFields };
    });
  };

  const handleToggleEdit = (index: number) => {
    setEditingFieldIndex(editingFieldIndex === index ? null : index);
  };

  const handleAddField = (field: Record<string, any>) => {
    setForm((prev: any) => ({
      ...prev,
      fields: [...prev.fields, field],
    }));
    setAddingField(false);
  };

  const hasFields = fields.length > 0;

  return (
    <div
      className={
        hasFields
          ? "grid gap-5 md:grid-cols-[minmax(0,1fr)_240px]"
          : "grid gap-5"
      }
    >
      {/* Main field list — unwrapped, blocks-builder style */}
      <div className="min-w-0 space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-[12px] font-semibold uppercase tracking-wide" style={{ color: "var(--fg-muted)" }}>
            Fields ({fields.length})
          </h3>
        </div>

        {fields.length === 0 && !addingField && (
          <EmptyState
            icon={ListPlus}
            title="No fields yet"
            description="Add your first field below"
          />
        )}

        <FieldList
          fields={fields}
          editingFieldIndex={editingFieldIndex}
          onToggleEdit={handleToggleEdit}
          onRemove={handleRemoveField}
          onMoveUp={(index) => handleMoveField(index, "up")}
          onMoveDown={(index) => handleMoveField(index, "down")}
          onReorder={handleReorderField}
          onDuplicate={handleDuplicateField}
          updateField={updateField}
        />

        {addingField ? (
          <AddFieldForm
            existingKeys={fields.map((f: any) => f.id)}
            onAdd={handleAddField}
            onCancel={() => setAddingField(false)}
          />
        ) : (
          <Button
            variant="outline"
            className="w-full border-dashed border-slate-300 text-slate-500 hover:text-indigo-600 hover:border-indigo-300 hover:bg-indigo-50/50 cursor-pointer"
            onClick={() => setAddingField(true)}
          >
            <Plus className="mr-2 h-4 w-4" /> Add Field
          </Button>
        )}
      </div>

      {/* Summary sidebar */}
      {hasFields && (
        <div className="min-w-0">
          <Card className="rounded-xl border border-slate-200 shadow-sm sticky top-6">
            <SectionHeader title="Fields Summary" />
            <CardContent className="p-3 space-y-1.5">
              {fields.map((f: any, i: number) => (
                <div key={i} className="flex items-center justify-between text-[12px]">
                  <span className="text-slate-700 truncate">{f.label || "Untitled"}</span>
                  <Chip>{typeLabelMap[f.type] || f.type}</Chip>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
