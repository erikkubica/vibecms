import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import {
  Plus,
  Pencil,
  Trash2,
  Loader2,
  Component,
  Filter,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import {
  getLayoutBlocksPaginated,
  deleteLayoutBlock,
  detachLayoutBlock,
  getLanguages,
  type LayoutBlock,
  type Language,
  type PaginationMeta,
} from "@/api/client";

export default function LayoutBlocksListPage() {
  const [layoutBlocks, setLayoutBlocks] = useState<LayoutBlock[]>([]);
  const [languages, setLanguages] = useState<Language[]>([]);
  const [loading, setLoading] = useState(true);
  const [languageFilter, setLanguageFilter] = useState<string>("all");
  const [deleteTarget, setDeleteTarget] = useState<LayoutBlock | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [page, setPage] = useState(1);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [detachingId, setDetachingId] = useState<number | null>(null);

  const fetchLayoutBlocks = useCallback(async () => {
    setLoading(true);
    try {
      const params: { language_id?: number; page: number; per_page: number } = { page, per_page: 25 };
      if (languageFilter && languageFilter !== "all") {
        params.language_id = Number(languageFilter);
      }
      const res = await getLayoutBlocksPaginated(params);
      setLayoutBlocks(res.data);
      setMeta(res.meta);
    } catch {
      toast.error("Failed to load layout blocks");
    } finally {
      setLoading(false);
    }
  }, [languageFilter, page]);

  const fetchLanguages = useCallback(async () => {
    try {
      const data = await getLanguages(true);
      setLanguages(data);
    } catch {
      // silent
    }
  }, []);

  useEffect(() => {
    fetchLanguages();
  }, [fetchLanguages]);

  useEffect(() => {
    setPage(1);
  }, [languageFilter]);

  useEffect(() => {
    fetchLayoutBlocks();
  }, [fetchLayoutBlocks]);

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteLayoutBlock(deleteTarget.id);
      toast.success("Layout block deleted successfully");
      setDeleteTarget(null);
      fetchLayoutBlocks();
    } catch {
      toast.error("Failed to delete layout block");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach(lb: LayoutBlock) {
    setDetachingId(lb.id);
    try {
      await detachLayoutBlock(lb.id);
      toast.success(`"${lb.name}" detached from ${lb.source}`);
      fetchLayoutBlocks();
    } catch {
      toast.error("Failed to detach layout block");
    } finally {
      setDetachingId(null);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900">Layout Blocks</h1>
        <Button asChild className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium">
          <Link to="/admin/layout-blocks/new">
            <Plus className="mr-2 h-4 w-4" />
            New Layout Block
          </Link>
        </Button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <Filter className="h-4 w-4 text-slate-400" />
        <Select value={languageFilter} onValueChange={setLanguageFilter}>
          <SelectTrigger className="w-48">
            <SelectValue placeholder="All Languages" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Languages</SelectItem>
            {languages.map((lang) => (
              <SelectItem key={lang.id} value={String(lang.id)}>
                {lang.flag} {lang.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Table */}
      <Card className="rounded-xl border border-slate-200 shadow-sm overflow-hidden py-0 gap-0">
        <CardContent className="p-0">
          {loading ? (
            <div className="flex h-64 items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
            </div>
          ) : layoutBlocks.length === 0 ? (
            <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
              <Component className="h-12 w-12" />
              <p className="text-lg font-medium">No layout blocks found</p>
              <p className="text-sm">Create your first layout block to get started</p>
              <Button asChild className="mt-2 bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium">
                <Link to="/admin/layout-blocks/new">
                  <Plus className="mr-2 h-4 w-4" />
                  New Layout Block
                </Link>
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-slate-50 hover:bg-slate-50">
                  <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Name</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Slug</TableHead>
                  <TableHead className="hidden text-xs font-semibold text-slate-500 uppercase tracking-wider sm:table-cell">Language</TableHead>
                  <TableHead className="hidden text-xs font-semibold text-slate-500 uppercase tracking-wider sm:table-cell">Source</TableHead>
                  <TableHead className="w-24 text-xs font-semibold text-slate-500 uppercase tracking-wider">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {layoutBlocks.map((lb) => (
                  <TableRow key={lb.id} className="hover:bg-slate-50">
                    <TableCell className="px-6 py-4 text-sm">
                      <Link
                        to={`/admin/layout-blocks/${lb.id}`}
                        className="font-medium text-slate-800 hover:text-indigo-600"
                      >
                        {lb.name}
                      </Link>
                    </TableCell>
                    <TableCell className="px-6 py-4 text-sm text-slate-500">
                      {lb.slug}
                    </TableCell>
                    <TableCell className="hidden px-6 py-4 text-sm text-slate-500 sm:table-cell">
                      {lb.language_id != null ? (languages.find(l => l.id === lb.language_id)?.name || String(lb.language_id)) : "All"}
                    </TableCell>
                    <TableCell className="hidden px-6 py-4 text-sm text-slate-500 sm:table-cell">
                      {lb.source === "theme" ? (
                        <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100 border-0 text-xs">Theme</Badge>
                      ) : lb.source === "extension" ? (
                        <Badge className="bg-purple-100 text-purple-700 hover:bg-purple-100 border-0 text-xs">Extension</Badge>
                      ) : (
                        <Badge className="bg-slate-100 text-slate-500 hover:bg-slate-100 border-0 text-xs">Custom</Badge>
                      )}
                    </TableCell>
                    <TableCell className="px-6 py-4 text-sm">
                      <div className="flex items-center gap-1">
                        {lb.source !== "custom" && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-amber-600 hover:text-amber-700"
                            onClick={() => handleDetach(lb)}
                            disabled={detachingId === lb.id}
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
                          <Link to={`/admin/layout-blocks/${lb.id}`}>
                            <Pencil className="h-4 w-4" />
                          </Link>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-red-500 hover:text-red-600"
                          disabled={lb.source !== "custom"}
                          onClick={() => setDeleteTarget(lb)}
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
            <DialogTitle>Delete Layout Block</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.name}&quot;?
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
