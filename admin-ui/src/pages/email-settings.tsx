import { useEffect, useState, type FormEvent } from "react";
import { Settings, Send, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { toast } from "sonner";
import {
  getEmailSettings,
  saveEmailSettings,
  sendTestEmail,
} from "@/api/client";

type Provider = "" | "smtp" | "resend";

export default function EmailSettingsPage() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  // Provider
  const [provider, setProvider] = useState<Provider>("");

  // SMTP fields
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("");
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpFromEmail, setSmtpFromEmail] = useState("");
  const [smtpFromName, setSmtpFromName] = useState("");

  // Resend fields
  const [resendApiKey, setResendApiKey] = useState("");
  const [resendFromEmail, setResendFromEmail] = useState("");
  const [resendFromName, setResendFromName] = useState("");

  async function fetchSettings() {
    try {
      const data = await getEmailSettings();
      const p = (data.provider || "") as Provider;
      setProvider(p);

      if (p === "smtp") {
        setSmtpHost(data.smtp_host || "");
        setSmtpPort(data.smtp_port || "");
        setSmtpUsername(data.smtp_username || "");
        setSmtpPassword(data.smtp_password || "");
        setSmtpFromEmail(data.from_email || "");
        setSmtpFromName(data.from_name || "");
      } else if (p === "resend") {
        setResendApiKey(data.resend_api_key || "");
        setResendFromEmail(data.from_email || "");
        setResendFromName(data.from_name || "");
      }
    } catch {
      toast.error("Failed to load email settings");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchSettings();
  }, []);

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    const data: Record<string, string> = { provider };

    if (provider === "smtp") {
      data.smtp_host = smtpHost;
      data.smtp_port = smtpPort;
      data.smtp_username = smtpUsername;
      data.smtp_password = smtpPassword;
      data.from_email = smtpFromEmail;
      data.from_name = smtpFromName;
    } else if (provider === "resend") {
      data.resend_api_key = resendApiKey;
      data.from_email = resendFromEmail;
      data.from_name = resendFromName;
    }

    setSaving(true);
    try {
      await saveEmailSettings(data);
      toast.success("Email settings saved successfully");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save email settings";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleTestEmail() {
    setTesting(true);
    try {
      await sendTestEmail();
      toast.success("Test email sent successfully");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to send test email";
      toast.error(message);
    } finally {
      setTesting(false);
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
      <div className="flex items-center gap-3">
        <Settings className="h-7 w-7 text-indigo-600" />
        <h1 className="text-2xl font-bold text-slate-900">Email Settings</h1>
      </div>

      <form onSubmit={handleSave} className="space-y-6 max-w-2xl">
        {/* Provider Picker */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-lg font-semibold text-slate-900">
              Email Provider
            </CardTitle>
            <CardDescription>
              Choose how VibeCMS sends transactional emails.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-3">
              <label className="flex items-center gap-3 cursor-pointer rounded-lg border border-slate-200 p-3 hover:bg-slate-50 transition-colors">
                <input
                  type="radio"
                  name="provider"
                  value=""
                  checked={provider === ""}
                  onChange={() => setProvider("")}
                  className="h-4 w-4 text-indigo-600 focus:ring-indigo-500"
                />
                <div>
                  <span className="text-sm font-medium text-slate-800">None</span>
                  <p className="text-xs text-slate-500">Email sending is disabled</p>
                </div>
              </label>
              <label className="flex items-center gap-3 cursor-pointer rounded-lg border border-slate-200 p-3 hover:bg-slate-50 transition-colors">
                <input
                  type="radio"
                  name="provider"
                  value="smtp"
                  checked={provider === "smtp"}
                  onChange={() => setProvider("smtp")}
                  className="h-4 w-4 text-indigo-600 focus:ring-indigo-500"
                />
                <div>
                  <span className="text-sm font-medium text-slate-800">SMTP</span>
                  <p className="text-xs text-slate-500">Connect via SMTP server</p>
                </div>
              </label>
              <label className="flex items-center gap-3 cursor-pointer rounded-lg border border-slate-200 p-3 hover:bg-slate-50 transition-colors">
                <input
                  type="radio"
                  name="provider"
                  value="resend"
                  checked={provider === "resend"}
                  onChange={() => setProvider("resend")}
                  className="h-4 w-4 text-indigo-600 focus:ring-indigo-500"
                />
                <div>
                  <span className="text-sm font-medium text-slate-800">Resend</span>
                  <p className="text-xs text-slate-500">Send via Resend API</p>
                </div>
              </label>
            </div>
          </CardContent>
        </Card>

        {/* Provider Config */}
        {provider === "" && (
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardContent className="p-6">
              <p className="text-sm text-slate-500">
                Email sending is disabled. Select a provider above to enable email delivery.
              </p>
            </CardContent>
          </Card>
        )}

        {provider === "smtp" && (
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">
                SMTP Configuration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="smtp-host" className="text-sm font-medium text-slate-700">Host</Label>
                  <Input
                    id="smtp-host"
                    placeholder="smtp.example.com"
                    value={smtpHost}
                    onChange={(e) => setSmtpHost(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-port" className="text-sm font-medium text-slate-700">Port</Label>
                  <Input
                    id="smtp-port"
                    placeholder="587"
                    value={smtpPort}
                    onChange={(e) => setSmtpPort(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="smtp-user" className="text-sm font-medium text-slate-700">Username</Label>
                  <Input
                    id="smtp-user"
                    placeholder="user@example.com"
                    value={smtpUsername}
                    onChange={(e) => setSmtpUsername(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-pass" className="text-sm font-medium text-slate-700">Password</Label>
                  <Input
                    id="smtp-pass"
                    type="password"
                    placeholder="********"
                    value={smtpPassword}
                    onChange={(e) => setSmtpPassword(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="smtp-from-email" className="text-sm font-medium text-slate-700">From Email</Label>
                  <Input
                    id="smtp-from-email"
                    placeholder="noreply@example.com"
                    value={smtpFromEmail}
                    onChange={(e) => setSmtpFromEmail(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-from-name" className="text-sm font-medium text-slate-700">From Name</Label>
                  <Input
                    id="smtp-from-name"
                    placeholder="My Site"
                    value={smtpFromName}
                    onChange={(e) => setSmtpFromName(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {provider === "resend" && (
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">
                Resend Configuration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="resend-key" className="text-sm font-medium text-slate-700">API Key</Label>
                <Input
                  id="resend-key"
                  type="password"
                  placeholder="re_..."
                  value={resendApiKey}
                  onChange={(e) => setResendApiKey(e.target.value)}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="resend-from-email" className="text-sm font-medium text-slate-700">From Email</Label>
                  <Input
                    id="resend-from-email"
                    placeholder="noreply@example.com"
                    value={resendFromEmail}
                    onChange={(e) => setResendFromEmail(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="resend-from-name" className="text-sm font-medium text-slate-700">From Name</Label>
                  <Input
                    id="resend-from-name"
                    placeholder="My Site"
                    value={resendFromName}
                    onChange={(e) => setResendFromName(e.target.value)}
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Actions */}
        <div className="flex items-center gap-3">
          <Button
            type="submit"
            className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg"
            disabled={saving}
          >
            {saving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              "Save Settings"
            )}
          </Button>
          {provider && (
            <Button
              type="button"
              variant="outline"
              onClick={handleTestEmail}
              disabled={testing}
              className="rounded-lg border-slate-300"
            >
              {testing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Sending...
                </>
              ) : (
                <>
                  <Send className="mr-2 h-4 w-4" />
                  Send Test Email
                </>
              )}
            </Button>
          )}
        </div>
      </form>
    </div>
  );
}
