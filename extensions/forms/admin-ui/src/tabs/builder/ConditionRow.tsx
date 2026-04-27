import React from "react";
import { X } from "@vibecms/icons";
import ConditionValueInput from "./ConditionValueInput";

const { Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Button } =
  (window as any).__VIBECMS_SHARED__.ui;

export interface Condition {
  field: string;
  operator: string;
  value?: any;
}

export type ConditionOperator =
  | "equals" | "not_equals"
  | "contains" | "not_contains"
  | "gt" | "gte" | "lt" | "lte"
  | "in" | "not_in"
  | "matches"
  | "is_empty" | "is_not_empty";

const OPERATORS: { value: ConditionOperator; label: string }[] = [
  { value: "equals",       label: "equals" },
  { value: "not_equals",   label: "not equals" },
  { value: "contains",     label: "contains" },
  { value: "not_contains", label: "does not contain" },
  { value: "gt",           label: "greater than" },
  { value: "gte",          label: "greater or equal" },
  { value: "lt",           label: "less than" },
  { value: "lte",          label: "less or equal" },
  { value: "in",           label: "is one of" },
  { value: "not_in",       label: "is not one of" },
  { value: "matches",      label: "matches regex" },
  { value: "is_empty",     label: "is empty" },
  { value: "is_not_empty", label: "is not empty" },
];

interface ConditionRowProps {
  condition: Condition;
  fields: any[];
  excludeFieldId?: string;
  onChange: (next: Condition) => void;
  onRemove: () => void;
}

/** Single condition row: Field dropdown + Operator dropdown + Value input + Remove button. */
export default function ConditionRow({
  condition,
  fields,
  excludeFieldId,
  onChange,
  onRemove,
}: ConditionRowProps): React.ReactElement {
  const availableFields = fields.filter((f) => f.id !== excludeFieldId);

  const handleFieldChange = (fieldId: string) => {
    onChange({ ...condition, field: fieldId });
  };

  const handleOperatorChange = (op: string) => {
    const updated: Condition = { ...condition, operator: op };
    // Clear value when switching to no-value operators
    if (op === "is_empty" || op === "is_not_empty") {
      delete updated.value;
    }
    // Reset to empty array for list operators
    if ((op === "in" || op === "not_in") && !Array.isArray(updated.value)) {
      updated.value = [];
    }
    onChange(updated);
  };

  const handleValueChange = (val: any) => {
    onChange({ ...condition, value: val });
  };

  return (
    <div className="flex items-start gap-1.5">
      {/* Field selector */}
      <Select value={condition.field || ""} onValueChange={handleFieldChange}>
        <SelectTrigger className="w-32 h-7 text-xs bg-white shrink-0">
          <SelectValue placeholder="Field" />
        </SelectTrigger>
        <SelectContent>
          {availableFields.map((f) => (
            <SelectItem key={f.id} value={f.id}>
              {f.label || f.id}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Operator selector */}
      <Select value={condition.operator || "equals"} onValueChange={handleOperatorChange}>
        <SelectTrigger className="w-36 h-7 text-xs bg-white shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {OPERATORS.map((op) => (
            <SelectItem key={op.value} value={op.value}>
              {op.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Value input (operator-aware) */}
      <ConditionValueInput
        operator={condition.operator || "equals"}
        value={condition.value}
        onChange={handleValueChange}
      />

      {/* Remove button */}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-slate-400 hover:text-red-500 shrink-0 cursor-pointer"
        onClick={onRemove}
        aria-label="Remove condition"
      >
        <X className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}
