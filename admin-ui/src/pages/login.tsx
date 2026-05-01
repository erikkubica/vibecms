import { useState, type FormEvent } from "react";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { toast } from "sonner";

export default function LoginPage() {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    try {
      await login(email, password);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Login failed. Please try again.";
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4" style={{ background: "var(--app-bg)" }}>
      <Card className="w-full max-w-md">
        <CardHeader className="text-center pb-2">
          <div
            className="mx-auto mb-4 flex h-12 w-12 items-center justify-center"
            style={{
              borderRadius: 10,
              background: "var(--accent)",
              color: "var(--accent-fg)",
              fontFamily: "var(--font-mono)",
              fontSize: 18,
              fontWeight: 600,
            }}
          >
            S
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight text-foreground">
            Squilla
          </CardTitle>
          <CardDescription className="text-muted-foreground">
            Sign in to your account
          </CardDescription>
        </CardHeader>
        <CardContent className="p-8 pt-4">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email" className="text-foreground text-sm font-medium">
                Email
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="admin@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoComplete="email"
                className="w-full px-4 py-3 rounded-lg border border-border bg-card text-foreground placeholder:text-muted-foreground focus:ring-2 text-sm"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password" className="text-foreground text-sm font-medium">
                Password
              </Label>
              <Input
                id="password"
                type="password"
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoComplete="current-password"
                className="w-full px-4 py-3 rounded-lg border border-border bg-card text-foreground placeholder:text-muted-foreground focus:ring-2 text-sm"
              />
            </div>
            <Button
              type="submit"
              className="w-full py-3 bg-primary text-white rounded-lg font-semibold shadow-sm"
              disabled={submitting}
            >
              {submitting ? "Signing in..." : "Sign in"}
            </Button>
          </form>

          <div className="mt-6 flex items-center justify-between text-sm">
            <a
              href="/register"
              className="hover: transition-colors font-medium" style={{color: "var(--accent-strong)"}}
            >
              Create account
            </a>
            <a
              href="/auth/forgot-password"
              className="hover: transition-colors font-medium" style={{color: "var(--accent-strong)"}}
            >
              Forgot password?
            </a>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
