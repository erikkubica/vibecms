import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Shield, Lock } from "lucide-react";
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
import { getRoles, deleteRole, type Role } from "@/api/client";
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
  Chip,
  TitleCell,
  RowActions,
  EmptyState,
  LoadingRow,
} from "@/components/ui/list-page";

interface RoleWithPerms extends Role {
  permissions?: unknown[] | Record<string, unknown>;
}

export default function RolesPage() {
  const navigate = useNavigate();
  const [roles, setRoles] = useState<RoleWithPerms[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  const [showDelete, setShowDelete] = useState(false);
  const [deletingRole, setDeletingRole] = useState<Role | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function fetchData() {
    try {
      const rolesData = await getRoles();
      setRoles(rolesData as RoleWithPerms[]);
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
      const message = err instanceof Error ? err.message : "Failed to delete role";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  function permissionCount(role: RoleWithPerms): number {
    const p = role.permissions;
    if (!p) return 0;
    if (Array.isArray(p)) return p.length;
    if (typeof p === "object") return Object.keys(p).length;
    return 0;
  }

  const q = search.toLowerCase();
  const filteredRoles = q
    ? roles.filter(
        (r) =>
          r.name.toLowerCase().includes(q) ||
          r.slug.toLowerCase().includes(q) ||
          (r.description || "").toLowerCase().includes(q),
      )
    : roles;

  return (
    <ListPageShell>
      <ListHeader
        title="Roles"
        tabs={[{ value: "all", label: "All", count: roles.length }]}
        activeTab="all"
        newLabel="Add Role"
        onNew={() => navigate("/admin/roles/new")}
      />

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder="Search roles…" />
      </ListToolbar>

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : roles.length === 0 ? (
          <EmptyState
            icon={Shield}
            title="No roles configured yet"
            description='Click "Add Role" to get started.'
          />
        ) : (
          <ListTable>
            <thead>
              <tr>
                <Th>Name</Th>
                <Th>Description</Th>
                <Th width={130}>Permissions</Th>
                <Th width={110}>Type</Th>
                <Th width={110} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {filteredRoles.map((role) => (
                <Tr key={role.id}>
                  <Td>
                    <TitleCell
                      to={`/admin/roles/${role.id}/edit`}
                      title={role.name}
                      slug={role.slug}
                    />
                  </Td>
                  <Td className="text-slate-600">
                    <span className="block max-w-md truncate" title={role.description || ""}>
                      {role.description || <span className="text-slate-400">—</span>}
                    </span>
                  </Td>
                  <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                    {permissionCount(role)}
                  </Td>
                  <Td>
                    {role.is_system ? (
                      <span className="inline-flex items-center gap-1 px-1.5 py-px text-[11px] font-medium text-slate-700 bg-slate-50 border border-slate-200 rounded-[2px]">
                        <Lock className="w-3 h-3" />
                        System
                      </span>
                    ) : (
                      <Chip>Custom</Chip>
                    )}
                  </Td>
                  <Td align="right" className="whitespace-nowrap">
                    <RowActions
                      onEdit={() => navigate(`/admin/roles/${role.id}/edit`)}
                      onDelete={role.is_system ? undefined : () => openDeleteDialog(role)}
                    />
                  </Td>
                </Tr>
              ))}
            </tbody>
          </ListTable>
        )}
      </ListCard>

      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Role</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingRole?.name}&quot;? This action cannot be undone. Users with this role will need to be reassigned.
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
