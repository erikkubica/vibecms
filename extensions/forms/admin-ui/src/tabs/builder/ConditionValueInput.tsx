import React, { useState } from "react";

const { Input, Badge } = (window as any).__VIBECMS_SHARED__.ui;

interface ConditionValueInputProps {
  operator: string;
  value: any;
  onChange: (value: any) => void;
}

/** Renders an operator-aware value input: tag editor for in/not_in, hidden for is_empty/is_not_empty, text otherwise. */
export default function ConditionValueInput({
  operator,
  value,
  onChange,
}: ConditionValueInputProps): React.ReactElement | null {
  const [tagInput, setTagInput] = useState("");

  if (operator === "is_empty" || operator === "is_not_empty") {
    return null;
  }

  if (operator === "in" || operator === "not_in") {
    const tags: string[] = Array.isArray(value) ? value : [];

    const addTag = (raw: string) => {
      const trimmed = raw.trim();
      if (trimmed && !tags.includes(trimmed)) {
        onChange([...tags, trimmed]);
      }
      setTagInput("");
    };

    const removeTag = (tag: string) => {
      onChange(tags.filter((t) => t !== tag));
    };

    return (
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap gap-1 p-1.5 border border-slate-200 rounded-md min-h-[32px] bg-white">
          {tags.map((tag) => (
            <span
              key={tag}
              className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] bg-indigo-50 text-indigo-700 border border-indigo-200"
            >
              {tag}
              <button
                type="button"
                className="hover:text-red-500"
                onClick={() => removeTag(tag)}
                aria-label={`Remove ${tag}`}
              >
                ×
              </button>
            </span>
          ))}
          <input
            className="flex-1 min-w-[80px] text-xs outline-none bg-transparent"
            placeholder="Type & press Enter"
            value={tagInput}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setTagInput(e.target.value)
            }
            onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
              if (e.key === "Enter" || e.key === ",") {
                e.preventDefault();
                addTag(tagInput);
              } else if (e.key === "Backspace" && tagInput === "" && tags.length > 0) {
                removeTag(tags[tags.length - 1]);
              }
            }}
            onBlur={() => {
              if (tagInput.trim()) addTag(tagInput);
            }}
          />
        </div>
        <p className="text-[9px] text-slate-400 mt-0.5">Press Enter or comma to add a value</p>
      </div>
    );
  }

  const strValue = value !== undefined && value !== null ? String(value) : "";

  return (
    <Input
      className="flex-1 h-7 text-xs"
      value={strValue}
      placeholder="value"
      onChange={(e: React.ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
    />
  );
}
