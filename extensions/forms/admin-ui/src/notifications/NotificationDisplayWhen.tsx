import React, { useState } from "react";
import ConditionBuilder, { ConditionGroup } from "../tabs/builder/ConditionBuilder";

const { AccordionRow } = (window as any).__VIBECMS_SHARED__.ui;

interface NotificationDisplayWhenProps {
  routeWhen: ConditionGroup;
  onChange: (next: ConditionGroup) => void;
  /** All fields available in the form for the condition field dropdown. */
  formFields: any[];
}

/** Collapsible "Send only when…" condition builder section for NotificationCard. */
export default function NotificationDisplayWhen({
  routeWhen,
  onChange,
  formFields,
}: NotificationDisplayWhenProps): React.ReactElement {
  const hasConditions = !!(routeWhen.all?.length || routeWhen.any?.length);
  const [expanded, setExpanded] = useState(hasConditions);

  return (
    <div className="border-t border-slate-100 pt-3">
      <AccordionRow
        open={expanded}
        onToggle={() => setExpanded(!expanded)}
        headerLeft={
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium" style={{ color: "var(--fg)" }}>Send only when…</span>
            {hasConditions && (
              <span className="text-[9px] font-semibold px-1.5 py-0.5 rounded-full bg-amber-100 text-amber-600">
                Active
              </span>
            )}
          </div>
        }
      >
        <div className="space-y-2">
          <p className="text-[10px] text-slate-400">
            This notification will only be sent when all selected conditions are met.
            Leave empty to always send.
          </p>
          <ConditionBuilder
            group={routeWhen}
            onChange={onChange}
            fields={formFields}
          />
          {hasConditions && (
            <button
              type="button"
              className="text-[10px] text-slate-400 hover:text-red-500"
              onClick={() => onChange({})}
            >
              Clear all conditions
            </button>
          )}
        </div>
      </AccordionRow>
    </div>
  );
}
