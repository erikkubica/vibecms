import React from "react";
import {
  CheckCircle2,
  XCircle,
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

  const getSetting = (key: string, defaultValue: any) => {
    return settings[key] !== undefined ? settings[key] : defaultValue;
  };

  const updateSettings = (key: string, value: any) => {
    setForm({
      ...form,
      settings: { ...form.settings, [key]: value },
    });
  };

  const captchaProvider = getSetting("captcha_provider", "none");

  return (
    <div className="space-y-6 max-w-2xl">
      <Card className="border-slate-200 shadow-none">
        <CardContent className="p-4 space-y-6">
          <div className="space-y-4">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-green-500" />
              Success & Error Messages
            </h3>
            <div className="space-y-2">
              <Label>Success Message</Label>
              <Input
                value={getSetting("success_message", "")}
                onChange={(e: any) =>
                  updateSettings("success_message", e.target.value)
                }
                placeholder="Thank you for your submission!"
              />
            </div>
            <div className="space-y-2">
              <Label>Error Message</Label>
              <Input
                value={getSetting("error_message", "")}
                onChange={(e: any) =>
                  updateSettings("error_message", e.target.value)
                }
                placeholder="There was an error processing your request."
              />
            </div>
          </div>

          <div className="space-y-4 pt-6 border-t border-slate-100">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <MousePointer2 className="h-5 w-5 text-indigo-500" />
              Button Settings
            </h3>
            <div className="space-y-2">
              <Label>Submit Button Text</Label>
              <Input
                value={getSetting("submit_button_text", "")}
                onChange={(e: any) =>
                  updateSettings("submit_button_text", e.target.value)
                }
                placeholder="Submit"
              />
            </div>
          </div>

          <div className="space-y-4 pt-6 border-t border-slate-100">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Globe className="h-5 w-5 text-blue-500" />
              Redirects
            </h3>
            <div className="space-y-2">
              <Label>Redirect URL (Optional)</Label>
              <Input
                value={getSetting("redirect_url", "")}
                onChange={(e: any) =>
                  updateSettings("redirect_url", e.target.value)
                }
                placeholder="e.g. /thank-you"
              />
              <p className="text-xs text-slate-400 italic">
                Leave empty to stay on the same page and show success message.
              </p>
            </div>
          </div>

          <div className="space-y-4 pt-6 border-t border-slate-100">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Shield className="h-5 w-5 text-amber-500" />
              Spam Protection
            </h3>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm">Honeypot Field</Label>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Adds a hidden trap field that bots fill in. Legitimate users
                    never see it.
                  </p>
                </div>
                <Switch
                  checked={getSetting("honeypot_enabled", true)}
                  onCheckedChange={(checked: boolean) =>
                    updateSettings("honeypot_enabled", checked)
                  }
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Rate Limit (submissions per IP per hour)</Label>
              <Input
                type="number"
                min={1}
                value={getSetting("rate_limit", 10)}
                onChange={(e: any) => {
                  const val = parseInt(e.target.value, 10);
                  if (!isNaN(val) && val >= 1) {
                    updateSettings("rate_limit", val);
                  }
                }}
                placeholder="10"
              />
            </div>
            <div className="space-y-2">
              <Label>CAPTCHA Provider</Label>
              <Select
                value={captchaProvider}
                onValueChange={(val: string) =>
                  updateSettings("captcha_provider", val)
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select provider" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">None</SelectItem>
                  <SelectItem value="recaptcha">reCAPTCHA v3</SelectItem>
                  <SelectItem value="hcaptcha">hCaptcha</SelectItem>
                  <SelectItem value="turnstile">
                    Cloudflare Turnstile
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            {captchaProvider !== "none" && (
              <div className="space-y-4 pl-1 border-l-2 border-slate-100 ml-1">
                <div className="space-y-2 pl-3">
                  <Label>CAPTCHA Site Key</Label>
                  <Input
                    value={getSetting("captcha_site_key", "")}
                    onChange={(e: any) =>
                      updateSettings("captcha_site_key", e.target.value)
                    }
                    placeholder="Enter your site key"
                  />
                </div>
                <div className="space-y-2 pl-3">
                  <Label>CAPTCHA Secret Key</Label>
                  <Input
                    type="password"
                    value={getSetting("captcha_secret_key", "")}
                    onChange={(e: any) =>
                      updateSettings("captcha_secret_key", e.target.value)
                    }
                    placeholder="Enter your secret key"
                  />
                </div>
              </div>
            )}
          </div>

          <div className="space-y-4 pt-6 border-t border-slate-100">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Database className="h-5 w-5 text-purple-500" />
              Data Retention
            </h3>
            <div className="space-y-2">
              <Label>Retention Period</Label>
              <Select
                value={String(getSetting("retention_period", "0"))}
                onValueChange={(val: string) =>
                  updateSettings("retention_period", val)
                }
              >
                <SelectTrigger>
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
              <p className="text-xs text-slate-400 italic">
                Submissions older than the retention period will be
                automatically deleted.
              </p>
            </div>
          </div>

          <div className="space-y-4 pt-6 border-t border-slate-100">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Lock className="h-5 w-5 text-teal-500" />
              Privacy
            </h3>
            <div className="space-y-2">
              <Label>Privacy Policy URL</Label>
              <Input
                value={getSetting("privacy_policy_url", "")}
                onChange={(e: any) =>
                  updateSettings("privacy_policy_url", e.target.value)
                }
                placeholder="e.g. /privacy-policy"
              />
              <p className="text-xs text-slate-400 italic">
                Used by GDPR consent fields.
              </p>
            </div>
          </div>

          <div className="space-y-4 pt-6 border-t border-slate-100">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Send className="h-5 w-5 text-cyan-500" />
              Submission Settings
            </h3>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm">Store IP Addresses</Label>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Disable for strict GDPR compliance.
                  </p>
                </div>
                <Switch
                  checked={getSetting("store_ip", true)}
                  onCheckedChange={(checked: boolean) =>
                    updateSettings("store_ip", checked)
                  }
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
