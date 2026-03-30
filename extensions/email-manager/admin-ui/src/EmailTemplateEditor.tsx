import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { ArrowLeft, Save, Loader2, Eye } from "@vibecms/icons";
import {
  Button,
  Input,
  Label,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
} from "@vibecms/ui";
import { toast } from "sonner";
import {
  getEmailTemplate,
  createEmailTemplate,
  updateEmailTemplate,
  getLanguages,
} from "@vibecms/api";

interface EmailTemplate {
  id: number;
  slug: string;
  name: string;
  language_id: number | null;
  subject_template: string;
  body_template: string;
  test_data: Record<string, any>;
}

interface Language {
  id: number;
  code: string;
  name: string;
  flag: string;
}

function renderPreview(bodyTemplate: string, testData: Record<string, any>): string {
  let html = bodyTemplate;
  html = html.replace(/\{\{\.\s*(\w+)\s*\}\}/g, (_match: string, key: string) => {
    return testData[key] !== undefined ? String(testData[key]) : `{{.${key}}}`;
  });
  return html;
}

export default function EmailTemplateEditor() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id && id !== "new";

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);

  const [languages, setLanguages] = useState<Language[]>([]);
  const [baseLayout, setBaseLayout] = useState("");

  // Form state
  const [formSlug, setFormSlug] = useState("");
  const [formName, setFormName] = useState("");
  const [formLanguageId, setFormLanguageId] = useState<string>("__universal__");
  const [formSubject, setFormSubject] = useState("");
  const [formBody, setFormBody] = useState("");
  const [formTestData, setFormTestData] = useState("{}");

  useEffect(() => {
    let cancelled = false;
    getLanguages().then((langs: Language[]) => {
      if (!cancelled) setLanguages(langs);
    }).catch(() => {});
    fetch("/admin/api/ext/email-manager/layouts", { credentials: "include" })
      .then((res) => res.json())
      .then((json) => {
        if (cancelled) return;
        // Find universal layout (language_id is null) as default for preview.
        const layouts = json.data || [];
        const universal = layouts.find((l: any) => l.language_id === null);
        if (universal) setBaseLayout(universal.body_template);
      })
      .catch(() => {});

    if (!isEdit) return;
    setLoading(true);
    getEmailTemplate(Number(id))
      .then((tpl: EmailTemplate) => {
        if (cancelled) return;
        setFormSlug(tpl.slug);
        setFormName(tpl.name);
        setFormLanguageId(tpl.language_id ? String(tpl.language_id) : "__universal__");
        setFormSubject(tpl.subject_template);
        setFormBody(tpl.body_template);
        // test_data may come back as a JSON string or an object
        let td = tpl.test_data;
        if (typeof td === "string") {
          try { td = JSON.parse(td); } catch { td = {}; }
        }
        setFormTestData(JSON.stringify(td || {}, null, 2));
      })
      .catch(() => {
        toast.error("Failed to load email template");
        navigate("/admin/ext/email-manager/templates", { replace: true });
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, isEdit, navigate]);

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!formSlug.trim() || !formName.trim()) {
      toast.error("Slug and name are required");
      return;
    }

    if (!formSubject.trim()) {
      toast.error("Subject template is required");
      return;
    }

    let testData: Record<string, any> = {};
    try {
      testData = JSON.parse(formTestData);
    } catch {
      toast.error("Test data must be valid JSON");
      return;
    }

    const data: Partial<EmailTemplate> & { language_id?: number | null } = {
      slug: formSlug.trim(),
      name: formName.trim(),
      language_id: formLanguageId === "__universal__" ? null : Number(formLanguageId),
      subject_template: formSubject,
      body_template: formBody,
      test_data: testData,
    };

    setSaving(true);
    try {
      if (isEdit) {
        await updateEmailTemplate(Number(id), data);
        toast.success("Email template updated successfully");
      } else {
        const created = await createEmailTemplate(data);
        toast.success("Email template created successfully");
        navigate(`/admin/ext/email-manager/templates/${created.id}`, { replace: true });
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save email template";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  function getPreviewHtml(): string {
    try {
      const testData = JSON.parse(formTestData);
      let html = renderPreview(formBody, testData);
      if (baseLayout) {
        html = baseLayout
          .replace(/\{\{\s*\.email_body\s*\}\}/g, html)
          .replace(/\{\{\s*\.site\.site_name\s*\}\}/g, testData.site_name || "My Site")
          .replace(/\{\{\s*\.site\.site_url\s*\}\}/g, testData.site_url || "#");
      }
      return html;
    } catch {
      return formBody;
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild className="h-8 w-8">
            <Link to="/admin/ext/email-manager/templates">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">
            {isEdit ? "Edit Email Template" : "New Email Template"}
          </h1>
        </div>
        <Button
          onClick={handleSave}
          disabled={saving}
          className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
        >
          <Save className="mr-2 h-4 w-4" />
          {saving ? "Saving..." : "Save"}
        </Button>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Slug, Name, Subject */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base font-semibold text-slate-800">Template Details</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="space-y-2">
                <Label htmlFor="tpl-slug" className="text-sm font-medium text-slate-700">
                  Slug
                </Label>
                <Input
                  id="tpl-slug"
                  placeholder="e.g. welcome-email"
                  value={formSlug}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFormSlug(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tpl-name" className="text-sm font-medium text-slate-700">
                  Name
                </Label>
                <Input
                  id="tpl-name"
                  placeholder="e.g. Welcome Email"
                  value={formName}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFormName(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">Language</Label>
                <Select value={formLanguageId} onValueChange={setFormLanguageId}>
                  <SelectTrigger className="rounded-lg border-slate-300">
                    <SelectValue placeholder="Universal" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__universal__">Universal (fallback)</SelectItem>
                    {languages.map((lang) => (
                      <SelectItem key={lang.id} value={String(lang.id)}>
                        {lang.flag} {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="tpl-subject" className="text-sm font-medium text-slate-700">
                  Subject Template
                </Label>
                <Input
                  id="tpl-subject"
                  placeholder="e.g. Welcome to {{.site_name}}"
                  value={formSubject}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFormSubject(e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Split pane: Body Template + Preview */}
        <div className="grid grid-cols-2 gap-4">
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-base font-semibold text-slate-800">Body Template</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={formBody}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setFormBody(e.target.value)}
                className="min-h-[400px] font-mono text-sm rounded-lg border-slate-300"
                placeholder={"<div style=\"font-family: sans-serif;\">\n  <h2>Hello {{.user_full_name}}</h2>\n  <p>Welcome to {{.site_name}}</p>\n</div>"}
              />
            </CardContent>
          </Card>
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <div className="flex items-center gap-2">
                <Eye className="h-4 w-4 text-slate-500" />
                <CardTitle className="text-base font-semibold text-slate-800">Preview</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="h-96 rounded-lg border border-slate-300 bg-white overflow-auto">
                <iframe
                  srcDoc={getPreviewHtml()}
                  title="Email Preview"
                  className="w-full h-full border-0"
                  sandbox=""
                />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Test Data */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base font-semibold text-slate-800">Test Data (JSON)</CardTitle>
          </CardHeader>
          <CardContent>
            <Textarea
              value={formTestData}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setFormTestData(e.target.value)}
              className="min-h-[150px] font-mono text-sm rounded-lg border-slate-300"
              placeholder='{"user_full_name": "John", "site_name": "My Site"}'
            />
          </CardContent>
        </Card>
      </form>
    </div>
  );
}
