import React from "react";
import { RotateCcw, Info } from "@vibecms/icons";

const { Button, Card, CardContent, Textarea, Label } = (window as any)
  .__VIBECMS_SHARED__.ui;

export default function LayoutTab({ form, setForm }: any) {
  const defaultLayout =
    '<!-- Default layout -->\n<div class="vibe-form">\n  {{range .fields_list}}\n    <div class="vibe-form-field">\n      <label>{{.label}}</label>\n      <input type="{{.type}}" name="{{.id}}" placeholder="{{.placeholder}}" {{if .required}}required{{end}} />\n    </div>\n  {{end}}\n  <button type="submit">Submit</button>\n</div>';

  return (
    <div className="space-y-4">
      <Card className="border-slate-200 shadow-none">
        <CardContent className="p-4 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Label className="text-lg font-semibold">
                Form Layout (HTML + Go Templates)
              </Label>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setForm({ ...form, layout: defaultLayout })}
              className="text-slate-500 hover:text-indigo-600"
            >
              <RotateCcw className="mr-2 h-4 w-4" /> Reset to Default
            </Button>
          </div>

          <div className="bg-slate-900 rounded-lg overflow-hidden border border-slate-800">
            <div className="px-4 py-2 bg-slate-800 border-b border-slate-700 flex items-center justify-between">
              <span className="text-xs font-mono text-slate-400">
                template.html
              </span>
              <div className="flex items-center gap-4">
                <span className="text-[10px] text-slate-500 font-mono">
                  Available: .fields_list, .fields, .id, .name
                </span>
              </div>
            </div>
            <Textarea
              value={form.layout}
              onChange={(e: any) =>
                setForm({ ...form, layout: e.target.value })
              }
              className="min-h-[400px] bg-transparent text-slate-300 font-mono text-sm border-none focus-visible:ring-0 p-4 resize-y"
              placeholder="Write your HTML here..."
            />
          </div>

          <div className="flex items-start gap-3 p-4 bg-blue-50 text-blue-800 rounded-lg text-sm border border-blue-100">
            <Info className="h-5 w-5 mt-0.5 shrink-0" />
            <div>
              <p className="font-semibold mb-1">Templating Guide</p>
              <p className="opacity-90">
                Use <code>{"{{range .fields_list}}...{{end}}"}</code> to loop
                through fields in order. Each field has <code>.id</code>,{" "}
                <code>.type</code>, <code>.label</code>,{" "}
                <code>.placeholder</code>, <code>.required</code>, and{" "}
                <code>.value</code>. For direct field access, use the{" "}
                <code>.fields</code> map:{" "}
                <code>{"{{.fields.email.label}}"}</code>, or the top-level
                shorthand: <code>{"{{.email.label}}"}</code>. Select options
                have <code>.label</code> and <code>.value</code>.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
