import React, { useEffect, useState, useRef } from "react";
import { Eye, RefreshCw, AlertCircle } from "lucide-react";

const { Button, Card, CardContent } = (window as any).__VIBECMS_SHARED__.ui;

export default function PreviewTab({ form }: any) {
    const [html, setHtml] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const containerRef = useRef<HTMLDivElement>(null);

    const fetchPreview = async () => {
        setLoading(true);
        setError(null);
        try {
            const res = await fetch("/admin/api/ext/forms/preview", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    layout: form.layout,
                    fields: form.fields
                }),
                credentials: "include"
            });
            
            if (res.ok) {
                const data = await res.json();
                setHtml(data.html);
            } else {
                const err = await res.json();
                setError(err.error || "Failed to render preview");
            }
        } catch (err) {
            setError("Connection error");
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchPreview();
    }, []);

    // Prevent form submission in preview
    useEffect(() => {
        if (containerRef.current) {
            const formEl = containerRef.current.querySelector("form");
            if (formEl) {
                formEl.onsubmit = (e) => {
                    e.preventDefault();
                    alert("Form submission is disabled in preview mode.");
                };
            }
        }
    }, [html]);

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-slate-800">Live Preview</h3>
                <Button variant="outline" size="sm" onClick={fetchPreview} disabled={loading}>
                    <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
                    Refresh Preview
                </Button>
            </div>

            <Card className="border-slate-200 bg-slate-50 min-h-[500px]">
                <CardContent className="p-8">
                    {loading && !html && (
                        <div className="flex flex-col items-center justify-center h-[400px] text-slate-400">
                            <RefreshCw className="h-8 w-8 animate-spin mb-2" />
                            <p>Rendering preview...</p>
                        </div>
                    )}

                    {error && (
                        <div className="flex flex-col items-center justify-center h-[400px] text-red-500 bg-red-50 rounded-lg border border-red-100 p-6 text-center">
                            <AlertCircle className="h-10 w-10 mb-2" />
                            <p className="font-semibold">Rendering Error</p>
                            <p className="text-sm opacity-80 max-w-md mt-2">{error}</p>
                            <Button variant="outline" className="mt-4 border-red-200 text-red-600 hover:bg-red-100" onClick={fetchPreview}>
                                Try Again
                            </Button>
                        </div>
                    )}

                    {!loading && !error && html && (
                        <div 
                            ref={containerRef}
                            className="bg-white p-8 rounded-xl shadow-sm border border-slate-200 max-w-2xl mx-auto"
                            dangerouslySetInnerHTML={{ __html: html }}
                        />
                    )}

                    {!loading && !error && !html && (
                        <div className="flex flex-col items-center justify-center h-[400px] text-slate-400">
                            <Eye className="h-8 w-8 mb-2" />
                            <p>No layout to preview.</p>
                        </div>
                    )}
                </CardContent>
            </Card>

            <div className="p-4 bg-amber-50 text-amber-800 rounded-lg text-xs border border-amber-100 flex gap-3">
                <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
                <p>
                    <strong>Note:</strong> The preview uses the current unsaved state of your layout and fields. 
                    Interactive features (like redirects) are disabled in this view.
                </p>
            </div>
        </div>
    );
}
