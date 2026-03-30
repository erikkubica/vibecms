export interface User {
  id: number;
  email: string;
  full_name: string;
  role_id: number;
  role: { id: number; slug: string; name: string; is_system: boolean } | string;
  capabilities?: Record<string, any>;
  last_login_at: string;
  created_at: string;
}

export interface NodeAccess {
  access: "none" | "read" | "write";
  scope: "all" | "own";
}

export function getNodeAccess(user: User | null, nodeType: string): NodeAccess {
  if (!user?.capabilities) return { access: "none", scope: "all" };
  const caps = user.capabilities;
  const nodes = caps.nodes as Record<string, NodeAccess> | undefined;
  if (nodes?.[nodeType]) return nodes[nodeType];
  if (caps.default_node_access) return caps.default_node_access as NodeAccess;
  return { access: "none", scope: "all" };
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
  blocks_data: Record<string, unknown>[];
  seo_settings: Record<string, unknown>;
  fields_data: Record<string, unknown>;
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

interface ApiResponse<T> {
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

async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const body = await res.json();

  if (!res.ok) {
    const err = body as ApiError;
    throw new ApiClientError(
      err.error?.message || "An unexpected error occurred",
      err.error?.code || "unknown",
      res.status
    );
  }

  return body as T;
}

export async function login(
  email: string,
  password: string
): Promise<{ user_id: number; email: string; role: string }> {
  const res = await api<ApiResponse<{ user_id: number; email: string; role: string }>>(
    "/auth/login",
    {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }
  );
  return res.data;
}

export async function logout(): Promise<void> {
  await api<void>("/auth/logout", { method: "POST" });
}

export async function getMe(): Promise<User> {
  const res = await api<ApiResponse<User>>("/me");
  return res.data;
}

export async function getNodes(params: {
  page?: number;
  per_page?: number;
  status?: string;
  node_type?: string;
  language_code?: string;
  search?: string;
}): Promise<{ data: ContentNode[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set("page", String(params.page));
  if (params.per_page) searchParams.set("per_page", String(params.per_page));
  if (params.status) searchParams.set("status", params.status);
  if (params.node_type) searchParams.set("node_type", params.node_type);
  if (params.language_code) searchParams.set("language_code", params.language_code);
  if (params.search) searchParams.set("search", params.search);

  const res = await api<{ data: ContentNode[]; meta: PaginationMeta }>(
    `/admin/api/nodes?${searchParams.toString()}`
  );
  return res;
}

export async function getNode(id: number | string): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>(`/admin/api/nodes/${id}`);
  return res.data;
}

export async function createNode(
  data: Partial<ContentNode>
): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>("/admin/api/nodes", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function updateNode(
  id: number | string,
  data: Partial<ContentNode>
): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>(`/admin/api/nodes/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

export async function deleteNode(id: number | string): Promise<void> {
  await api<void>(`/admin/api/nodes/${id}`, { method: "DELETE" });
}

export async function getUsers(params?: {
  page?: number;
  per_page?: number;
}): Promise<{ data: User[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.per_page) searchParams.set("per_page", String(params.per_page));

  const res = await api<{ data: User[]; meta: PaginationMeta }>(
    `/admin/api/users?${searchParams.toString()}`
  );
  return res;
}

export async function getNodeTranslations(id: number | string): Promise<ContentNode[]> {
  const res = await api<ApiResponse<ContentNode[]>>(`/admin/api/nodes/${id}/translations`);
  return res.data;
}

export async function createNodeTranslation(id: number | string, languageCode: string): Promise<ContentNode> {
  const res = await api<ApiResponse<ContentNode>>(`/admin/api/nodes/${id}/translations`, {
    method: "POST",
    body: JSON.stringify({ language_code: languageCode }),
  });
  return res.data;
}

export async function getHomepageId(): Promise<number> {
  const res = await api<ApiResponse<{ homepage_node_id: number }>>("/admin/api/nodes/homepage");
  return res.data.homepage_node_id;
}

export async function setHomepage(nodeId: number | string): Promise<void> {
  await api<void>(`/admin/api/nodes/${nodeId}/homepage`, { method: "POST" });
}

export interface NodeTypeField {
  key: string;
  label: string;
  type: "text" | "textarea" | "number" | "date" | "select" | "image" | "toggle" | "link" | "group" | "repeater" | "node" | "color" | "email" | "url" | "richtext" | "range" | "file" | "gallery" | "radio" | "checkbox";
  required?: boolean;
  options?: string[];            // for select, radio, checkbox types
  placeholder?: string;          // for text, textarea, number, email, url
  default_value?: string;        // default value for any field
  help_text?: string;            // instructions/description shown below field
  sub_fields?: NodeTypeField[];  // for group and repeater
  node_type_filter?: string;     // for node selector - filter by content type
  multiple?: boolean;            // for node selector, file - multi-select
  min?: number;                  // for number, range
  max?: number;                  // for number, range
  step?: number;                 // for number, range
  min_length?: number;           // for text, textarea
  max_length?: number;           // for text, textarea
  rows?: number;                 // for textarea
  prepend?: string;              // text before input (text, number, email, url)
  append?: string;               // text after input (text, number, email, url)
  allowed_types?: string;        // for file - comma-separated mime types or extensions
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

export interface NodeType {
  id: number;
  slug: string;
  label: string;
  icon: string;
  description: string;
  field_schema: NodeTypeField[];
  url_prefixes: Record<string, string>;
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
  block_config: TemplateBlockConfig[];
  created_at: string;
  updated_at: string;
}

export async function getBlockTypes(): Promise<BlockType[]> {
  const res = await api<ApiResponse<BlockType[]>>("/admin/api/block-types");
  return res.data;
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

export async function getSiteSettings(): Promise<Record<string, string>> {
  const res = await api<ApiResponse<Record<string, string>>>("/admin/api/settings");
  return res.data;
}

export async function updateSiteSettings(settings: Record<string, string>): Promise<void> {
  await api<ApiResponse<{ message: string }>>("/admin/api/settings", {
    method: "PUT",
    body: JSON.stringify(settings),
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

export async function previewBlockTemplate(htmlTemplate: string, testData: Record<string, unknown>): Promise<string> {
  const res = await api<{ html: string }>(`/admin/api/block-types/preview`, {
    method: "POST",
    body: JSON.stringify({ html_template: htmlTemplate, test_data: testData }),
  });
  return res.html;
}

export async function getTemplates(): Promise<Template[]> {
  const res = await api<ApiResponse<Template[]>>("/admin/api/templates");
  return res.data;
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

// --- Layout Blocks ---

export interface LayoutBlock {
  id: number;
  slug: string;
  name: string;
  description: string;
  language_id: number | null;
  template_code: string;
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
