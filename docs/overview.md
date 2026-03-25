# VibeCMS Product Overview

VibeCMS is a high-performance, AI-native content management system built in Go, engineered specifically for agencies and developers who refuse to compromise between speed and flexibility. In an era where "WordPress bloat" often results in multi-second page loads, VibeCMS aims for a sub-50ms Time to First Byte (TTFB). It achieves this by combining a compiled Go backend with an innovative "zero-rebuild" extension architecture, allowing developers to inject custom logic via the Tengo scripting language without restarting the server. 

The system treats content as structured JSON data, making it natively "AI-friendly" and allowing for seamless integration with Large Language Models for content generation and SEO optimization. Designed for a hybrid open-core model, VibeCMS is deployed as a single, independent instance per website, ensuring total data isolation and predictable performance for agency-managed portfolios.

## About This Document

**Purpose:** This document anchors the entire bundle by defining product scope, target users, and problem statement. All other bundle documents derive their boundaries from this.

**How AI tools should use this:** Read this document first; use the product scope to constrain feature suggestions; reject code changes outside the described scope.

**Consistency requirements:** All other bundle documents (goals.md, product-requirements.md, tech-stack.md, architecture.md, database-schema.md, api-spec.md, folder-structure.md, development-phases.md) must stay within the product boundaries defined here.

### Problem Statement

*   **The "WP-Bloat" & Performance Tax:** Standard CMS solutions often take 500ms to 2s+ to respond due to heavy plugin architectures and inefficient database abstraction layers.
    *   *Current Workaround:* Users rely on aggressive caching layers (Varnish/Cloudflare) or high-spec servers.
    *   *Inadequacy:* Caching layer complexity introduces "cache invalidation" bugs, and high-spec servers increase monthly overhead for simple sites.
*   **The Rebuild Bottleneck:** Modern static site generators (SSGs) or compiled languages like Go usually require a deployment pipeline or binary recompilation to change business logic.
    *   *Current Workaround:* Developers use complex CI/CD pipelines to trigger rebuilds for every minor logic change.
    *   *Inadequacy:* This creates a time lag between a business requirement and its deployment, complicating the workflow for agencies managing hundreds of sites.
*   **AI Integration Friction:** Most CMS platforms treat AI as an afterthought, forcing "text-area" hacks rather than structured, schema-first generation.
    *   *Current Workaround:* Copy-pasting content from ChatGPT into a CMS text editor.
    *   *Inadequacy:* This loses the structural integrity of the data and makes it impossible for AI to understand the relationships between different content blocks.
*   **Agency Management Overhead:** Managing hundreds of independent site deployments usually requires manual login to dozens of separate admin panels to check health.
    *   *ARCHITECTED PAIN:* Because the system uses a 1-deployment-per-site model, the risk of "silent failure" increases as the agency grows. The system needs a standardized way to report health without a manual check.

### Target Users

*   **The Performance-Obsessed Agency (Primary):**
    *   **Role:** Technical leads at digital agencies managing 50-500 client sites.
    *   **Goal:** To deliver ultra-fast, SEO-dominant websites that are easy to maintain and monitor centrally.
    *   **Frustration:** High maintenance costs and security vulnerabilities associated with PHP-based platforms.
    *   **How VibeCMS helps:** Provides a secure, "set-and-forget" Go binary with central health monitoring and native high performance.
*   **The AI-First Developer:**
    *   **Role:** Individual developers building modern web apps that leverage LLMs for content.
    *   **Goal:** To build sites where the layout and SEO schema are programmatically generated and updated by AI.
    *   **Frustration:** Rigid database schemas in traditional CMSs that don't play well with JSON-based AI responses.
    *   **How VibeCMS helps:** Uses a JSON-schema first approach (JSONB) that allows LLMs to "understand" and populate blocks directly.
*   **The Enterprise Site Manager:**
    *   **Role:** Internal IT or Marketing leads for a specific high-traffic brand.
    *   **Goal:** Maximum uptime and fast TTFB for better search rankings and conversion.
    *   **Frustration:** Complexity of managing heavy infrastructure for a site that "just needs to be fast."
    *   **How VibeCMS helps:** Simplifies the stack to a single binary and a database, reducing the surface area for infrastructure failure.

### Core Workflows

#### 1. The Block-Based Content Edit
*   **Context:** An Editor wants to add a "Call to Action" section to the homepage.
*   **Steps:**
    1.  Editor selects "Add Block" within the Admin UI.
    2.  The UI (rendered via HTMX) fetches the JSON schema for the available CTA block.
    3.  Editor fills in the fields (Title, Button Text, Image).
    4.  Editor hits "Save."
*   **System Response:** The system validates the input against the JSON schema, updates the `blocks_data` column in the PostgreSQL `content_nodes` table, and invalidates the in-memory SEO sitemap.
*   **Outcome:** The new block appears instantly on the site with zero rebuild required.

#### 2. Zero-Rebuild Logic Extension
*   **Context:** A Developer needs to add a custom contact form that sends data to a specific third-party CRM.
*   **Steps:**
    1.  Developer creates a `contact_handler.tgo` file in the theme's script directory.
    2.  The script defines logic to intercept a POST request, parse the JSON, and call the CRM API.
    3.  Developer uploads the file via the Media Manager or SFTP.
*   **System Response:** The Tengo VM detects the new script and executes it when the specific hook (e.g., `on_form_submit`) is triggered.
*   **Outcome:** New business logic is live without stopping the Go executable or recompiling the binary.

#### 3. AI-Assisted SEO Generation
*   **Context:** An Editor has written a long-form article but lacks meta-tags and Schema.org markup.
*   **Steps:**
    1.  Editor clicks "Suggest SEO" in the sidebar.
    2.  The system sends the structured block data to the configured AI provider (OpenAI/Anthropic).
    3.  The system receives a JSON response containing Meta Title, Description, and FAQ Schema.
*   **System Response:** The Admin UI populates the SEO fields with the suggestions, highlighting them for review.
*   **Outcome:** High-quality SEO metadata is generated in seconds based on the actual structural data of the page.

### Out of Scope

*   **Multi-Website Support (Same Instance):** One deployment = one website. This prevents a single point of failure and simplifies the logic for sub-50ms TTFB.
*   **Centralized Master Dashboard (UI):** VibeCMS provides the API for monitoring, but the actual dashboard app to view all sites at once is a separate product/project.
*   **Drag-and-Drop Visual Editor:** To maintain the performance of the HTMX/Alpine.js stack, the MVP focuses on a structured form-based block editor.
*   **Internal API Proxying/Credit Pooling:** The CMS will not manage API credits for Ahrefs/Semrush; each instance must be configured with its own credentials or the agency's shared key.
*   **WASM Extensibility:** While considered, this is deferred in favor of Tengo scripting to keep the initial development footprint small and more focused on Go-like syntax.
*   **Self-Updating Binaries:** Because the app is aimed at agencies with CI/CD or Docker workflows, the CMS will notify of updates but will not attempt to overwrite its own executable.