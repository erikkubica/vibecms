import React, { useState } from "react";
import { Plus, ChevronUp, ChevronDown, Pencil, X } from "@vibecms/icons";

const {
  Button,
  Card,
  CardContent,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
  Textarea,
} = (window as any).__VIBECMS_SHARED__.ui;

interface OptionPair {
  label: string;
  value: string;
}

function keyify(str: string): string {
  return str
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .replace(/_+/g, "_");
}

function normalizeOptions(options: any): OptionPair[] {
  if (!options) return [];
  if (!Array.isArray(options)) return [];
  if (options.length === 0) return [];
  if (
    typeof options[0] === "object" &&
    options[0] !== null &&
    "label" in options[0]
  ) {
    return options;
  }
  return options
    .filter((o: any) => typeof o === "string" && o.trim())
    .map((o: string) => ({ label: o.trim(), value: o.trim() }));
}

const allTypeOptions: { value: string; label: string }[] = [
  { value: "text", label: "Text" },
  { value: "email", label: "Email" },
  { value: "tel", label: "Phone" },
  { value: "url", label: "URL" },
  { value: "number", label: "Number" },
  { value: "range", label: "Range" },
  { value: "textarea", label: "Textarea" },
  { value: "select", label: "Select" },
  { value: "checkbox", label: "Checkbox" },
  { value: "radio", label: "Radio" },
  { value: "date", label: "Date" },
  { value: "file", label: "File Upload" },
  { value: "hidden", label: "Hidden" },
  { value: "gdpr_consent", label: "GDPR Consent" },
];

const typeLabelMap: Record<string, string> = {};
allTypeOptions.forEach((t) => {
  typeLabelMap[t.value] = t.label;
});

export default function BuilderTab({ form, setForm }: any) {
  const [editingFieldIndex, setEditingFieldIndex] = useState<number | null>(
    null,
  );
  const [addingField, setAddingField] = useState(false);
  const [newFieldLabel, setNewFieldLabel] = useState("");
  const [newFieldKey, setNewFieldKey] = useState("");
  const [newFieldType, setNewFieldType] = useState("text");
  const [newFieldPlaceholder, setNewFieldPlaceholder] = useState("");
  const [newFieldRequired, setNewFieldRequired] = useState(false);
  const [newFieldOptions, setNewFieldOptions] = useState<OptionPair[]>([]);
  const [autoFieldKey, setAutoFieldKey] = useState(true);

  const [manuallyEditedKeys, setManuallyEditedKeys] = useState<Set<number>>(
    new Set(),
  );

  const fields = form.fields || [];

  const updateField = (index: number, updates: Record<string, any>) => {
    setForm((prev: any) => {
      const newFields = [...prev.fields];
      newFields[index] = { ...newFields[index], ...updates };
      return { ...prev, fields: newFields };
    });
  };

  const handleLabelChange = (index: number, label: string) => {
    const updates: Record<string, any> = { label };
    if (!manuallyEditedKeys.has(index)) {
      updates.id = keyify(label);
    }
    updateField(index, updates);
  };

  const handleKeyChange = (index: number, key: string) => {
    const sanitized = key.replace(/[^a-z0-9_]/g, "");
    setManuallyEditedKeys((prev) => {
      const next = new Set(prev);
      next.add(index);
      return next;
    });
    updateField(index, { id: sanitized });
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
    setManuallyEditedKeys((prev) => {
      const next = new Set<number>();
      prev.forEach((k) => {
        if (k < index) next.add(k);
        else if (k > index) next.add(k - 1);
      });
      return next;
    });
  };

  const handleMoveField = (index: number, direction: "up" | "down") => {
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= fields.length) return;
    setForm((prev: any) => {
      const newFields = [...prev.fields];
      [newFields[index], newFields[targetIndex]] = [
        newFields[targetIndex],
        newFields[index],
      ];
      return { ...prev, fields: newFields };
    });
    if (editingFieldIndex === index) setEditingFieldIndex(targetIndex);
    else if (editingFieldIndex === targetIndex) setEditingFieldIndex(index);
    setManuallyEditedKeys((prev) => {
      const next = new Set<number>();
      prev.forEach((k) => {
        if (k === index) next.add(targetIndex);
        else if (k === targetIndex) next.add(index);
        else next.add(k);
      });
      return next;
    });
  };

  const handleNewFieldLabelChange = (val: string) => {
    setNewFieldLabel(val);
    if (autoFieldKey) {
      setNewFieldKey(keyify(val));
    }
  };

  const resetAddFieldForm = () => {
    setNewFieldLabel("");
    setNewFieldKey("");
    setNewFieldType("text");
    setNewFieldPlaceholder("");
    setNewFieldRequired(false);
    setNewFieldOptions([]);
    setAutoFieldKey(true);
    setAddingField(false);
  };

  const handleAddField = () => {
    if (!newFieldLabel.trim() || !newFieldKey.trim()) return;
    if (fields.some((f: any) => f.id === newFieldKey)) return;

    const actualType = newFieldType;
    const newField: Record<string, any> = {
      id: newFieldKey,
      type: actualType,
      label: newFieldLabel,
      placeholder: newFieldPlaceholder,
      required: actualType === "gdpr_consent" ? true : newFieldRequired,
    };

    if (actualType === "gdpr_consent") {
      newField.consent_text =
        "I agree to the Privacy Policy and consent to having my data stored.";
    }

    if (
      (actualType === "select" || actualType === "radio") &&
      newFieldOptions.length > 0
    ) {
      const validOptions = newFieldOptions.filter(
        (o: OptionPair) => o.label.trim() || o.value.trim(),
      );
      if (validOptions.length > 0) {
        newField.options = validOptions;
      }
    }

    if (actualType === "file") {
      newField.allowed_types = "";
      newField.max_size = 5;
      newField.multiple = false;
    }

    if (actualType === "number" || actualType === "range") {
      newField.step = 1;
    }

    setForm((prev: any) => ({
      ...prev,
      fields: [...prev.fields, newField],
    }));
    resetAddFieldForm();
  };

  // --- Option row helpers ---

  const updateOptionLabel = (
    fieldIndex: number,
    optIdx: number,
    label: string,
  ) => {
    const opts = normalizeOptions(fields[fieldIndex].options);
    const expectedAuto = keyify(opts[optIdx]?.label || "");
    const newVal =
      opts[optIdx]?.value === expectedAuto || !opts[optIdx]?.value
        ? keyify(label)
        : opts[optIdx]?.value;
    const updated = [...opts];
    updated[optIdx] = { label, value: newVal };
    updateField(fieldIndex, { options: updated });
  };

  const updateOptionValue = (
    fieldIndex: number,
    optIdx: number,
    value: string,
  ) => {
    const opts = normalizeOptions(fields[fieldIndex].options);
    const updated = [...opts];
    updated[optIdx] = { label: opts[optIdx].label, value };
    updateField(fieldIndex, { options: updated });
  };

  const removeOption = (fieldIndex: number, optIdx: number) => {
    const opts = normalizeOptions(fields[fieldIndex].options);
    updateField(fieldIndex, { options: opts.filter((_, i) => i !== optIdx) });
  };

  const addOption = (fieldIndex: number) => {
    const opts = normalizeOptions(fields[fieldIndex].options);
    updateField(fieldIndex, {
      options: [...opts, { label: "", value: "" }],
    });
  };

  // --- New field option helpers ---

  const updateNewOptionLabel = (optIdx: number, label: string) => {
    const opts = [...newFieldOptions];
    const expectedAuto = keyify(opts[optIdx]?.label || "");
    const newVal =
      opts[optIdx]?.value === expectedAuto || !opts[optIdx]?.value
        ? keyify(label)
        : opts[optIdx]?.value;
    opts[optIdx] = { label, value: newVal };
    setNewFieldOptions(opts);
  };

  const updateNewOptionValue = (optIdx: number, value: string) => {
    const opts = [...newFieldOptions];
    opts[optIdx] = { label: opts[optIdx].label, value };
    setNewFieldOptions(opts);
  };

  const removeNewOption = (optIdx: number) => {
    setNewFieldOptions(newFieldOptions.filter((_, i) => i !== optIdx));
  };

  const addNewOption = () => {
    setNewFieldOptions([...newFieldOptions, { label: "", value: "" }]);
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
      <div className="md:col-span-2 space-y-3">
        {/* Field list */}
        {fields.length === 0 && !addingField && (
          <div className="text-center py-8 rounded-lg border border-dashed border-slate-200 text-slate-400 text-sm italic">
            No fields defined yet. Add your first field below.
          </div>
        )}

        {fields.map((field: any, index: number) => {
          const normalizedOpts = normalizeOptions(field.options);
          const optCount = normalizedOpts.length;
          const isGDPR = field.type === "gdpr_consent";
          const hasOptions =
            (field.type === "select" || field.type === "radio") && optCount > 0;

          return (
            <div
              key={index}
              className={`rounded-lg border ${
                editingFieldIndex === index
                  ? "border-indigo-300 bg-indigo-50/30"
                  : "border-slate-200 bg-white"
              }`}
            >
              {/* Collapsed header */}
              <div className="flex items-center gap-2 p-2 px-3">
                <div className="flex flex-col gap-0.5">
                  <button
                    type="button"
                    onClick={() => handleMoveField(index, "up")}
                    disabled={index === 0}
                    className="text-slate-400 hover:text-slate-600 disabled:opacity-30"
                  >
                    <ChevronUp className="h-3.5 w-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => handleMoveField(index, "down")}
                    disabled={index === fields.length - 1}
                    className="text-slate-400 hover:text-slate-600 disabled:opacity-30"
                  >
                    <ChevronDown className="h-3.5 w-3.5" />
                  </button>
                </div>
                <div
                  className="flex-1 min-w-0 cursor-pointer"
                  onClick={() =>
                    setEditingFieldIndex(
                      editingFieldIndex === index ? null : index,
                    )
                  }
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-slate-800">
                      {field.label || "Untitled Field"}
                    </span>
                    <span className="text-[10px] text-slate-400 font-mono">
                      {field.id}
                    </span>
                    {field.required && (
                      <span className="text-[9px] text-red-400 font-semibold">
                        REQ
                      </span>
                    )}
                  </div>
                  <div className="text-[10px] text-slate-500 font-medium uppercase">
                    {typeLabelMap[field.type] || field.type}
                    {hasOptions &&
                      ` · ${optCount} option${optCount !== 1 ? "s" : ""}`}
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-slate-400 hover:text-indigo-600"
                  onClick={() =>
                    setEditingFieldIndex(
                      editingFieldIndex === index ? null : index,
                    )
                  }
                >
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-red-400 hover:text-red-600"
                  onClick={() => handleRemoveField(index)}
                >
                  <X className="h-3.5 w-3.5" />
                </Button>
              </div>

              {/* Expanded inline editor */}
              {editingFieldIndex === index && (
                <div className="border-t border-indigo-100 p-3 space-y-3 bg-white rounded-b-lg">
                  {/* Row 1: Label + Key */}
                  <div className="grid gap-2 sm:grid-cols-2">
                    <div className="space-y-1">
                      <Label className="text-[10px] text-slate-500 uppercase">
                        Label
                      </Label>
                      <Input
                        value={field.label}
                        onChange={(e: any) =>
                          handleLabelChange(index, e.target.value)
                        }
                        className="h-8 text-sm"
                        placeholder="e.g. Your Email"
                      />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-[10px] text-slate-500 uppercase">
                        Key
                      </Label>
                      <Input
                        value={field.id}
                        onChange={(e: any) =>
                          handleKeyChange(index, e.target.value)
                        }
                        className="h-8 text-sm font-mono"
                        placeholder="e.g. user_email"
                      />
                    </div>
                  </div>

                  {/* Row 2: Type + Placeholder */}
                  <div className="grid gap-2 sm:grid-cols-2">
                    <div className="space-y-1">
                      <Label className="text-[10px] text-slate-500 uppercase">
                        Type
                      </Label>
                      <Select
                        value={field.type}
                        onValueChange={(val: string) => {
                          const updates: Record<string, any> = { type: val };
                          if (val === "gdpr_consent") {
                            updates.required = true;
                            if (!field.consent_text) {
                              updates.consent_text =
                                "I agree to the Privacy Policy and consent to having my data stored.";
                            }
                          }
                          if (val === "file") {
                            if (!field.allowed_types)
                              updates.allowed_types = "";
                            if (!field.max_size) updates.max_size = 5;
                            if (field.multiple === undefined)
                              updates.multiple = false;
                          }
                          if (
                            (val === "number" || val === "range") &&
                            field.step === undefined
                          ) {
                            updates.step = 1;
                          }
                          updateField(index, updates);
                        }}
                      >
                        <SelectTrigger className="h-8 text-sm bg-white">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {allTypeOptions.map((t) => (
                            <SelectItem key={t.value} value={t.value}>
                              {t.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    {!isGDPR && field.type !== "hidden" && (
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Placeholder
                        </Label>
                        <Input
                          value={field.placeholder || ""}
                          onChange={(e: any) =>
                            updateField(index, {
                              placeholder: e.target.value,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="Optional placeholder text"
                        />
                      </div>
                    )}
                  </div>

                  {/* Row 3: Default Value + Help Text */}
                  {field.type !== "hidden" && !isGDPR && (
                    <div className="grid gap-2 sm:grid-cols-2">
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Default Value
                        </Label>
                        <Input
                          value={field.default_value || ""}
                          onChange={(e: any) =>
                            updateField(index, {
                              default_value: e.target.value,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="Optional default value"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Help Text
                        </Label>
                        <Input
                          value={field.help || ""}
                          onChange={(e: any) =>
                            updateField(index, { help: e.target.value })
                          }
                          className="h-8 text-sm"
                          placeholder="Shown below the field"
                        />
                      </div>
                    </div>
                  )}

                  {/* Hidden: only default value */}
                  {field.type === "hidden" && (
                    <div className="grid gap-2 sm:grid-cols-2">
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Default Value
                        </Label>
                        <Input
                          value={field.default_value || ""}
                          onChange={(e: any) =>
                            updateField(index, {
                              default_value: e.target.value,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="Hidden field value"
                        />
                      </div>
                    </div>
                  )}

                  {/* Row 4a: Text validation (text, textarea, url, email, tel) */}
                  {["text", "textarea", "url", "email", "tel"].includes(
                    field.type,
                  ) && (
                    <div className="grid gap-2 sm:grid-cols-2">
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Min Length
                        </Label>
                        <Input
                          type="number"
                          min={0}
                          value={field.min_length ?? ""}
                          onChange={(e: any) =>
                            updateField(index, {
                              min_length: e.target.value
                                ? Number(e.target.value)
                                : undefined,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="No minimum"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Max Length
                        </Label>
                        <Input
                          type="number"
                          min={0}
                          value={field.max_length ?? ""}
                          onChange={(e: any) =>
                            updateField(index, {
                              max_length: e.target.value
                                ? Number(e.target.value)
                                : undefined,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="No maximum"
                        />
                      </div>
                    </div>
                  )}

                  {/* Row 4b: Number/Range validation */}
                  {(field.type === "number" || field.type === "range") && (
                    <div className="grid gap-2 sm:grid-cols-3">
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Min
                        </Label>
                        <Input
                          type="number"
                          value={field.min ?? ""}
                          onChange={(e: any) =>
                            updateField(index, {
                              min: e.target.value
                                ? Number(e.target.value)
                                : undefined,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="No min"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Max
                        </Label>
                        <Input
                          type="number"
                          value={field.max ?? ""}
                          onChange={(e: any) =>
                            updateField(index, {
                              max: e.target.value
                                ? Number(e.target.value)
                                : undefined,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="No max"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-[10px] text-slate-500 uppercase">
                          Step
                        </Label>
                        <Input
                          type="number"
                          min={0.01}
                          step={0.01}
                          value={field.step ?? 1}
                          onChange={(e: any) =>
                            updateField(index, {
                              step: e.target.value ? Number(e.target.value) : 1,
                            })
                          }
                          className="h-8 text-sm"
                          placeholder="1"
                        />
                      </div>
                    </div>
                  )}

                  {/* Row 4c: File upload options */}
                  {field.type === "file" && (
                    <>
                      <div className="grid gap-2 sm:grid-cols-2">
                        <div className="space-y-1">
                          <Label className="text-[10px] text-slate-500 uppercase">
                            Allowed Types
                          </Label>
                          <Input
                            value={field.allowed_types || ""}
                            onChange={(e: any) =>
                              updateField(index, {
                                allowed_types: e.target.value,
                              })
                            }
                            className="h-8 text-sm"
                            placeholder="pdf, doc, jpg, png"
                          />
                          <p className="text-[9px] text-slate-400">
                            Comma-separated file extensions
                          </p>
                        </div>
                        <div className="space-y-1">
                          <Label className="text-[10px] text-slate-500 uppercase">
                            Max Size (MB)
                          </Label>
                          <Input
                            type="number"
                            min={0.1}
                            step={0.1}
                            value={field.max_size ?? 5}
                            onChange={(e: any) =>
                              updateField(index, {
                                max_size: e.target.value
                                  ? Number(e.target.value)
                                  : 5,
                              })
                            }
                            className="h-8 text-sm"
                            placeholder="5"
                          />
                        </div>
                      </div>
                      <div className="flex items-center justify-between pt-1">
                        <div>
                          <Label className="text-[10px] text-slate-500 uppercase">
                            Allow Multiple Files
                          </Label>
                        </div>
                        <Switch
                          checked={!!field.multiple}
                          onCheckedChange={(checked: boolean) =>
                            updateField(index, { multiple: checked })
                          }
                        />
                      </div>
                    </>
                  )}

                  {/* Row 4d: GDPR consent text */}
                  {isGDPR && (
                    <div className="space-y-1">
                      <Label className="text-[10px] text-slate-500 uppercase">
                        Consent Text
                      </Label>
                      <Textarea
                        value={
                          field.consent_text ||
                          "I agree to the Privacy Policy and consent to having my data stored."
                        }
                        onChange={(e: any) =>
                          updateField(index, {
                            consent_text: e.target.value,
                          })
                        }
                        className="min-h-[60px] text-sm"
                        rows={2}
                        placeholder="I agree to the Privacy Policy..."
                      />
                      <p className="text-[9px] text-slate-400">
                        Shown next to the consent checkbox. Include a link to
                        your privacy policy.
                      </p>
                    </div>
                  )}

                  {/* Row 5: Options for select/radio */}
                  {(field.type === "select" || field.type === "radio") && (
                    <div className="space-y-2">
                      <Label className="text-[10px] text-slate-500 uppercase">
                        Options
                      </Label>
                      {normalizedOpts.length > 0 && (
                        <div className="grid grid-cols-[1fr_1fr_auto] gap-1 text-[9px] text-slate-400 uppercase px-0.5">
                          <span>Label</span>
                          <span>Value</span>
                          <span className="w-6" />
                        </div>
                      )}
                      <div className="space-y-1.5">
                        {normalizedOpts.map(
                          (opt: OptionPair, optIdx: number) => (
                            <div
                              key={optIdx}
                              className="grid grid-cols-[1fr_1fr_auto] gap-1.5 items-center"
                            >
                              <Input
                                value={opt.label}
                                onChange={(e: any) =>
                                  updateOptionLabel(
                                    index,
                                    optIdx,
                                    e.target.value,
                                  )
                                }
                                className="h-7 text-xs"
                                placeholder="Label"
                              />
                              <Input
                                value={opt.value}
                                onChange={(e: any) =>
                                  updateOptionValue(
                                    index,
                                    optIdx,
                                    e.target.value.replace(/[^a-z0-9_]/gi, ""),
                                  )
                                }
                                className="h-7 text-xs font-mono"
                                placeholder="value"
                              />
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-slate-300 hover:text-red-500 shrink-0"
                                onClick={() => removeOption(index, optIdx)}
                              >
                                <X className="h-3 w-3" />
                              </Button>
                            </div>
                          ),
                        )}
                      </div>
                      <button
                        type="button"
                        onClick={() => addOption(index)}
                        className="text-[11px] text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1"
                      >
                        <Plus className="h-3 w-3" /> Add option
                      </button>
                    </div>
                  )}

                  {/* Row 6: Required toggle */}
                  <div className="flex items-center justify-between pt-1">
                    <div>
                      <Label className="text-[10px] text-slate-500 uppercase">
                        Required
                      </Label>
                      {isGDPR && (
                        <p className="text-[9px] text-slate-400">
                          GDPR consent fields are always required.
                        </p>
                      )}
                    </div>
                    <Switch
                      checked={!!field.required}
                      onCheckedChange={(checked: boolean) =>
                        updateField(index, { required: checked })
                      }
                      disabled={isGDPR}
                    />
                  </div>
                </div>
              )}
            </div>
          );
        })}

        {/* Add field form */}
        {addingField ? (
          <div className="p-4 rounded-xl border border-indigo-200 bg-indigo-50/50 space-y-4">
            <div className="space-y-2">
              <Label className="text-xs font-semibold">Field Label</Label>
              <Input
                placeholder="e.g. Your Email"
                value={newFieldLabel}
                onChange={(e) => handleNewFieldLabelChange(e.target.value)}
                className="h-9"
              />
            </div>
            <div className="space-y-2">
              <Label className="text-xs font-semibold">
                Field Key{" "}
                <span className="font-normal text-slate-400">
                  (used in templates)
                </span>
              </Label>
              <Input
                placeholder="e.g. user_email"
                value={newFieldKey}
                onChange={(e) => {
                  setNewFieldKey(e.target.value.replace(/[^a-z0-9_]/g, ""));
                  setAutoFieldKey(false);
                }}
                className="h-9 font-mono"
              />
            </div>
            <div className="space-y-2">
              <Label className="text-xs font-semibold">Field Type</Label>
              <Select
                value={newFieldType}
                onValueChange={(val: string) => setNewFieldType(val)}
              >
                <SelectTrigger className="h-9 bg-white">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {allTypeOptions.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {(newFieldType === "select" || newFieldType === "radio") && (
              <div className="space-y-2">
                <Label className="text-xs font-semibold">Options</Label>
                {newFieldOptions.length > 0 && (
                  <div className="grid grid-cols-[1fr_1fr_auto] gap-1 text-[9px] text-slate-400 uppercase px-0.5">
                    <span>Label</span>
                    <span>Value</span>
                    <span className="w-6" />
                  </div>
                )}
                <div className="space-y-1.5">
                  {newFieldOptions.map((opt: OptionPair, optIdx: number) => (
                    <div
                      key={optIdx}
                      className="grid grid-cols-[1fr_1fr_auto] gap-1.5 items-center"
                    >
                      <Input
                        value={opt.label}
                        onChange={(e: any) =>
                          updateNewOptionLabel(optIdx, e.target.value)
                        }
                        className="h-9 text-xs"
                        placeholder="Label"
                      />
                      <Input
                        value={opt.value}
                        onChange={(e: any) =>
                          updateNewOptionValue(
                            optIdx,
                            e.target.value.replace(/[^a-z0-9_]/gi, ""),
                          )
                        }
                        className="h-9 text-xs font-mono"
                        placeholder="value"
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-9 w-9 text-slate-300 hover:text-red-500 shrink-0"
                        onClick={() => removeNewOption(optIdx)}
                      >
                        <X className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  ))}
                </div>
                <button
                  type="button"
                  onClick={addNewOption}
                  className="text-[11px] text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1"
                >
                  <Plus className="h-3 w-3" /> Add option
                </button>
              </div>
            )}
            {newFieldType !== "hidden" && newFieldType !== "gdpr_consent" && (
              <div className="space-y-2">
                <Label className="text-xs font-semibold">Placeholder</Label>
                <Input
                  placeholder="Optional placeholder text"
                  value={newFieldPlaceholder}
                  onChange={(e) => setNewFieldPlaceholder(e.target.value)}
                  className="h-9"
                />
              </div>
            )}
            {newFieldType === "gdpr_consent" ? (
              <div className="p-3 bg-amber-50 text-amber-800 rounded-lg text-xs border border-amber-100">
                <p className="font-semibold">GDPR Consent Field</p>
                <p className="opacity-80 mt-1">
                  This field will be automatically marked as required. You can
                  customize the consent text after adding the field.
                </p>
              </div>
            ) : (
              <div className="flex items-center justify-between">
                <Label className="text-xs font-semibold">Required</Label>
                <Switch
                  checked={newFieldRequired}
                  onCheckedChange={(checked: boolean) =>
                    setNewFieldRequired(checked)
                  }
                />
              </div>
            )}
            <div className="flex gap-2">
              <Button
                size="sm"
                className="flex-1 bg-indigo-600 hover:bg-indigo-700"
                onClick={handleAddField}
              >
                Add Field
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="flex-1"
                onClick={resetAddFieldForm}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <Button
            variant="outline"
            className="w-full rounded-lg border-dashed border-slate-300 text-slate-500 hover:text-indigo-600 hover:border-indigo-300 hover:bg-indigo-50/50"
            onClick={() => setAddingField(true)}
          >
            <Plus className="mr-2 h-4 w-4" /> Add Field
          </Button>
        )}
      </div>

      {/* Form Details sidebar */}
      <div className="space-y-4">
        <Card className="border-slate-200 shadow-none sticky top-6">
          <CardContent className="p-4 space-y-4">
            <h3 className="font-semibold text-slate-900">Form Details</h3>
            <div className="space-y-2">
              <Label>Form Name</Label>
              <Input
                value={form.name}
                onChange={(e: any) =>
                  setForm((prev: any) => ({ ...prev, name: e.target.value }))
                }
                placeholder="Contact Us"
              />
            </div>
            <div className="space-y-2">
              <Label>Form Slug</Label>
              <Input
                value={form.slug}
                onChange={(e: any) =>
                  setForm((prev: any) => ({
                    ...prev,
                    slug: e.target.value.replace(/\s+/g, "-").toLowerCase(),
                  }))
                }
                placeholder="contact-us"
              />
            </div>
            {fields.length > 0 && (
              <div className="pt-2 border-t border-slate-100">
                <p className="text-[10px] text-slate-400 uppercase tracking-wider mb-2">
                  Fields Summary
                </p>
                <div className="space-y-1">
                  {fields.map((f: any, i: number) => (
                    <div
                      key={i}
                      className="flex items-center justify-between text-xs"
                    >
                      <span className="text-slate-600 truncate">
                        {f.label || "Untitled"}
                      </span>
                      <span className="text-[9px] text-slate-400 font-mono ml-2 shrink-0">
                        {typeLabelMap[f.type] || f.type}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
