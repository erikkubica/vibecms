import React, { useEffect, useState, useCallback, useRef } from "react";
import {
  Eye,
  RefreshCw,
  AlertCircle,
  Monitor,
  Tablet,
  Smartphone,
  Moon,
  Sun,
} from "@vibecms/icons";

const { Button, Card, CardContent, SectionHeader } =
  (window as any).__VIBECMS_SHARED__.ui;

type DeviceMode = "desktop" | "tablet" | "phone";

const DEVICE_CONFIG: Record<DeviceMode, { maxWidth: number; label: string }> = {
  desktop: { maxWidth: 1200, label: "Desktop" },
  tablet: { maxWidth: 768, label: "Tablet" },
  phone: { maxWidth: 375, label: "Phone" },
};

function buildFakeValue(field: any): string {
  switch (field.type) {
    case "email": return "john@example.com";
    case "textarea": return "This is a test message.";
    case "select": return field.placeholder || "Option 1";
    case "checkbox": return "true";
    case "number": return "42";
    case "tel": return "+1 (555) 123-4567";
    case "url": return "https://example.com";
    case "date": return "2025-01-15";
    case "password": return "••••••••";
    case "file": return "document.pdf";
    case "hidden": return "";
    default: return "John Doe";
  }
}

function buildTestData(fields: any[]): Record<string, unknown> {
  const fieldsById: Record<string, any> = {};
  for (const field of fields) {
    fieldsById[field.id] = { ...field, value: buildFakeValue(field) };
  }
  return {
    ...fieldsById,
    fields: fieldsById,
    fields_list: fields.map((f) => ({ ...f, value: buildFakeValue(f) })),
    fields_by_id: fieldsById,
  };
}

export default function PreviewTab({ form }: any) {
  const [previewHtml, setPreviewHtml] = useState("");
  const [previewHead, setPreviewHead] = useState("");
  const [previewBodyClass, setPreviewBodyClass] = useState("");
  const [loading, setLoading] = useState(false);
  const [debouncing, setDebouncing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [device, setDevice] = useState<DeviceMode>("desktop");
  const [darkMode, setDarkMode] = useState(false);

  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isMountedRef = useRef(true);
  const formRef = useRef(form);
  formRef.current = form;

  const layoutSnapshot = form?.layout || "";
  const fieldsSnapshot = JSON.stringify(form?.fields || []);

  const fetchPreview = useCallback(async () => {
    const currentForm = formRef.current;
    if (!currentForm?.layout) {
      setPreviewHtml("");
      setPreviewHead("");
      setPreviewBodyClass("");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const testData = buildTestData(currentForm.fields || []);
      const res = await fetch("/admin/api/block-types/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ html_template: currentForm.layout, test_data: testData }),
        credentials: "include",
      });
      if (!isMountedRef.current) return;
      if (res.ok) {
        const data = await res.json();
        setPreviewHtml(data.html || "");
        setPreviewHead(data.head || "");
        setPreviewBodyClass(data.body_class || "");
      } else {
        let errData: any;
        try { errData = await res.json(); } catch { errData = { error: `HTTP ${res.status}` }; }
        setError(errData.error || errData.message || "Failed to render preview");
      }
    } catch (err: any) {
      if (!isMountedRef.current) return;
      setError(err.message || "Connection error");
    } finally {
      if (isMountedRef.current) { setLoading(false); setDebouncing(false); }
    }
  }, []);

  useEffect(() => {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    if (!form?.layout) {
      setPreviewHtml(""); setPreviewHead(""); setPreviewBodyClass("");
      return;
    }
    setDebouncing(true);
    debounceTimerRef.current = setTimeout(() => { fetchPreview(); }, 1500);
    return () => { if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current); };
  }, [layoutSnapshot, fieldsSnapshot, fetchPreview]);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    };
  }, []);

  const handleManualRefresh = useCallback(() => {
    if (debounceTimerRef.current) { clearTimeout(debounceTimerRef.current); debounceTimerRef.current = null; }
    setDebouncing(false);
    fetchPreview();
  }, [fetchPreview]);

  const darkClass = darkMode ? "dark bg-gray-900 text-gray-100" : "";
  // Always load Tailwind CDN — form layouts are authored in raw Tailwind utility
  // classes. The theme's <head> (returned as previewHead) is appended after, so
  // theme-specific styles still apply on top.
  const iframeSrcDoc =
    previewHtml || error
      ? `<!doctype html><html><head><meta charset="utf-8"><script src="https://cdn.tailwindcss.com"></script>${previewHead || ""}<style>body{margin:0;padding:1.5rem;} form button[type="submit"],form input[type="submit"]{pointer-events:none;opacity:0.7;}</style></head><body class="${[previewBodyClass, darkClass].filter(Boolean).join(" ")}">${previewHtml || ""}</body></html>`
      : "";

  const deviceToggle = (
    <div className="flex items-center rounded-lg border border-slate-200 overflow-hidden">
      {(["desktop", "tablet", "phone"] as DeviceMode[]).map((d) => {
        const Icon = d === "desktop" ? Monitor : d === "tablet" ? Tablet : Smartphone;
        return (
          <button
            key={d}
            type="button"
            title={DEVICE_CONFIG[d].label}
            onClick={() => setDevice(d)}
            className={`px-2.5 py-1.5 transition-colors ${
              device === d
                ? "bg-indigo-600 text-white"
                : "bg-white text-slate-500 hover:bg-slate-50"
            }`}
          >
            <Icon className="h-4 w-4" />
          </button>
        );
      })}
    </div>
  );

  return (
    <Card className="rounded-xl border border-slate-200 shadow-sm">
      <SectionHeader
        title={`Live Preview${debouncing && !loading ? " — Auto-updating…" : ""}`}
        icon={<Eye className="h-4 w-4 text-indigo-500" />}
        actions={
          <div className="flex items-center gap-2">
            {deviceToggle}
            <button
              type="button"
              title={darkMode ? "Light mode" : "Dark mode"}
              onClick={() => setDarkMode(!darkMode)}
              className={`rounded-lg border px-2.5 py-1.5 transition-colors ${
                darkMode
                  ? "bg-slate-800 text-yellow-300 border-slate-600"
                  : "bg-white text-slate-500 border-slate-200 hover:bg-slate-50"
              }`}
            >
              {darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </button>
            <Button variant="outline" size="sm" onClick={handleManualRefresh} disabled={loading}>
              <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          </div>
        }
      />

      <CardContent className="p-0">
        <div className="min-h-[500px] bg-slate-50">
          {loading && !previewHtml && (
            <div className="flex flex-col items-center justify-center h-[500px] text-slate-400">
              <RefreshCw className="h-8 w-8 animate-spin mb-2" />
              <p>Rendering preview...</p>
            </div>
          )}

          {error && !previewHtml && (
            <div className="flex flex-col items-center justify-center h-[500px] text-red-500 p-6 text-center">
              <AlertCircle className="h-10 w-10 mb-2" />
              <p className="font-semibold">Template Error</p>
              <p className="text-sm opacity-80 max-w-md mt-2">{error}</p>
              <Button
                variant="outline"
                className="mt-4 border-red-200 text-red-600 hover:bg-red-100"
                onClick={handleManualRefresh}
              >
                <RefreshCw className="mr-2 h-4 w-4" />
                Try Again
              </Button>
            </div>
          )}

          {!loading && !error && !previewHtml && (
            <div className="flex flex-col items-center justify-center h-[500px] text-slate-400">
              <Eye className="h-8 w-8 mb-2" />
              <p>No layout template to preview.</p>
              <p className="text-xs mt-1">Add a layout template to see a live preview.</p>
            </div>
          )}

          {(previewHtml || (loading && previewHtml)) && (
            <div
              className={`relative flex justify-center ${device !== "desktop" ? "bg-slate-100 py-4" : ""}`}
            >
              {(loading || debouncing) && (
                <div className="absolute top-2 right-2 z-10 bg-white/80 backdrop-blur-sm rounded-md px-2 py-1 text-xs text-slate-500 flex items-center gap-1">
                  <RefreshCw className="h-3 w-3 animate-spin" />
                  {debouncing && !loading ? "Auto-updating…" : "Updating..."}
                </div>
              )}
              <div
                style={{ maxWidth: DEVICE_CONFIG[device].maxWidth, width: "100%" }}
                className={device !== "desktop" ? "rounded-xl overflow-hidden border border-slate-300 shadow-lg bg-white" : ""}
              >
                <iframe
                  title="Form preview"
                  className="h-[600px] w-full border-0 bg-white"
                  sandbox="allow-same-origin allow-scripts"
                  srcDoc={iframeSrcDoc}
                />
              </div>
            </div>
          )}
        </div>

        <div className="p-4 border-t border-slate-100">
          <div className="p-3 bg-amber-50 text-amber-800 rounded-lg text-xs border border-amber-100 flex gap-3">
            <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold mb-0.5">Simulated Preview Data</p>
              <p>
                This preview uses generated fake values. Interactive features like form
                submission are disabled.
              </p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
