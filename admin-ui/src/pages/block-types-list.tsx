import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import {
  Plus,
  Pencil,
  Trash2,
  Loader2,
  Square,
  Unplug,
} from "lucide-react";
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
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import {
  getBlockTypesPaginated,
  deleteBlockType,
  detachBlockType,
  type BlockType,
  type PaginationMeta,
} from "@/api/client";

export default function BlockTypesListPage() {
  const [blockTypes, setBlockTypes] = useState<BlockType[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleteTarget, setDeleteTarget] = useState<BlockType | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [page, setPage] = useState(1);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [detachingId, setDetachingId] = useState<number | null>(null);

  const fetchBlockTypes = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getBlockTypesPaginated({ page, per_page: 25 });
      setBlockTypes(res.data);
      setMeta(res.meta);
    } catch {
      toast.error("Failed to load block types");
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    fetchBlockTypes();
  }, [fetchBlockTypes]);

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteBlockType(deleteTarget.id);
      toast.success("Block type deleted successfully");
      setDeleteTarget(null);
      fetchBlockTypes();
    } catch {
      toast.error("Failed to delete block type");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach(bt: BlockType) {
    setDetachingId(bt.id);
    try {
      await detachBlockType(bt.id);
      toast.success(`"${bt.label}" detached from ${bt.source}`);
      fetchBlockTypes();
    } catch {
      toast.error("Failed to detach block type");
    } finally {
      setDetachingId(null);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900">Block Types</h1>
        <Button asChild className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium">
          <Link to="/admin/block-types/new">
            <Plus className="mr-2 h-4 w-4" />
            New Block Type
          </Link>
        </Button>
      </div>

      {/* Table */}
      <Card className="rounded-xl border border-slate-200 shadow-sm overflow-hidden py-0 gap-0">
        <CardContent className="p-0">
          {loading ? (
            <div className="flex h-64 items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
            </div>
          ) : blockTypes.length === 0 ? (
            <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
              <Square className="h-12 w-12" />
              <p className="text-lg font-medium">No block types found</p>
              <p className="text-sm">Create your first block type to get started</p>
              <Button asChild className="mt-2 bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium">
                <Link to="/admin/block-types/new">
                  <Plus className="mr-2 h-4 w-4" />
                  New Block Type
                </Link>
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-slate-50 hover:bg-slate-50">
                  <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Label</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Slug</TableHead>
                  <TableHead className="hidden text-xs font-semibold text-slate-500 uppercase tracking-wider md:table-cell">Fields</TableHead>
                  <TableHead className="hidden text-xs font-semibold text-slate-500 uppercase tracking-wider sm:table-cell">Source</TableHead>
                  <TableHead className="hidden text-xs font-semibold text-slate-500 uppercase tracking-wider sm:table-cell">Description</TableHead>
                  <TableHead className="w-24 text-xs font-semibold text-slate-500 uppercase tracking-wider">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {blockTypes.map((bt) => (
                  <TableRow key={bt.id} className="hover:bg-slate-50">
                    <TableCell className="px-6 py-4 text-sm">
                      <Link
                        to={`/admin/block-types/${bt.id}/edit`}
                        className="font-medium text-slate-800 hover:text-indigo-600"
                      >
                        {bt.label}
                      </Link>
                    </TableCell>
                    <TableCell className="px-6 py-4 text-sm text-slate-500">
                      {bt.slug}
                    </TableCell>
                    <TableCell className="hidden px-6 py-4 text-sm text-slate-500 md:table-cell">
                      {bt.field_schema?.length ?? 0}
                    </TableCell>
                    <TableCell className="hidden px-6 py-4 text-sm text-slate-500 sm:table-cell">
                      {bt.source === "theme" ? (
                        <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-0 text-xs">Theme</Badge>
                      ) : bt.source === "extension" ? (
                        <Badge className="bg-purple-100 text-purple-700 hover:bg-purple-100 border-0 text-xs">Extension</Badge>
                      ) : (
                        <Badge className="bg-slate-100 text-slate-500 hover:bg-slate-100 border-0 text-xs">Custom</Badge>
                      )}
                    </TableCell>
                    <TableCell className="hidden px-6 py-4 text-sm text-slate-500 sm:table-cell">
                      <span className="block max-w-xs truncate" title={bt.description || ""}>{bt.description || "—"}</span>
                    </TableCell>
                    <TableCell className="px-6 py-4 text-sm">
                      <div className="flex items-center gap-1">
                        {bt.source !== "custom" && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-amber-600 hover:text-amber-700"
                            onClick={() => handleDetach(bt)}
                            disabled={detachingId === bt.id}
                            title="Detach from source"
                          >
                            <Unplug className="h-4 w-4" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="icon"
                          asChild
                          className="h-8 w-8"
                        >
                          <Link to={`/admin/block-types/${bt.id}/edit`}>
                            <Pencil className="h-4 w-4" />
                          </Link>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-red-500 hover:text-red-600"
                          disabled={bt.source !== "custom"}
                          onClick={() => setDeleteTarget(bt)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {meta && meta.total_pages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-slate-500">
            Showing {(meta.page - 1) * meta.per_page + 1} to{" "}
            {Math.min(meta.page * meta.per_page, meta.total)} of {meta.total}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
              className="rounded-lg border-slate-300"
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= meta.total_pages}
              onClick={() => setPage((p) => p + 1)}
              className="rounded-lg border-slate-300"
            >
              Next
            </Button>
          </div>
        </div>
      )}

      {/* Delete dialog */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Block Type</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.label}&quot;?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteTarget(null)}
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
