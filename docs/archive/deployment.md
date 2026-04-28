# VibeCMS Deployment Guide

## About This Document

**Purpose:** Step-by-step environment setup and deployment procedures for every environment this project targets. The authoritative runbook for getting the application running.

**How AI tools should use this:** Reference this document when scaffolding CI/CD pipelines, Dockerfiles, environment configs, or deployment scripts; do not invent configuration values not listed here.

**Consistency requirements:** Technology choices must match tech-stack.md; service names must match architecture.md; environment variables must map to the configuration requirements implied by api-spec.md and database-schema.md.

VibeCMS is designed as a high-performance, single-site CMS. This guide ensures that agencies can deploy independent instances for each client website with consistent performance and reliability. By following these steps, you will establish a production-ready Go environment, configure the necessary PostgreSQL and S3 integrations, and set up a robust CI/CD pipeline for automated updates.

---

### Prerequisites

Before starting the setup, ensure the following tools are installed and accounts are active.

*   **Go 1.22.x**: The core programming language.
    *   Verification: `go version` should print `go version go1.22.x`.
*   **PostgreSQL 16.x**: Required for content storage and JSONB support.
    *   Verification: `psql --version` should print `psql (PostgreSQL) 16.x`.
*   **Docker & Docker Compose v2.x**: For local database orchestration.
    *   Verification: `docker compose version` should print `v2.x.x`.
*   **Node.js 20.x & npm**: Required for Tailwind CSS compilation.
    *   Verification: `node -v` should print `v20.x.x`.
*   **Templ CLI**: To compile type-safe Go components.
    *   Installation: `go install github.com/a-h/templ/cmd/templ@latest`.
    *   Verification: `templ version`.
*   **S3-Compatible Bucket**: AWS S3, DigitalOcean Spaces, or Cloudflare R2 bucket for media storage.
*   **Resend.com API Key**: For transactional email delivery.

---

### Environment Variables

VibeCMS uses environment variables for all sensitive and environment-specific configurations. Copy `.env.example` to `.env` and fill in the values.

| Variable | Required | Example Value | Description |
|----------|----------|---------------|-------------|
| `PORT` | No | `8080` | The port the Fiber server listens on. |
| `DATABASE_URL` | Yes | `postgres://user:pass@localhost:5432/vibecms` | PostgreSQL connection string. |
| `LICENSE_KEY` | Yes | `vibe_live_abc123...` | Ed25519 signed license key for the domain. |
| `VIBE_ENV` | No | `production` | Environment mode (`development` or `production`). |
| `MONITORING_TOKEN` | Yes | `sk_mon_987654321` | Static Bearer token for the Health API endpoints. |
| `STORAGE_DRIVER` | No | `s3` | Storage backend (`local` or `s3`). |
| `S3_ENDPOINT` | Conditional | `https://s3.amazonaws.com` | S3 endpoint (Required if driver is `s3`). |
| `S3_ACCESS_KEY` | Conditional | `AKIA...` | S3 Access Key. |
| `S3_SECRET_KEY` | Conditional | `secret_abc...` | S3 Secret Key. |
| `S3_BUCKET` | Conditional | `my-vibe-assets` | S3 Bucket Name. |
| `RESEND_API_KEY` | Yes | `re_123456789` | API Key for Resend.com mail integration. |
| `OPENAI_API_KEY` | No | `sk-proj-abc...` | Key for AI-native SEO/Block suggestions. |

---

### Local Development Setup

Follow these steps to get VibeCMS running on your local workstation.

1.  **Clone the Repository:**
    `git clone https://github.com/your-org/vibecms.git && cd vibecms`
    *Performance: Standard git clone.*

2.  **Spin up Postgres via Docker:**
    `docker compose up -d db`
    *Outcome: A PostgreSQL container is running on port 5432.*

3.  **Install Node Dependencies:**
    `npm install`
    *Outcome: Tailwind CSS and build tools are installed.*

4.  **Install Go Dependencies:**
    `go mod download`
    *Outcome: All libraries from go.mod are cached locally.*

5.  **Generate Templ Components:**
    `templ generate`
    *Outcome: Go source files are generated in `ui/admin/`.*

6.  **Compile Tailwind CSS:**
    `npx tailwindcss -i ./ui/assets/css/input.css -o ./ui/assets/css/style.css`
    *Outcome: `style.css` is ready for the Admin UI.*

7.  **Run Migrations and Start App:**
    `go run cmd/vibecms/main.go`
    *Outcome: You should see "Fiber server started on :8080" and "Migrations completed".*

---

### CI/CD Pipeline

The following GitHub Actions workflow automates testing, asset compilation, and deployment to a production server via SSH.

```yaml
name: VibeCMS CI/CD

on:
  push:
    branches: [ main ]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build Assets
        run: |
          npm install
          npx tailwindcss -o ./ui/assets/css/style.css --minify
          go install github.com/a-h/templ/cmd/templ@latest
          templ generate

      - name: Run Tests
        run: go test ./...

      - name: Build Binary
        run: |
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o vibecms cmd/vibecms/main.go

      - name: Deploy via SSH
        uses: appleboy/scp-action@master
        with:
          host: ${{ secrets.PROD_HOST }}
          username: ${{ secrets.PROD_USER }}
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          source: "./vibecms,./themes/,./ui/assets/"
          target: "/var/www/vibecms"

      - name: Restart Service
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.PROD_HOST }}
          username: ${{ secrets.PROD_USER }}
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          script: |
            sudo systemctl restart vibecms
            curl -f -H "Authorization: Bearer ${{ secrets.MONITORING_TOKEN }}" http://localhost:8080/api/health
```

---

### Production Deployment

1.  **Pre-deployment Check:** Verify that the target production server has the environment variables defined in `/etc/vibecms/.env`.
2.  **Binary Update:** Transfer the new `vibecms` binary to the target directory.
3.  **Template Update:** sync the `themes/` directory to ensure new Jet templates are available.
4.  **Database Migration:** Restart the service. The VibeCMS binary automatically detects version changes and executes SQL migrations in `internal/db/migrations/` within a transaction.
5.  **Post-deployment Verification:**
    *   Run `curl -I https://yourdomain.com` to check for `HTTP/2 200` and TTFB < 50ms.
    *   Log in to `/admin` to verify HTMX interactions work (e.g., adding a block).
6.  **Downtime:** Approximately 2–5 seconds during the systemd service restart.

---

### Rollback Procedure

In the event of a critical failure (e.g., 500 errors on the public site or database corruption):

1.  **Identify Previous Version:** CI/CD keeps the last stable binary as `vibecms.old`.
2.  **Execute Rollback Command:**
    `ssh user@host "cd /var/www/vibecms && mv vibecms.old vibecms && sudo systemctl restart vibecms"`
3.  **Database Rollback:** If the failure was due to a migration, manually revert the database to the latest snapshot taken by the Internal Cron System before deployment.
    *   Command: `pg_restore -d vibecms /var/www/vibecms/storage/backups/pre-deploy.dump`.
4.  **Verification:** Check the Health Monitoring API (`/api/health`) to confirm the status is `OK`.