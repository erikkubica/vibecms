import React from "react";
import { Plus, X } from "@vibecms/icons";
import { OptionPair } from "./types";

const { Button, Input, Label } = (window as any).__VIBECMS_SHARED__.ui;

interface OptionsEditorProps {
  options: OptionPair[];
  onUpdateLabel: (optIdx: number, label: string) => void;
  onUpdateValue: (optIdx: number, value: string) => void;
  onRemove: (optIdx: number) => void;
  onAdd: () => void;
}

export default function OptionsEditor({
  options,
  onUpdateLabel,
  onUpdateValue,
  onRemove,
  onAdd,
}: OptionsEditorProps) {
  return (
    <div className="space-y-2">
      <Label className="text-[10px] text-slate-500 uppercase">Options</Label>
      {options.length > 0 && (
        <div className="flex items-center gap-1.5 text-[9px] text-slate-400 uppercase px-0.5">
          <span className="flex-1">Label</span>
          <span className="flex-1">Value</span>
          <span className="w-7" />
        </div>
      )}
      <div className="space-y-1.5">
        {options.map((opt: OptionPair, optIdx: number) => (
          <div key={optIdx} className="flex items-center gap-1.5">
            <Input
              value={opt.label}
              onChange={(e: any) => onUpdateLabel(optIdx, e.target.value)}
              className="h-7 text-xs flex-1 min-w-0"
              placeholder="Label"
            />
            <Input
              value={opt.value}
              onChange={(e: any) =>
                onUpdateValue(
                  optIdx,
                  e.target.value.replace(/[^a-z0-9_]/gi, ""),
                )
              }
              className="h-7 text-xs font-mono flex-1 min-w-0"
              placeholder="value"
            />
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-slate-300 hover:text-red-500 shrink-0"
              onClick={() => onRemove(optIdx)}
            >
              <X className="h-3 w-3" />
            </Button>
          </div>
        ))}
      </div>
      <button
        type="button"
        onClick={onAdd}
        className="text-[11px] text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1"
      >
        <Plus className="h-3 w-3" /> Add option
      </button>
    </div>
  );
}
