import React from "react";
import { Plus, PlusSquare, X } from "@vibecms/icons";
import ConditionRow, { Condition } from "./ConditionRow";

export interface ConditionGroup {
  all?: (Condition | ConditionGroup)[];
  any?: (Condition | ConditionGroup)[];
}

interface ConditionBuilderProps {
  group: ConditionGroup;
  onChange: (next: ConditionGroup) => void;
  fields: any[];
  excludeFieldId?: string;
  /** Nesting depth — used for visual indentation. Root = 0. */
  depth?: number;
  /** Called when a nested group asks to remove itself. Only set for depth > 0. */
  onRemoveGroup?: () => void;
}

// Root + one sub-level only. Deeper nesting becomes unreadable.
const MAX_DEPTH = 1;

function isConditionGroup(item: Condition | ConditionGroup): item is ConditionGroup {
  return "all" in item || "any" in item;
}

function emptyCondition(): Condition {
  return { field: "", operator: "equals", value: "" };
}

/** Recursive AND/OR condition group editor. */
export default function ConditionBuilder({
  group,
  onChange,
  fields,
  excludeFieldId,
  depth = 0,
  onRemoveGroup,
}: ConditionBuilderProps): React.ReactElement {
  // Determine mode: "all" (AND) or "any" (OR). Default to "all".
  const mode: "all" | "any" =
    "any" in group && (!("all" in group) || !group.all?.length) ? "any" : "all";
  const items: (Condition | ConditionGroup)[] = group[mode] ?? [];

  const setMode = (newMode: "all" | "any") => {
    onChange({ [newMode]: items } as ConditionGroup);
  };

  const updateItem = (idx: number, next: Condition | ConditionGroup) => {
    const updated = [...items];
    updated[idx] = next;
    onChange({ [mode]: updated } as ConditionGroup);
  };

  const removeItem = (idx: number) => {
    const updated = items.filter((_, i) => i !== idx);
    onChange({ [mode]: updated } as ConditionGroup);
  };

  const addCondition = () => {
    onChange({ [mode]: [...items, emptyCondition()] } as ConditionGroup);
  };

  const addGroup = () => {
    onChange({ [mode]: [...items, { all: [emptyCondition()] }] } as ConditionGroup);
  };

  const indentClass = depth > 0 ? "ml-4 border-l-2 border-indigo-100 pl-3" : "";

  return (
    <div className={`space-y-2 ${indentClass}`}>
      {/* Mode toggle (+ remove-group button when nested) */}
      <div className="flex items-center gap-1">
        <span className="text-[10px] text-slate-500 uppercase font-medium mr-1">Match</span>
        {(["all", "any"] as const).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => setMode(m)}
            className={`px-2 py-0.5 rounded text-[10px] font-semibold border transition-colors cursor-pointer ${
              mode === m
                ? "bg-indigo-600 text-white border-indigo-600"
                : "bg-white text-slate-500 border-slate-200 hover:border-slate-300"
            }`}
          >
            {m === "all" ? "ALL (AND)" : "ANY (OR)"}
          </button>
        ))}
        {onRemoveGroup && (
          <button
            type="button"
            onClick={onRemoveGroup}
            className="ml-auto flex items-center gap-1 text-[10px] text-slate-400 hover:text-red-500 font-medium cursor-pointer"
            aria-label="Remove group"
          >
            <X className="h-3 w-3" />
            Remove group
          </button>
        )}
      </div>

      {/* Condition rows and nested groups */}
      {items.map((item, idx) =>
        isConditionGroup(item) ? (
          <ConditionBuilder
            key={idx}
            group={item}
            onChange={(next) => updateItem(idx, next)}
            fields={fields}
            excludeFieldId={excludeFieldId}
            depth={(depth ?? 0) + 1}
            onRemoveGroup={() => removeItem(idx)}
          />
        ) : (
          <ConditionRow
            key={idx}
            condition={item}
            fields={fields}
            excludeFieldId={excludeFieldId}
            onChange={(next) => updateItem(idx, next)}
            onRemove={() => removeItem(idx)}
          />
        )
      )}

      {/* Add actions */}
      <div className="flex items-center gap-2 pt-1">
        <button
          type="button"
          onClick={addCondition}
          className="flex items-center gap-1 text-[10px] text-indigo-600 hover:text-indigo-800 font-medium cursor-pointer"
        >
          <Plus className="h-3 w-3" />
          Add condition
        </button>
        {depth < MAX_DEPTH && (
          <button
            type="button"
            onClick={addGroup}
            className="flex items-center gap-1 text-[10px] text-slate-500 hover:text-slate-700 font-medium cursor-pointer"
          >
            <PlusSquare className="h-3 w-3" />
            Add group
          </button>
        )}
      </div>
    </div>
  );
}
