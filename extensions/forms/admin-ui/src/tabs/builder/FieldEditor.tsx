import React, { useState } from "react";
import { FIELD_TYPES } from "./types";
import { normalizeOptions, keyify } from "./key-utils";
import FieldValidationRules from "./FieldValidationRules";
import FieldFileOptions from "./FieldFileOptions";
import FieldGDPROptions from "./FieldGDPROptions";
import OptionsEditor from "./OptionsEditor";
import ConditionBuilder, { ConditionGroup } from "./ConditionBuilder";

const {
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
  AccordionRow,
} = (window as any).__VIBECMS_SHARED__.ui;

interface FieldEditorProps {
  field: any;
  updateField: (updates: Record<string, any>) => void;
  allFields?: any[];
}

export default function FieldEditor({ field, updateField, allFields = [] }: FieldEditorProps) {
  const isGDPR = field.type === "gdpr_consent";
  const normalizedOpts = normalizeOptions(field.options);
  const [showConditions, setShowConditions] = useState(
    !!(field.display_when && (field.display_when.all?.length || field.display_when.any?.length)),
  );

  const displayWhenGroup: ConditionGroup = field.display_when || {};

  const updateOptionLabel = (optIdx: number, label: string) => {
    const opts = normalizeOptions(field.options);
    const expectedAuto = keyify(opts[optIdx]?.label || "");
    const newVal =
      opts[optIdx]?.value === expectedAuto || !opts[optIdx]?.value
        ? keyify(label)
        : opts[optIdx]?.value;
    const updated = [...opts];
    updated[optIdx] = { label, value: newVal };
    updateField({ options: updated });
  };

  const updateOptionValue = (optIdx: number, value: string) => {
    const opts = normalizeOptions(field.options);
    const updated = [...opts];
    updated[optIdx] = { label: opts[optIdx].label, value };
    updateField({ options: updated });
  };

  const removeOption = (optIdx: number) => {
    const opts = normalizeOptions(field.options);
    updateField({ options: opts.filter((_, i) => i !== optIdx) });
  };

  const addOption = () => {
    const opts = normalizeOptions(field.options);
    updateField({ options: [...opts, { label: "", value: "" }] });
  };

  return (
    <div className="space-y-4">
      {/* Basic fields */}
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label className="text-[10px] text-slate-500 uppercase">Label</Label>
          <Input
            value={field.label}
            onChange={(e: any) => updateField({ label: e.target.value })}
            className="h-8 text-sm"
            placeholder="e.g. Your Email"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-[10px] text-slate-500 uppercase">Key</Label>
          <Input
            value={field.id}
            onChange={(e: any) =>
              updateField({ id: e.target.value.replace(/[^a-z0-9_]/g, "") })
            }
            className="h-8 text-sm font-mono"
            placeholder="e.g. user_email"
          />
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label className="text-[10px] text-slate-500 uppercase">Type</Label>
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
                if (!field.allowed_types) updates.allowed_types = "";
                if (!field.max_size) updates.max_size = 5;
                if (field.multiple === undefined) updates.multiple = false;
              }
              if ((val === "number" || val === "range") && field.step === undefined) {
                updates.step = 1;
              }
              updateField(updates);
            }}
          >
            <SelectTrigger className="h-8 text-sm bg-white">
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
        {!isGDPR && field.type !== "hidden" && (
          <div className="space-y-1.5">
            <Label className="text-[10px] text-slate-500 uppercase">Placeholder</Label>
            <Input
              value={field.placeholder || ""}
              onChange={(e: any) => updateField({ placeholder: e.target.value })}
              className="h-8 text-sm"
              placeholder="Optional placeholder text"
            />
          </div>
        )}
      </div>

      {field.type !== "hidden" && !isGDPR && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label className="text-[10px] text-slate-500 uppercase">Default Value</Label>
            <Input
              value={field.default_value || ""}
              onChange={(e: any) => updateField({ default_value: e.target.value })}
              className="h-8 text-sm"
              placeholder="Optional default value"
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-[10px] text-slate-500 uppercase">Help Text</Label>
            <Input
              value={field.help || ""}
              onChange={(e: any) => updateField({ help: e.target.value })}
              className="h-8 text-sm"
              placeholder="Shown below the field"
            />
          </div>
        </div>
      )}

      {field.type === "hidden" && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label className="text-[10px] text-slate-500 uppercase">Default Value</Label>
            <Input
              value={field.default_value || ""}
              onChange={(e: any) => updateField({ default_value: e.target.value })}
              className="h-8 text-sm"
              placeholder="Hidden field value"
            />
          </div>
        </div>
      )}

      <FieldValidationRules field={field} updateField={updateField} />

      {field.type === "file" && (
        <FieldFileOptions field={field} updateField={updateField} />
      )}

      {isGDPR && <FieldGDPROptions field={field} updateField={updateField} />}

      {(field.type === "select" || field.type === "radio") && (
        <OptionsEditor
          options={normalizedOpts}
          onUpdateLabel={updateOptionLabel}
          onUpdateValue={updateOptionValue}
          onRemove={removeOption}
          onAdd={addOption}
        />
      )}

      {/* Width selector */}
      <div className="space-y-1.5">
        <Label className="text-[10px] text-slate-500 uppercase">Width</Label>
        <div className="flex gap-1">
          {(["full", "half", "third"] as const).map((w) => (
            <button
              key={w}
              type="button"
              onClick={() => updateField({ width: w })}
              className={`flex-1 py-1 text-xs rounded border transition-colors ${
                (field.width || "full") === w
                  ? "border-indigo-400 bg-indigo-50 text-indigo-700 font-semibold"
                  : "border-slate-200 text-slate-500 hover:border-slate-300"
              }`}
            >
              {w === "full" ? "Full" : w === "half" ? "1/2" : "1/3"}
            </button>
          ))}
        </div>
      </div>

      {/* Required toggle */}
      <div className="flex items-center justify-between">
        <div>
          <Label className="text-[10px] text-slate-500 uppercase">Required</Label>
          {isGDPR && (
            <p className="text-[9px] text-slate-400">GDPR consent fields are always required.</p>
          )}
        </div>
        <Switch
          checked={!!field.required}
          onCheckedChange={(checked: boolean) => updateField({ required: checked })}
          disabled={isGDPR}
        />
      </div>

      {/* Conditional visibility */}
      <div style={{ borderTop: "1px solid var(--border)", paddingTop: 12, marginTop: 4 }}>
        <AccordionRow
          open={showConditions}
          onToggle={() => setShowConditions(!showConditions)}
          headerLeft={
            <div className="flex items-center gap-2">
              <span className="text-xs font-medium" style={{ color: "var(--fg)" }}>Show this field when…</span>
              {(displayWhenGroup.all?.length || displayWhenGroup.any?.length) ? (
                <span className="text-[9px] font-semibold px-1.5 py-0.5 rounded-full bg-indigo-100 text-indigo-600">
                  Active
                </span>
              ) : null}
            </div>
          }
        >
          <div>
            <ConditionBuilder
              group={displayWhenGroup}
              onChange={(next) => updateField({ display_when: next })}
              fields={allFields}
              excludeFieldId={field.id}
            />
            {(displayWhenGroup.all?.length || displayWhenGroup.any?.length) ? (
              <button
                type="button"
                className="mt-2 text-[10px] text-slate-400 hover:text-red-500"
                onClick={() => updateField({ display_when: {} })}
              >
                Clear all conditions
              </button>
            ) : null}
          </div>
        </AccordionRow>
      </div>
    </div>
  );
}
