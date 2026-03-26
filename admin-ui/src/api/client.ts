export interface User {
  id: number;
  email: string;
  full_name: string;
  role: string;
  last_login_at: string;
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
