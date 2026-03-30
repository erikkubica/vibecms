import { useState, useEffect } from "react";
import { Settings, Send, Loader2, Puzzle, Check, Power } from "@vibecms/icons";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  Badge,
} from "@vibecms/ui";
import { toast } from "sonner";
import { getEmailSettings, saveEmailSettings, sendTestEmail } from "@vibecms/api";

interface ExtensionManifest {
  slug: string;
  name: string;
  manifest: {
    provides?: string[];
  };
}

function getShared() {
  return (window as any).__VIBECMS_SHARED__;
}

function ProviderSettingsInner({ providerSlug, useExtensions }: { providerSlug: string; useExtensions: () => any }) {
  const { getSlotExtensions, loading } = useExtensions();

  if (loading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
      </div>
    );
  }

  const extensions = getSlotExtensions("email-settings");
  const match = extensions.find(
    (ext: { slug: string }) => ext.slug === providerSlug
  );

  if (!match) return null;

  const Component = match.Component;
  return <Component />;
}

function ProviderSettings({ providerSlug }: { providerSlug: string }) {
  const shared = getShared();
  const useExtensions = shared?.useExtensions;
  if (!useExtensions) return null;

  return <ProviderSettingsInner providerSlug={providerSlug} useExtensions={useExtensions} />;
}

export default function EmailSettings() {
  const [testing, setTesting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [activating, setActivating] = useState<string | null>(null);
  const [providers, setProviders] = useState<ExtensionManifest[]>([]);
  const [activeProvider, setActiveProvider] = useState("");

  useEffect(() => {
    async function load() {
      try {
        const data = await getEmailSettings();
        // GetSettings strips the "email_" prefix from keys
        setActiveProvider(data.provider || "");

        const res = await fetch("/admin/api/extensions/manifests", {
          credentials: "include",
        });
        if (res.ok) {
          const json = await res.json();
          const allExts: ExtensionManifest[] = json.data || [];
          setProviders(
            allExts.filter((ext) =>
              ext.manifest.provides?.includes("email.provider")
            )
          );
        }
      } catch {
        toast.error("Failed to load email settings");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  async function handleActivate(slug: string) {
    setActivating(slug);
    try {
      await saveEmailSettings({ email_provider: slug });
      setActiveProvider(slug);
      const name = providers.find((p) => p.slug === slug)?.name || slug;
      toast.success(`${name} is now the active email provider`);
    } catch {
      toast.error("Failed to activate provider");
    } finally {
      setActivating(null);
    }
  }

  async function handleDeactivate() {
    setActivating("__none__");
    try {
      await saveEmailSettings({ email_provider: "" });
      setActiveProvider("");
      toast.success("Email delivery disabled");
    } catch {
      toast.error("Failed to deactivate provider");
    } finally {
      setActivating(null);
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
          disabled={testing || !activeProvider}
          className="rounded-lg border-slate-300"
          title={!activeProvider ? "Activate a provider first" : ""}
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

      {/* No providers */}
      {providers.length === 0 && (
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardContent className="flex flex-col items-center justify-center gap-3 py-10 text-slate-400">
            <Puzzle className="h-10 w-10" />
            <p className="text-sm text-center max-w-md">
              No email provider extensions are active. Go to{" "}
              <strong>Extensions</strong> and activate SMTP or Resend provider.
            </p>
          </CardContent>
        </Card>
      )}

      {/* Provider cards */}
      {providers.map((provider) => {
        const isActive = activeProvider === provider.slug;
        return (
          <Card
            key={provider.slug}
            className={`rounded-xl border shadow-sm transition-colors ${
              isActive
                ? "border-green-300 bg-green-50/30"
                : "border-slate-200"
            }`}
          >
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <CardTitle className="text-lg font-semibold text-slate-900">
                    {provider.name}
                  </CardTitle>
                  {isActive && (
                    <Badge className="bg-green-100 text-green-700 hover:bg-green-100 border-0 text-xs">
                      <Check className="mr-1 h-3 w-3" />
                      Active
                    </Badge>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {isActive ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleDeactivate}
                      disabled={activating !== null}
                      className="rounded-lg border-slate-300 text-slate-600 hover:text-red-600 hover:border-red-300"
                    >
                      {activating === "__none__" ? (
                        <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                      ) : (
                        <Power className="mr-1.5 h-3.5 w-3.5" />
                      )}
                      Deactivate
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      onClick={() => handleActivate(provider.slug)}
                      disabled={activating !== null}
                      className="rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white"
                    >
                      {activating === provider.slug ? (
                        <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                      ) : (
                        <Power className="mr-1.5 h-3.5 w-3.5" />
                      )}
                      Set as Active
                    </Button>
                  )}
                </div>
              </div>
              <CardDescription>
                {isActive
                  ? "This provider is handling outgoing emails. Configure it below."
                  : "Configure this provider, then activate it when ready."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ProviderSettings providerSlug={provider.slug} />
            </CardContent>
          </Card>
        );
      })}

    </div>
  );
}
