# VibeCMS Product Overview

VibeCMS is a high-performance, AI-native content management system built in Go, engineered specifically for agencies and developers who refuse to compromise between speed and flexibility. In an era where "WordPress bloat" often results in multi-second page loads, VibeCMS aims for a sub-50ms Time to First Byte (TTFB). It achieves this by combining a compiled Go Kernel with an innovative "Kernel + Extensions" architecture, allowing developers to inject heavy business logic via decoupled gRPC plugins, and execute fast event hooks via the Tengo scripting engine.

The system treats content as structured JSON data (JSONB), making it natively "AI-friendly" and allowing for seamless integration with Large Language Models for content generation and SEO optimization. Designed for a hybrid open-core model, VibeCMS is deployed as a single, independent instance per website, ensuring total data isolation and predictable performance for agency-managed portfolios.

## About This Document

**Purpose:** This document anchors the entire bundle by defining product scope, target users, and problem statement. All other bundle documents derive their boundaries from this.

**Consistency requirements:** All other bundle documents (goals.md, product-requirements.md, tech-stack.md, architecture.md, database-schema.md, api-spec.md, folder-structure.md, development-phases.md) must stay within the product boundaries defined here, including the React MFE and Go gRPC Plugin stack.

### Problem Statement

*   **The "WP-Bloat" & Performance Tax:** Standard CMS solutions often take 500ms to 2s+ to respond due to heavy plugin architectures and inefficient database abstraction layers.
*   **The Rebuild Bottleneck:** Modern static site generators (SSGs) or compiled languages like Go usually require a deployment pipeline or binary recompilation to change business logic.
    *   *How we fix it:* Decoupled gRPC extensions and Tengo scripts allow new endpoints and logic to be hot-swapped or booted alongside the core Kernel without bringing the site down.
*   **AI Integration Friction:** Most CMS platforms treat AI as an afterthought, forcing "text-area" hacks rather than structured, schema-first generation.
    *   *How we fix it:* VibeCMS uses a JSON-schema first approach (JSONB) that allows LLMs to "understand" and populate blocks directly.
*   **Monolithic Admin UIs:** Traditional CMS admin panels become slow and fragile as plugins inject conflicting CSS/JS.
    *   *How we fix it:* The React Micro-Frontend (MFE) architecture completely isolates Extension UIs while sharing core dependencies (`react`, `@vibecms/ui`), guaranteeing the admin panel is pristine.

### Target Users

*   **The Performance-Obsessed Agency (Primary):**
    *   **Role:** Technical leads at digital agencies managing 50-500 client sites.
    *   **Goal:** To deliver ultra-fast, SEO-dominant websites that are easy to maintain and monitor centrally.
    *   **How VibeCMS helps:** Provides a secure, "set-and-forget" Go binary with central health monitoring and native high performance.
*   **The AI-First Developer:**
    *   **Role:** Individual developers building modern web apps that leverage LLMs for content.
    *   **Goal:** To build sites where the layout and SEO schema are programmatically generated and updated by AI.
*   **The Enterprise Site Manager:**
    *   **Role:** Internal IT or Marketing leads for a specific high-traffic brand.
    *   **Goal:** Maximum uptime and fast TTFB for better search rankings and conversion.

### Core Workflows

#### 1. The Block-Based Content Edit
*   **Context:** An Editor wants to add a "Call to Action" section to the homepage.
*   **Steps:**
    1.  Editor selects "Add Block" within the Admin UI.
    2.  The React SPA fetches the JSON schema for the available CTA block via the Core API.
    3.  Editor fills in the fields (Title, Button Text, Image).
    4.  Editor hits "Save."
*   **System Response:** The system validates the input against the JSON schema, updates the `blocks_data` column in the PostgreSQL table, and invalidates the in-memory route cache.
*   **Outcome:** The new block appears instantly on the site with zero rebuild required.

#### 2. gRPC / Scripting Extension
*   **Context:** A Developer needs to add a custom contact form that sends data to a specific third-party CRM.
*   **Steps:**
    1.  Developer creates a new gRPC extension binary (or a `.tgo` script if simple enough).
    2.  The Extension uses `CoreAPI.RegisterRoute` to intercept POST requests, or hooks into the event bus for `form.submitted`.
    3.  Developer mounts the extension in the `extensions/` directory.
*   **System Response:** The Plugin Manager detects the extension, evaluates its requested capability manifest (`"events:subscribe"`), and permits it to run.
*   **Outcome:** New business logic safely executes outside the public-facing Go Kernel.

#### 3. AI-Assisted SEO Generation
*   **Context:** An Editor has written a long-form article but lacks meta-tags and Schema.org markup.
*   **Steps:**
    1.  Editor clicks "Suggest SEO" in the sidebar.
    2.  The system sends the structured block data to the configured AI provider.
    3.  The system receives a JSON response containing Meta Title, Description, and FAQ Schema.
*   **System Response:** The Admin UI populates the SEO fields with the suggestions, highlighting them for review.

### Out of Scope

*   **Multi-Website Support (Same Instance):** One deployment = one website. This prevents a single point of failure and simplifies the logic for sub-50ms TTFB.
*   **Centralized Master Dashboard (UI):** VibeCMS provides the API for monitoring, but the actual dashboard app to view all sites at once is a separate product/project.
*   **WASM Extensibility:** While considered, this is deferred in favor of Tengo scripting and gRPC extensions.
*   **Self-Updating Binaries:** Because the app is aimed at agencies with CI/CD or Docker workflows, the CMS will notify of updates but will not attempt to overwrite its own executable.