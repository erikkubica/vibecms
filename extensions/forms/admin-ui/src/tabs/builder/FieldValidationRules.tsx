import React from "react";

const { Input, Label } = (window as any).__VIBECMS_SHARED__.ui;

interface FieldValidationRulesProps {
  field: any;
  updateField: (updates: Record<string, any>) => void;
}

export default function FieldValidationRules({
  field,
  updateField,
}: FieldValidationRulesProps) {
  const isText = ["text", "textarea", "url", "email", "tel"].includes(
    field.type,
  );
  const isNumeric = field.type === "number" || field.type === "range";

  if (isText) {
    return (
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
              updateField({
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
              updateField({
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
    );
  }

  if (isNumeric) {
    return (
      <div className="grid gap-2 sm:grid-cols-3">
        <div className="space-y-1">
          <Label className="text-[10px] text-slate-500 uppercase">Min</Label>
          <Input
            type="number"
            value={field.min ?? ""}
            onChange={(e: any) =>
              updateField({
                min: e.target.value ? Number(e.target.value) : undefined,
              })
            }
            className="h-8 text-sm"
            placeholder="No min"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-[10px] text-slate-500 uppercase">Max</Label>
          <Input
            type="number"
            value={field.max ?? ""}
            onChange={(e: any) =>
              updateField({
                max: e.target.value ? Number(e.target.value) : undefined,
              })
            }
            className="h-8 text-sm"
            placeholder="No max"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-[10px] text-slate-500 uppercase">Step</Label>
          <Input
            type="number"
            min={0.01}
            step={0.01}
            value={field.step ?? 1}
            onChange={(e: any) =>
              updateField({ step: e.target.value ? Number(e.target.value) : 1 })
            }
            className="h-8 text-sm"
            placeholder="1"
          />
        </div>
      </div>
    );
  }

  return null;
}
