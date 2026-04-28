import React, { useEffect, useRef, useMemo, useState } from "react";
import {
  Save,
  ArrowLeft,
  Layout,
  Settings,
  Mail,
  ListPlus,
  Eye,
  Webhook,
  Download,
  Trash2,
} from "@vibecms/icons";
import { typeLabelMap } from "./tabs/builder/types";

const {
  Button,
  Card,
  CardContent,
  SectionHeader,
  Separator,
  Chip,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  LoadingRow,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} = (window as any).__VIBECMS_SHARED__.ui;
const { useParams, useNavigate } = (window as any).__VIBECMS_SHARED__
  .ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

import BuilderTab from "./tabs/BuilderTab";
import LayoutTab from "./tabs/LayoutTab";
import PreviewTab from "./tabs/PreviewTab";
import NotificationsTab from "./tabs/NotificationsTab";
import SettingsTab from "./tabs/SettingsTab";
import WebhooksTab from "./tabs/WebhooksTab";

function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function FormEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(id ? true : false);
  const [saving, setSaving] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [form, setForm] = useState({
    name: "",
    slug: "",
    fields: [] as any[],
    layout: "",
    notifications: [
      {
        name: "Admin Notification",
        enabled: true,
        recipients: "{{.SiteEmail}}",
        subject: "New submission: {{.FormName}}",
        body: "You have a new submission.\n\n{{range .Data}}\n{{.Label}}: {{.Value}}\n{{end}}",
        reply_to: "",
      },
    ] as any[],
    settings: {
      success_message: "Thank you! Your message has been sent.",
      error_message: "Oops! Something went wrong.",
      redirect_url: "",
    } as Record<string, any>,
    created_at: "" as string,
    updated_at: "" as string,
  });

  const initialFormRef = useRef<any>(null);
  const [autoSlug, setAutoSlug] = useState(!id || id === "new");

  const isDirty = useMemo(() => {
    if (!initialFormRef.current) return false;
    return JSON.stringify(form) !== JSON.stringify(initialFormRef.current);
  }, [form]);

  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (isDirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [isDirty]);

  useEffect(() => {
    if (id && id !== "new") {
      fetch(`/admin/api/ext/forms/${id}`, { credentials: "include" })
        .then((res) => res.json())
        .then((data) => {
          const loaded = {
            ...data,
            fields: data.fields || [],
            notifications: data.notifications || [],
            settings: data.settings || {},
          };
          setForm(loaded);
          setAutoSlug(false);
          initialFormRef.current = JSON.parse(JSON.stringify(loaded));
          setLoading(false);
        })
        .catch(() => {
          toast.error("Failed to load form");
          navigate("/admin/ext/forms");
        });
    } else {
      fetch("/admin/api/ext/forms/defaults/layout", { credentials: "include" })
        .then((res) => res.json())
        .then((data) => {
          setForm((prev) => {
            const updated = { ...prev, layout: data.layout || "" };
            initialFormRef.current = JSON.parse(JSON.stringify(updated));
            return updated;
          });
        })
        .catch(() => {
          initialFormRef.current = JSON.parse(JSON.stringify(form));
        });
    }
  }, [id]);

  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const validateLocally = (): Record<string, string> => {
    const errs: Record<string, string> = {};
    const name = (form.name || "").trim();
    const slug = (form.slug || "").trim();
    if (!name) errs.name = "Name is required";
    if (!slug) errs.slug = "Slug is required";
    else if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug))
      errs.slug = "Lowercase letters, numbers, and hyphens only";
    return errs;
  };

  const handleNameChange = (val: string) => {
    setForm((prev: any) => {
      const next: any = { ...prev, name: val };
      if (autoSlug) next.slug = slugify(val);
      return next;
    });
    if (fieldErrors.name) setFieldErrors((p) => ({ ...p, name: "" }));
    if (autoSlug && fieldErrors.slug) setFieldErrors((p) => ({ ...p, slug: "" }));
  };

  const handleSave = async () => {
    const local = validateLocally();
    if (Object.keys(local).length > 0) {
      setFieldErrors(local);
      toast.error(Object.values(local)[0]);
      return;
    }
    setFieldErrors({});
    setSaving(true);

    const method = id && id !== "new" ? "PUT" : "POST";
    const url =
      id && id !== "new"
        ? `/admin/api/ext/forms/${id}`
        : "/admin/api/ext/forms/";

    try {
      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
        credentials: "include",
      });
      if (res.ok) {
        toast.success("Form saved successfully");
        if (method === "POST") {
          const data = await res.json();
          initialFormRef.current = JSON.parse(JSON.stringify(form));
          navigate(`/admin/ext/forms/edit/${data.id}`);
        } else {
          initialFormRef.current = JSON.parse(JSON.stringify(form));
        }
      } else {
        const err = await res.json();
        if (err.fields && typeof err.fields === "object") {
          setFieldErrors(err.fields);
          const first = Object.values(err.fields)[0];
          toast.error(typeof first === "string" ? first : err.message || "Validation failed");
        } else {
          toast.error(err.message || err.error || "Failed to save form");
        }
      }
    } catch {
      toast.error("An error occurred while saving");
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = () => {
    if (isDirty && !window.confirm("Discard unsaved changes?")) return;
    navigate("/admin/ext/forms");
  };

  const handleDelete = async () => {
    if (!id || id === "new") return;
    setDeleting(true);
    try {
      const res = await fetch(`/admin/api/ext/forms/${id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (res.ok) {
        toast.success("Form deleted");
        navigate("/admin/ext/forms");
      } else {
        toast.error("Failed to delete form");
      }
    } catch {
      toast.error("Failed to delete form");
    } finally {
      setDeleting(false);
      setShowDelete(false);
    }
  };

  if (loading)
    return (
      <div className="w-full pb-8">
        <LoadingRow />
      </div>
    );

  const isEdit = !!(id && id !== "new");

  return (
    <div className="w-full pb-8">
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Main content (col 1): pill + tabs */}
        <div className="space-y-4 min-w-0">
      {/* Compact pill header */}
      <div
        className="flex items-center gap-1.5"
        style={{
          padding: 6,
          background: "var(--card-bg)",
          border: "1px solid var(--border)",
          borderRadius: "var(--radius-lg)",
          boxShadow: "var(--shadow-sm)",
        }}
      >
        <Button
          variant="ghost"
          size="icon"
          onClick={handleCancel}
          className="h-7 w-7 shrink-0"
          aria-label="Back"
        >
          <ArrowLeft className="h-3.5 w-3.5" style={{ color: "var(--fg-muted)" }} />
        </Button>

        <div className="flex items-center gap-1.5 flex-[1_1_60%] min-w-0 px-1">
          <span
            className="shrink-0 uppercase"
            style={{ fontSize: 10.5, fontWeight: 600, color: "var(--fg-muted)", letterSpacing: "0.06em" }}
          >
            Form Name
          </span>
          <input
            placeholder="Contact Us"
            value={form.name}
            onChange={(e: any) => handleNameChange(e.target.value)}
            required
            className="flex-1 min-w-0 bg-transparent outline-none"
            style={{ border: "none", padding: "6px 4px", fontSize: 14, fontWeight: 500, color: "var(--fg)" }}
          />
        </div>

        <div className="w-px h-5 shrink-0" style={{ background: "var(--border)" }} />

        <div className="flex items-center gap-1 flex-[1_1_40%] min-w-0 px-1">
          <span className="shrink-0" style={{ fontSize: 11, color: "var(--fg-subtle)", fontFamily: "var(--font-mono)" }}>/</span>
          <input
            placeholder="auto-generated"
            value={form.slug}
            onChange={(e: any) => {
              setAutoSlug(false);
              setForm((prev: any) => ({ ...prev, slug: e.target.value.replace(/\s+/g, "-").toLowerCase() }));
              if (fieldErrors.slug) setFieldErrors((p) => ({ ...p, slug: "" }));
            }}
            disabled={autoSlug}
            required
            className="flex-1 min-w-0 bg-transparent outline-none disabled:opacity-60"
            style={{ border: "none", padding: "6px 0", fontSize: 12.5, color: "var(--fg)", fontFamily: "var(--font-mono)" }}
          />
          <button
            type="button"
            className="shrink-0 px-1.5 py-0.5 rounded text-[10.5px] font-medium uppercase"
            style={{
              color: autoSlug ? "var(--accent)" : "var(--fg-muted)",
              background: autoSlug ? "color-mix(in oklab, var(--accent) 12%, transparent)" : "var(--sub-bg)",
              border: "1px solid var(--border)",
              letterSpacing: "0.04em",
            }}
            onClick={() => setAutoSlug(!autoSlug)}
            title={autoSlug ? "Click to edit slug manually" : "Click to auto-generate from name"}
          >
            {autoSlug ? "Auto" : "Edit"}
          </button>
        </div>
      </div>

      {/* Field errors below pill */}
      {(fieldErrors.name || fieldErrors.slug) && (
        <div className="flex gap-6 px-2 -mt-1">
          {fieldErrors.name && <p className="text-xs text-rose-600">{fieldErrors.name}</p>}
          {fieldErrors.slug && <p className="text-xs text-rose-600">{fieldErrors.slug}</p>}
        </div>
      )}

          <Tabs defaultValue="builder" className="w-full mt-1">
            <TabsList className="grid w-full grid-cols-6 mb-4">
              <TabsTrigger value="builder">
                <ListPlus className="mr-1.5 h-3.5 w-3.5" /> Builder
              </TabsTrigger>
              <TabsTrigger value="layout">
                <Layout className="mr-1.5 h-3.5 w-3.5" /> Layout
              </TabsTrigger>
              <TabsTrigger value="preview">
                <Eye className="mr-1.5 h-3.5 w-3.5" /> Preview
              </TabsTrigger>
              <TabsTrigger value="notifications">
                <Mail className="mr-1.5 h-3.5 w-3.5" /> Notifs
              </TabsTrigger>
              <TabsTrigger value="settings">
                <Settings className="mr-1.5 h-3.5 w-3.5" /> Settings
              </TabsTrigger>
              <TabsTrigger value="webhooks">
                <Webhook className="mr-1.5 h-3.5 w-3.5" /> Webhooks
              </TabsTrigger>
            </TabsList>

            <TabsContent value="builder">
              <BuilderTab form={form} setForm={setForm} />
            </TabsContent>
            <TabsContent value="layout">
              <LayoutTab form={form} setForm={setForm} />
            </TabsContent>
            <TabsContent value="preview">
              <PreviewTab form={form} />
            </TabsContent>
            <TabsContent value="notifications">
              <NotificationsTab form={form} setForm={setForm} />
            </TabsContent>
            <TabsContent value="settings">
              <SettingsTab form={form} setForm={setForm} />
            </TabsContent>
            <TabsContent value="webhooks">
              <WebhooksTab form={form} setForm={setForm} />
            </TabsContent>
          </Tabs>
        </div>

        {/* Sidebar (col 2) */}
        <div className="space-y-4">
          {/* Publish card — canonical pattern */}
          <Card>
            <SectionHeader title="Publish" />
            <CardContent className="space-y-4 p-4">
              <div className="relative">
                {isDirty && (
                  <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-amber-400 border border-white z-10" />
                )}
                <Button
                  className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm h-9 text-sm"
                  onClick={handleSave}
                  disabled={saving}
                >
                  <Save className="mr-1.5 h-3.5 w-3.5" />
                  {saving ? "Saving…" : "Save"}
                </Button>
              </div>

              {isEdit && (
                <>
                  <Separator />
                  <Button
                    variant="outline"
                    className="w-full bg-red-50 text-red-700 border-red-200 hover:bg-red-100 rounded-lg font-medium h-8 text-xs"
                    onClick={() => setShowDelete(true)}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete
                  </Button>
                </>
              )}

              {isEdit && (
                <>
                  <Separator />
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-400">
                    <div className="flex justify-between">
                      <span>Fields</span>
                      <span className="text-slate-600">{form.fields?.length || 0}</span>
                    </div>
                    {form.created_at && (
                      <div className="flex justify-between">
                        <span>Created</span>
                        <span className="text-slate-600">{new Date(form.created_at).toLocaleDateString()}</span>
                      </div>
                    )}
                    {form.updated_at && (
                      <div className="flex justify-between">
                        <span>Updated</span>
                        <span className="text-slate-600">{new Date(form.updated_at).toLocaleDateString()}</span>
                      </div>
                    )}
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* Fields summary — visible on every tab */}
          {form.fields && form.fields.length > 0 && (
            <Card>
              <SectionHeader title={`Fields (${form.fields.length})`} />
              <CardContent className="p-3 space-y-1.5">
                {form.fields.map((f: any, i: number) => (
                  <div key={i} className="flex items-center gap-2 text-[12px] min-w-0">
                    <div className="flex-1 min-w-0">
                      <div className="truncate text-slate-700 font-medium">{f.label || "Untitled"}</div>
                      <div className="truncate text-[10.5px] font-mono" style={{ color: "var(--fg-subtle)" }}>{f.id || f.key || "—"}</div>
                    </div>
                    <Chip>{typeLabelMap[f.type] || f.type}</Chip>
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          {/* Actions card */}
          {isEdit && (
            <Card>
              <SectionHeader title="Actions" />
              <CardContent className="p-3">
                <Button
                  variant="ghost"
                  size="sm"
                  className="w-full justify-start text-slate-500 hover:text-indigo-600"
                  onClick={() => {
                    window.location.href = `/admin/api/ext/forms/${id}/export`;
                  }}
                >
                  <Download className="mr-1.5 h-3.5 w-3.5" /> Export JSON
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Delete dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Form</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{form.name}&quot;?
              This will also delete all submissions for this form. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDelete(false)} disabled={deleting}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
