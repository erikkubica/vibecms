import { useEffect, useState } from "react";
import { Key, Loader2, Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import {
  ListPageShell,
  ListHeader,
  ListToolbar,
  ListSearch,
  ListCard,
  ListTable,
  Th,
  Tr,
  Td,
  StatusPill,
  Chip,
  TitleCell,
  RowActions,
  EmptyState,
  LoadingRow,
} from "@/components/ui/list-page";

interface McpToken {
  id: number;
  name: string;
  token_prefix: string;
  scope: "full" | "content" | "read";
  last_used_at?: string | null;
  expires_at?: string | null;
  created_at: string;
  revoked_at?: string | null;
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
  if (res.status === 204) return undefined as T;
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body?.error?.message || body?.error || "Unexpected error");
  }
  return body?.data as T;
}

function formatDateTime(s?: string | null): string {
  if (!s) return "—";
  const d = new Date(s);
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleString();
}

const scopeLabels: Record<string, string> = {
  full: "Full access",
  content: "Content only",
  read: "Read-only",
};

function tokenStatus(t: McpToken): "active" | "danger" | "warning" {
  if (t.revoked_at) return "danger";
  if (t.expires_at) {
    const exp = new Date(t.expires_at).getTime();
    if (!isNaN(exp) && exp < Date.now()) return "warning";
  }
  return "active";
}

function tokenStatusLabel(t: McpToken): string {
  const s = tokenStatus(t);
  if (s === "danger") return "revoked";
  if (s === "warning") return "expired";
  return "active";
}

export default function McpTokensPage() {
  const [tokens, setTokens] = useState<McpToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<McpToken | null>(null);
  const [revealed, setRevealed] = useState<{ token: string; record: McpToken } | null>(null);
  const [copied, setCopied] = useState(false);

  const [name, setName] = useState("");
  const [scope, setScope] = useState<"full" | "content" | "read">("full");
  const [expiresAt, setExpiresAt] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function load() {
    try {
      const data = await apiFetch<McpToken[]>("/admin/api/mcp-tokens");
      setTokens(data || []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to load tokens");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setSubmitting(true);
    try {
      const body: Record<string, unknown> = { name: name.trim(), scope };
      if (expiresAt) body.expires_at = new Date(expiresAt).toISOString();
      const result = await apiFetch<{ token: string; record: McpToken }>(
        "/admin/api/mcp-tokens",
        { method: "POST", body: JSON.stringify(body) }
      );
      setRevealed(result);
      setCreateOpen(false);
      setName("");
      setScope("full");
      setExpiresAt("");
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create token");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete() {
    if (!confirmDelete) return;
    try {
      await apiFetch(`/admin/api/mcp-tokens/${confirmDelete.id}`, { method: "DELETE" });
      toast.success(`Token "${confirmDelete.name}" revoked`);
      setConfirmDelete(null);
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to revoke");
    }
  }

  async function copyToken() {
    if (!revealed) return;
    try {
      await navigator.clipboard.writeText(revealed.token);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error("Copy failed — select and copy manually");
    }
  }

  const q = search.toLowerCase();
  const filteredTokens = q
    ? tokens.filter((t) => t.name.toLowerCase().includes(q) || t.token_prefix.toLowerCase().includes(q))
    : tokens;

  return (
    <ListPageShell>
      <ListHeader
        title="MCP Tokens"
        tabs={[{ value: "all", label: "All", count: tokens.length }]}
        activeTab="all"
        newLabel="New token"
        onNew={() => setCreateOpen(true)}
      />

      <p className="mb-3 text-[12px] text-slate-500 max-w-3xl">
        Bearer tokens that let AI clients control this VibeCMS instance via the Model Context
        Protocol. Each token is shown once at creation — store it somewhere safe.
      </p>

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search tokens…" />
      </ListToolbar>

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : tokens.length === 0 ? (
          <EmptyState
            icon={Key}
            title="No tokens yet"
            description="Create one to connect Claude Code, Cursor, or any other MCP client."
          />
        ) : (
          <ListTable>
            <thead>
              <tr>
                <Th>Name</Th>
                <Th width={140}>Scope</Th>
                <Th width={100}>Status</Th>
                <Th width={170}>Last used</Th>
                <Th width={170}>Expires</Th>
                <Th width={170}>Created</Th>
                <Th width={80} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {filteredTokens.map((t) => (
                <Tr key={t.id}>
                  <Td>
                    <TitleCell title={t.name} slug={`${t.token_prefix}…`} />
                  </Td>
                  <Td>
                    <Chip>{scopeLabels[t.scope] || t.scope}</Chip>
                  </Td>
                  <Td>
                    <StatusPill status={tokenStatus(t)} label={tokenStatusLabel(t)} />
                  </Td>
                  <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                    {formatDateTime(t.last_used_at)}
                  </Td>
                  <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                    {formatDateTime(t.expires_at)}
                  </Td>
                  <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                    {formatDateTime(t.created_at)}
                  </Td>
                  <Td align="right" className="whitespace-nowrap">
                    <RowActions
                      onDelete={() => setConfirmDelete(t)}
                      deleteTitle="Revoke token"
                    />
                  </Td>
                </Tr>
              ))}
            </tbody>
          </ListTable>
        )}
      </ListCard>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New MCP token</DialogTitle>
            <DialogDescription>
              The raw token is shown once. Copy it into your MCP client config immediately.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div>
              <label className="text-sm font-medium text-slate-700">Name</label>
              <input
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
                placeholder="Claude Code (laptop)"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div>
              <label className="text-sm font-medium text-slate-700">Scope</label>
              <select
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
                value={scope}
                onChange={(e) => setScope(e.target.value as "full" | "content" | "read")}
              >
                <option value="full">Full access — every tool (incl. settings, themes, raw SQL if enabled)</option>
                <option value="content">Content only — nodes, taxonomies, menus, media, files</option>
                <option value="read">Read-only — no mutations</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium text-slate-700">
                Expires at <span className="text-slate-400 font-normal">(optional)</span>
              </label>
              <input
                type="datetime-local"
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Create token
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!revealed} onOpenChange={(o) => !o && setRevealed(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Token "{revealed?.record.name}"</DialogTitle>
            <DialogDescription>Store this securely. It won't be shown again.</DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-sm">
            <span className="flex-1 break-all">{revealed?.token}</span>
            <Button size="icon" variant="ghost" onClick={copyToken} aria-label="Copy">
              {copied ? (
                <Check className="h-4 w-4 text-emerald-600" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
          <div className="mt-3 rounded-md bg-indigo-50 p-3 text-xs text-indigo-900">
            <p className="font-medium mb-1">Claude Code setup</p>
            <pre className="whitespace-pre-wrap break-all">
              {`claude mcp add vibecms ${window.location.origin}/mcp \\
  --header "Authorization: Bearer ${revealed?.token || "<token>"}"`}
            </pre>
          </div>
          <DialogFooter>
            <Button onClick={() => setRevealed(null)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!confirmDelete} onOpenChange={(o) => !o && setConfirmDelete(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke "{confirmDelete?.name}"?</DialogTitle>
            <DialogDescription>
              Any MCP client using this token will stop working immediately.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setConfirmDelete(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              Revoke
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ListPageShell>
  );
}
