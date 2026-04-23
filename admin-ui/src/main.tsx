import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, useNavigate } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
import App from "@/App";
import { queryClient } from "@/sdui/query-client";
import { useSSE } from "@/hooks/use-sse";
import { registerBuiltinComponents } from "@/sdui/register-builtins";
import { setNavigate } from "@/sdui/action-handler";
import "./index.css";

// Register SDUI built-in components before any rendering.
registerBuiltinComponents();

// Expose shared libraries for extension micro-frontends
import * as React from "react";
import * as ReactDOM from "react-dom/client";
import * as ReactRouterDOM from "react-router-dom";
import * as Sonner from "sonner";
import * as LucideReact from "lucide-react";

// shadcn/ui components — explicit named imports for reliable sharing
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
  CardAction,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableFooter,
  TableCaption,
} from "@/components/ui/table";
import { Switch } from "@/components/ui/switch";

// API client
import * as apiClient from "@/api/client";

// Host components for extension slot rendering
import { ExtensionSlot } from "@/components/extension-slot";
import { useExtensions } from "@/hooks/use-extensions";

declare global {
  interface Window {
    __VIBECMS_SHARED__: Record<string, unknown>;
  }
}

window.__VIBECMS_SHARED__ = {
  React,
  ReactDOM,
  ReactRouterDOM,
  Sonner,
  icons: LucideReact,
  ui: {
    Button,
    Card,
    CardContent,
    CardHeader,
    CardTitle,
    CardDescription,
    CardFooter,
    CardAction,
    Input,
    Label,
    Badge,
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Tabs,
    TabsList,
    TabsTrigger,
    TabsContent,
    Textarea,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
    TableFooter,
    TableCaption,
    Switch,
  },
  api: apiClient,
  ExtensionSlot,
  useExtensions,
};

// ---------------------------------------------------------------------------
// SDUI integration components — must live inside BrowserRouter so they can
// call useNavigate() from the router context.
// ---------------------------------------------------------------------------

/** Reads navigate from router context and wires it into the SDUI action handler. */
function NavigateBridge() {
  const navigate = useNavigate();
  React.useEffect(() => {
    setNavigate(navigate);
  }, [navigate]);
  return null;
}

/** Activates the SSE connection for real-time state invalidation. */
function SduiProviders({ children }: { children: React.ReactNode }) {
  useSSE();
  return <>{children}</>;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <NavigateBridge />
        <SduiProviders>
          <App />
          <Toaster position="top-right" richColors />
        </SduiProviders>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
