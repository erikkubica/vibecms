export interface User {
  id: number;
  email: string;
  full_name: string;
  role_id: number;
  role: { id: number; slug: string; name: string; is_system: boolean } | string;
  capabilities?: Record<string, any>;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
  language_id?: number | null;
}

export interface ContentNode {
  id: number;
  uuid: string;
  parent_id: number | null;
  node_type: string;
  status: string;
  language_code: string;
  slug: string;
  full_url: string;
  title: string;
  featured_image: Record<string, unknown>;
  excerpt: string;
  taxonomies: Record<string, string[]>;
  blocks_data: Record<string, unknown>[];
  seo_settings: Record<string, unknown>;
  fields_data: Record<string, unknown>;
  layout_data: Record<string, Record<string, unknown>>;
  layout_id: number | null;
  translation_group_id: string | null;
  version: number;
  published_at: string | null;
  created_at: string;
  updated_at: string;
  is_homepage?: boolean;
}

export interface PaginationMeta {
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface ApiResponse<T> {
  data: T;
  meta?: PaginationMeta;
}

interface ApiError {
  error: {
    code: string;
    message: string;
  };
}

class ApiClientError extends Error {
  code: string;
  status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = "ApiClientError";
    this.code = code;
    this.status = status;
  }
}

// adminLangHeader reads the admin's currently-selected language code from
// localStorage (mirrors AdminLanguageProvider's STORAGE_KEY) so every API
// request carries it. Locale-aware backend handlers (theme-settings and any
// future settings endpoints) use it to scope reads/writes. "all" or empty
// becomes the empty string, which the backend treats as the fallback row.
function adminLangHeader(): string {
  if (typeof localStorage === "undefined") return "";
  const code = localStorage.getItem("squilla_admin_lang") || "";
  return code === "all" ? "" : code;
}

async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const lang = adminLangHeader();
  const res = await fetch(path, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(lang ? { "X-Admin-Language": lang } : {}),
      ...options?.headers,
    },
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const body = await res.json();

  if (!res.ok) {
    const error = (body as ApiError).error;
    throw new ApiClientError(
      error?.message || "An unexpected error occurred",
      error?.code || "unknown_error",
      res.status
    );
  }

  return body;
}

export async function login(data: Record<string, string>): Promise<User> {
  const res = await api<ApiResponse<User>>("/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function logout(): Promise<void> {
  await api("/auth/logout", { method: "POST" });
}

export async function getMe(): Promise<User> {
  const res = await api<ApiResponse<User>>("/me");
  return res.data;
}

export interface GetNodesParams {
  page?: number;
  per_page?: number;
  status?: string;
  node_type?: string;
  language_code?: string;
  search?: string;
  tax_query?: Record<string, string[]>;
}

export async function getNodes(params: GetNodesParams): Promise<{ data: ContentNode[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set("page", String(params.page));
  if (params.per_page) searchParams.set("per_page", String(params.per_page));
  if (params.status) searchParams.set("status", params.status);
  if (params.node_type) searchParams.set("node_type", params.node_type);
  if (params.language_code) searchParams.set("language_code", params.language_code);
  if (params.search) searchParams.set("search", params.search);
  if (params.tax_query) searchParams.set("tax_query", JSON.stringify(params.tax_query));

  const res = await api<{ data: ContentNode[]; meta: PaginationMeta }>(
    `/admin/api/nodes?${searchParams.toString()}`
  );
  return res;
}

export async function getNode(id: number | string): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>(`/admin/api/nodes/${id}`);
  return res.data;
}

export async function createNode(data: Partial<ContentNode>): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>("/admin/api/nodes", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateNode(id: number | string, data: Partial<ContentNode>): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>(`/admin/api/nodes/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteNode(id: number | string): Promise<void> {
  await api(`/admin/api/nodes/${id}`, { method: "DELETE" });
}

export async function getNodeTranslations(id: number | string): Promise<ContentNode[]> {
  const res = await api<ApiResponse<ContentNode[]>>(`/admin/api/nodes/${id}/translations`);
  return res.data;
}

export async function createNodeTranslation(id: number | string, data: { language_code: string }): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>(`/admin/api/nodes/${id}/translations`, {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function getHomepageId(): Promise<number> {
  const settings = await getSiteSettings();
  return Number(settings.homepage_node_id) || 0;
}

export async function setHomepage(nodeId: number | string): Promise<void> {
  await updateSiteSettings({ homepage_node_id: String(nodeId) });
}

export interface NodeTypeField {
  name: string;
  key: string;
  label: string;
  type: string;
  required?: boolean;
  options?: string[];
  placeholder?: string;
  default_value?: any;
  help?: string;
  sub_fields?: NodeTypeField[];
  node_type_filter?: string;
  taxonomy?: string;
  term_node_type?: string;
  multiple?: boolean;
  min?: number;
  max?: number;
  step?: number;
  min_length?: number;
  max_length?: number;
  rows?: number;
  prepend?: string;
  append?: string;
  allowed_types?: string;
  width?: number;
}

export interface NodeSearchResult {
  id: number;
  title: string;
  slug: string;
  node_type: string;
  status: string;
  language_code: string;
}

export async function searchNodes(params: {
  q?: string;
  node_type?: string;
  limit?: number;
}): Promise<NodeSearchResult[]> {
  const searchParams = new URLSearchParams();
  if (params.q) searchParams.set("q", params.q);
  if (params.node_type) searchParams.set("node_type", params.node_type);
  if (params.limit) searchParams.set("limit", String(params.limit));
  const res = await api<ApiResponse<NodeSearchResult[]>>(
    `/admin/api/nodes/search?${searchParams.toString()}`
  );
  return res.data;
}

export interface TaxonomyDefinition {
  slug: string;
  label: string;
  multiple: boolean;
}

export interface NodeType {
  id: number;
  slug: string;
  label: string;
  label_plural?: string;
  icon: string;
  description: string;
  taxonomies: TaxonomyDefinition[];
  field_schema: NodeTypeField[];
  url_prefixes: Record<string, string>;
  supports_blocks?: boolean;
  created_at: string;
  updated_at: string;
}

export async function getNodeTypes(): Promise<NodeType[]> {
  const res = await api<ApiResponse<NodeType[]>>("/admin/api/node-types");
  return res.data;
}

export async function getNodeType(id: number | string): Promise<NodeType> {
  const res = await api<ApiResponse<NodeType>>(`/admin/api/node-types/${id}`);
  return res.data;
}

export async function createNodeType(data: Partial<NodeType>): Promise<NodeType> {
  const res = await api<ApiResponse<NodeType>>("/admin/api/node-types", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateNodeType(id: number | string, data: Partial<NodeType>): Promise<NodeType> {
  const res = await api<ApiResponse<NodeType>>(`/admin/api/node-types/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteNodeType(id: number | string): Promise<void> {
  await api<void>(`/admin/api/node-types/${id}`, { method: "DELETE" });
}

export interface Taxonomy {
  id: number;
  slug: string;
  label: string;
  label_plural?: string;
  description: string;
  node_types: string[];
  field_schema: NodeTypeField[];
  hierarchical?: boolean;
  show_ui?: boolean;
  created_at: string;
  updated_at: string;
}

export async function getTaxonomies(): Promise<Taxonomy[]> {
  const res = await api<ApiResponse<Taxonomy[]>>("/admin/api/taxonomies");
  return res.data;
}

export async function getTaxonomy(slug: string): Promise<Taxonomy> {
  const res = await api<ApiResponse<Taxonomy>>(`/admin/api/taxonomies/${slug}`);
  return res.data;
}

export async function createTaxonomy(data: Partial<Taxonomy>): Promise<Taxonomy> {
  const res = await api<ApiResponse<Taxonomy>>("/admin/api/taxonomies", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateTaxonomy(slug: string, data: Partial<Taxonomy>): Promise<Taxonomy> {
  const res = await api<ApiResponse<Taxonomy>>(`/admin/api/taxonomies/${slug}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteTaxonomy(slug: string): Promise<void> {
  await api(`/admin/api/taxonomies/${slug}`, {
    method: "DELETE",
  });
}

export interface TaxonomyTerm {
  id: number;
  node_type: string;
  taxonomy: string;
  language_code: string;
  translation_group_id?: string;
  slug: string;
  name: string;
  description: string;
  parent_id?: number;
  count: number;
  fields_data: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export async function listTerms(
  nodeType: string,
  taxonomy: string,
  opts?: { language_code?: string },
): Promise<TaxonomyTerm[]> {
  // Pass language as a query param so the call doesn't depend on the
  // admin's current header language. Callers that want every locale (e.g.
  // an audit view) pass language_code: "all".
  const qs = opts?.language_code
    ? `?language_code=${encodeURIComponent(opts.language_code)}`
    : "";
  const res = await api<ApiResponse<TaxonomyTerm[]>>(
    `/admin/api/terms/${nodeType}/${taxonomy}${qs}`,
  );
  return res.data;
}

export async function getTerm(id: number): Promise<TaxonomyTerm> {
  const res = await api<ApiResponse<TaxonomyTerm>>(`/admin/api/terms/${id}`);
  return res.data;
}

export async function createTerm(nodeType: string, taxonomy: string, data: Partial<TaxonomyTerm>): Promise<TaxonomyTerm> {
  const res = await api<ApiResponse<TaxonomyTerm>>(`/admin/api/terms/${nodeType}/${taxonomy}`, {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateTerm(id: number, data: Partial<TaxonomyTerm>): Promise<TaxonomyTerm> {
  const res = await api<ApiResponse<TaxonomyTerm>>(`/admin/api/terms/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteTerm(id: number): Promise<void> {
  await api(`/admin/api/terms/${id}`, {
    method: "DELETE",
  });
}

export async function getTermTranslations(id: number): Promise<TaxonomyTerm[]> {
  const res = await api<ApiResponse<TaxonomyTerm[]>>(`/admin/api/terms/${id}/translations`);
  return res.data;
}

export async function createTermTranslation(
  id: number,
  data: { language_code: string },
): Promise<TaxonomyTerm> {
  const res = await api<ApiResponse<TaxonomyTerm>>(`/admin/api/terms/${id}/translations`, {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function listTaxonomyTerms(nodeType: string, taxonomy: string): Promise<string[]> {
  const res = await api<ApiResponse<string[]>>(`/admin/api/taxonomies/${nodeType}/${taxonomy}/terms`);
  return res.data;
}

export interface Language {
  id: number;
  code: string;
  slug: string;
  name: string;
  native_name: string;
  flag: string;
  is_default: boolean;
  is_active: boolean;
  hide_prefix: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export async function getLanguages(activeOnly?: boolean): Promise<Language[]> {
  const params = activeOnly ? "?active=true" : "";
  const res = await api<ApiResponse<Language[]>>(`/admin/api/languages${params}`);
  return res.data;
}

export async function getLanguage(id: number | string): Promise<Language> {
  const res = await api<ApiResponse<Language>>(`/admin/api/languages/${id}`);
  return res.data;
}

export async function createLanguage(data: Partial<Language>): Promise<Language> {
  const res = await api<ApiResponse<Language>>("/admin/api/languages", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateLanguage(id: number | string, data: Partial<Language>): Promise<Language> {
  const res = await api<ApiResponse<Language>>(`/admin/api/languages/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteLanguage(id: number | string): Promise<void> {
  await api<void>(`/admin/api/languages/${id}`, { method: "DELETE" });
}

export interface BlockType {
  id: number;
  slug: string;
  label: string;
  icon: string;
  description: string;
  field_schema: NodeTypeField[];
  html_template: string;
  test_data: Record<string, unknown>;
  source: string;
  theme_name: string | null;
  cache_output: boolean;
  created_at: string;
  updated_at: string;
}

export interface TemplateBlockConfig {
  block_type_slug: string;
  default_values: Record<string, unknown>;
}

export interface Template {
  id: number;
  slug: string;
  label: string;
  description: string;
  source: string;
  theme_name: string | null;
  block_config: TemplateBlockConfig[];
  created_at: string;
  updated_at: string;
}

export async function getBlockTypes(): Promise<BlockType[]> {
  // Request a high per_page so all block types are returned in one page —
  // node-editor and template-editor need the full set to resolve field
  // schemas for any block type referenced in content.
  const res = await api<ApiResponse<BlockType[]>>("/admin/api/block-types?per_page=1000");
  return res.data;
}

export async function getBlockTypesPaginated(params?: { page?: number; per_page?: number }): Promise<{ data: BlockType[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.per_page) searchParams.set("per_page", String(params.per_page));
  const qs = searchParams.toString();
  const res = await api<{ data: BlockType[]; meta: PaginationMeta }>(`/admin/api/block-types${qs ? `?${qs}` : ""}`);
  return res;
}

export async function getBlockType(id: number | string): Promise<BlockType> {
  const res = await api<ApiResponse<BlockType>>(`/admin/api/block-types/${id}`);
  return res.data;
}

export async function createBlockType(data: Partial<BlockType>): Promise<BlockType> {
  const res = await api<ApiResponse<BlockType>>("/admin/api/block-types", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateBlockType(id: number | string, data: Partial<BlockType>): Promise<BlockType> {
  const res = await api<ApiResponse<BlockType>>(`/admin/api/block-types/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteBlockType(id: number | string): Promise<void> {
  await api<void>(`/admin/api/block-types/${id}`, { method: "DELETE" });
}

export async function detachBlockType(id: number | string): Promise<BlockType> {
  const res = await api<ApiResponse<BlockType>>(`/admin/api/block-types/${id}/detach`, {
    method: "POST",
  });
  return res.data;
}

export async function reattachBlockType(id: number | string): Promise<BlockType> {
  const res = await api<ApiResponse<BlockType>>(`/admin/api/block-types/${id}/reattach`, {
    method: "POST",
  });
  return res.data;
}

export async function getSiteSettings(
  locale?: string,
): Promise<Record<string, string>> {
  const res = await api<ApiResponse<Record<string, string>>>(
    "/admin/api/settings",
    { headers: siteSettingsLocaleHeader(locale) },
  );
  return res.data;
}

export async function updateSiteSettings(
  settings: Record<string, string>,
  locale?: string,
): Promise<void> {
  await api<ApiResponse<{ message: string }>>("/admin/api/settings", {
    method: "PUT",
    body: JSON.stringify(settings),
    headers: siteSettingsLocaleHeader(locale),
  });
}

// siteSettingsLocaleHeader is the same shape as localeHeader (defined further
// down in this file). Re-declared here so this block doesn't depend on source
// ordering — the helper is trivial.
function siteSettingsLocaleHeader(locale?: string): Record<string, string> {
  if (!locale || locale === "all") return {};
  return { "X-Admin-Language": locale };
}

// ---------------------------------------------------------------------------
// Theme settings
// ---------------------------------------------------------------------------

export interface ThemeSettingsPageSummary {
  slug: string;
  name: string;
  icon?: string;
}

export interface ThemeSettingsListResponse {
  active_theme_slug: string;
  pages: ThemeSettingsPageSummary[];
}

export interface ThemeSettingsField {
  key: string;
  label: string;
  type: string;
  default?: unknown;
  config?: Record<string, unknown>;
}

export interface ThemeSettingsPage {
  slug: string;
  name: string;
  description?: string;
  icon?: string;
  fields: ThemeSettingsField[];
}

export interface ThemeSettingsValue {
  value: unknown;
  compatible: boolean;
  raw: string;
}

export interface ThemeSettingsPageResponse {
  page: ThemeSettingsPage;
  values: Record<string, ThemeSettingsValue>;
}

export async function getThemeSettingsPages(): Promise<ThemeSettingsListResponse> {
  const res = await api<ApiResponse<ThemeSettingsListResponse>>("/admin/api/theme-settings");
  return res.data;
}

// localeHeader builds the X-Admin-Language override header. An empty string
// or "all" collapses to no header so the api wrapper falls back to the global
// admin language. Used by editors that pin themselves to a specific locale
// independent of the header default.
function localeHeader(locale?: string): Record<string, string> {
  if (!locale || locale === "all") return {};
  return { "X-Admin-Language": locale };
}

export async function getThemeSettingsPage(
  slug: string,
  locale?: string,
): Promise<ThemeSettingsPageResponse> {
  const res = await api<ApiResponse<ThemeSettingsPageResponse>>(
    `/admin/api/theme-settings/${encodeURIComponent(slug)}`,
    { headers: localeHeader(locale) },
  );
  return res.data;
}

export async function saveThemeSettingsPage(
  slug: string,
  values: Record<string, unknown>,
  locale?: string,
): Promise<void> {
  await api(`/admin/api/theme-settings/${encodeURIComponent(slug)}`, {
    method: "PUT",
    body: JSON.stringify({ values }),
    headers: localeHeader(locale),
  });
}

export async function clearCache(): Promise<{ message: string }> {
  const res = await api<ApiResponse<{ message: string }>>("/admin/api/cache/clear", {
    method: "POST",
  });
  return res.data;
}

export async function getCacheStats(): Promise<Record<string, unknown>> {
  const res = await api<ApiResponse<Record<string, unknown>>>("/admin/api/cache/stats");
  return res.data;
}

export async function previewBlockTemplate(htmlTemplate: string, testData: Record<string, unknown>): Promise<{ html: string; head: string; body_class: string }> {
  const res = await api<{ html: string; head?: string; body_class?: string }>(`/admin/api/block-types/preview`, {
    method: "POST",
    body: JSON.stringify({ html_template: htmlTemplate, test_data: testData }),
  });
  return { html: res.html, head: res.head || "", body_class: res.body_class || "" };
}

export async function getTemplates(): Promise<Template[]> {
  const res = await api<ApiResponse<Template[]>>("/admin/api/templates");
  return res.data;
}

export async function getTemplatesPaginated(params?: { page?: number; per_page?: number }): Promise<{ data: Template[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.per_page) searchParams.set("per_page", String(params.per_page));
  const qs = searchParams.toString();
  const res = await api<{ data: Template[]; meta: PaginationMeta }>(`/admin/api/templates${qs ? `?${qs}` : ""}`);
  return res;
}

export async function getTemplate(id: number | string): Promise<Template> {
  const res = await api<ApiResponse<Template>>(`/admin/api/templates/${id}`);
  return res.data;
}

export async function createTemplate(data: Partial<Template>): Promise<Template> {
  const res = await api<ApiResponse<Template>>("/admin/api/templates", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateTemplate(id: number | string, data: Partial<Template>): Promise<Template> {
  const res = await api<ApiResponse<Template>>(`/admin/api/templates/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteTemplate(id: number | string): Promise<void> {
  await api<void>(`/admin/api/templates/${id}`, { method: "DELETE" });
}

export async function detachTemplate(id: number | string): Promise<Template> {
  const res = await api<ApiResponse<Template>>(`/admin/api/templates/${id}/detach`, {
    method: "POST",
  });
  return res.data;
}

export async function reattachTemplate(id: number | string): Promise<Template> {
  const res = await api<ApiResponse<Template>>(`/admin/api/templates/${id}/reattach`, {
    method: "POST",
  });
  return res.data;
}

// --- Layouts ---

export interface Layout {
  id: number;
  slug: string;
  name: string;
  description: string;
  language_id: number | null;
  template_code: string;
  source: string;
  theme_name: string | null;
  is_default: boolean;
  supports_blocks?: boolean;
  created_at: string;
  updated_at: string;
}

export async function getLayouts(params?: { language_id?: number; source?: string }): Promise<Layout[]> {
  const searchParams = new URLSearchParams();
  if (params?.language_id != null) searchParams.set("language_id", String(params.language_id));
  if (params?.source) searchParams.set("source", params.source);
  const qs = searchParams.toString();
  const res = await api<ApiResponse<Layout[]>>(`/admin/api/layouts${qs ? `?${qs}` : ""}`);
  return res.data;
}

export async function getLayoutsPaginated(params?: { language_id?: number; source?: string; page?: number; per_page?: number }): Promise<{ data: Layout[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params?.language_id != null) searchParams.set("language_id", String(params.language_id));
  if (params?.source) searchParams.set("source", params.source);
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.per_page) searchParams.set("per_page", String(params.per_page));
  const qs = searchParams.toString();
  const res = await api<{ data: Layout[]; meta: PaginationMeta }>(`/admin/api/layouts${qs ? `?${qs}` : ""}`);
  return res;
}

export async function getLayout(id: number | string): Promise<Layout> {
  const res = await api<ApiResponse<Layout>>(`/admin/api/layouts/${id}`);
  return res.data;
}

export async function createLayout(data: Partial<Layout>): Promise<Layout> {
  const res = await api<ApiResponse<Layout>>("/admin/api/layouts", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateLayout(id: number | string, data: Partial<Layout>): Promise<Layout> {
  const res = await api<ApiResponse<Layout>>(`/admin/api/layouts/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteLayout(id: number | string): Promise<void> {
  await api<void>(`/admin/api/layouts/${id}`, { method: "DELETE" });
}

export async function detachLayout(id: number | string): Promise<Layout> {
  const res = await api<ApiResponse<Layout>>(`/admin/api/layouts/${id}/detach`, {
    method: "POST",
  });
  return res.data;
}

export async function reattachLayout(id: number | string): Promise<Layout> {
  const res = await api<ApiResponse<Layout>>(`/admin/api/layouts/${id}/reattach`, {
    method: "POST",
  });
  return res.data;
}

export async function getLayoutPartials(layoutId: number | string): Promise<LayoutBlock[]> {
  const res = await api<ApiResponse<LayoutBlock[]>>(`/admin/api/layouts/${layoutId}/partials`);
  return res.data;
}

// --- Layout Blocks ---

export interface LayoutBlock {
  id: number;
  slug: string;
  name: string;
  description: string;
  language_id: number | null;
  template_code: string;
  field_schema: NodeTypeField[];
  source: string;
  theme_name: string | null;
  created_at: string;
  updated_at: string;
}

export async function getLayoutBlocks(params?: { language_id?: number; source?: string }): Promise<LayoutBlock[]> {
  const searchParams = new URLSearchParams();
  if (params?.language_id != null) searchParams.set("language_id", String(params.language_id));
  if (params?.source) searchParams.set("source", params.source);
  const qs = searchParams.toString();
  const res = await api<ApiResponse<LayoutBlock[]>>(`/admin/api/layout-blocks${qs ? `?${qs}` : ""}`);
  return res.data;
}

export async function getLayoutBlocksPaginated(params?: { language_id?: number; source?: string; page?: number; per_page?: number }): Promise<{ data: LayoutBlock[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params?.language_id != null) searchParams.set("language_id", String(params.language_id));
  if (params?.source) searchParams.set("source", params.source);
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.per_page) searchParams.set("per_page", String(params.per_page));
  const qs = searchParams.toString();
  const res = await api<{ data: LayoutBlock[]; meta: PaginationMeta }>(`/admin/api/layout-blocks${qs ? `?${qs}` : ""}`);
  return res;
}

export async function getLayoutBlock(id: number | string): Promise<LayoutBlock> {
  const res = await api<ApiResponse<LayoutBlock>>(`/admin/api/layout-blocks/${id}`);
  return res.data;
}

export async function createLayoutBlock(data: Partial<LayoutBlock>): Promise<LayoutBlock> {
  const res = await api<ApiResponse<LayoutBlock>>("/admin/api/layout-blocks", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateLayoutBlock(id: number | string, data: Partial<LayoutBlock>): Promise<LayoutBlock> {
  const res = await api<ApiResponse<LayoutBlock>>(`/admin/api/layout-blocks/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteLayoutBlock(id: number | string): Promise<void> {
  await api<void>(`/admin/api/layout-blocks/${id}`, { method: "DELETE" });
}

export async function detachLayoutBlock(id: number | string): Promise<LayoutBlock> {
  const res = await api<ApiResponse<LayoutBlock>>(`/admin/api/layout-blocks/${id}/detach`, {
    method: "POST",
  });
  return res.data;
}

export async function reattachLayoutBlock(id: number | string): Promise<LayoutBlock> {
  const res = await api<ApiResponse<LayoutBlock>>(`/admin/api/layout-blocks/${id}/reattach`, {
    method: "POST",
  });
  return res.data;
}

// --- Menus ---

export interface MenuItem {
  id?: number;
  _uid?: string;
  title: string;
  item_type: "node" | "custom";
  node_id?: number | null;
  url?: string;
  target: string;
  css_class?: string;
  children?: MenuItem[];
}

export interface Menu {
  id: number;
  slug: string;
  name: string;
  language_id: number | null;
  version: number;
  items: MenuItem[];
  created_at: string;
  updated_at: string;
}

export async function getMenus(params?: { language_id?: number }): Promise<Menu[]> {
  const searchParams = new URLSearchParams();
  if (params?.language_id != null) searchParams.set("language_id", String(params.language_id));
  const qs = searchParams.toString();
  const res = await api<ApiResponse<Menu[]>>(`/admin/api/menus${qs ? `?${qs}` : ""}`);
  return res.data;
}

export async function getMenu(id: number | string): Promise<Menu> {
  const res = await api<ApiResponse<Menu>>(`/admin/api/menus/${id}`);
  return res.data;
}

export async function createMenu(data: Partial<Menu>): Promise<Menu> {
  const res = await api<ApiResponse<Menu>>("/admin/api/menus", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateMenu(id: number | string, data: Partial<Menu>): Promise<Menu> {
  const res = await api<ApiResponse<Menu>>(`/admin/api/menus/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteMenu(id: number | string): Promise<void> {
  await api<void>(`/admin/api/menus/${id}`, { method: "DELETE" });
}

export async function replaceMenuItems(id: number | string, version: number, items: MenuItem[]): Promise<Menu> {
  const res = await api<ApiResponse<Menu>>(`/admin/api/menus/${id}/items`, {
    method: "PUT",
    body: JSON.stringify({ version, items }),
  });
  return res.data;
}

// --- Roles ---

export interface Role {
  id: number;
  slug: string;
  name: string;
  description: string;
  is_system: boolean;
  capabilities: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export async function getRoles(): Promise<Role[]> {
  const res = await api<ApiResponse<Role[]>>("/admin/api/roles");
  return res.data;
}

export async function getRole(id: number): Promise<Role> {
  const res = await api<ApiResponse<Role>>(`/admin/api/roles/${id}`);
  return res.data;
}

export async function createRole(data: Partial<Role>): Promise<Role> {
  const res = await api<ApiResponse<Role>>("/admin/api/roles", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateRole(id: number, data: Partial<Role>): Promise<Role> {
  const res = await api<ApiResponse<Role>>(`/admin/api/roles/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteRole(id: number): Promise<void> {
  await api<void>(`/admin/api/roles/${id}`, { method: "DELETE" });
}

// --- System Actions ---

export interface SystemAction {
  id: number;
  slug: string;
  label: string;
  category: string;
  description: string;
}

export async function getSystemActions(): Promise<SystemAction[]> {
  const res = await api<ApiResponse<SystemAction[]>>("/admin/api/system-actions");
  return res.data;
}

// --- Users ---

export async function getUsers(params?: { page?: number; per_page?: number }): Promise<{ data: User[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.per_page) searchParams.set("per_page", String(params.per_page));
  const qs = searchParams.toString();
  const res = await api<{ data: User[]; meta: PaginationMeta }>(`/admin/api/users${qs ? `?${qs}` : ""}`);
  return res;
}

export async function getUser(id: number): Promise<User> {
  const res = await api<ApiResponse<User>>(`/admin/api/users/${id}`);
  return res.data;
}

export async function createUser(data: Partial<User>): Promise<User> {
  const res = await api<ApiResponse<User>>("/admin/api/users", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateUser(id: number, data: Partial<User>): Promise<User> {
  const res = await api<ApiResponse<User>>(`/admin/api/users/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteUser(id: number): Promise<void> {
  await api<void>(`/admin/api/users/${id}`, { method: "DELETE" });
}

// --- Email Templates ---

export interface EmailTemplate {
  id: number;
  slug: string;
  name: string;
  language_id: number | null;
  subject_template: string;
  body_template: string;
  test_data: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export async function getEmailTemplates(): Promise<EmailTemplate[]> {
  const res = await api<ApiResponse<EmailTemplate[]>>("/admin/api/ext/email-manager/templates");
  return res.data;
}

export async function getEmailTemplate(id: number): Promise<EmailTemplate> {
  const res = await api<ApiResponse<EmailTemplate>>(`/admin/api/ext/email-manager/templates/${id}`);
  return res.data;
}

export async function createEmailTemplate(data: Partial<EmailTemplate>): Promise<EmailTemplate> {
  const res = await api<ApiResponse<EmailTemplate>>("/admin/api/ext/email-manager/templates", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateEmailTemplate(id: number, data: Partial<EmailTemplate>): Promise<EmailTemplate> {
  const res = await api<ApiResponse<EmailTemplate>>(`/admin/api/ext/email-manager/templates/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteEmailTemplate(id: number): Promise<void> {
  await api<void>(`/admin/api/ext/email-manager/templates/${id}`, { method: "DELETE" });
}

// --- Email Rules ---

export interface EmailRule {
  id: number;
  action: string;
  node_type: string | null;
  template_id: number;
  template?: EmailTemplate;
  recipient_type: string;
  recipient_value: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export async function getEmailRules(): Promise<EmailRule[]> {
  const res = await api<ApiResponse<EmailRule[]>>("/admin/api/ext/email-manager/rules");
  return res.data;
}

export async function getEmailRule(id: number): Promise<EmailRule> {
  const res = await api<ApiResponse<EmailRule>>(`/admin/api/ext/email-manager/rules/${id}`);
  return res.data;
}

export async function createEmailRule(data: Partial<EmailRule>): Promise<EmailRule> {
  const res = await api<ApiResponse<EmailRule>>("/admin/api/ext/email-manager/rules", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateEmailRule(id: number, data: Partial<EmailRule>): Promise<EmailRule> {
  const res = await api<ApiResponse<EmailRule>>(`/admin/api/ext/email-manager/rules/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteEmailRule(id: number): Promise<void> {
  await api<void>(`/admin/api/ext/email-manager/rules/${id}`, { method: "DELETE" });
}

// --- Email Logs ---

export interface EmailLog {
  id: number;
  rule_id: number | null;
  template_slug: string;
  action: string;
  recipient_email: string;
  subject: string;
  rendered_body: string;
  status: string;
  error_message: string | null;
  provider: string | null;
  created_at: string;
}

export async function getEmailLogs(params?: {
  status?: string;
  action?: string;
  recipient?: string;
  date_from?: string;
  date_to?: string;
  page?: number;
  per_page?: number;
}): Promise<{ data: EmailLog[]; total: number; page: number; per_page: number }> {
  const searchParams = new URLSearchParams();
  if (params?.status) searchParams.set("status", params.status);
  if (params?.action) searchParams.set("action", params.action);
  if (params?.recipient) searchParams.set("recipient", params.recipient);
  if (params?.date_from) searchParams.set("date_from", params.date_from);
  if (params?.date_to) searchParams.set("date_to", params.date_to);
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.per_page) searchParams.set("per_page", String(params.per_page));
  const qs = searchParams.toString();
  const res = await api<{ data: EmailLog[]; meta: { total: number; page: number; per_page: number; total_pages: number } }>(
    `/admin/api/ext/email-manager/logs${qs ? `?${qs}` : ""}`
  );
  return { data: res.data, total: res.meta.total, page: res.meta.page, per_page: res.meta.per_page };
}

export async function getEmailLog(id: number): Promise<EmailLog> {
  const res = await api<ApiResponse<EmailLog>>(`/admin/api/ext/email-manager/logs/${id}`);
  return res.data;
}

export async function resendEmail(id: number): Promise<void> {
  await api<void>(`/admin/api/ext/email-manager/logs/${id}/resend`, { method: "POST" });
}

// --- Email Settings ---

export async function getEmailSettings(): Promise<Record<string, string>> {
  const res = await api<ApiResponse<Record<string, string>>>("/admin/api/ext/email-manager/settings");
  return res.data;
}

export async function saveEmailSettings(data: Record<string, string>): Promise<void> {
  await api<void>("/admin/api/ext/email-manager/settings", {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function sendTestEmail(): Promise<void> {
  await api<void>("/admin/api/ext/email-manager/settings/test", { method: "POST" });
}

// --- Themes ---

export interface Theme {
  id: number;
  slug: string;
  name: string;
  description: string;
  version: string;
  author: string;
  source: string;
  git_url: string | null;
  git_branch: string;
  has_git_token: boolean;
  is_active: boolean;
  path: string;
  thumbnail: string | null;
  created_at: string;
  updated_at: string;
}

export async function getThemes(): Promise<Theme[]> {
  const res = await api<ApiResponse<Theme[]>>("/admin/api/themes");
  return res.data;
}

export async function getTheme(id: number): Promise<Theme> {
  const res = await api<ApiResponse<Theme>>(`/admin/api/themes/${id}`);
  return res.data;
}

export async function uploadTheme(file: File): Promise<Theme> {
  const form = new FormData();
  form.append("file", file);
  const res = await fetch("/admin/api/themes/upload", {
    method: "POST",
    body: form,
    credentials: "include",
  });
  if (!res.ok) {
    const body = await res.json();
    throw new ApiClientError(
      body.error?.message || "Upload failed",
      body.error?.code || "upload_failed",
      res.status
    );
  }
  const body = await res.json();
  return (body as ApiResponse<Theme>).data;
}

export async function installThemeFromGit(data: {
  git_url: string;
  git_branch: string;
  git_token?: string;
}): Promise<Theme> {
  const res = await api<ApiResponse<Theme>>("/admin/api/themes/git", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function activateTheme(id: number): Promise<void> {
  await api<void>(`/admin/api/themes/${id}/activate`, { method: "POST" });
}

export async function deactivateTheme(id: number): Promise<void> {
  await api<void>(`/admin/api/themes/${id}/deactivate`, { method: "POST" });
}

export async function pullTheme(id: number): Promise<Theme> {
  const res = await api<ApiResponse<Theme>>(`/admin/api/themes/${id}/pull`, {
    method: "POST",
  });
  return res.data;
}

export async function deleteTheme(id: number): Promise<void> {
  await api<void>(`/admin/api/themes/${id}`, { method: "DELETE" });
}

export async function updateThemeGitConfig(
  id: number,
  data: { git_url?: string; git_branch?: string; git_token?: string }
): Promise<void> {
  await api<void>(`/admin/api/themes/${id}/git-config`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

// --- Extensions ---

export interface Extension {
  id: number;
  slug: string;
  name: string;
  version: string;
  description: string;
  author: string;
  path: string;
  is_active: boolean;
  priority: number;
  settings: Record<string, unknown>;
  installed_at: string;
  updated_at: string;
}

export async function getExtensions(): Promise<Extension[]> {
  const res = await api<ApiResponse<Extension[]>>("/admin/api/extensions");
  return res.data;
}

export async function activateExtension(slug: string): Promise<void> {
  await api<void>(`/admin/api/extensions/${slug}/activate`, { method: "POST" });
}

export async function deactivateExtension(slug: string): Promise<void> {
  await api<void>(`/admin/api/extensions/${slug}/deactivate`, { method: "POST" });
}

export async function uploadExtension(file: File): Promise<void> {
  const form = new FormData();
  form.append("file", file);
  const res = await fetch("/admin/api/extensions/upload", {
    method: "POST",
    body: form,
    credentials: "include",
  });
  if (!res.ok) {
    const body = await res.json();
    throw new ApiClientError(
      body.error?.message || "Upload failed",
      body.error?.code || "upload_failed",
      res.status
    );
  }
}

export async function deleteExtension(slug: string): Promise<void> {
  await api<void>(`/admin/api/extensions/${slug}`, { method: "DELETE" });
}

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

// Role Access helper
export function getNodeAccess(user: User | null, nodeType: string): { access: "none" | "read" | "write" | "all" } {
  if (!user) return { access: "none" };
  const caps = user.capabilities || {};

  // System admin or content manager has full access
  if (caps["*"] || caps["admin_access"] || caps["manage_content"]) return { access: "all" };

  // Check explicit node type access
  const typeAccess = caps[`nodes:${nodeType}`];
  if (typeAccess === "write" || typeAccess === "all") return { access: "all" };
  if (typeAccess === "read") return { access: "read" };

  // Check general node access
  const generalAccess = caps["nodes:*"];
  if (generalAccess === "write" || generalAccess === "all") return { access: "all" };
  if (generalAccess === "read") return { access: "read" };

  // Check default_node_access from role capabilities
  const defaultAccess = caps["default_node_access"] as Record<string, string> | undefined;
  if (defaultAccess?.access === "write" || defaultAccess?.access === "all") return { access: "all" };
  if (defaultAccess?.access === "read") return { access: "read" };

  return { access: "none" };
}
