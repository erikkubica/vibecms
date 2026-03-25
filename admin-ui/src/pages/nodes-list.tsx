import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import {
  Plus,
  Search,
  Loader2,
  Pencil,
  Trash2,
  FileText,
  Home,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "sonner";
import {
  getNodes,
  deleteNode,
  type ContentNode,
  type PaginationMeta,
} from "@/api/client";

interface NodesListProps {
  nodeType: "page" | "post";
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case "published":
      return "bg-emerald-100 text-emerald-700 hover:bg-emerald-100";
    case "draft":
      return "bg-amber-100 text-amber-700 hover:bg-amber-100";
    case "archived":
      return "bg-slate-100 text-slate-600 hover:bg-slate-100";
    default:
      return "bg-slate-100 text-slate-600 hover:bg-slate-100";
  }
}

export default function NodesListPage({ nodeType }: NodesListProps) {
  const label = nodeType === "page" ? "Page" : "Post";
  const labelPlural = nodeType === "page" ? "Pages" : "Posts";
  const basePath = nodeType === "page" ? "/admin/pages" : "/admin/posts";

  const [nodes, setNodes] = useState<ContentNode[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [deleteTarget, setDeleteTarget] = useState<ContentNode | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [searchDebounce, setSearchDebounce] = useState("");

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => setSearchDebounce(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const fetchNodes = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getNodes({
        page,
        per_page: 20,
        node_type: nodeType,
        status: status === "all" ? undefined : status,
        search: searchDebounce || undefined,
      });
      setNodes(res.data);
      setMeta(res.meta);
    } catch {
      toast.error(`Failed to load ${labelPlural.toLowerCase()}`);
    } finally {
      setLoading(false);
    }
  }, [page, nodeType, status, searchDebounce, labelPlural]);

  useEffect(() => {
    setPage(1);
  }, [searchDebounce, status, nodeType]);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteNode(deleteTarget.id);
      toast.success(`${label} deleted successfully`);
      setDeleteTarget(null);
      fetchNodes();
    } catch {
      toast.error(`Failed to delete ${label.toLowerCase()}`);
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900">{labelPlural}</h1>
        <Button asChild className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium">
          <Link to={`${basePath}/new`}>
            <Plus className="mr-2 h-4 w-4" />
            New {label}
          </Link>
        </Button>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm sm:flex-row">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <Input
            placeholder={`Search ${labelPlural.toLowerCase()}...`}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
          />
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-full rounded-lg border-slate-300 sm:w-40">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="published">Published</SelectItem>
            <SelectItem value="draft">Draft</SelectItem>
            <SelectItem value="archived">Archived</SelectItem>
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
          ) : nodes.length === 0 ? (
            <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
              <FileText className="h-12 w-12" />
              <p className="text-lg font-medium">
                No {labelPlural.toLowerCase()} found
              </p>
              <p className="text-sm">
                {searchDebounce || status !== "all"
                  ? "Try adjusting your filters"
                  : `Create your first ${label.toLowerCase()} to get started`}
              </p>
              {!searchDebounce && status === "all" && (
                <Button asChild className="mt-2 bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium">
                  <Link to={`${basePath}/new`}>
                    <Plus className="mr-2 h-4 w-4" />
                    New {label}
                  </Link>
                </Button>
              )}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-slate-50 hover:bg-slate-50">
                  <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Title</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Status</TableHead>
                  <TableHead className="hidden text-xs font-semibold text-slate-500 uppercase tracking-wider md:table-cell">Slug</TableHead>
                  <TableHead className="hidden text-xs font-semibold text-slate-500 uppercase tracking-wider sm:table-cell">
                    Updated
                  </TableHead>
                  <TableHead className="w-24 text-xs font-semibold text-slate-500 uppercase tracking-wider">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {nodes.map((node) => (
                  <TableRow key={node.id} className="hover:bg-slate-50">
                    <TableCell className="px-6 py-4 text-sm">
                      <div className="flex items-center gap-2">
                        <Link
                          to={`${basePath}/${node.id}/edit`}
                          className="font-medium text-slate-800 hover:text-indigo-600"
                        >
                          {node.title}
                        </Link>
                        {node.is_homepage && (
                          <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100 border-0 gap-1">
                            <Home className="h-3 w-3" />
                            Home
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="px-6 py-4 text-sm">
                      <Badge className={`${statusBadgeClass(node.status)} border-0 font-medium`}>
                        {node.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="hidden px-6 py-4 text-sm text-slate-500 md:table-cell">
                      /{node.slug}
                    </TableCell>
                    <TableCell className="hidden px-6 py-4 text-sm text-slate-500 sm:table-cell">
                      {new Date(node.updated_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="px-6 py-4 text-sm">
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          asChild
                          className="h-8 w-8"
                        >
                          <Link to={`${basePath}/${node.id}/edit`}>
                            <Pencil className="h-4 w-4" />
                          </Link>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-red-500 hover:text-red-600"
                          onClick={() => setDeleteTarget(node)}
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

      {/* Pagination */}
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
            <DialogTitle>Delete {label}</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.title}&quot;?
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
