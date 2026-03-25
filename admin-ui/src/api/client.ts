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
  search?: string;
}): Promise<{ data: ContentNode[]; meta: PaginationMeta }> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set("page", String(params.page));
  if (params.per_page) searchParams.set("per_page", String(params.per_page));
  if (params.status) searchParams.set("status", params.status);
  if (params.node_type) searchParams.set("node_type", params.node_type);
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
