import React from "react";

const { Input, Label, Switch } = (window as any).__VIBECMS_SHARED__.ui;

interface FieldFileOptionsProps {
  field: any;
  updateField: (updates: Record<string, any>) => void;
}

export default function FieldFileOptions({
  field,
  updateField,
}: FieldFileOptionsProps) {
  return (
    <>
      <div className="grid gap-2 sm:grid-cols-2">
        <div className="space-y-1">
          <Label className="text-[10px] text-slate-500 uppercase">
            Allowed Types
          </Label>
          <Input
            value={field.allowed_types || ""}
            onChange={(e: any) =>
              updateField({ allowed_types: e.target.value })
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
              updateField({
                max_size: e.target.value ? Number(e.target.value) : 5,
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
            updateField({ multiple: checked })
          }
        />
      </div>
    </>
  );
}
