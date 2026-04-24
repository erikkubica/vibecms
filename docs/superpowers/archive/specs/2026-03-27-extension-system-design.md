# Extension System Design: Pluggable Admin UI & Email Providers

**Date:** 2026-03-27
**Status:** Approved
**Goal:** Make VibeCMS extensible via dynamic React micro-frontends and hookable backend providers, demonstrated by SMTP and Resend.com email provider extensions.

---

## 1. Overview

VibeCMS extensions currently support Tengo scripts for backend hooks but have no way to inject UI into the admin React SPA. This design adds three capabilities:

1. **Admin UI micro-frontends** — Extensions ship pre-built React bundles that the admin dynamically loads at runtime, sharing React, shadcn/ui, and the router.
2. **Provider pattern** — A convention on top of the existing EventBus where exactly one extension "provides" a capability (e.g., email transport).
3. **Extension settings API** — Extensions can declare, read, and write their own scoped settings.

The first two extensions built with this system will be **SMTP Provider** and **Resend.com Provider**, replacing the current hardcoded email provider logic.

---

## 2. Extension Manifest (extension.json)

The existing manifest gains an `admin_ui` section and a `settings_schema` section:

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

### Full-featured extension example (Contact Forms):

```json
{
  "name": "Contact Forms",
  "slug": "contact-forms",
  "version": "1.0.0",
  "author": "VibeCMS",
  "description": "Drag-and-drop contact form builder with submission management",
  "priority": 50,

  "node_types": [
    { "slug": "contact-form", "label": "Contact Form", "icon": "mail" },
    { "slug": "form-submission", "label": "Form Submission", "icon": "file-text" }
  ],

  "admin_ui": {
    "entry": "admin-ui/dist/index.js",
    "routes": [
      { "path": "contact-forms", "component": "ContactFormList" },
      { "path": "contact-forms/:id", "component": "ContactFormEditor" },
      { "path": "contact-forms/submissions", "component": "SubmissionList" },
      { "path": "contact-forms/settings", "component": "FormSettings" }
    ],
    "menu": {
      "label": "Contact Forms",
      "icon": "mail",
      "position": "after:posts",
      "children": [
        { "label": "Forms", "route": "contact-forms" },
        { "label": "Submissions", "route": "contact-forms/submissions" },
        { "label": "Settings", "route": "contact-forms/settings" }
      ]
    },
    "slots": {}
  }
}
```

### Manifest fields:

| Field | Type | Description |
|-------|------|-------------|
| `admin_ui.entry` | string | Path to the pre-built JS bundle (ES module) relative to extension root |
| `admin_ui.slots` | object | Keyed by slot name (e.g., `email-settings`). Each slot has `component` (exported name) and `label` (tab label) |
| `admin_ui.routes` | array | Each route has `path` (relative to `/admin/`) and `component` (exported name) |
| `admin_ui.menu` | object or null | Sidebar menu item with `label`, `icon` (lucide name), `position` (placement hint), `children` |
| `settings_schema` | object | Keyed by setting name. Each has `type`, `label`, `required`, `default`, `sensitive`, `enum` |
| `provides` | array | Capabilities this extension provides (e.g., `["email.provider"]`). Used by the Go loader to wire up provider patterns. |
| `node_types` | array | Content types to auto-register. Each has `slug`, `label`, `icon` |

---

## 3. Backend Architecture

### 3.1 Extension Loader Changes

`internal/cms/extension_loader.go` — `ScanAndRegister()` is extended to:

1. Parse the new manifest fields (`admin_ui`, `settings_schema`, `node_types`)
2. Store parsed manifest in a new `Manifest` JSONB column on the Extension model
3. Auto-register `node_types` declared in the manifest as ContentType records
4. Serve extension admin-ui assets via a new route

**New Extension model field:**
```go
Manifest  datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"manifest"`
```

### 3.2 Extension Admin API

New endpoints under `/admin/api/extensions/`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/api/extensions/manifests` | Returns all active extensions' parsed manifests (routes, slots, menus) for the React app to consume |
| GET | `/admin/api/extensions/:slug/assets/*filepath` | Serves static files from the extension's `admin-ui/dist/` directory |
| GET | `/admin/api/extensions/:slug/settings` | Returns extension's current settings values |
| PUT | `/admin/api/extensions/:slug/settings` | Updates extension's settings (validates against schema, encrypts sensitive fields) |

### 3.3 Extension Settings Storage

Extension settings are stored in the existing `site_settings` table with a namespaced key pattern:

```
ext.<slug>.<key>
```

Example: `ext.smtp-provider.host`, `ext.smtp-provider.port`

Sensitive fields (marked `sensitive: true` in schema) use the existing encryption support in the settings store.

### 3.4 Provider Pattern

The provider pattern is a convention built on top of the existing email system, not a new abstraction. The change is:

**Current flow:**
1. Email dispatcher has hardcoded SMTP and Resend providers
2. `site_settings.email_provider` selects which one ("smtp" or "resend")
3. Provider is instantiated with settings from `site_settings`

**New flow:**
1. Email dispatcher asks: "which extensions provide `email.provider`?"
2. Looks at `site_settings.email_provider` to find the active provider slug (e.g., "smtp-provider")
3. Calls a new internal method that loads the matching extension's Tengo `email_send` handler, OR uses a Go-native provider registered by the extension loader
4. Falls back to "no provider configured" error if none active

**Implementation approach — Go-native providers via extension loader:**

Rather than routing email through Tengo (which adds latency and complexity for a critical path), extensions register Go-native email providers. The extension loader:

1. Reads the manifest's `provides` field (new): `"provides": ["email.provider"]`
2. For known provider types like `email.provider`, the loader reads the extension's settings and instantiates the appropriate Go `Provider` implementation
3. The SMTP and Resend Go provider code already exists — it just moves from hardcoded selection to extension-driven selection

This means:
- `internal/email/smtp.go` and `internal/email/resend.go` stay as-is
- The extension loader maps `slug: "smtp-provider"` → `email.NewSMTPProvider(settings)`
- The extension loader maps `slug: "resend-provider"` → `email.NewResendProvider(settings)`
- Future extensions with custom transport would implement the Provider interface in Go (compiled into VibeCMS) or use Tengo for simple webhook-based providers

### 3.5 Tengo API Extensions

New functions added to the `cms/settings` module for extension-scoped access:

```
settings.get_ext(slug, key)           // Read extension setting
settings.set_ext(slug, key, value)    // Write extension setting
settings.get_ext_all(slug)            // Read all settings for extension
```

New function in `cms/email` module:

```
email.get_provider()                  // Returns active provider slug
```

---

## 4. Admin UI Architecture

### 4.1 Shared Globals

The admin Vite build exposes core libraries as global variables that extension bundles import from. This is configured in `vite.config.ts`:

```typescript
// vite.config.ts addition
build: {
  rollupOptions: {
    output: {
      globals: {}, // not needed for main app
    },
  },
},
```

The admin app's `main.tsx` exposes globals on `window`:

```typescript
// In main.tsx, before React render:
import * as React from 'react';
import * as ReactDOM from 'react-dom/client';
import * as ReactRouterDOM from 'react-router-dom';
import * as Sonner from 'sonner';
// shadcn/ui components
import * as Button from '@/components/ui/button';
import * as Card from '@/components/ui/card';
import * as Input from '@/components/ui/input';
import * as Label from '@/components/ui/label';
import * as Badge from '@/components/ui/badge';
import * as Dialog from '@/components/ui/dialog';
import * as Select from '@/components/ui/select';
import * as Tabs from '@/components/ui/tabs';
// ... other commonly used components

window.__VIBECMS_SHARED__ = {
  React,
  ReactDOM,
  ReactRouterDOM,
  Sonner,
  ui: { Button, Card, Input, Label, Badge, Dialog, Select, Tabs },
  api: apiClient, // the existing API client
  icons: LucideIcons, // re-export of lucide-react
};
```

### 4.2 Extension Bundle Format

Extension authors build their React components using a Vite config that externalizes shared dependencies:

```typescript
// extensions/smtp-provider/admin-ui/vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  build: {
    lib: {
      entry: 'src/index.tsx',
      formats: ['es'],
      fileName: 'index',
    },
    rollupOptions: {
      external: (id) => {
        // Everything from shared globals is external
        return id === 'react' || id === 'react-dom' || id === 'react-router-dom' ||
               id === 'sonner' || id.startsWith('@vibecms/');
      },
      output: {
        globals: {}, // ES modules, no globals needed
      },
    },
  },
});
```

The extension's `src/index.tsx` exports named components:

```typescript
// extensions/smtp-provider/admin-ui/src/index.tsx
export { default as SmtpSettings } from './SmtpSettings';
```

The built bundle uses import maps or a runtime resolver to map `react` → `window.__VIBECMS_SHARED__.React`, etc.

### 4.3 Extension Loader (React)

A new module `admin-ui/src/lib/extension-loader.ts`:

```typescript
interface ExtensionManifest {
  slug: string;
  name: string;
  admin_ui: {
    entry: string;
    slots: Record<string, { component: string; label: string }>;
    routes: Array<{ path: string; component: string }>;
    menu: {
      label: string;
      icon: string;
      position: string;
      children: Array<{ label: string; route: string }>;
    } | null;
  };
}

interface LoadedExtension {
  manifest: ExtensionManifest;
  module: Record<string, React.ComponentType>;
}

// Fetches all active extension manifests from the API
async function fetchExtensionManifests(): Promise<ExtensionManifest[]>;

// Dynamically imports an extension's JS bundle via native import()
// Bare imports (react, @vibecms/ui, etc.) are resolved by the import map in index.html
async function loadExtensionModule(slug: string, entry: string): Promise<Record<string, React.ComponentType>>;

// Cache of loaded extensions
const extensionCache: Map<string, LoadedExtension>;
```

**Import rewriting strategy:**

Since extension bundles use bare imports (`import React from 'react'`), but we serve them as static files (not through a bundler), we need a runtime import resolver. Two options:

**Option A: Import maps** (preferred) — The admin app injects an `<script type="importmap">` that maps bare specifiers to blob URLs wrapping `window.__VIBECMS_SHARED__`:

```html
<script type="importmap">
{
  "imports": {
    "react": "/admin/api/shims/react.js",
    "react-dom": "/admin/api/shims/react-dom.js",
    "react-router-dom": "/admin/api/shims/react-router-dom.js",
    "sonner": "/admin/api/shims/sonner.js",
    "@vibecms/ui": "/admin/api/shims/ui.js",
    "@vibecms/api": "/admin/api/shims/api.js",
    "@vibecms/icons": "/admin/api/shims/icons.js"
  }
}
</script>
```

Each shim is a tiny ES module that re-exports from `window.__VIBECMS_SHARED__`:

```javascript
// /admin/api/shims/react.js
const R = window.__VIBECMS_SHARED__.React;
export default R;
export const { useState, useEffect, useCallback, useRef, useMemo, createContext, useContext, Fragment, createElement } = R;
```

**Option B: Fetch-and-rewrite** — Fetch the bundle as text, replace bare imports, convert to blob URL, then `import()`. More complex, fragile with source maps.

**We'll use Option A (import maps)** — it's a web standard, no text rewriting, and works with native `import()`.

### 4.4 Slot System

Slots are named injection points in existing admin pages. The first slot is `email-settings`.

```typescript
// admin-ui/src/components/extension-slot.tsx
interface ExtensionSlotProps {
  name: string; // e.g., "email-settings"
}

function ExtensionSlot({ name }: ExtensionSlotProps) {
  // 1. Get all loaded extensions that declare this slot
  // 2. Render as tabs (one tab per extension)
  // 3. Each tab content is wrapped in <ErrorBoundary>
  // 4. Lazy-load the component on first tab activation
}
```

The `email-settings` page changes from hardcoded SMTP/Resend forms to:

```tsx
// Simplified email-settings.tsx
function EmailSettingsPage() {
  return (
    <div>
      <h1>Email Settings</h1>
      <ExtensionSlot name="email-settings" />
      {/* Falls back to "No email provider installed" if no extensions claim this slot */}
    </div>
  );
}
```

### 4.5 Route Registration

Extension routes are injected into the React Router at app startup:

```typescript
// In App.tsx, inside the admin layout route:
{extensionRoutes.map((route) => (
  <Route
    key={route.path}
    path={route.path}
    element={
      <ErrorBoundary fallback={<ExtensionError />}>
        <Suspense fallback={<Loader />}>
          <ExtensionPage slug={route.slug} component={route.component} />
        </Suspense>
      </ErrorBoundary>
    }
  />
))}
```

### 4.6 Menu Registration

Extension menu items are injected into the sidebar dynamically:

```typescript
// In admin-layout.tsx
// After building staticNavTop + customNavItems + staticNavBottom:
// Insert extension menu items based on their position hints

extensionMenus.forEach((menu) => {
  const navGroup: NavGroup = {
    label: menu.label,
    icon: iconMap[menu.icon] || Puzzle,
    children: menu.children.map((child) => ({
      to: `/admin/${child.route}`,
      label: child.label,
      icon: iconMap[menu.icon] || Puzzle,
    })),
  };
  // Insert based on menu.position ("after:posts", "before:appearance", etc.)
  // Default: insert before the "Appearance" group
});
```

### 4.7 Error Boundary

Every extension component render point is wrapped in an ErrorBoundary:

```typescript
// admin-ui/src/components/extension-error-boundary.tsx
class ExtensionErrorBoundary extends React.Component {
  state = { hasError: false, error: null };

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      return (
        <Card className="border-red-200 bg-red-50">
          <CardContent className="p-6 text-center">
            <AlertCircle className="h-8 w-8 text-red-400 mx-auto mb-2" />
            <h3 className="font-medium text-red-800">Extension Error</h3>
            <p className="text-sm text-red-600 mt-1">
              This extension encountered an error and was disabled to protect the admin.
            </p>
            <Button variant="outline" size="sm" className="mt-3"
              onClick={() => this.setState({ hasError: false })}>
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

---

## 5. Extension Directory Structure

### SMTP Provider Extension:

```
extensions/smtp-provider/
├── extension.json
├── scripts/
│   └── extension.tengo        # Registers as email provider
├── admin-ui/
│   ├── src/
│   │   ├── index.tsx           # Exports SmtpSettings component
│   │   └── SmtpSettings.tsx    # SMTP configuration form
│   ├── vite.config.ts          # Build config with externals
│   └── dist/
│       └── index.js            # Pre-built ES module bundle
```

### Resend Provider Extension:

```
extensions/resend-provider/
├── extension.json
├── scripts/
│   └── extension.tengo
├── admin-ui/
│   ├── src/
│   │   ├── index.tsx
│   │   └── ResendSettings.tsx  # Resend API key + domain config form
│   ├── vite.config.ts
│   └── dist/
│       └── index.js
```

---

## 6. Email System Refactor

The current email system has hardcoded SMTP/Resend providers in Go. The refactor:

### Current state (to be removed):
- `internal/email/provider.go` — Provider interface + factory that switches on `site_settings.email_provider`
- `internal/email/smtp.go` — SMTP implementation
- `internal/email/resend.go` — Resend implementation
- `admin-ui/src/pages/email-settings.tsx` — Hardcoded SMTP/Resend forms

### New state:
- `internal/email/provider.go` — Provider interface stays. Factory function changes to look up the active provider extension and instantiate the matching Go provider.
- `internal/email/smtp.go` — Stays as-is, but instantiated by the SMTP extension loader, not the factory.
- `internal/email/resend.go` — Stays as-is, but instantiated by the Resend extension loader.
- `internal/cms/extension_loader.go` — Extended to handle `"provides": ["email.provider"]` in manifests. Maps known provider slugs to Go provider constructors.
- `admin-ui/src/pages/email-settings.tsx` — Gutted. Becomes a thin wrapper around `<ExtensionSlot name="email-settings" />` with a fallback message.

### Provider resolution:
```go
func (l *ExtensionLoader) GetEmailProvider() (email.Provider, error) {
    activeSlug := settings.Get("email_provider") // e.g., "smtp-provider"
    if activeSlug == "" {
        return nil, fmt.Errorf("no email provider configured")
    }

    ext, err := l.GetActiveExtension(activeSlug)
    if err != nil {
        return nil, fmt.Errorf("email provider extension %q not found or inactive", activeSlug)
    }

    switch ext.Slug {
    case "smtp-provider":
        return email.NewSMTPProvider(l.getExtSettings(ext.Slug))
    case "resend-provider":
        return email.NewResendProvider(l.getExtSettings(ext.Slug))
    default:
        // Future: look for a Tengo-based provider handler
        return nil, fmt.Errorf("unknown email provider type: %s", ext.Slug)
    }
}
```

---

## 7. Extension Vite Template

A reusable Vite config template for extension authors:

```typescript
// Template: extensions/<slug>/admin-ui/vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    outDir: 'dist',
    lib: {
      entry: 'src/index.tsx',
      formats: ['es'],
      fileName: 'index',
    },
    rollupOptions: {
      external: [
        'react', 'react-dom', 'react-dom/client',
        'react-router-dom', 'sonner',
        '@vibecms/ui', '@vibecms/api', '@vibecms/icons',
      ],
    },
    cssCodeSplit: false, // Bundle CSS into JS
  },
});
```

Extension components use `@vibecms/ui` for shadcn components:

```tsx
import React, { useState, useEffect } from 'react';
import { Input, Label, Button, Card, CardContent } from '@vibecms/ui';
import { toast } from 'sonner';
import { api } from '@vibecms/api';

export default function SmtpSettings() {
  const [settings, setSettings] = useState({});

  useEffect(() => {
    api.getExtensionSettings('smtp-provider').then(setSettings);
  }, []);

  async function handleSave() {
    await api.updateExtensionSettings('smtp-provider', settings);
    toast.success('SMTP settings saved');
  }

  return (
    <Card>
      <CardContent className="space-y-4 p-6">
        <div className="space-y-2">
          <Label>SMTP Host</Label>
          <Input value={settings.host || ''} onChange={...} />
        </div>
        {/* ... more fields ... */}
        <Button onClick={handleSave}>Save Settings</Button>
      </CardContent>
    </Card>
  );
}
```

---

## 8. Safety & Error Handling

| Scenario | Handling |
|----------|----------|
| Extension JS bundle fails to load (404, syntax error) | `import()` promise rejects → ErrorBoundary shows fallback. Other extensions unaffected. |
| Extension component throws during render | React ErrorBoundary catches → shows "Extension Error" card with retry button |
| Extension component infinite loops | Browser's own tab crash protection. Future: wrap in `requestIdleCallback` timeout. |
| Extension manifest has invalid JSON | Go loader logs warning, skips extension. Admin works fine. |
| Extension declares invalid routes | React router ignores invalid paths. No crash. |
| Email provider extension is deactivated | `GetEmailProvider()` returns error → dispatcher logs "no provider" → email not sent, logged as failed. |
| Two extensions claim same slot | Both render as tabs. User sees both. No conflict. |
| Extension settings validation fails | API returns 400 with field-level errors. Extension form shows errors. |

---

## 9. Migration Path

1. Build the extension infrastructure (loader, API, shared globals, slot system, error boundaries)
2. Create SMTP and Resend extensions that mirror current hardcoded functionality
3. Refactor email-settings page to use `<ExtensionSlot>`
4. Remove hardcoded provider selection from email settings
5. Ship default extensions in `extensions/` directory (pre-installed, inactive by default)
6. Users activate their preferred email provider extension

**Backwards compatibility:** The refactored email system should detect if no provider extension is installed and fall back to the current `site_settings`-based provider configuration, logging a deprecation warning.

---

## 10. Out of Scope (Future Work)

- **Extension marketplace / registry** — extensions are installed manually (ZIP/Git) for now
- **Extension dependency system** — extensions don't declare dependencies on other extensions
- **Tengo-based email providers** — only Go-native providers for now (SMTP, Resend)
- **Extension versioning/updates** — no auto-update mechanism
- **Extension permissions/capabilities** — no fine-grained permission model (trust-based)
- **CSS isolation** — extensions share the admin's Tailwind classes (feature, not bug)
