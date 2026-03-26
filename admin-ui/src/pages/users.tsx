import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Pencil, Trash2, Users, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { toast } from "sonner";
import { getUsers, type User, type PaginationMeta } from "@/api/client";

// ---------- Local API helpers ----------

interface UserDetail extends User {
  role_id: number;
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

// ---------- Role badge colors ----------

const roleBadgeColors: Record<string, string> = {
  admin: "bg-red-100 text-red-700",
  editor: "bg-blue-100 text-blue-700",
  author: "bg-green-100 text-green-700",
  viewer: "bg-slate-100 text-slate-600",
};

function getRoleSlug(role: User["role"]): string {
  if (!role) return "unknown";
  if (typeof role === "string") return role.toLowerCase();
  return role.slug || "unknown";
}

function getRoleName(role: User["role"]): string {
  if (!role) return "Unknown";
  if (typeof role === "string") return role;
  return role.name || "Unknown";
}

function getRoleBadgeClass(role: User["role"]): string {
  const slug = getRoleSlug(role);
  return roleBadgeColors[slug] || "bg-indigo-100 text-indigo-700";
}

// ---------- Date formatting ----------

function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return "Never";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return "Never";
  return d.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
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

// ---------- Component ----------

export default function UsersPage() {
  const [users, setUsers] = useState<UserDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [_meta, setMeta] = useState<PaginationMeta | undefined>();

  // Delete confirmation
  const [showDelete, setShowDelete] = useState(false);
  const [deletingUser, setDeletingUser] = useState<UserDetail | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function fetchData() {
    try {
      const usersRes = await getUsers();
      setUsers(usersRes.data as UserDetail[]);
      setMeta(usersRes.meta);
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
      const message =
        err instanceof Error ? err.message : "Failed to delete user";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <Users className="h-7 w-7 text-indigo-600" />
            <h1 className="text-2xl font-bold text-slate-900">Users</h1>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            Manage user accounts and their roles.
          </p>
        </div>
        <Button
          asChild
          className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
        >
          <Link to="/admin/users/new">
            <Plus className="mr-2 h-4 w-4" />
            Add User
          </Link>
        </Button>
      </div>

      {/* Table */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <CardHeader>
          <CardTitle className="text-lg font-semibold text-slate-900">
            All Users
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="border-slate-200 hover:bg-transparent">
                <TableHead className="text-slate-500 font-medium">Full Name</TableHead>
                <TableHead className="text-slate-500 font-medium">Email</TableHead>
                <TableHead className="text-slate-500 font-medium">Role</TableHead>
                <TableHead className="text-slate-500 font-medium">Last Login</TableHead>
                <TableHead className="text-slate-500 font-medium">Created</TableHead>
                <TableHead className="text-slate-500 font-medium text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-12 text-slate-400">
                    No users found. Click &quot;Add User&quot; to create one.
                  </TableCell>
                </TableRow>
              )}
              {users.map((user) => (
                <TableRow key={user.id} className="border-slate-100">
                  <TableCell className="font-medium text-slate-800">
                    {user.full_name}
                  </TableCell>
                  <TableCell className="text-slate-600">{user.email}</TableCell>
                  <TableCell>
                    <span
                      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${getRoleBadgeClass(user.role)}`}
                    >
                      {getRoleName(user.role)}
                    </span>
                  </TableCell>
                  <TableCell className="text-slate-500 text-sm">
                    {formatDateTime(user.last_login_at)}
                  </TableCell>
                  <TableCell className="text-slate-500 text-sm">
                    {formatDate(user.created_at)}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        asChild
                        className="h-8 w-8 text-slate-500 hover:text-indigo-600"
                      >
                        <Link to={`/admin/users/${user.id}/edit`}>
                          <Pencil className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-slate-500 hover:text-red-600"
                        onClick={() => openDeleteDialog(user)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Delete confirmation dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete User</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingUser?.full_name}&quot;
              ({deletingUser?.email})? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDelete(false)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
