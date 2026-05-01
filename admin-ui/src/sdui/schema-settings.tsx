import { useEffect, useState, useCallback } from "react";
import { Save, Loader2, RefreshCw } from "lucide-react";
import { useAdminLanguage } from "@/hooks/use-admin-language";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import { SidebarCard } from "@/components/ui/sidebar-card";
import { LanguageSelect } from "@/components/ui/language-select";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import {
  clearCache,
  getNodes,
  getRoles,
  getSettingsSchema,
  saveSettingsSchema,
  type ContentNode,
  type Role,
  type SettingsSchema,
  type SettingsSchemaField,
} from "@/api/client";
import { iconMap } from "./sdui-components";

// SchemaSettings is the schema-driven counterpart of SettingsForm. The
// React component is purely a renderer — the schema (sections, fields,
// per-field translatable flag) comes from the server. One mixed page can
// have translatable fields (per-language) alongside global fields
// (language_code='') with the framework routing storage transparently.
//
// SDUI emits { type: "SchemaSettings", props: { schema_id, show_clear_cache } }.
// The component fetches the schema + current values for the admin's
// locale and posts diffs back to /admin/api/settings/schemas/<id>.

const ICON_COLORS: Record<string, string> = {
  Globe: "var(--accent-strong)",
  Home: "var(--success)",
  FileText: "var(--warning)",
  Code: "var(--warning)",
  Settings: "var(--muted-foreground)",
  Shield: "var(--danger)",
};

function renderIcon(name: string | undefined) {
  if (!name) return null;
  const Icon = iconMap[name];
  if (!Icon) return null;
  const color = ICON_COLORS[name] || "var(--accent-strong)";
  return <Icon className="h-4 w-4" style={{color}} />;
}

export interface SchemaSettingsProps {
  schema_id: string;
  show_clear_cache?: boolean;
}

export function SchemaSettings({
  schema_id,
  show_clear_cache = false,
}: SchemaSettingsProps) {
  const { languages, currentCode } = useAdminLanguage();
  const [schema, setSchema] = useState<SettingsSchema | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [original, setOriginal] = useState<Record<string, string>>({});
  const [pages, setPages] = useState<ContentNode[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  // pageLocale only matters when at least one field is translatable.
  // For all-global schemas the picker is hidden and saves omit the
  // locale header (server falls back to '' for non-translatable rows
  // regardless).
  const [pageLocale, setPageLocale] = useState<string>(currentCode);
  useEffect(() => {
    setPageLocale(currentCode);
  }, [currentCode]);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const env = await getSettingsSchema(schema_id, pageLocale);
      const needsPages = env.schema.sections.some((s) =>
        s.fields.some((f) => f.type === "node_select"),
      );
      const needsRoles = env.schema.sections.some((s) =>
        s.fields.some((f) => f.type === "role_select"),
      );
      const sidePromises: Promise<unknown>[] = [];
      if (needsPages) {
        sidePromises.push(getNodes({ page: 1, per_page: 200, status: "published" }));
      } else {
        sidePromises.push(Promise.resolve(undefined));
      }
      if (needsRoles) sidePromises.push(getRoles());
      else sidePromises.push(Promise.resolve(undefined));

      const [pagesRes, rolesRes] = (await Promise.all(sidePromises)) as [
        { data: ContentNode[] } | undefined,
        Role[] | undefined,
      ];

      // Initialise every key the schema declares so onChange handlers
      // never produce undefined → controlled-input warnings.
      const initial: Record<string, string> = {};
      for (const sec of env.schema.sections) {
        for (const f of sec.fields) {
          initial[f.key] = env.values[f.key] ?? f.default ?? "";
        }
      }
      setSchema(env.schema);
      setValues(initial);
      setOriginal(initial);
      if (pagesRes) setPages(pagesRes.data);
      if (rolesRes) setRoles(rolesRes);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to load settings";
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  }, [schema_id, pageLocale]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const hasTranslatable = schema?.sections.some((s) =>
    s.fields.some((f) => f.translatable),
  ) ?? false;

  async function handleSave() {
    if (!schema) return;
    setSaving(true);
    try {
      const diff: Record<string, string> = {};
      for (const key of Object.keys(values)) {
        if (values[key] !== original[key]) diff[key] = values[key];
      }
      if (Object.keys(diff).length === 0) {
        toast.info("No changes to save");
        return;
      }
      // Only forward the locale when the schema actually has
      // translatable fields. For all-global schemas the server doesn't
      // need it and forwarding it would tie an unrelated header to
      // the request.
      const localeArg = hasTranslatable ? pageLocale : undefined;
      await saveSettingsSchema(schema.id, diff, localeArg);
      setOriginal({ ...values });
      toast.success("Settings saved");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to save settings";
      toast.error(msg);
    } finally {
      setSaving(false);
    }
  }

  async function handleClearCache() {
    setClearing(true);
    try {
      await clearCache();
      toast.success("All caches cleared");
    } catch {
      toast.error("Failed to clear caches");
    } finally {
      setClearing(false);
    }
  }

  if (loading || !schema) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin" style={{color: "var(--accent-strong)"}} />
      </div>
    );
  }

  const hasChanges = JSON.stringify(values) !== JSON.stringify(original);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-foreground">{schema.title}</h1>
        {schema.description && (
          <p className="text-sm text-muted-foreground mt-0.5">{schema.description}</p>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div className="space-y-4 min-w-0">
          {schema.sections.map((section, idx) => (
            <Card
              key={idx}
              className="rounded-xl border border-border shadow-sm"
            >
              <SectionHeader title={section.title} icon={renderIcon(section.icon)} />
              <CardContent className="space-y-4">
                {section.description && (
                  <p className="text-xs text-muted-foreground -mt-1">{section.description}</p>
                )}
                <div className="space-y-4">
                  {section.fields.map((field) => (
                    <SchemaField
                      key={field.key}
                      field={field}
                      value={values[field.key] ?? ""}
                      pages={pages}
                      roles={roles}
                      onChange={(v) =>
                        setValues((prev) => ({ ...prev, [field.key]: v }))
                      }
                    />
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        <aside className="lg:sticky lg:top-4 lg:self-start" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <SidebarCard title="Publish">
            <div style={{ display: "flex", flexDirection: "column", gap: 11 }}>
              {hasTranslatable && languages.length > 0 ? (
                <div style={{ display: "flex", flexDirection: "column", gap: 5 }}>
                  <Label style={{ fontSize: 12, fontWeight: 500, color: "var(--fg)", letterSpacing: "-0.005em" }}>Language</Label>
                  <LanguageSelect
                    languages={languages}
                    value={pageLocale}
                    onChange={setPageLocale}
                  />
                  <span style={{ fontSize: 11.5, color: "var(--fg-muted)", lineHeight: 1.45, letterSpacing: "-0.005em" }}>
                    Translatable fields store a separate value per language. Fields marked “Global” apply to every language.
                  </span>
                </div>
              ) : (
                <span style={{ fontSize: 11.5, color: "var(--fg-muted)", lineHeight: 1.45, letterSpacing: "-0.005em" }}>
                  These settings apply to every language.
                </span>
              )}

              <hr style={{ border: "none", borderTop: "1px solid var(--divider)", margin: "4px 0" }} />

              <Button
                onClick={handleSave}
                disabled={saving || !hasChanges}
                className="w-full"
              >
                <Save className="mr-1.5 h-3.5 w-3.5" />
                {saving ? "Saving…" : "Save changes"}
              </Button>

              {show_clear_cache && (
                <Button
                  variant="outline"
                  onClick={handleClearCache}
                  disabled={clearing}
                  className="w-full"
                >
                  <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${clearing ? "animate-spin" : ""}`} />
                  {clearing ? "Clearing…" : "Clear cache"}
                </Button>
              )}
            </div>
          </SidebarCard>
        </aside>
      </div>
    </div>
  );
}

function SchemaField({
  field,
  value,
  pages,
  roles,
  onChange,
}: {
  field: SettingsSchemaField;
  value: string;
  pages: ContentNode[];
  roles: Role[];
  onChange: (v: string) => void;
}) {
  const inputClasses =
    "rounded-lg focus:ring-2";

  // Always render translatability — operators repeatedly asked "did
  // it actually take effect?" and an absent badge isn't proof. On
  // all-global schemas the sidebar still says so up top; the badge
  // is redundant there but cheap, and it removes any "did the flag
  // wire through?" doubt.
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2">
        <Label className="text-sm font-medium text-foreground">{field.label}</Label>
        {field.translatable ? (
          <span
            className="rounded-full px-2 py-0.5 text-[10px] font-medium ring-1 ring-inset"
            style={{background: "var(--accent-weak)", color: "var(--accent-strong)", boxShadow: "inset 0 0 0 1px var(--accent-mid)"}}
            title="This field stores a separate value per language"
          >
            Translatable
          </span>
        ) : (
          <span
            className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground ring-1 ring-inset ring-border"
            title="This field applies to every language"
          >
            Global
          </span>
        )}
      </div>
      {field.type === "text" && (
        <Input
          placeholder={field.placeholder}
          value={value}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
          className={inputClasses}
        />
      )}
      {field.type === "textarea" && (
        <Textarea
          placeholder={field.placeholder}
          value={value}
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => onChange(e.target.value)}
          rows={field.rows ?? 4}
          className={`${inputClasses} resize-none ${field.font_mono ? "font-mono text-xs" : ""}`}
        />
      )}
      {field.type === "toggle" && (() => {
        const trueVal = field.true_value ?? "true";
        const falseVal = field.false_value ?? "false";
        const effective = value === "" ? (field.default ?? falseVal) : value;
        const checked = effective === trueVal;
        return (
          <div className="flex items-center gap-3 pt-1">
            <Switch
              checked={checked}
              onCheckedChange={(v: boolean) => onChange(v ? trueVal : falseVal)}
            />
            <span className="text-xs text-muted-foreground">
              {checked ? "On" : "Off"}
            </span>
          </div>
        );
      })()}
      {field.type === "node_select" && (
        <Select
          value={value || "__none__"}
          onValueChange={(v) => onChange(v === "__none__" ? "" : v)}
        >
          <SelectTrigger className={inputClasses}>
            <SelectValue placeholder={field.placeholder ?? "Select..."} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">{field.empty_label ?? "None"}</SelectItem>
            {pages.map((p) => (
              <SelectItem key={p.id} value={String(p.id)}>
                {p.title} ({p.full_url}) [{p.language_code.toUpperCase()}]
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {field.type === "role_select" && (
        <Select value={value || ""} onValueChange={onChange}>
          <SelectTrigger className={inputClasses}>
            <SelectValue placeholder={field.placeholder ?? "Select a role..."} />
          </SelectTrigger>
          <SelectContent>
            {roles.map((r) => (
              <SelectItem key={r.id} value={r.slug}>
                {r.name} ({r.slug})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {field.type === "select" && (
        <Select value={value || ""} onValueChange={onChange}>
          <SelectTrigger className={inputClasses}>
            <SelectValue placeholder={field.placeholder ?? "Select..."} />
          </SelectTrigger>
          <SelectContent>
            {(field.options ?? []).map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {field.help && <p className="text-[11px]" style={{color: "var(--fg-subtle)"}}>{field.help}</p>}
      {field.warning && (
        <div className="flex gap-2 rounded-md border px-2.5 py-1.5" style={{borderColor: "var(--warning)", background: "var(--warning-bg)"}}>
          <span className="text-[11px] leading-tight" style={{color: "var(--warning)"}} aria-hidden="true">⚠</span>
          <p className="text-[11px] leading-snug" style={{color: "var(--warning)"}}>{field.warning}</p>
        </div>
      )}
    </div>
  );
}
