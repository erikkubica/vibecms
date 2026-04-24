# Extension System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make VibeCMS extensible via dynamic React micro-frontends and extension-driven email providers (SMTP + Resend), replacing hardcoded email provider logic.

**Architecture:** Extensions declare UI components, routes, menu items, and settings in `extension.json`. The Go backend serves extension assets and settings. The React admin dynamically loads extension bundles at runtime via import maps and shared globals, wrapping each in ErrorBoundary for crash isolation. Email providers move from hardcoded Go factories to extension-driven resolution.

**Tech Stack:** Go/Fiber (backend APIs), React/TypeScript (admin UI micro-frontends), Vite (extension builds), ES import maps (shared dependency resolution)

**Spec:** `docs/superpowers/specs/2026-03-27-extension-system-design.md`

---

## Task 1: Extend Extension Model with Manifest JSONB Column

**Files:**
- Modify: `internal/models/extension.go`
- Modify: `internal/db/migrations.go` (auto-migrate picks it up)

- [ ] **Step 1: Add Manifest field to Extension model**

In `internal/models/extension.go`, add the `Manifest` field after `Settings`:

```go
// Extension represents an installed CMS extension (plugin).
type Extension struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Slug        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	Name        string    `gorm:"type:varchar(150);not null" json:"name"`
	Version     string    `gorm:"type:varchar(50);not null;default:'1.0.0'" json:"version"`
	Description string    `gorm:"type:text;not null;default:''" json:"description"`
	Author      string    `gorm:"type:varchar(150);not null;default:''" json:"author"`
	Path        string    `gorm:"type:text;not null" json:"path"`
	IsActive    bool      `gorm:"not null;default:false" json:"is_active"`
	Priority    int       `gorm:"not null;default:50" json:"priority"`
	Settings    JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	Manifest    JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"manifest"`
	InstalledAt time.Time `gorm:"autoCreateTime" json:"installed_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
```

- [ ] **Step 2: Verify the app compiles and auto-migration adds the column**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles successfully. On next startup, GORM auto-migrates the `manifest` column.

- [ ] **Step 3: Commit**

```bash
git add internal/models/extension.go
git commit -m "feat: add Manifest JSONB column to Extension model"
```

---

## Task 2: Extend ExtensionManifest and Loader to Parse Full Manifest

**Files:**
- Modify: `internal/cms/extension_loader.go`

- [ ] **Step 1: Expand ExtensionManifest struct to include all new fields**

Replace the `ExtensionManifest` struct in `internal/cms/extension_loader.go`:

```go
// AdminUISlot describes a component injected into a named slot.
type AdminUISlot struct {
	Component string `json:"component"`
	Label     string `json:"label"`
}

// AdminUIRoute describes a route registered by an extension.
type AdminUIRoute struct {
	Path      string `json:"path"`
	Component string `json:"component"`
}

// AdminUIMenuItem describes a menu child item.
type AdminUIMenuItem struct {
	Label string `json:"label"`
	Route string `json:"route"`
}

// AdminUIMenu describes a sidebar menu group registered by an extension.
type AdminUIMenu struct {
	Label    string            `json:"label"`
	Icon     string            `json:"icon"`
	Position string            `json:"position"`
	Children []AdminUIMenuItem `json:"children"`
}

// AdminUIManifest describes the admin UI components an extension provides.
type AdminUIManifest struct {
	Entry  string                  `json:"entry"`
	Slots  map[string]AdminUISlot  `json:"slots"`
	Routes []AdminUIRoute          `json:"routes"`
	Menu   *AdminUIMenu            `json:"menu"`
}

// SettingsField describes a single setting field in the schema.
type SettingsField struct {
	Type      string   `json:"type"`
	Label     string   `json:"label"`
	Required  bool     `json:"required"`
	Default   any      `json:"default"`
	Sensitive bool     `json:"sensitive"`
	Enum      []string `json:"enum"`
}

// ExtensionManifest represents the extension.json manifest file.
type ExtensionManifest struct {
	Name           string                    `json:"name"`
	Slug           string                    `json:"slug"`
	Version        string                    `json:"version"`
	Author         string                    `json:"author"`
	Description    string                    `json:"description"`
	Priority       int                       `json:"priority"`
	Provides       []string                  `json:"provides"`
	AdminUI        *AdminUIManifest          `json:"admin_ui"`
	SettingsSchema map[string]SettingsField   `json:"settings_schema"`
}
```

- [ ] **Step 2: Update ScanAndRegister to store full manifest JSON**

In `ScanAndRegister()`, after unmarshalling the manifest into the struct, store the raw JSON in the `Manifest` field. Modify the section after `json.Unmarshal`:

```go
		// Store raw manifest JSON for the API to serve to the admin UI.
		ext := models.Extension{
			Slug:        manifest.Slug,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Author:      manifest.Author,
			Path:        extDir,
			Priority:    manifest.Priority,
			Manifest:    models.JSONB(data), // raw JSON from extension.json
		}
```

Also update the upsert `DoUpdates` to include `"manifest"`:

```go
		result := l.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "version", "description", "author", "path", "priority", "manifest",
			}),
		}).Create(&ext)
```

- [ ] **Step 3: Verify compilation**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles successfully.

- [ ] **Step 4: Commit**

```bash
git add internal/cms/extension_loader.go
git commit -m "feat: parse full extension manifest including admin_ui, settings_schema, provides"
```

---

## Task 3: Extension Settings API (Backend)

**Files:**
- Modify: `internal/cms/extension_handler.go`
- Modify: `admin-ui/src/api/client.ts` (add TS functions later in Task 7)

- [ ] **Step 1: Add Manifests endpoint to ExtensionHandler**

Add these methods to `internal/cms/extension_handler.go`:

```go
// Manifests handles GET /extensions/manifests — returns admin_ui manifests for all active extensions.
func (h *ExtensionHandler) Manifests(c *fiber.Ctx) error {
	exts, err := h.loader.GetActive()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list extensions")
	}

	type manifestEntry struct {
		Slug     string          `json:"slug"`
		Name     string          `json:"name"`
		Manifest json.RawMessage `json:"manifest"`
	}

	entries := make([]manifestEntry, 0, len(exts))
	for _, ext := range exts {
		entries = append(entries, manifestEntry{
			Slug:     ext.Slug,
			Name:     ext.Name,
			Manifest: json.RawMessage(ext.Manifest),
		})
	}
	return api.Success(c, entries)
}
```

- [ ] **Step 2: Add GetSettings and UpdateSettings endpoints**

```go
// GetSettings handles GET /extensions/:slug/settings — returns extension settings.
func (h *ExtensionHandler) GetSettings(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// Verify extension exists
	if _, err := h.loader.GetBySlug(slug); err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
	}

	prefix := "ext." + slug + "."
	var settings []models.SiteSetting
	h.db.Where("key LIKE ?", prefix+"%").Find(&settings)

	result := make(map[string]string)
	for _, s := range settings {
		key := strings.TrimPrefix(s.Key, prefix)
		if s.Value != nil {
			result[key] = *s.Value
		}
	}
	return api.Success(c, result)
}

// UpdateSettings handles PUT /extensions/:slug/settings — updates extension settings.
func (h *ExtensionHandler) UpdateSettings(c *fiber.Ctx) error {
	slug := c.Params("slug")

	ext, err := h.loader.GetBySlug(slug)
	if err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
	}

	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Parse manifest to check for sensitive fields
	var manifest ExtensionManifest
	_ = json.Unmarshal(ext.Manifest, &manifest)

	prefix := "ext." + slug + "."
	for key, value := range body {
		setting := models.SiteSetting{
			Key:   prefix + key,
			Value: &value,
		}

		h.db.Where("key = ?", setting.Key).Assign(models.SiteSetting{Value: &value}).FirstOrCreate(&setting)
	}

	return api.Success(c, fiber.Map{"message": "Settings saved"})
}
```

- [ ] **Step 3: Add ServeAsset endpoint for extension JS bundles**

```go
// ServeAsset handles GET /extensions/:slug/assets/*filepath — serves static files from extension admin-ui/dist/.
func (h *ExtensionHandler) ServeAsset(c *fiber.Ctx) error {
	slug := c.Params("slug")
	filePath := c.Params("*")

	ext, err := h.loader.GetBySlug(slug)
	if err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
	}

	// Resolve and validate path
	fullPath := filepath.Join(ext.Path, "admin-ui", "dist", filePath)
	cleanPath := filepath.Clean(fullPath)
	basePath := filepath.Clean(filepath.Join(ext.Path, "admin-ui", "dist"))

	if !strings.HasPrefix(cleanPath, basePath) {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_PATH", "Path traversal not allowed")
	}

	return c.SendFile(cleanPath)
}
```

- [ ] **Step 4: Register new routes**

Update the `RegisterRoutes` method in `extension_handler.go`:

```go
func (h *ExtensionHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/extensions", auth.CapabilityRequired("manage_settings"))
	g.Get("/manifests", h.Manifests)
	g.Get("/", h.List)
	g.Get("/:slug/files", h.BrowseFiles)
	g.Get("/:slug/settings", h.GetSettings)
	g.Put("/:slug/settings", h.UpdateSettings)
	g.Get("/:slug/assets/*", h.ServeAsset)
	g.Get("/:slug", h.Get)
	g.Post("/:slug/activate", h.Activate)
	g.Post("/:slug/deactivate", h.Deactivate)
	g.Post("/upload", h.Upload)
	g.Delete("/:slug", h.Delete)
}
```

Note: `/manifests` must be registered before `/:slug` to avoid the wildcard catching it.

- [ ] **Step 5: Add missing imports to extension_handler.go**

Ensure the import block includes `"encoding/json"` and `"strings"` (both already present).

- [ ] **Step 6: Verify compilation**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles successfully.

- [ ] **Step 7: Commit**

```bash
git add internal/cms/extension_handler.go
git commit -m "feat: extension settings API, manifests endpoint, and asset serving"
```

---

## Task 4: Refactor Email Provider Resolution

**Files:**
- Modify: `internal/email/provider.go`
- Modify: `internal/email/dispatcher.go`

- [ ] **Step 1: Add extension-aware provider factory**

Update `internal/email/provider.go` to support extension-based provider resolution:

```go
package email

// Provider defines the interface for sending emails.
type Provider interface {
	Name() string
	Send(to []string, subject string, html string) error
}

// NewProvider creates a provider from site settings map.
// Supports both legacy names ("smtp", "resend") and extension slugs ("smtp-provider", "resend-provider").
func NewProvider(name string, settings map[string]string) Provider {
	switch name {
	case "smtp", "smtp-provider":
		return NewSMTPProvider(settings)
	case "resend", "resend-provider":
		return NewResendProvider(settings)
	default:
		return nil
	}
}

// NewProviderFromExtension creates a provider using extension-scoped settings.
// The providerSlug determines which Go provider to instantiate.
// Settings should already be the extension-scoped settings (without the ext.slug. prefix).
func NewProviderFromExtension(providerSlug string, extSettings map[string]string) Provider {
	switch providerSlug {
	case "smtp-provider":
		return NewSMTPProvider(extSettings)
	case "resend-provider":
		return NewResendProvider(extSettings)
	default:
		return nil
	}
}
```

- [ ] **Step 2: Update dispatcher to resolve extension-based providers**

In `internal/email/dispatcher.go`, update the `HandleEvent` method's provider resolution (lines 54-56):

Replace:
```go
	providerName := settings["email_provider"]
	provider := NewProvider(providerName, settings)
```

With:
```go
	providerName := settings["email_provider"]
	var provider Provider

	// Try extension-based provider first (ext. prefixed settings)
	if providerName != "" && providerName != "smtp" && providerName != "resend" {
		// Extension provider: load extension-scoped settings
		extPrefix := "ext." + providerName + "."
		extSettings := make(map[string]string)
		for k, v := range settings {
			if strings.HasPrefix(k, extPrefix) {
				extSettings[strings.TrimPrefix(k, extPrefix)] = v
			}
		}
		// Map extension settings to legacy field names for provider constructors
		if providerName == "smtp-provider" {
			extSettings["smtp_host"] = extSettings["host"]
			extSettings["smtp_port"] = extSettings["port"]
			extSettings["smtp_username"] = extSettings["username"]
			extSettings["smtp_password"] = extSettings["password"]
		} else if providerName == "resend-provider" {
			extSettings["resend_api_key"] = extSettings["api_key"]
		}
		provider = NewProviderFromExtension(providerName, extSettings)
	}

	// Fallback to legacy provider resolution
	if provider == nil {
		provider = NewProvider(providerName, settings)
	}
```

Also add `"strings"` to the import block if not present.

- [ ] **Step 3: Verify compilation**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles successfully.

- [ ] **Step 4: Commit**

```bash
git add internal/email/provider.go internal/email/dispatcher.go
git commit -m "feat: extension-aware email provider resolution with legacy fallback"
```

---

## Task 5: Import Map Shims and Shared Globals

**Files:**
- Modify: `admin-ui/index.html`
- Modify: `admin-ui/src/main.tsx`
- Create: `admin-ui/public/shims/react.js`
- Create: `admin-ui/public/shims/react-dom.js`
- Create: `admin-ui/public/shims/react-router-dom.js`
- Create: `admin-ui/public/shims/sonner.js`
- Create: `admin-ui/public/shims/vibecms-ui.js`
- Create: `admin-ui/public/shims/vibecms-api.js`
- Create: `admin-ui/public/shims/vibecms-icons.js`

- [ ] **Step 1: Add import map to index.html**

Update `admin-ui/index.html`:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>admin-ui</title>
    <script type="importmap">
    {
      "imports": {
        "react": "/admin/shims/react.js",
        "react-dom": "/admin/shims/react-dom.js",
        "react-dom/client": "/admin/shims/react-dom.js",
        "react-router-dom": "/admin/shims/react-router-dom.js",
        "sonner": "/admin/shims/sonner.js",
        "@vibecms/ui": "/admin/shims/vibecms-ui.js",
        "@vibecms/api": "/admin/shims/vibecms-api.js",
        "@vibecms/icons": "/admin/shims/vibecms-icons.js"
      }
    }
    </script>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 2: Expose shared globals in main.tsx**

Update `admin-ui/src/main.tsx` to expose globals before rendering:

```tsx
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
import * as TabsModule from "@/components/ui/tabs";
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
    ...TabsModule,
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
```

- [ ] **Step 3: Create shim modules**

Create `admin-ui/public/shims/react.js`:
```javascript
const R = window.__VIBECMS_SHARED__.React;
export default R;
export const {
  useState, useEffect, useCallback, useRef, useMemo, useContext,
  createContext, createElement, Fragment, forwardRef, memo,
  Suspense, lazy, Children, cloneElement, isValidElement,
} = R;
```

Create `admin-ui/public/shims/react-dom.js`:
```javascript
const RD = window.__VIBECMS_SHARED__.ReactDOM;
export default RD;
export const { createRoot, hydrateRoot } = RD;
```

Create `admin-ui/public/shims/react-router-dom.js`:
```javascript
const RRD = window.__VIBECMS_SHARED__.ReactRouterDOM;
export default RRD;
export const {
  useNavigate, useParams, useLocation, useSearchParams,
  Link, NavLink, Navigate, Outlet, Route, Routes,
  BrowserRouter, useMatch,
} = RRD;
```

Create `admin-ui/public/shims/sonner.js`:
```javascript
const S = window.__VIBECMS_SHARED__.Sonner;
export default S;
export const { toast, Toaster } = S;
```

Create `admin-ui/public/shims/vibecms-ui.js`:
```javascript
const UI = window.__VIBECMS_SHARED__.ui;
export default UI;
// Re-export all components
for (const [key, value] of Object.entries(UI)) {
  Object.defineProperty(exports || {}, key, { get: () => value });
}
// Named exports for tree-shaking compatibility — use dynamic approach
export const Button = UI.Button;
export const Card = UI.Card;
export const CardContent = UI.CardContent;
export const CardHeader = UI.CardHeader;
export const CardTitle = UI.CardTitle;
export const CardDescription = UI.CardDescription;
export const CardFooter = UI.CardFooter;
export const Input = UI.Input;
export const Label = UI.Label;
export const Badge = UI.Badge;
export const Dialog = UI.Dialog;
export const DialogContent = UI.DialogContent;
export const DialogHeader = UI.DialogHeader;
export const DialogTitle = UI.DialogTitle;
export const DialogDescription = UI.DialogDescription;
export const DialogFooter = UI.DialogFooter;
export const Select = UI.Select;
export const SelectContent = UI.SelectContent;
export const SelectItem = UI.SelectItem;
export const SelectTrigger = UI.SelectTrigger;
export const SelectValue = UI.SelectValue;
export const Tabs = UI.Tabs;
export const TabsList = UI.TabsList;
export const TabsTrigger = UI.TabsTrigger;
export const TabsContent = UI.TabsContent;
export const Switch = UI.Switch;
export const Textarea = UI.Textarea;
```

Create `admin-ui/public/shims/vibecms-api.js`:
```javascript
const API = window.__VIBECMS_SHARED__.api;
export default API;
export const {
  getExtensionSettings, updateExtensionSettings,
  getEmailSettings, saveEmailSettings, sendTestEmail,
} = API;
```

Create `admin-ui/public/shims/vibecms-icons.js`:
```javascript
const Icons = window.__VIBECMS_SHARED__.icons;
export default Icons;
// Re-export all icons dynamically
for (const [key, value] of Object.entries(Icons)) {
  Object.defineProperty(exports || {}, key, { get: () => value });
}
// Common icons as named exports
export const {
  Settings, Loader2, Check, X, Plus, Trash2, Edit, Save,
  Mail, Send, Server, Key, Globe, Shield, AlertCircle,
  ChevronRight, ChevronDown, ExternalLink, RefreshCw,
} = Icons;
```

- [ ] **Step 4: Verify the admin UI builds**

Run: `cd /root/projects/vibecms/admin-ui && npx tsc --noEmit && npx vite build`
Expected: Compiles and builds successfully.

- [ ] **Step 5: Commit**

```bash
git add admin-ui/index.html admin-ui/src/main.tsx admin-ui/public/shims/
git commit -m "feat: shared globals and import map shims for extension micro-frontends"
```

---

## Task 6: Extension Loader, ErrorBoundary, and Slot System (React)

**Files:**
- Create: `admin-ui/src/lib/extension-loader.ts`
- Create: `admin-ui/src/components/extension-error-boundary.tsx`
- Create: `admin-ui/src/components/extension-slot.tsx`
- Create: `admin-ui/src/hooks/use-extensions.tsx`

- [ ] **Step 1: Create the extension loader module**

Create `admin-ui/src/lib/extension-loader.ts`:

```typescript
export interface AdminUISlot {
  component: string;
  label: string;
}

export interface AdminUIRoute {
  path: string;
  component: string;
}

export interface AdminUIMenuItem {
  label: string;
  route: string;
}

export interface AdminUIMenu {
  label: string;
  icon: string;
  position: string;
  children: AdminUIMenuItem[];
}

export interface AdminUIManifest {
  entry: string;
  slots: Record<string, AdminUISlot>;
  routes: AdminUIRoute[];
  menu: AdminUIMenu | null;
}

export interface ExtensionManifestEntry {
  slug: string;
  name: string;
  manifest: {
    admin_ui?: AdminUIManifest;
    provides?: string[];
    settings_schema?: Record<string, unknown>;
  };
}

export interface LoadedExtension {
  entry: ExtensionManifestEntry;
  module: Record<string, React.ComponentType<unknown>>;
}

const extensionCache = new Map<string, LoadedExtension>();

export async function fetchExtensionManifests(): Promise<ExtensionManifestEntry[]> {
  const res = await fetch("/admin/api/extensions/manifests");
  if (!res.ok) return [];
  const json = await res.json();
  return json.data || [];
}

export async function loadExtensionModule(
  slug: string,
  entry: string,
): Promise<Record<string, React.ComponentType<unknown>>> {
  const url = `/admin/api/extensions/${slug}/assets/${entry.replace(/^admin-ui\/dist\//, "")}`;
  try {
    const mod = await import(/* @vite-ignore */ url);
    return mod;
  } catch (err) {
    console.error(`[extensions] Failed to load module for ${slug}:`, err);
    throw err;
  }
}

export async function loadExtension(
  entry: ExtensionManifestEntry,
): Promise<LoadedExtension | null> {
  if (extensionCache.has(entry.slug)) {
    return extensionCache.get(entry.slug)!;
  }

  const adminUI = entry.manifest.admin_ui;
  if (!adminUI?.entry) return null;

  try {
    const module = await loadExtensionModule(entry.slug, adminUI.entry);
    const loaded: LoadedExtension = { entry, module };
    extensionCache.set(entry.slug, loaded);
    return loaded;
  } catch {
    return null;
  }
}

export function getExtensionCache(): Map<string, LoadedExtension> {
  return extensionCache;
}
```

- [ ] **Step 2: Create the extensions context provider**

Create `admin-ui/src/hooks/use-extensions.tsx`:

```tsx
import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import {
  fetchExtensionManifests,
  loadExtension,
  type ExtensionManifestEntry,
  type LoadedExtension,
  type AdminUIRoute,
  type AdminUIMenu,
} from "@/lib/extension-loader";

interface ExtensionsContextValue {
  manifests: ExtensionManifestEntry[];
  loaded: Map<string, LoadedExtension>;
  loading: boolean;
  getSlotExtensions: (
    slotName: string,
  ) => Array<{ slug: string; label: string; Component: React.ComponentType<unknown> }>;
  routes: Array<AdminUIRoute & { slug: string }>;
  menus: Array<AdminUIMenu & { slug: string }>;
}

const ExtensionsContext = createContext<ExtensionsContextValue>({
  manifests: [],
  loaded: new Map(),
  loading: true,
  getSlotExtensions: () => [],
  routes: [],
  menus: [],
});

export function ExtensionsProvider({ children }: { children: ReactNode }) {
  const [manifests, setManifests] = useState<ExtensionManifestEntry[]>([]);
  const [loaded, setLoaded] = useState<Map<string, LoadedExtension>>(new Map());
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function init() {
      const entries = await fetchExtensionManifests();
      if (cancelled) return;
      setManifests(entries);

      const loadedMap = new Map<string, LoadedExtension>();
      await Promise.allSettled(
        entries.map(async (entry) => {
          const ext = await loadExtension(entry);
          if (ext) loadedMap.set(entry.slug, ext);
        }),
      );

      if (!cancelled) {
        setLoaded(loadedMap);
        setLoading(false);
      }
    }

    init();
    return () => { cancelled = true; };
  }, []);

  function getSlotExtensions(slotName: string) {
    const results: Array<{
      slug: string;
      label: string;
      Component: React.ComponentType<unknown>;
    }> = [];

    for (const [slug, ext] of loaded) {
      const adminUI = ext.entry.manifest.admin_ui;
      if (!adminUI?.slots?.[slotName]) continue;

      const slotDef = adminUI.slots[slotName];
      const Component = ext.module[slotDef.component];
      if (Component) {
        results.push({ slug, label: slotDef.label, Component });
      }
    }

    return results;
  }

  const routes: Array<AdminUIRoute & { slug: string }> = [];
  const menus: Array<AdminUIMenu & { slug: string }> = [];

  for (const [slug, ext] of loaded) {
    const adminUI = ext.entry.manifest.admin_ui;
    if (!adminUI) continue;

    if (adminUI.routes) {
      for (const route of adminUI.routes) {
        routes.push({ ...route, slug });
      }
    }

    if (adminUI.menu) {
      menus.push({ ...adminUI.menu, slug });
    }
  }

  return (
    <ExtensionsContext.Provider
      value={{ manifests, loaded, loading, getSlotExtensions, routes, menus }}
    >
      {children}
    </ExtensionsContext.Provider>
  );
}

export function useExtensions() {
  return useContext(ExtensionsContext);
}
```

- [ ] **Step 3: Create ExtensionErrorBoundary**

Create `admin-ui/src/components/extension-error-boundary.tsx`:

```tsx
import { Component, type ReactNode } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { AlertCircle } from "lucide-react";

interface Props {
  extensionName?: string;
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ExtensionErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error) {
    console.error(
      `[extensions] Error in ${this.props.extensionName || "extension"}:`,
      error,
    );
  }

  render() {
    if (this.state.hasError) {
      return (
        <Card className="border-red-200 bg-red-50 rounded-xl">
          <CardContent className="p-6 text-center">
            <AlertCircle className="h-8 w-8 text-red-400 mx-auto mb-2" />
            <h3 className="font-medium text-red-800">Extension Error</h3>
            <p className="text-sm text-red-600 mt-1">
              {this.props.extensionName
                ? `"${this.props.extensionName}" encountered an error.`
                : "This extension encountered an error."}
            </p>
            <p className="text-xs text-red-500 mt-1 font-mono">
              {this.state.error?.message}
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={() => this.setState({ hasError: false, error: null })}
            >
              Retry
            </Button>
          </CardContent>
        </Card>
      );
    }
    return this.props.children;
  }
}
```

- [ ] **Step 4: Create ExtensionSlot component**

Create `admin-ui/src/components/extension-slot.tsx`:

```tsx
import { Suspense, useState } from "react";
import { Loader2, Puzzle } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent } from "@/components/ui/card";
import { useExtensions } from "@/hooks/use-extensions";
import { ExtensionErrorBoundary } from "@/components/extension-error-boundary";

interface ExtensionSlotProps {
  name: string;
  fallback?: React.ReactNode;
}

export function ExtensionSlot({ name, fallback }: ExtensionSlotProps) {
  const { getSlotExtensions, loading } = useExtensions();
  const extensions = getSlotExtensions(name);
  const [activeTab, setActiveTab] = useState<string>("");

  if (loading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
      </div>
    );
  }

  if (extensions.length === 0) {
    return (
      fallback || (
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardContent className="flex h-32 flex-col items-center justify-center gap-2 text-slate-400">
            <Puzzle className="h-8 w-8" />
            <p className="text-sm">No extensions available for this section</p>
          </CardContent>
        </Card>
      )
    );
  }

  if (extensions.length === 1) {
    const { slug, Component } = extensions[0];
    return (
      <ExtensionErrorBoundary extensionName={slug}>
        <Suspense
          fallback={
            <div className="flex h-32 items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
            </div>
          }
        >
          <Component />
        </Suspense>
      </ExtensionErrorBoundary>
    );
  }

  const defaultTab = activeTab || extensions[0].slug;

  return (
    <Tabs value={defaultTab} onValueChange={setActiveTab}>
      <TabsList>
        {extensions.map((ext) => (
          <TabsTrigger key={ext.slug} value={ext.slug}>
            {ext.label}
          </TabsTrigger>
        ))}
      </TabsList>
      {extensions.map((ext) => (
        <TabsContent key={ext.slug} value={ext.slug}>
          <ExtensionErrorBoundary extensionName={ext.slug}>
            <Suspense
              fallback={
                <div className="flex h-32 items-center justify-center">
                  <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
                </div>
              }
            >
              <ext.Component />
            </Suspense>
          </ExtensionErrorBoundary>
        </TabsContent>
      ))}
    </Tabs>
  );
}
```

- [ ] **Step 5: Verify TypeScript**

Run: `cd /root/projects/vibecms/admin-ui && npx tsc --noEmit`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add admin-ui/src/lib/extension-loader.ts admin-ui/src/hooks/use-extensions.tsx admin-ui/src/components/extension-error-boundary.tsx admin-ui/src/components/extension-slot.tsx
git commit -m "feat: extension loader, error boundary, slot system, and extensions context"
```

---

## Task 7: Wire Extensions into App Router and Sidebar

**Files:**
- Modify: `admin-ui/src/App.tsx`
- Modify: `admin-ui/src/components/layout/admin-layout.tsx`
- Modify: `admin-ui/src/api/client.ts`

- [ ] **Step 1: Add extension API functions to client.ts**

Add to `admin-ui/src/api/client.ts`:

```typescript
// Extension settings
export async function getExtensionSettings(slug: string): Promise<Record<string, string>> {
  const res = await api<ApiResponse<Record<string, string>>>(`/admin/api/extensions/${slug}/settings`);
  return res.data;
}

export async function updateExtensionSettings(slug: string, data: Record<string, string>): Promise<void> {
  await api<void>(`/admin/api/extensions/${slug}/settings`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

// Extension manifests
export async function getExtensionManifests(): Promise<Array<{
  slug: string;
  name: string;
  manifest: Record<string, unknown>;
}>> {
  const res = await api<ApiResponse<Array<{
    slug: string;
    name: string;
    manifest: Record<string, unknown>;
  }>>>("/admin/api/extensions/manifests");
  return res.data;
}
```

- [ ] **Step 2: Wrap admin routes with ExtensionsProvider**

In `admin-ui/src/App.tsx`, add the import and wrap the protected route:

Add import at top:
```typescript
import { ExtensionsProvider } from "@/hooks/use-extensions";
import { ExtensionPageLoader } from "@/components/extension-page-loader";
```

Wrap the admin layout route element:
```tsx
<Route
  path="/admin"
  element={
    <ProtectedRoute>
      <AdminLanguageProvider>
        <ExtensionsProvider>
          <AdminLayout />
        </ExtensionsProvider>
      </AdminLanguageProvider>
    </ProtectedRoute>
  }
>
```

Add a catch-all extension route before the final `</Route>`:

```tsx
        {/* Extension routes (dynamic) */}
        <Route path="ext/:slug/*" element={<ExtensionPageLoader />} />
```

- [ ] **Step 3: Create ExtensionPageLoader component**

Create `admin-ui/src/components/extension-page-loader.tsx`:

```tsx
import { Suspense } from "react";
import { useParams, useLocation } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useExtensions } from "@/hooks/use-extensions";
import { ExtensionErrorBoundary } from "@/components/extension-error-boundary";

export function ExtensionPageLoader() {
  const { slug } = useParams<{ slug: string }>();
  const location = useLocation();
  const { loaded, loading } = useExtensions();

  if (loading || !slug) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  const ext = loaded.get(slug);
  if (!ext) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-2 text-slate-400">
        <p className="text-lg font-medium">Extension not found</p>
        <p className="text-sm">The extension "{slug}" is not loaded.</p>
      </div>
    );
  }

  // Find matching route from manifest
  const adminUI = ext.entry.manifest.admin_ui;
  if (!adminUI?.routes) return null;

  // Extract the path after /admin/ext/:slug/
  const basePath = `/admin/ext/${slug}/`;
  const subPath = location.pathname.startsWith(basePath)
    ? location.pathname.slice(basePath.length)
    : "";

  // Find the matching route (simple prefix match)
  const matchedRoute = adminUI.routes.find((r) => {
    const routePath = r.path.replace(/:\w+/g, "[^/]+");
    return new RegExp(`^${routePath}$`).test(subPath) || subPath === r.path;
  });

  if (!matchedRoute) {
    // Default to first route's component
    const defaultComponent = adminUI.routes[0]?.component;
    const Component = defaultComponent ? ext.module[defaultComponent] : null;
    if (!Component) return null;

    return (
      <ExtensionErrorBoundary extensionName={ext.entry.name}>
        <Suspense fallback={<Loader2 className="h-8 w-8 animate-spin text-indigo-500" />}>
          <Component />
        </Suspense>
      </ExtensionErrorBoundary>
    );
  }

  const Component = ext.module[matchedRoute.component];
  if (!Component) return null;

  return (
    <ExtensionErrorBoundary extensionName={ext.entry.name}>
      <Suspense fallback={<Loader2 className="h-8 w-8 animate-spin text-indigo-500" />}>
        <Component />
      </Suspense>
    </ExtensionErrorBoundary>
  );
}
```

- [ ] **Step 4: Add extension menus to sidebar**

In `admin-ui/src/components/layout/admin-layout.tsx`, import and use extensions context:

Add import:
```typescript
import { useExtensions } from "@/hooks/use-extensions";
```

Inside `AdminLayout()` function, after the existing hooks:
```typescript
const { menus: extensionMenus } = useExtensions();
```

Update `navEntries` construction to include extension menus. Replace:
```typescript
const navEntries: NavEntry[] = [...staticNavTop, ...customNavItems, ...staticNavBottom];
```

With:
```typescript
// Build extension nav groups
const extensionNavGroups: NavEntry[] = extensionMenus.map((menu) => ({
  label: menu.label,
  icon: iconMap[menu.icon] || Puzzle,
  children: menu.children.map((child) => ({
    to: `/admin/ext/${menu.slug}/${child.route}`,
    label: child.label,
    icon: iconMap[menu.icon] || Puzzle,
  })),
}));

// Insert extension menus before the "Appearance" group
const appearanceIdx = staticNavBottom.findIndex(
  (e) => "label" in e && e.label === "Appearance"
);
const bottomWithExtensions = [...staticNavBottom];
if (appearanceIdx >= 0) {
  bottomWithExtensions.splice(appearanceIdx, 0, ...extensionNavGroups);
} else {
  bottomWithExtensions.unshift(...extensionNavGroups);
}

const navEntries: NavEntry[] = [...staticNavTop, ...customNavItems, ...bottomWithExtensions];
```

- [ ] **Step 5: Verify TypeScript**

Run: `cd /root/projects/vibecms/admin-ui && npx tsc --noEmit`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add admin-ui/src/App.tsx admin-ui/src/components/layout/admin-layout.tsx admin-ui/src/api/client.ts admin-ui/src/components/extension-page-loader.tsx
git commit -m "feat: wire extensions into router, sidebar, and API client"
```

---

## Task 8: Refactor Email Settings Page to Use ExtensionSlot

**Files:**
- Modify: `admin-ui/src/pages/email-settings.tsx`

- [ ] **Step 1: Replace hardcoded email settings with ExtensionSlot**

Rewrite `admin-ui/src/pages/email-settings.tsx`:

```tsx
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
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /root/projects/vibecms/admin-ui && npx tsc --noEmit`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add admin-ui/src/pages/email-settings.tsx
git commit -m "refactor: email settings page uses ExtensionSlot instead of hardcoded forms"
```

---

## Task 9: Create SMTP Provider Extension

**Files:**
- Create: `extensions/smtp-provider/extension.json`
- Create: `extensions/smtp-provider/admin-ui/src/index.tsx`
- Create: `extensions/smtp-provider/admin-ui/src/SmtpSettings.tsx`
- Create: `extensions/smtp-provider/admin-ui/vite.config.ts`
- Create: `extensions/smtp-provider/admin-ui/package.json`
- Create: `extensions/smtp-provider/admin-ui/tsconfig.json`
- Build: `extensions/smtp-provider/admin-ui/dist/index.js`

- [ ] **Step 1: Create extension.json manifest**

Create `extensions/smtp-provider/extension.json`:

```json
{
  "name": "SMTP Provider",
  "slug": "smtp-provider",
  "version": "1.0.0",
  "author": "VibeCMS",
  "description": "Send emails via any SMTP server",
  "priority": 50,
  "provides": ["email.provider"],
  "admin_ui": {
    "entry": "admin-ui/dist/index.js",
    "slots": {
      "email-settings": {
        "component": "SmtpSettings",
        "label": "SMTP"
      }
    },
    "routes": [],
    "menu": null
  },
  "settings_schema": {
    "host": { "type": "string", "label": "SMTP Host", "required": true },
    "port": { "type": "number", "label": "SMTP Port", "default": 587 },
    "username": { "type": "string", "label": "Username" },
    "password": { "type": "string", "label": "Password", "sensitive": true },
    "from_email": { "type": "string", "label": "From Email", "required": true },
    "from_name": { "type": "string", "label": "From Name" },
    "encryption": { "type": "string", "label": "Encryption", "enum": ["none", "tls", "starttls"], "default": "tls" }
  }
}
```

- [ ] **Step 2: Create package.json and tsconfig**

Create `extensions/smtp-provider/admin-ui/package.json`:

```json
{
  "name": "smtp-provider-admin-ui",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "vite build"
  },
  "dependencies": {},
  "devDependencies": {
    "@types/react": "^19.0.0",
    "@vitejs/plugin-react": "^4.3.0",
    "typescript": "^5.6.0",
    "vite": "^6.0.0"
  }
}
```

Create `extensions/smtp-provider/admin-ui/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "outDir": "dist"
  },
  "include": ["src"]
}
```

- [ ] **Step 3: Create Vite config**

Create `extensions/smtp-provider/admin-ui/vite.config.ts`:

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
  build: {
    outDir: "dist",
    lib: {
      entry: "src/index.tsx",
      formats: ["es"],
      fileName: "index",
    },
    rollupOptions: {
      external: [
        "react",
        "react-dom",
        "react-dom/client",
        "react-router-dom",
        "sonner",
        "@vibecms/ui",
        "@vibecms/api",
        "@vibecms/icons",
      ],
    },
    cssCodeSplit: false,
  },
});
```

- [ ] **Step 4: Create SmtpSettings component**

Create `extensions/smtp-provider/admin-ui/src/SmtpSettings.tsx`:

```tsx
import { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  Input,
  Label,
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@vibecms/ui";
import { toast } from "sonner";
import { getExtensionSettings, updateExtensionSettings } from "@vibecms/api";
import { Loader2, Server } from "@vibecms/icons";

const SLUG = "smtp-provider";

export default function SmtpSettings() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [host, setHost] = useState("");
  const [port, setPort] = useState("587");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [fromEmail, setFromEmail] = useState("");
  const [fromName, setFromName] = useState("");
  const [encryption, setEncryption] = useState("tls");

  useEffect(() => {
    getExtensionSettings(SLUG)
      .then((data) => {
        setHost(data.host || "");
        setPort(data.port || "587");
        setUsername(data.username || "");
        setPassword(data.password || "");
        setFromEmail(data.from_email || "");
        setFromName(data.from_name || "");
        setEncryption(data.encryption || "tls");
      })
      .catch(() => toast.error("Failed to load SMTP settings"))
      .finally(() => setLoading(false));
  }, []);

  async function handleSave() {
    setSaving(true);
    try {
      await updateExtensionSettings(SLUG, {
        host,
        port,
        username,
        password,
        from_email: fromEmail,
        from_name: fromName,
        encryption,
      });
      toast.success("SMTP settings saved");
    } catch {
      toast.error("Failed to save SMTP settings");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <Card className="rounded-xl border border-slate-200 shadow-sm">
      <CardHeader>
        <div className="flex items-center gap-2">
          <Server className="h-5 w-5 text-indigo-500" />
          <CardTitle className="text-lg font-semibold text-slate-900">
            SMTP Configuration
          </CardTitle>
        </div>
        <CardDescription>
          Configure your SMTP server for sending transactional emails.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">Host</Label>
            <Input
              placeholder="smtp.example.com"
              value={host}
              onChange={(e) => setHost(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">Port</Label>
            <Input
              placeholder="587"
              value={port}
              onChange={(e) => setPort(e.target.value)}
            />
          </div>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">
              Username
            </Label>
            <Input
              placeholder="user@example.com"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">
              Password
            </Label>
            <Input
              type="password"
              placeholder="********"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">
              From Email
            </Label>
            <Input
              placeholder="noreply@example.com"
              value={fromEmail}
              onChange={(e) => setFromEmail(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">
              From Name
            </Label>
            <Input
              placeholder="My Site"
              value={fromName}
              onChange={(e) => setFromName(e.target.value)}
            />
          </div>
        </div>
        <div className="space-y-2 max-w-xs">
          <Label className="text-sm font-medium text-slate-700">
            Encryption
          </Label>
          <Select value={encryption} onValueChange={setEncryption}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="tls">TLS</SelectItem>
              <SelectItem value="starttls">STARTTLS</SelectItem>
              <SelectItem value="none">None</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button
          onClick={handleSave}
          disabled={saving}
          className="bg-indigo-600 hover:bg-indigo-700 text-white"
        >
          {saving ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Saving...
            </>
          ) : (
            "Save SMTP Settings"
          )}
        </Button>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 5: Create index.tsx entry point**

Create `extensions/smtp-provider/admin-ui/src/index.tsx`:

```tsx
export { default as SmtpSettings } from "./SmtpSettings";
```

- [ ] **Step 6: Install dependencies and build**

```bash
cd /root/projects/vibecms/extensions/smtp-provider/admin-ui && npm install && npm run build
```

Expected: `dist/index.js` is created.

- [ ] **Step 7: Commit**

```bash
git add extensions/smtp-provider/
git commit -m "feat: SMTP provider extension with admin UI settings component"
```

---

## Task 10: Create Resend Provider Extension

**Files:**
- Create: `extensions/resend-provider/extension.json`
- Create: `extensions/resend-provider/admin-ui/src/index.tsx`
- Create: `extensions/resend-provider/admin-ui/src/ResendSettings.tsx`
- Create: `extensions/resend-provider/admin-ui/vite.config.ts`
- Create: `extensions/resend-provider/admin-ui/package.json`
- Create: `extensions/resend-provider/admin-ui/tsconfig.json`
- Build: `extensions/resend-provider/admin-ui/dist/index.js`

- [ ] **Step 1: Create extension.json manifest**

Create `extensions/resend-provider/extension.json`:

```json
{
  "name": "Resend Provider",
  "slug": "resend-provider",
  "version": "1.0.0",
  "author": "VibeCMS",
  "description": "Send emails via Resend.com API",
  "priority": 50,
  "provides": ["email.provider"],
  "admin_ui": {
    "entry": "admin-ui/dist/index.js",
    "slots": {
      "email-settings": {
        "component": "ResendSettings",
        "label": "Resend"
      }
    },
    "routes": [],
    "menu": null
  },
  "settings_schema": {
    "api_key": { "type": "string", "label": "API Key", "required": true, "sensitive": true },
    "from_email": { "type": "string", "label": "From Email", "required": true },
    "from_name": { "type": "string", "label": "From Name" }
  }
}
```

- [ ] **Step 2: Create package.json, tsconfig, and vite config**

Create `extensions/resend-provider/admin-ui/package.json`:

```json
{
  "name": "resend-provider-admin-ui",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "vite build"
  },
  "dependencies": {},
  "devDependencies": {
    "@types/react": "^19.0.0",
    "@vitejs/plugin-react": "^4.3.0",
    "typescript": "^5.6.0",
    "vite": "^6.0.0"
  }
}
```

Create `extensions/resend-provider/admin-ui/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "outDir": "dist"
  },
  "include": ["src"]
}
```

Create `extensions/resend-provider/admin-ui/vite.config.ts`:

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
  build: {
    outDir: "dist",
    lib: {
      entry: "src/index.tsx",
      formats: ["es"],
      fileName: "index",
    },
    rollupOptions: {
      external: [
        "react",
        "react-dom",
        "react-dom/client",
        "react-router-dom",
        "sonner",
        "@vibecms/ui",
        "@vibecms/api",
        "@vibecms/icons",
      ],
    },
    cssCodeSplit: false,
  },
});
```

- [ ] **Step 3: Create ResendSettings component**

Create `extensions/resend-provider/admin-ui/src/ResendSettings.tsx`:

```tsx
import { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  Input,
  Label,
  Button,
} from "@vibecms/ui";
import { toast } from "sonner";
import { getExtensionSettings, updateExtensionSettings } from "@vibecms/api";
import { Loader2, Key } from "@vibecms/icons";

const SLUG = "resend-provider";

export default function ResendSettings() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [apiKey, setApiKey] = useState("");
  const [fromEmail, setFromEmail] = useState("");
  const [fromName, setFromName] = useState("");

  useEffect(() => {
    getExtensionSettings(SLUG)
      .then((data) => {
        setApiKey(data.api_key || "");
        setFromEmail(data.from_email || "");
        setFromName(data.from_name || "");
      })
      .catch(() => toast.error("Failed to load Resend settings"))
      .finally(() => setLoading(false));
  }, []);

  async function handleSave() {
    setSaving(true);
    try {
      await updateExtensionSettings(SLUG, {
        api_key: apiKey,
        from_email: fromEmail,
        from_name: fromName,
      });
      toast.success("Resend settings saved");
    } catch {
      toast.error("Failed to save Resend settings");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <Card className="rounded-xl border border-slate-200 shadow-sm">
      <CardHeader>
        <div className="flex items-center gap-2">
          <Key className="h-5 w-5 text-indigo-500" />
          <CardTitle className="text-lg font-semibold text-slate-900">
            Resend Configuration
          </CardTitle>
        </div>
        <CardDescription>
          Configure your Resend.com API key for sending transactional emails.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label className="text-sm font-medium text-slate-700">API Key</Label>
          <Input
            type="password"
            placeholder="re_..."
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">
              From Email
            </Label>
            <Input
              placeholder="noreply@example.com"
              value={fromEmail}
              onChange={(e) => setFromEmail(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label className="text-sm font-medium text-slate-700">
              From Name
            </Label>
            <Input
              placeholder="My Site"
              value={fromName}
              onChange={(e) => setFromName(e.target.value)}
            />
          </div>
        </div>
        <Button
          onClick={handleSave}
          disabled={saving}
          className="bg-indigo-600 hover:bg-indigo-700 text-white"
        >
          {saving ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Saving...
            </>
          ) : (
            "Save Resend Settings"
          )}
        </Button>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 4: Create index.tsx entry point**

Create `extensions/resend-provider/admin-ui/src/index.tsx`:

```tsx
export { default as ResendSettings } from "./ResendSettings";
```

- [ ] **Step 5: Install dependencies and build**

```bash
cd /root/projects/vibecms/extensions/resend-provider/admin-ui && npm install && npm run build
```

Expected: `dist/index.js` is created.

- [ ] **Step 6: Commit**

```bash
git add extensions/resend-provider/
git commit -m "feat: Resend provider extension with admin UI settings component"
```

---

## Task 11: Integration Test — Full Docker Build and Verify

**Files:** None (verification only)

- [ ] **Step 1: Build and start Docker**

```bash
cd /root/projects/vibecms && docker compose up --build -d
```

Expected: App starts without errors.

- [ ] **Step 2: Check extension loader logs**

```bash
docker compose logs app 2>&1 | grep -i extension
```

Expected: Logs show `[extensions] scanned 3 extensions from /app/extensions` (hello-extension + smtp-provider + resend-provider).

- [ ] **Step 3: Verify manifests API**

```bash
curl -s http://localhost:8099/admin/api/extensions/manifests -H "Cookie: <session>" | jq .
```

Expected: Returns JSON with active extensions and their admin_ui manifests.

- [ ] **Step 4: Verify extension asset serving**

```bash
curl -sI http://localhost:8099/admin/api/extensions/smtp-provider/assets/index.js
```

Expected: 200 OK with `application/javascript` content type.

- [ ] **Step 5: Verify email settings page loads extension tabs**

Navigate to `http://<host>:8099/admin/email-settings` in browser.
Expected: Shows tab UI with SMTP and/or Resend settings based on which extensions are active.

- [ ] **Step 6: Commit any fixes**

```bash
git add -A && git commit -m "fix: integration fixes for extension system"
```

---

## Task 12: Update Extension Preview Images

**Files:**
- Modify: `admin-ui/public/previews/default-extension.svg`

- [ ] **Step 1: Create SMTP and Resend-specific preview images**

Create `admin-ui/public/previews/smtp-provider.svg` and `admin-ui/public/previews/resend-provider.svg` with appropriate branding.

- [ ] **Step 2: Update extension cards to use thumbnail from manifest (future)**

This is optional — for now the default extension preview works. Extensions could declare a `thumbnail` field in their manifest in the future.

- [ ] **Step 3: Commit**

```bash
git add admin-ui/public/previews/
git commit -m "feat: extension-specific preview images"
```
