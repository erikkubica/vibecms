# VibeCMS Goals and Success Metrics

## About This Document

**Purpose:** Defines measurable success criteria. Features and implementation choices in other documents should be evaluated against these goals.

**How AI tools should use this:** Reference these goals when prioritizing features; suggest only work that advances at least one goal.

**Consistency requirements:** Stay aligned with overview.md (product scope); every goal should map to at least one requirement in product-requirements.md.

VibeCMS is built on the premise that speed is a feature and extensibility shouldn't require a compilation pipeline. For agencies managing vast portfolios of sites, "success" is defined by two primary vectors: the performance experienced by the end-user (SEO and conversion) and the operational efficiency experienced by the developer (maintenance and updates). This document establishes the quantitative benchmarks that the Go-based architecture must hit to fulfill its promise as a high-performance alternative to traditional CMS platforms.

---

## Stated Goals

These metrics represent the core value propositions explicitly requested for the VibeCMS project.

**[P0] Core Performance: Time to First Byte (TTFB) — < 50ms**
- Baseline: N/A (new product)
- Target: Under 50ms for 95% of requests (P95) on a standard $5/mo VPS.
- Measurement method: Synthetic monitoring (e.g., OhDear or Datadog) targeting the `/` home route with default theme.
- Timeline: By launch.
- Rationale: This is the flagship architectural requirement; achieving this eliminates the "WP-bloat" performance tax mentioned in the vision.

**[P0] Extension Flexibility: Zero-Rebuild Logic Density — 100%**
- Baseline: N/A (new product)
- Target: 100% of custom business logic (API integrations, form handlers) achievable via Tengo scripting without Go recompilation.
- Measurement method: Internal QA audit of 5 standard agency use-cases (Contact Form -> CRM, Dynamic Price Fetching, Custom Auth Hook).
- Timeline: By launch.
- Rationale: Necessary to support the "Zero-Rebuild" requirement for agency agility.

**[P1] Monitoring Efficiency: Centralized API Response Time — < 100ms**
- Baseline: N/A (new product)
- Target: Health monitoring endpoint (`/api/v1/health`) must return full status (DB, Disk, Updates) in < 100ms.
- Measurement method: `curl` timing on the authenticated health endpoint under 10 concurrent requests.
- Timeline: By launch.
- Rationale: High-speed monitoring is essential for agencies managing hundreds of independent instances via external dashboards.

---

## Architected Goals

These metrics are recommended based on the technical stack (Go, JSONB, Tengo) and the agency-focused distribution model.

**[P0] Database Efficiency: Block Retrieval Latency — < 10ms**
- Baseline: N/A (new product)
- Target: Average latency for fetching structured `blocks_data` (JSONB) from `content_nodes` < 10ms.
- Measurement method: PostgreSQL `EXPLAIN ANALYZE` logs and Go internal telemetry.
- Timeline: By launch.
- Rationale: Because the app relies on fetching and parsing JSONB for every page render, DB retrieval is the primary bottleneck for the 50ms TTFB goal.

**[P1] Media Optimization: Image Payload Reduction — 60%**
- Baseline: N/A (new product)
- Target: Automatic conversion of uploaded JPEG/PNG to WebP resulting in average 60% file size reduction.
- Measurement method: Comparison of `original_size` vs `optimized_size` in the Media Manager database logs.
- Timeline: Within 30 days of v1.0 launch.
- Rationale: Large images negate TTFB gains; native WebP optimization ensures the high-performance promise extends to the full page load.

**[P1] Extension Performance: Script Execution Overhead — < 5ms**
- Baseline: N/A (new product)
- Target: Tengo script execution for standard hooks (e.g., `before_render`) must add < 5ms to total request time.
- Measurement method: Go `time.Since()` benchmarks surrounding the Tengo VM execution block.
- Timeline: By launch.
- Rationale: Since scripts run in a sandboxed VM per request, execution overhead must be negligible to maintain the sub-50ms TTFB target.

**[P2] SEO Automation: Schema Accuracy Score — 90%**
- Baseline: N/A (new product)
- Target: 90% of AI-generated Schema.org blocks must pass the Google Rich Results Test without manual correction.
- Measurement method: Random sample audit of 50 AI-generated nodes using the Google Search Console API.
- Timeline: Within 90 days of v1.0 launch.
- Rationale: To fulfill the "AI-Native" goal, the automated output must be high-quality enough to reduce manual agency labor.

---

### Anti-Goals

- **Multi-Tenant Consolidation:** It is explicitly NOT a goal to support multiple websites within a single PostgreSQL schema or Go process. This keeps the codebase simple and ensures one site's traffic peak cannot crash an agency's entire portfolio.
- **Client-Side Framework Parity:** It is NOT a goal to match the interactivity of React or Vue SPAs in the Admin UI. The focus is on a high-speed, server-rendered HTMX experience to minimize JS shipping to the browser.
- **Universal Scripting Access:** It is NOT a goal for Tengo scripts to have access to the underlying OS or host filesystem. Security and sandbox isolation take priority over script power to prevent instance-takeover via the extension engine.