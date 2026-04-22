import React, { useEffect, useState, useCallback, useRef } from "react";
import { Eye, RefreshCw, AlertCircle } from "@vibecms/icons";

const { Button, Card, CardContent } = (window as any).__VIBECMS_SHARED__.ui;

function buildFakeValue(field: any): string {
  switch (field.type) {
    case "email":
      return "john@example.com";
    case "textarea":
      return "This is a test message that demonstrates how a longer text input would appear in the form preview.";
    case "select":
      return field.placeholder || "Option 1";
    case "checkbox":
      return "true";
    case "number":
      return "42";
    case "tel":
      return "+1 (555) 123-4567";
    case "url":
      return "https://example.com";
    case "date":
      return "2025-01-15";
    case "password":
      return "••••••••";
    case "file":
      return "document.pdf";
    case "hidden":
      return "";
    default:
      return "John Doe";
  }
}

function buildTestData(fields: any[]): Record<string, unknown> {
  const fieldsById: Record<string, any> = {};

  for (const field of fields) {
    fieldsById[field.id] = { ...field, value: buildFakeValue(field) };
  }

  return {
    ...fieldsById,
    // Map keyed by field ID — enables {{.fields.name.label}} direct access
    fields: fieldsById,
    // Ordered array — use {{range .fields_list}} for iteration
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

  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isMountedRef = useRef(true);
  const formRef = useRef(form);
  formRef.current = form;

  // Serialise layout + fields for deep comparison (used as effect deps)
  const layoutSnapshot = form?.layout || "";
  const fieldsSnapshot = JSON.stringify(form?.fields || []);

  // Stable fetchPreview — reads form data from ref, not closure
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
        body: JSON.stringify({
          html_template: currentForm.layout,
          test_data: testData,
        }),
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
        try {
          errData = await res.json();
        } catch {
          errData = { error: `HTTP ${res.status}: ${res.statusText}` };
        }
        setError(
          errData.error || errData.message || "Failed to render preview",
        );
      }
    } catch (err: any) {
      if (!isMountedRef.current) return;
      setError(err.message || "Connection error");
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
        setDebouncing(false);
      }
    }
  }, []);

  // Debounced auto-refresh when layout or fields change
  useEffect(() => {
    // Clear any pending debounce timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // If no layout, skip
    if (!form?.layout) {
      setPreviewHtml("");
      setPreviewHead("");
      setPreviewBodyClass("");
      return;
    }

    setDebouncing(true);

    debounceTimerRef.current = setTimeout(() => {
      fetchPreview();
    }, 1500);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [layoutSnapshot, fieldsSnapshot, fetchPreview]);

  // Cleanup on unmount
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  // Manual refresh (bypasses debounce)
  const handleManualRefresh = useCallback(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }
    setDebouncing(false);
    fetchPreview();
  }, [fetchPreview]);

  const iframeSrcDoc =
    previewHtml || error
      ? `<!doctype html><html><head><meta charset="utf-8">${previewHead || '<script src="https://cdn.tailwindcss.com"></script>'}<style>body{margin:0;padding:1.5rem;} form button[type="submit"],form input[type="submit"]{pointer-events:none;opacity:0.7;}</style></head><body class="${previewBodyClass}">${previewHtml || ""}</body></html>`
      : "";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-800 flex items-center gap-2">
          <Eye className="h-5 w-5" />
          Live Preview
          {debouncing && !loading && (
            <span className="ml-2 text-xs font-normal text-slate-400 flex items-center gap-1 animate-pulse">
              <RefreshCw className="h-3 w-3 animate-spin" />
              Auto-updating…
            </span>
          )}
        </h3>
        <Button
          variant="outline"
          size="sm"
          onClick={handleManualRefresh}
          disabled={loading}
        >
          <RefreshCw
            className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`}
          />
          Refresh Preview
        </Button>
      </div>

      <Card className="border-slate-200 bg-slate-50 min-h-[500px] overflow-hidden">
        <CardContent className="p-0">
          {loading && !previewHtml && (
            <div className="flex flex-col items-center justify-center h-[500px] text-slate-400">
              <RefreshCw className="h-8 w-8 animate-spin mb-2" />
              <p>Rendering preview...</p>
            </div>
          )}

          {error && !previewHtml && (
            <div className="flex flex-col items-center justify-center h-[500px] text-red-500 bg-red-50 rounded-lg border border-red-100 p-6 text-center">
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
              <p className="text-xs mt-1">
                Add a layout template to see a live preview.
              </p>
            </div>
          )}

          {(previewHtml || (loading && previewHtml)) && (
            <div className="relative">
              {(loading || debouncing) && (
                <div className="absolute top-2 right-2 z-10 bg-white/80 backdrop-blur-sm rounded-md px-2 py-1 text-xs text-slate-500 flex items-center gap-1">
                  <RefreshCw className="h-3 w-3 animate-spin" />
                  {debouncing && !loading ? "Auto-updating…" : "Updating..."}
                </div>
              )}
              <iframe
                title="Form preview"
                className="h-[600px] w-full border-0 bg-white"
                sandbox="allow-same-origin allow-scripts"
                srcDoc={iframeSrcDoc}
              />
            </div>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="p-4 bg-amber-50 text-amber-800 rounded-lg text-xs border border-amber-100 flex gap-3">
          <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold mb-1">Simulated Preview Data</p>
            <p>
              This preview uses automatically generated fake values for each
              field. The actual form submission will contain real user input.
              Interactive features like form submission are disabled in this
              view.
            </p>
          </div>
        </div>

        <div className="p-4 bg-blue-50 text-blue-800 rounded-lg text-xs border border-blue-100 flex gap-3">
          <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold mb-1">Template Variables</p>
            <div className="mt-1.5 space-y-1 font-mono text-[11px]">
              <p>
                <span className="bg-blue-100 px-1 rounded">
                  {"{{range .fields_list}}"}
                </span>{" "}
                Loop all fields — access
                <span className="bg-blue-100 px-0.5 rounded">.label</span>,{" "}
                <span className="bg-blue-100 px-0.5 rounded">.type</span>,{" "}
                <span className="bg-blue-100 px-0.5 rounded">.id</span>,{" "}
                <span className="bg-blue-100 px-0.5 rounded">.placeholder</span>
                , <span className="bg-blue-100 px-0.5 rounded">.required</span>,{" "}
                <span className="bg-blue-100 px-0.5 rounded">.value</span>
              </p>
              <p>
                <span className="bg-blue-100 px-1 rounded">
                  {"{{.email.label}}"}
                </span>{" "}
                Shorthand: access any field by ID at top level
              </p>
              <p>
                <span className="bg-blue-100 px-1 rounded">
                  {"{{.fields.email.label}}"}
                </span>{" "}
                Access field by ID via the fields map
              </p>
              <p>
                <span className="bg-blue-100 px-1 rounded">
                  {"{{.fields_by_id.email.label}}"}
                </span>{" "}
                Explicit access by ID (same result)
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
