import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Shield, Plus, Pencil, Trash2, Lock, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import {
  getRoles,
  deleteRole,
  type Role,
} from "@/api/client";

export default function RolesPage() {
  const navigate = useNavigate();
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);

  // Delete confirmation
  const [showDelete, setShowDelete] = useState(false);
  const [deletingRole, setDeletingRole] = useState<Role | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function fetchData() {
    try {
      const rolesData = await getRoles();
      setRoles(rolesData);
    } catch {
      toast.error("Failed to load roles");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchData();
  }, []);

  function openDeleteDialog(role: Role) {
    setDeletingRole(role);
    setShowDelete(true);
  }

  async function handleDelete() {
    if (!deletingRole) return;
    setDeleting(true);
    try {
      await deleteRole(deletingRole.id);
      toast.success("Role deleted successfully");
      setShowDelete(false);
      setDeletingRole(null);
      await fetchData();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to delete role";
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
        <div className="flex items-center gap-3">
          <Shield className="h-7 w-7 text-indigo-600" />
          <h1 className="text-2xl font-bold text-slate-900">Roles</h1>
        </div>
        <Button
          className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
          onClick={() => navigate("/admin/roles/new")}
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Role
        </Button>
      </div>

      {/* Table */}
      <Card className="rounded-xl border border-slate-200 shadow-sm">
        <CardHeader>
          <CardTitle className="text-lg font-semibold text-slate-900">
            All Roles
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="border-slate-200 hover:bg-transparent">
                <TableHead className="text-slate-500 font-medium">Name</TableHead>
                <TableHead className="text-slate-500 font-medium">Slug</TableHead>
                <TableHead className="text-slate-500 font-medium">Description</TableHead>
                <TableHead className="text-slate-500 font-medium">System</TableHead>
                <TableHead className="text-slate-500 font-medium text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {roles.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-12 text-slate-400">
                    No roles configured yet. Click &quot;Add Role&quot; to get started.
                  </TableCell>
                </TableRow>
              )}
              {roles.map((role) => (
                <TableRow key={role.id} className="border-slate-100">
                  <TableCell className="font-medium text-slate-800">{role.name}</TableCell>
                  <TableCell>
                    <span className="font-mono text-sm text-slate-600">{role.slug}</span>
                  </TableCell>
                  <TableCell className="text-slate-600 max-w-xs truncate">
                    {role.description || <span className="text-slate-300">--</span>}
                  </TableCell>
                  <TableCell>
                    {role.is_system && (
                      <Badge className="bg-slate-100 text-slate-600 hover:bg-slate-100 border-0 text-xs gap-1">
                        <Lock className="h-3 w-3" />
                        System
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-slate-500 hover:text-indigo-600"
                        onClick={() => navigate(`/admin/roles/${role.id}/edit`)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      {!role.is_system && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-slate-500 hover:text-red-600"
                          onClick={() => openDeleteDialog(role)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      )}
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
            <DialogTitle>Delete Role</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingRole?.name}&quot;? This action cannot be
              undone. Users with this role will need to be reassigned.
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
