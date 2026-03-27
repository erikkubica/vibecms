import { useState } from "react";
import { Settings, Send, Loader2, Puzzle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { toast } from "sonner";
import { sendTestEmail } from "@/api/client";
import { ExtensionSlot } from "@/components/extension-slot";

export default function EmailSettingsPage() {
  const [testing, setTesting] = useState(false);

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

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Settings className="h-7 w-7 text-indigo-600" />
          <h1 className="text-2xl font-bold text-slate-900">Email Settings</h1>
        </div>
        <Button
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
      </div>

      {/* Extension-provided settings */}
      <ExtensionSlot
        name="email-settings"
        fallback={
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <CardHeader>
              <CardTitle className="text-lg font-semibold text-slate-900">
                No Email Provider
              </CardTitle>
              <CardDescription>
                Install and activate an email provider extension to enable email sending.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col items-center justify-center gap-3 py-8 text-slate-400">
              <Puzzle className="h-12 w-12" />
              <p className="text-sm text-center max-w-md">
                Go to <strong>Extensions</strong> and activate an email provider
                (SMTP or Resend) to configure email delivery.
              </p>
            </CardContent>
          </Card>
        }
      />
    </div>
  );
}
