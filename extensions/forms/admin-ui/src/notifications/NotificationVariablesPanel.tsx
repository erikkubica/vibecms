import React, { useState } from "react";
import { Code } from "@vibecms/icons";

const { AccordionRow } = (window as any).__VIBECMS_SHARED__.ui;

const TEMPLATE_VARS = [
  { syntax: "{{.FormName}}", desc: "Form display name" },
  { syntax: "{{.FormSlug}}", desc: "Form URL slug" },
  { syntax: "{{.FormID}}", desc: "Form database ID" },
  { syntax: "{{.SubmittedAt}}", desc: "Submission timestamp" },
  {
    syntax: "{{range .Data}}",
    desc: "Loop all submitted fields",
    children: [
      { syntax: "  {{.Label}}", desc: "Field label" },
      { syntax: "  {{.Value}}", desc: "Submitted value" },
      { syntax: "  {{.Key}}", desc: "Field key" },
      { syntax: "{{end}}", desc: "End loop" },
    ],
  },
  { syntax: "{{.Field.email}}", desc: "Direct access to specific field value" },
  { syntax: "{{.Field.name}}", desc: "Replace email/name with your field keys" },
];

export default function NotificationVariablesPanel() {
  const [open, setOpen] = useState(false);

  return (
    <AccordionRow
      open={open}
      onToggle={() => setOpen(!open)}
      headerLeft={
        <div className="flex items-center gap-2">
          <Code className="h-4 w-4 text-indigo-500 shrink-0" />
          <span className="text-[13px] font-semibold text-slate-700">
            Template Variables Reference
          </span>
        </div>
      }
    >
      <div className="bg-slate-50 rounded-lg p-4 font-mono text-xs leading-relaxed text-slate-700 border border-slate-100">
        <p className="text-slate-500 text-[11px] mb-2 font-sans font-medium">
          Available in Subject and Body:
        </p>
        {TEMPLATE_VARS.map((v, i) => (
          <React.Fragment key={i}>
            <div className="flex gap-3 py-0.5">
              <span className="text-indigo-600 whitespace-nowrap min-w-[180px]">
                {v.syntax}
              </span>
              <span className="text-slate-500 font-sans">— {v.desc}</span>
            </div>
            {v.children &&
              v.children.map((child, j) => (
                <div key={`${i}-${j}`} className="flex gap-3 py-0.5">
                  <span className="text-indigo-600 whitespace-nowrap min-w-[180px]">
                    {child.syntax}
                  </span>
                  <span className="text-slate-500 font-sans">— {child.desc}</span>
                </div>
              ))}
          </React.Fragment>
        ))}
      </div>
    </AccordionRow>
  );
}
