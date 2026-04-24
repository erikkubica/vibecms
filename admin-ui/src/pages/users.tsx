import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Users, Plus } from "lucide-react";
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
import { getUsers, type User } from "@/api/client";
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

interface UserDetail extends User {
  role_id: number;
  is_active?: boolean;
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
    throw new Error(body?.error?.message || "An unexpected error occurred");
  }
  return body as T;
}

async function deleteUser(id: number): Promise<void> {
  await apiFetch<void>(`/admin/api/users/${id}`, { method: "DELETE" });
}

function getRoleName(role: User["role"]): string {
  if (!role) return "Unknown";
  if (typeof role === "string") return role;
  return role.name || "Unknown";
}

function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return "Never";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return "Never";
  return d.toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" });
}

function formatDateTime(dateStr: string | null | undefined): string {
  if (!dateStr) return "Never";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return "Never";
  return d.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function UsersPage() {
  const [users, setUsers] = useState<UserDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [filterRole, setFilterRole] = useState("");
  const [filterStatus, setFilterStatus] = useState("");

  const [showDelete, setShowDelete] = useState(false);
  const [deletingUser, setDeletingUser] = useState<UserDetail | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function fetchData() {
    try {
      const usersRes = await getUsers();
      setUsers(usersRes.data as UserDetail[]);
    } catch {
      toast.error("Failed to load users");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchData();
  }, []);

  function openDeleteDialog(user: UserDetail) {
    setDeletingUser(user);
    setShowDelete(true);
  }

  async function handleDelete() {
    if (!deletingUser) return;
    setDeleting(true);
    try {
      await deleteUser(deletingUser.id);
      toast.success("User deleted successfully");
      setShowDelete(false);
      setDeletingUser(null);
      await fetchData();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete user";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  const uniqueRoles = Array.from(new Set(users.map((u) => getRoleName(u.role)))).sort();

  const q = search.toLowerCase();
  const filteredUsers = users.filter((u) => {
    if (q && !u.full_name.toLowerCase().includes(q) && !u.email.toLowerCase().includes(q)) return false;
    if (filterRole && getRoleName(u.role) !== filterRole) return false;
    if (filterStatus === "active" && u.is_active === false) return false;
    if (filterStatus === "inactive" && u.is_active !== false) return false;
    return true;
  });

  const selectCls = "h-[30px] pl-2 pr-7 bg-white border border-slate-300 rounded text-[13px] text-slate-700 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/30 outline-none appearance-none cursor-pointer";

  return (
    <ListPageShell>
      <ListHeader
        title="Users"
        tabs={[{ value: "all", label: "All", count: users.length }]}
        activeTab="all"
        newLabel="Add User"
        newHref="/admin/users/new"
      />

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search users…" />
        <select value={filterRole} onChange={(e) => setFilterRole(e.target.value)} className={selectCls}>
          <option value="">All roles</option>
          {uniqueRoles.map((r) => <option key={r} value={r}>{r}</option>)}
        </select>
        <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} className={selectCls}>
          <option value="">All statuses</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </ListToolbar>

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : users.length === 0 ? (
          <EmptyState
            icon={Users}
            title="No users found"
            description='Click "Add User" to create one.'
            action={
              <Link
                to="/admin/users/new"
                className="h-[30px] px-3 inline-flex items-center gap-1.5 text-[13px] font-medium text-white bg-indigo-600 rounded hover:bg-indigo-700"
              >
                <Plus className="w-3.5 h-3.5" />
                Add User
              </Link>
            }
          />
        ) : (
          <ListTable>
            <thead>
              <tr>
                <Th>Full Name</Th>
                <Th>Email</Th>
                <Th width={140}>Role</Th>
                <Th width={110}>Status</Th>
                <Th width={170}>Last Login</Th>
                <Th width={130}>Created</Th>
                <Th width={110} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((user) => {
                const isActive = user.is_active !== false;
                return (
                  <Tr key={user.id}>
                    <Td>
                      <TitleCell to={`/admin/users/${user.id}/edit`} title={user.full_name} />
                    </Td>
                    <Td className="text-slate-600">{user.email}</Td>
                    <Td>
                      <Chip>{getRoleName(user.role)}</Chip>
                    </Td>
                    <Td>
                      <StatusPill status={isActive ? "active" : "inactive"} />
                    </Td>
                    <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                      {formatDateTime(user.last_login_at)}
                    </Td>
                    <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                      {formatDate(user.created_at)}
                    </Td>
                    <Td align="right" className="whitespace-nowrap">
                      <RowActions
                        editTo={`/admin/users/${user.id}/edit`}
                        onDelete={() => openDeleteDialog(user)}
                      />
                    </Td>
                  </Tr>
                );
              })}
            </tbody>
          </ListTable>
        )}
      </ListCard>

      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete User</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingUser?.full_name}&quot; ({deletingUser?.email})? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDelete(false)} disabled={deleting}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ListPageShell>
  );
}
