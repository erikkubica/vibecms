import React from "react";
import {
  CheckCircle2,
  MousePointer2,
  Globe,
  Shield,
  Database,
  Lock,
  Send,
} from "@vibecms/icons";

const {
  Card,
  CardContent,
  SectionHeader,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
} = (window as any).__VIBECMS_SHARED__.ui;

export default function SettingsTab({ form, setForm }: any) {
  const settings = form.settings || {};

  const getSetting = (key: string, defaultValue: any) =>
    settings[key] !== undefined ? settings[key] : defaultValue;

  const updateSettings = (key: string, value: any) => {
    setForm({ ...form, settings: { ...form.settings, [key]: value } });
  };

  const captchaProvider = getSetting("captcha_provider", "none");

  return (
    <div className="space-y-6 max-w-2xl">
      {/* General */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <SectionHeader title="General" icon={<CheckCircle2 className="h-4 w-4 text-emerald-500" />} />
        <CardContent className="p-4 space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4">
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-slate-500">Success Message</Label>
              <Input
                value={getSetting("success_message", "")}
                onChange={(e: any) => updateSettings("success_message", e.target.value)}
                placeholder="Thank you for your submission!"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-slate-500">Error Message</Label>
              <Input
                value={getSetting("error_message", "")}
                onChange={(e: any) => updateSettings("error_message", e.target.value)}
                placeholder="There was an error processing your request."
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Redirects */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <SectionHeader title="Redirects" icon={<Globe className="h-4 w-4 text-blue-500" />} />
        <CardContent className="p-4 space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Redirect URL (Optional)</Label>
          <Input
            value={getSetting("redirect_url", "")}
            onChange={(e: any) => updateSettings("redirect_url", e.target.value)}
            placeholder="e.g. /thank-you"
            className="max-w-2xl"
          />
          <p className="text-xs text-slate-400">
            Leave empty to stay on the same page and show the success message.
          </p>
        </CardContent>
      </Card>

      {/* Styling */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <SectionHeader title="Styling" icon={<MousePointer2 className="h-4 w-4 text-indigo-500" />} />
        <CardContent className="p-4 space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Form CSS Class</Label>
          <Input
            value={getSetting("form_css_class", "")}
            onChange={(e: any) => updateSettings("form_css_class", e.target.value)}
            placeholder="e.g. my-newsletter dark-form"
            className="max-w-2xl"
          />
        </CardContent>
      </Card>

      {/* Anti-spam */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <SectionHeader title="Anti-spam" icon={<Shield className="h-4 w-4 text-amber-500" />} />
        <CardContent className="p-4 space-y-4">
          <div className="flex items-center justify-between py-2">
            <div>
              <Label className="text-sm font-medium text-slate-700">Honeypot Field</Label>
              <p className="text-xs text-slate-400 mt-0.5">
                Adds a hidden trap field that bots fill in. Legitimate users never see it.
              </p>
            </div>
            <Switch
              checked={getSetting("honeypot_enabled", true)}
              onCheckedChange={(checked: boolean) =>
                updateSettings("honeypot_enabled", checked)
              }
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4">
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-slate-500">Rate Limit (submissions)</Label>
              <Input
                type="number"
                inputMode="numeric"
                min={1}
                value={getSetting("rate_limit", 10)}
                onChange={(e: any) => {
                  const val = parseInt(e.target.value, 10);
                  if (!isNaN(val) && val >= 1) updateSettings("rate_limit", val);
                }}
                placeholder="10"
                className="max-w-[8rem]"
              />
              <p className="text-xs text-slate-400">Per IP per window</p>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-slate-500">Window (seconds)</Label>
              <Input
                type="number"
                inputMode="numeric"
                min={1}
                value={getSetting("rate_limit_window", 3600)}
                onChange={(e: any) => {
                  const val = parseInt(e.target.value, 10);
                  if (!isNaN(val) && val >= 1) updateSettings("rate_limit_window", val);
                }}
                placeholder="3600"
                className="max-w-[8rem]"
              />
              <p className="text-xs text-slate-400">Default: 3600 (1 hour)</p>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-500">CAPTCHA Provider</Label>
            <Select
              value={captchaProvider}
              onValueChange={(val: string) => updateSettings("captcha_provider", val)}
            >
              <SelectTrigger className="max-w-xs">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">None</SelectItem>
                <SelectItem value="recaptcha">reCAPTCHA v3</SelectItem>
                <SelectItem value="hcaptcha">hCaptcha</SelectItem>
                <SelectItem value="turnstile">Cloudflare Turnstile</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {captchaProvider !== "none" && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4 pl-3 border-l-2 border-slate-200">
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">CAPTCHA Site Key</Label>
                <Input
                  value={getSetting("captcha_site_key", "")}
                  onChange={(e: any) => updateSettings("captcha_site_key", e.target.value)}
                  placeholder="Enter your site key"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">CAPTCHA Secret Key</Label>
                <Input
                  type="password"
                  value={getSetting("captcha_secret_key", "")}
                  onChange={(e: any) => updateSettings("captcha_secret_key", e.target.value)}
                  placeholder="Enter your secret key"
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Privacy */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <SectionHeader title="Privacy" icon={<Lock className="h-4 w-4 text-teal-500" />} />
        <CardContent className="p-4 space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Privacy Policy URL</Label>
          <Input
            value={getSetting("privacy_policy_url", "")}
            onChange={(e: any) => updateSettings("privacy_policy_url", e.target.value)}
            placeholder="e.g. /privacy-policy"
            className="max-w-2xl"
          />
          <p className="text-xs text-slate-400">Used by GDPR consent fields.</p>
        </CardContent>
      </Card>

      {/* Data Retention */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <SectionHeader title="Data Retention" icon={<Database className="h-4 w-4 text-violet-500" />} />
        <CardContent className="p-4 space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Retention Period</Label>
          <Select
            value={String(getSetting("retention_period", "0"))}
            onValueChange={(val: string) => updateSettings("retention_period", val)}
          >
            <SelectTrigger className="max-w-[12rem]">
              <SelectValue placeholder="Select retention period" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">Keep forever</SelectItem>
              <SelectItem value="30">30 days</SelectItem>
              <SelectItem value="60">60 days</SelectItem>
              <SelectItem value="90">90 days</SelectItem>
              <SelectItem value="180">180 days</SelectItem>
              <SelectItem value="365">365 days</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-xs text-slate-400">
            Submissions older than the retention period will be automatically deleted.
          </p>
        </CardContent>
      </Card>

      {/* Submission Settings */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <SectionHeader title="Submission Settings" icon={<Send className="h-4 w-4 text-cyan-500" />} />
        <CardContent className="p-4">
          <div className="flex items-center justify-between py-2">
            <div>
              <Label className="text-sm font-medium text-slate-700">Store IP Addresses</Label>
              <p className="text-xs text-slate-400 mt-0.5">Disable for strict GDPR compliance.</p>
            </div>
            <Switch
              checked={getSetting("store_ip", true)}
              onCheckedChange={(checked: boolean) => updateSettings("store_ip", checked)}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
