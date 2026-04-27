import React, { useCallback, useState } from "react";
import { RotateCcw } from "@vibecms/icons";

const {
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  CodeWindow,
} = (window as any).__VIBECMS_SHARED__.ui;

const STARTER_TEMPLATES = [
  { value: "simple", label: "Simple (default)" },
  { value: "grid", label: "Grid (multi-column)" },
  { value: "card", label: "Card (shadowed box)" },
  { value: "inline", label: "Inline (compact, no labels)" },
];

async function fetchDefaultLayout(style?: string): Promise<string> {
  const url = style
    ? `/admin/api/ext/forms/defaults/layout?style=${encodeURIComponent(style)}`
    : "/admin/api/ext/forms/defaults/layout";
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error("Failed to fetch default layout");
  const data = await res.json();
  return data.layout as string;
}

export default function LayoutTab({ form, setForm }: any) {
  const [starterStyle, setStarterStyle] = useState<string>("");

  const handleReset = useCallback(async () => {
    try {
      const layout = await fetchDefaultLayout();
      setForm({ ...form, layout });
      setStarterStyle("");
    } catch { /* ignore */ }
  }, [form, setForm]);

  const handleStarterChange = useCallback(
    async (style: string) => {
      if (!style) return;
      const currentLayout = form.layout || "";
      if (
        currentLayout.trim() !== "" &&
        !window.confirm("Replace the current layout with the selected starter template?")
      ) {
        return;
      }
      try {
        const layout = await fetchDefaultLayout(style === "simple" ? undefined : style);
        setForm({ ...form, layout });
        setStarterStyle(style);
      } catch { /* ignore */ }
    },
    [form, setForm],
  );

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h3
            className="font-semibold"
            style={{ fontSize: 13, color: "var(--fg)", margin: 0 }}
          >
            Form Layout
          </h3>
          <p
            style={{
              fontSize: 11,
              color: "var(--fg-muted)",
              margin: 0,
              marginTop: 2,
            }}
          >
            HTML + Go templates
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-48">
            <Select value={starterStyle} onValueChange={handleStarterChange}>
              <SelectTrigger className="h-7 text-[12px]">
                <SelectValue placeholder="Starter template…" />
              </SelectTrigger>
              <SelectContent>
                {STARTER_TEMPLATES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>
                    {t.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleReset}
            className="text-slate-500 hover:text-indigo-600"
          >
            <RotateCcw className="mr-1.5 h-3.5 w-3.5" /> Reset
          </Button>
        </div>
      </div>
      <CodeWindow
        title="layout.html"
        value={form.layout || ""}
        onChange={(v: string) => setForm((p: any) => ({ ...p, layout: v }))}
        height="500px"
      />

      <div className="p-3 bg-blue-50 text-blue-800 rounded-lg text-xs border border-blue-100">
        <p className="font-semibold mb-2">Template Variables</p>
        <div className="grid gap-1 sm:grid-cols-3 font-mono text-[11px]">
          <p>
            <span className="bg-blue-100 px-1 rounded">{"{{range .fields_list}}"}</span>{" "}
            Loop all fields
          </p>
          <p>
            <span className="bg-blue-100 px-1 rounded">{"{{.email.label}}"}</span>{" "}
            Shorthand field access
          </p>
          <p>
            <span className="bg-blue-100 px-1 rounded">{"{{.fields.email.label}}"}</span>{" "}
            Via fields map
          </p>
        </div>
      </div>
    </div>
  );
}
