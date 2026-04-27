import React, { useState } from "react";
import { Plus, X } from "@vibecms/icons";
import { FIELD_TYPES, OptionPair } from "./types";
import { keyify } from "./key-utils";

const {
  Button,
  Card,
  CardContent,
  SectionHeader,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
} = (window as any).__VIBECMS_SHARED__.ui;

interface AddFieldFormProps {
  existingKeys: string[];
  onAdd: (field: Record<string, any>) => void;
  onCancel: () => void;
}

export default function AddFieldForm({
  existingKeys,
  onAdd,
  onCancel,
}: AddFieldFormProps) {
  const [newFieldLabel, setNewFieldLabel] = useState("");
  const [newFieldKey, setNewFieldKey] = useState("");
  const [newFieldType, setNewFieldType] = useState("text");
  const [newFieldPlaceholder, setNewFieldPlaceholder] = useState("");
  const [newFieldRequired, setNewFieldRequired] = useState(false);
  const [newFieldOptions, setNewFieldOptions] = useState<OptionPair[]>([]);
  const [autoFieldKey, setAutoFieldKey] = useState(true);

  const handleLabelChange = (val: string) => {
    setNewFieldLabel(val);
    if (autoFieldKey) setNewFieldKey(keyify(val));
  };

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

  const handleAddField = () => {
    if (!newFieldLabel.trim() || !newFieldKey.trim()) return;
    if (existingKeys.includes(newFieldKey)) return;

    const newField: Record<string, any> = {
      id: newFieldKey,
      type: newFieldType,
      label: newFieldLabel,
      placeholder: newFieldPlaceholder,
      required: newFieldType === "gdpr_consent" ? true : newFieldRequired,
    };

    if (newFieldType === "gdpr_consent") {
      newField.consent_text =
        "I agree to the Privacy Policy and consent to having my data stored.";
    }

    if (
      (newFieldType === "select" || newFieldType === "radio") &&
      newFieldOptions.length > 0
    ) {
      const validOptions = newFieldOptions.filter(
        (o: OptionPair) => o.label.trim() || o.value.trim(),
      );
      if (validOptions.length > 0) newField.options = validOptions;
    }

    if (newFieldType === "file") {
      newField.allowed_types = "";
      newField.max_size = 5;
      newField.multiple = false;
    }

    if (newFieldType === "number" || newFieldType === "range") {
      newField.step = 1;
    }

    onAdd(newField);
  };

  return (
    <Card className="rounded-xl border border-indigo-200 shadow-sm">
      <SectionHeader title="Add New Field" />
      <CardContent className="p-4 space-y-4">
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Field Label</Label>
          <Input
            placeholder="e.g. Your Email"
            value={newFieldLabel}
            onChange={(e: any) => handleLabelChange(e.target.value)}
          />
        </div>

        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">
            Field Key{" "}
            <span className="font-normal text-slate-400">(used in templates)</span>
          </Label>
          <Input
            placeholder="e.g. user_email"
            value={newFieldKey}
            onChange={(e: any) => {
              setNewFieldKey(e.target.value.replace(/[^a-z0-9_]/g, ""));
              setAutoFieldKey(false);
            }}
            className="font-mono"
          />
        </div>

        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Field Type</Label>
          <Select value={newFieldType} onValueChange={(val: string) => setNewFieldType(val)}>
            <SelectTrigger className="bg-white">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {FIELD_TYPES.map((t) => (
                <SelectItem key={t.value} value={t.value}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {(newFieldType === "select" || newFieldType === "radio") && (
          <div className="space-y-2">
            <Label className="text-xs font-medium text-slate-500">Options</Label>
            {newFieldOptions.length > 0 && (
              <div className="grid grid-cols-[1fr_1fr_auto] gap-1 text-[9px] text-slate-400 uppercase px-0.5">
                <span>Label</span>
                <span>Value</span>
                <span className="w-6" />
              </div>
            )}
            <div className="space-y-1.5">
              {newFieldOptions.map((opt: OptionPair, optIdx: number) => (
                <div key={optIdx} className="grid grid-cols-[1fr_1fr_auto] gap-1.5 items-center">
                  <Input
                    value={opt.label}
                    onChange={(e: any) => updateNewOptionLabel(optIdx, e.target.value)}
                    className="text-xs"
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
                    className="text-xs font-mono"
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
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-500">Placeholder</Label>
            <Input
              placeholder="Optional placeholder text"
              value={newFieldPlaceholder}
              onChange={(e: any) => setNewFieldPlaceholder(e.target.value)}
            />
          </div>
        )}

        {newFieldType === "gdpr_consent" ? (
          <div className="p-3 bg-amber-50 text-amber-800 rounded-lg text-xs border border-amber-100">
            <p className="font-semibold">GDPR Consent Field</p>
            <p className="opacity-80 mt-1">
              This field will be automatically marked as required. You can customize
              the consent text after adding the field.
            </p>
          </div>
        ) : (
          <div className="flex items-center justify-between">
            <Label className="text-xs font-medium text-slate-500">Required</Label>
            <Switch
              checked={newFieldRequired}
              onCheckedChange={(checked: boolean) => setNewFieldRequired(checked)}
            />
          </div>
        )}

        <div className="flex gap-2 pt-2">
          <Button size="sm" className="flex-1" onClick={handleAddField}>
            Add Field
          </Button>
          <Button size="sm" variant="outline" className="flex-1" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
