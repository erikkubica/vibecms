import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";
import App from "@/App";
import "./index.css";

// Expose shared libraries for extension micro-frontends
import * as React from "react";
import * as ReactDOM from "react-dom/client";
import * as ReactRouterDOM from "react-router-dom";
import * as Sonner from "sonner";
import * as LucideReact from "lucide-react";

// shadcn/ui components
import * as ButtonModule from "@/components/ui/button";
import * as CardModule from "@/components/ui/card";
import * as InputModule from "@/components/ui/input";
import * as LabelModule from "@/components/ui/label";
import * as BadgeModule from "@/components/ui/badge";
import * as DialogModule from "@/components/ui/dialog";
import * as SelectModule from "@/components/ui/select";
import * as SwitchModule from "@/components/ui/switch";
import * as TextareaModule from "@/components/ui/textarea";

// API client
import * as apiClient from "@/api/client";

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
    ...ButtonModule,
    ...CardModule,
    ...InputModule,
    ...LabelModule,
    ...BadgeModule,
    ...DialogModule,
    ...SelectModule,
    ...SwitchModule,
    ...TextareaModule,
  },
  api: apiClient,
};

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
      <Toaster position="top-right" richColors />
    </BrowserRouter>
  </StrictMode>
);
