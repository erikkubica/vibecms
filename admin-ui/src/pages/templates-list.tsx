import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { LayoutTemplate, Unplug, Plus } from "lucide-react";
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
  getTemplatesPaginated,
  deleteTemplate,
  detachTemplate,
  type Template,
} from "@/api/client";
import {
  ListPageShell,
  ListHeader,
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

type SourceFilter = "all" | "custom" | "theme" | "extension";

export default function TemplatesListPage() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleteTarget, setDeleteTarget] = useState<Template | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [source, setSource] = useState<SourceFilter>("all");
  const [detachingId, setDetachingId] = useState<number | null>(null);

  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getTemplatesPaginated({ page: 1, per_page: 500 });
      setTemplates(res.data);
    } catch {
      toast.error("Failed to load templates");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteTemplate(deleteTarget.id);
      toast.success("Template deleted successfully");
      setDeleteTarget(null);
      fetchTemplates();
    } catch {
      toast.error("Failed to delete template");
    } finally {
      setDeleting(false);
    }
  }

  async function handleDetach(tpl: Template) {
    setDetachingId(tpl.id);
    try {
      await detachTemplate(tpl.id);
      toast.success(`"${tpl.label}" detached from ${tpl.source}`);
      fetchTemplates();
    } catch {
      toast.error("Failed to detach template");
    } finally {
      setDetachingId(null);
    }
  }

  const countBy = (s: string) => templates.filter((t) => t.source === s).length;
  const sourceTabs = [
    { value: "all", label: "All", count: templates.length },
    { value: "custom", label: "Custom", count: countBy("custom") },
    { value: "theme", label: "Theme", count: countBy("theme") },
    { value: "extension", label: "Extension", count: countBy("extension") },
  ].filter((t) => t.value === "all" || t.count > 0);

  const displayed = source === "all" ? templates : templates.filter((t) => t.source === source);

  return (
    <ListPageShell>
      <ListHeader
        title="Templates"
        tabs={sourceTabs}
        activeTab={source}
        onTabChange={(v) => setSource(v as SourceFilter)}
        newLabel="New Template"
        newHref="/admin/templates/new"
      />

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : displayed.length === 0 ? (
          <EmptyState
            icon={LayoutTemplate}
            title={source === "all" ? "No templates found" : `No ${source} templates`}
            description={source === "all" ? "Create your first template to get started" : ""}
            action={
              source === "all" ? (
                <Link
                  to="/admin/templates/new"
                  className="h-[30px] px-3 inline-flex items-center gap-1.5 text-[13px] font-medium text-white bg-indigo-600 rounded hover:bg-indigo-700"
                >
                  <Plus className="w-3.5 h-3.5" />
                  New Template
                </Link>
              ) : undefined
            }
          />
        ) : (
          <ListTable>
            <thead>
              <tr>
                <Th>Label</Th>
                <Th width={100}>Blocks</Th>
                <Th width={140}>Source</Th>
                <Th>Description</Th>
                <Th width={120} align="right">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {displayed.map((tpl) => (
                <Tr key={tpl.id}>
                  <Td>
                    <TitleCell to={`/admin/templates/${tpl.id}/edit`} title={tpl.label} slug={tpl.slug} />
                  </Td>
                  <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                    {tpl.block_config?.length ?? 0}
                  </Td>
                  <Td>
                    {tpl.source === "theme" ? (
                      <Chip>{tpl.theme_name || "Theme"}</Chip>
                    ) : tpl.source === "extension" ? (
                      <Chip>Extension</Chip>
                    ) : (
                      <Chip>Custom</Chip>
                    )}
                  </Td>
                  <Td className="text-slate-500">
                    <span className="block max-w-xs truncate" title={tpl.description || ""}>
                      {tpl.description || "—"}
                    </span>
                  </Td>
                  <Td align="right" className="whitespace-nowrap">
                    <RowActions
                      editTo={`/admin/templates/${tpl.id}/edit`}
                      onDelete={() => setDeleteTarget(tpl)}
                      disableDelete={tpl.source !== "custom"}
                      deleteTitle={tpl.source !== "custom" ? "Only custom templates can be deleted" : "Delete"}
                      extra={
                        tpl.source !== "custom" ? (
                          <button
                            type="button"
                            title="Detach from source"
                            onClick={() => handleDetach(tpl)}
                            disabled={detachingId === tpl.id}
                            className="w-[26px] h-[26px] grid place-items-center text-amber-600 hover:text-amber-700 hover:bg-amber-50 hover:border-amber-200 border border-transparent rounded-[2px] cursor-pointer bg-transparent disabled:opacity-40"
                          >
                            <Unplug className="w-3 h-3" />
                          </button>
                        ) : null
                      }
                    />
                  </Td>
                </Tr>
              ))}
            </tbody>
          </ListTable>
        )}
      </ListCard>

      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Template</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.label}&quot;? This action cannot be undone.
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
