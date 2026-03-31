# Extensions

Extensions are self-contained feature packages — like WordPress plugins. Each extension owns its full stack: Go plugin binary, admin UI frontend, Tengo scripts, SQL migrations, and manifest.

## Structure

```
extensions/
  my-extension/
    extension.json          # Manifest (capabilities, routes, menus, public_routes)
    bin/my-extension        # Compiled Go plugin binary (built by developer)
    cmd/plugin/main.go      # Go plugin source
    admin-ui/
      src/                  # React/TypeScript source
      dist/index.js         # Built frontend bundle (built by developer)
      package.json
      vite.config.ts
    scripts/                # Tengo (.tgo) scripts
    migrations/             # SQL migrations (run on activation)
```

## Building Extensions

Extensions must be pre-built before deployment. The Dockerfile does NOT build extensions — it only builds the VibeCMS core binary and admin SPA.

### Go Plugin Binary

From the project root:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o extensions/my-extension/bin/my-extension ./extensions/my-extension/cmd/plugin/
```

Note: Binaries must be statically linked for Alpine Docker. Build inside Docker or use `CGO_ENABLED=0` with a compatible toolchain. The `docker compose build` step handles this automatically for built-in extensions using the golang:alpine builder.

### Admin UI Frontend

```bash
cd extensions/my-extension/admin-ui
npm ci
npm run build
```

The built `dist/index.js` is committed to the repo and served as an ES module micro-frontend.

## How Extensions Work

- **Go plugin**: Runs as a separate gRPC process managed by core. Communicates via CoreAPI (data, settings, events, files, email, etc.)
- **Admin UI**: ES module loaded at runtime by the admin SPA shell. Uses shared deps via import maps (`@vibecms/ui`, `@vibecms/icons`, `@vibecms/api`)
- **Public routes**: Extensions can declare `public_routes` in their manifest to serve public URLs (no auth) via the core proxy
- **Tengo scripts**: Event hooks, HTTP routes, and filters registered via the scripting engine

## Manifest (extension.json)

Key fields:
- `capabilities`: CoreAPI permissions the extension needs
- `plugins`: Go binary paths and event subscriptions
- `public_routes`: Public URL patterns proxied to the plugin
- `admin_ui.routes`: Admin SPA routes and components
- `admin_ui.menu`: Sidebar menu entry
- `admin_ui.settings_menu`: Items injected into the Settings sidebar group
- `admin_ui.field_types`: Custom field types for the node editor

See `docs/extension_api.md` for the full API reference.
