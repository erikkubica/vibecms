import { useEffect, useState, useCallback } from "react";
import { LayoutTemplate, Check, Unplug } from "lucide-react";
import { Button } from "@/components/ui/button";
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
import { toast } from "sonner";
import {
  getLayoutsPaginated,
  deleteLayout,
  detachLayout,
  getLanguages,
  type Layout,
  type Language,
} from "@/api/client";
import {
  ListPageShell,
  ListHeader,
  ListToolbar,
  ListCard,
  ListTable,
  Th,
  Tr,
  Td,
  TitleCell,
  RowActions,
  Chip,
  EmptyState,
  LoadingRow,
} from "@/components/ui/list-page";

type SourceFilter = "all" | "custom" | "theme" | "extension";

export default function LayoutsListPage() {
  const [layouts, setLayouts] = useState<Layout[]>([]);
  const [languages, setLanguages] = useState<Language[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleteTarget, setDeleteTarget] = useState<Layout | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [langFilter, setLangFilter] = useState("all");
  const [source, setSource] = useState<SourceFilter>("all");
  const [detachingId, setDetachingId] = useState<number | null>(null);

  const fetchLayouts = useCallback(async () => {
    setLoading(true);
    try {
      const params: { language_id?: number; page: number; per_page: number } = { page: 1, per_page: 500 };
      if (langFilter && langFilter !== "all") params.language_id = Number(langFilter);
      const res = await getLayoutsPaginated(params);
      setLayouts(res.data);
    } catch {
      toast.error("Failed to load layouts");
    } finally {
      setLoading(false);
    }
  }, [langFilter]);

  useEffect(() => {
    fetchLayouts();
  }, [fetchLayouts]);

  useEffect(() => {
    getLanguages(true).then(setLanguages).catch(() => {});
  }, []);

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteLayout(deleteTarget.id);
      toast.success("Layout deleted successfully");
      setDeleteTarget(null);
      fetchLayouts();
    } catch {
      toast.error("Failed to delete layout");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach(layout: Layout) {
    setDetachingId(layout.id);
    try {
      await detachLayout(layout.id);
      toast.success(`"${layout.name}" detached from ${layout.source}`);
      fetchLayouts();
    } catch {
      toast.error("Failed to detach layout");
    } finally {
      setDetachingId(null);
    }
  }

  const countBy = (s: string) => layouts.filter((t) => t.source === s).length;
  const sourceTabs = [
    { value: "all", label: "All", count: layouts.length },
    { value: "custom", label: "Custom", count: countBy("custom") },
    { value: "theme", label: "Theme", count: countBy("theme") },
    { value: "extension", label: "Extension", count: countBy("extension") },
  ].filter((t) => t.value === "all" || t.count > 0);

  const displayed = source === "all" ? layouts : layouts.filter((t) => t.source === source);

  return (
    <ListPageShell>
      <ListHeader
        title="Layouts"
        tabs={sourceTabs}
        activeTab={source}
        onTabChange={(v) => setSource(v as SourceFilter)}
        newLabel="New Layout"
        newHref="/admin/layouts/new"
      />

      <ListToolbar>
        <Select value={langFilter} onValueChange={setLangFilter}>
          <SelectTrigger className="h-[30px] w-[200px] text-[13px] bg-white border-slate-300 rounded">
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
      </ListToolbar>

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : displayed.length === 0 ? (
          <EmptyState
            icon={LayoutTemplate}
            title={source === "all" ? "No layouts found" : `No ${source} layouts`}
            description={source === "all" ? "Create your first layout to get started" : ""}
          />
        ) : (
          <ListTable>
            <thead>
              <tr>
                <Th>Name</Th>
                <Th width={200}>Slug</Th>
                <Th width={140}>Language</Th>
                <Th width={140}>Source</Th>
                <Th width={110}>Default</Th>
                <Th width={140} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {displayed.map((layout) => {
                const isCustom = layout.source === "custom";
                return (
                  <Tr key={layout.id}>
                    <Td>
                      <TitleCell to={`/admin/layouts/${layout.id}`} title={layout.name} />
                    </Td>
                    <Td className="font-mono text-[12px] text-slate-500">{layout.slug}</Td>
                    <Td className="text-slate-600">
                      {layout.language_id != null
                        ? (languages.find(l => l.id === layout.language_id)?.name || String(layout.language_id))
                        : "All"}
                    </Td>
                    <Td>
                      {layout.source === "theme" ? (
                        <Chip>{layout.theme_name || "Theme"}</Chip>
                      ) : layout.source === "extension" ? (
                        <Chip>Extension</Chip>
                      ) : (
                        <Chip>Custom</Chip>
                      )}
                    </Td>
                    <Td>
                      {layout.is_default ? (
                        <span className="inline-flex items-center gap-1 px-1.5 py-px text-[11px] font-medium text-indigo-700 bg-indigo-50 border border-indigo-200 rounded-[2px]">
                          <Check className="w-2.5 h-2.5" />
                          Default
                        </span>
                      ) : (
                        <span className="text-slate-400 text-[12px]">—</span>
                      )}
                    </Td>
                    <Td align="right" className="whitespace-nowrap">
                      <RowActions
                        editTo={`/admin/layouts/${layout.id}`}
                        onDelete={isCustom ? () => setDeleteTarget(layout) : undefined}
                        disableDelete={!isCustom}
                        deleteTitle={isCustom ? "Delete" : "Built-in, cannot delete"}
                        extra={
                          !isCustom ? (
                            <button
                              type="button"
                              title="Detach from source"
                              onClick={() => handleDetach(layout)}
                              disabled={detachingId === layout.id}
                              className="w-[26px] h-[26px] grid place-items-center text-amber-600 hover:bg-amber-50 hover:border-amber-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent disabled:opacity-40"
                            >
                              <Unplug className="w-3 h-3" />
                            </button>
                          ) : undefined
                        }
                      />
                    </Td>
                  </Tr>
                );
              })}
            </tbody>
          </ListTable>
        )}
      </ListCard>

      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Layout</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.name}&quot;? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)} disabled={deleting}>
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
