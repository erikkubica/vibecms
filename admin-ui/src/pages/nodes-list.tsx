import { useEffect, useState, useCallback } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { FileText, Home, Globe, Tag, X, ChevronDown, Plus } from "lucide-react";
import { useAdminLanguage } from "@/hooks/use-admin-language";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { usePageMeta } from "@/components/layout/page-meta";
import {
  getNodes,
  deleteNode,
  getNodeTypes,
  type ContentNode,
  type PaginationMeta,
  type NodeType,
} from "@/api/client";
import {
  ListPageShell,
  ListHeader,
  ListToolbar,
  ListSearch,
  ListCard,
  ListTable,
  ListFooter,
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

interface NodesListProps {
  nodeType: string;
}

export default function NodesListPage({ nodeType }: NodesListProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const basePath = nodeType === "page" ? "/admin/pages" : nodeType === "post" ? "/admin/posts" : `/admin/content/${nodeType}`;

  const { languages, currentCode: globalLangCode } = useAdminLanguage();
  const [langFilter, setLangFilter] = useState<string>("global");

  const [nodes, setNodes] = useState<ContentNode[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(10);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [deleteTarget, setDeleteTarget] = useState<ContentNode | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [searchDebounce, setSearchDebounce] = useState("");
  const [nodeTypeDef, setNodeTypeDef] = useState<NodeType | null>(null);

  const label = nodeTypeDef?.label ?? "";
  const labelPlural = nodeTypeDef?.label_plural ?? "";

  usePageMeta(labelPlural ? [labelPlural] : null);

  const taxQuery: Record<string, string[]> = {};
  const activeTaxFilters: { taxonomy: string; term: string; label: string }[] = [];

  useEffect(() => {
    getNodeTypes().then(types => {
      const def = types.find(t => t.slug === nodeType);
      setNodeTypeDef(def || null);
    });
  }, [nodeType]);

  if (nodeTypeDef?.taxonomies) {
    nodeTypeDef.taxonomies.forEach(tax => {
      const term = searchParams.get(tax.slug);
      if (term) {
        taxQuery[tax.slug] = [term];
        activeTaxFilters.push({ taxonomy: tax.slug, term, label: tax.label });
      }
    });
  }

  const effectiveLangCode = langFilter === "global"
    ? (globalLangCode === "all" ? undefined : globalLangCode)
    : langFilter === "all" ? undefined : langFilter;

  useEffect(() => {
    const timer = setTimeout(() => setSearchDebounce(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const fetchNodes = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getNodes({
        page,
        per_page: perPage,
        node_type: nodeType,
        status: status === "all" ? undefined : status,
        language_code: effectiveLangCode || undefined,
        search: searchDebounce || undefined,
        tax_query: Object.keys(taxQuery).length > 0 ? taxQuery : undefined,
      });
      setNodes(res.data);
      setMeta(res.meta);
    } catch {
      toast.error(`Failed to load ${labelPlural.toLowerCase()}`);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, perPage, nodeType, status, effectiveLangCode, searchDebounce, labelPlural, JSON.stringify(taxQuery)]);

  useEffect(() => {
    setPage(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchDebounce, status, nodeType, effectiveLangCode, JSON.stringify(taxQuery)]);

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

  const removeTaxFilter = (taxonomy: string) => {
    const p = new URLSearchParams(searchParams);
    p.delete(taxonomy);
    setSearchParams(p);
  };

  const statusTabs = [
    { value: "all", label: "All" },
    { value: "published", label: "Published" },
    { value: "draft", label: "Drafts" },
    { value: "archived", label: "Archived" },
  ];

  if (!nodeTypeDef) {
    return (
      <ListPageShell>
        <ListCard>
          <LoadingRow />
        </ListCard>
      </ListPageShell>
    );
  }

  return (
    <ListPageShell>
      <ListHeader
        title={labelPlural}
        count={meta?.total}
        tabs={statusTabs}
        activeTab={status}
        onTabChange={setStatus}
        newLabel={`New ${label}`}
        newHref={`${basePath}/new`}
      />

      {activeTaxFilters.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-2.5">
          {activeTaxFilters.map((f) => (
            <span
              key={f.taxonomy}
              className="inline-flex items-center gap-1.5 px-2 py-0.5 text-[11px] font-medium text-indigo-700 bg-indigo-50 border border-indigo-200 rounded"
            >
              <Tag className="w-3 h-3" />
              {f.label}: <strong>{f.term}</strong>
              <button
                type="button"
                onClick={() => removeTaxFilter(f.taxonomy)}
                className="hover:text-red-500 cursor-pointer bg-transparent border-0"
              >
                <X className="w-3 h-3" />
              </button>
            </span>
          ))}
          <button
            type="button"
            onClick={() => setSearchParams({})}
            className="text-[11px] text-slate-500 hover:text-slate-700 cursor-pointer bg-transparent border-0"
          >
            Clear all
          </button>
        </div>
      )}

      <ListToolbar>
        <ListSearch value={search} onChange={setSearch} placeholder={`Search by title or slug…`} />
        <Select value={langFilter} onValueChange={setLangFilter}>
          <SelectTrigger className="h-[30px] w-[160px] text-[13px] bg-white border-slate-300 rounded">
            <Globe className="mr-1 h-3.5 w-3.5 text-slate-400" />
            <SelectValue placeholder="Language" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="global">
              {globalLangCode === "all" ? "All (global)" : `Global (${globalLangCode})`}
            </SelectItem>
            <SelectItem value="all">All languages</SelectItem>
            {languages.map((lang) => (
              <SelectItem key={lang.code} value={lang.code}>
                {lang.flag} {lang.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </ListToolbar>

      <ListCard>
        {loading ? (
          <LoadingRow />
        ) : nodes.length === 0 ? (
          <EmptyState
            icon={FileText}
            title={`No ${labelPlural.toLowerCase()} found`}
            description={
              searchDebounce || status !== "all"
                ? "Try adjusting your filters"
                : `Create your first ${label.toLowerCase()} to get started`
            }
            action={
              !searchDebounce && status === "all" ? (
                <Link
                  to={`${basePath}/new`}
                  className="h-[30px] px-3 inline-flex items-center gap-1.5 text-[13px] font-medium text-white bg-indigo-600 rounded hover:bg-indigo-700"
                >
                  <Plus className="w-3.5 h-3.5" />
                  New {label}
                </Link>
              ) : undefined
            }
          />
        ) : (
          <>
            <ListTable>
              <thead>
                <tr>
                  <Th>Title</Th>
                  <Th width={120}>Status</Th>
                  <Th width={240}>Taxonomies</Th>
                  <Th width={80}>Lang</Th>
                  <Th width={110}>
                    <span className="inline-flex items-center gap-1 text-slate-900">
                      Updated <ChevronDown className="w-2.5 h-2.5" />
                    </span>
                  </Th>
                  <Th width={110} align="right">Actions</Th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((node) => {
                  const lang = languages.find((l) => l.code === node.language_code);
                  return (
                    <Tr key={node.id}>
                      <Td>
                        <TitleCell
                          to={`${basePath}/${node.id}/edit`}
                          title={node.title}
                          slug={node.slug}
                          extra={
                            node.is_homepage ? (
                              <span className="inline-flex items-center gap-1 px-1.5 py-px text-[10px] font-medium text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-[2px]">
                                <Home className="w-2.5 h-2.5" />
                                Home
                              </span>
                            ) : undefined
                          }
                        />
                      </Td>
                      <Td>
                        <StatusPill status={node.status} />
                      </Td>
                      <Td>
                        {Object.entries(node.taxonomies || {}).length === 0 ? (
                          <span className="text-slate-400 text-[12px]">—</span>
                        ) : (
                          <div className="flex gap-1 flex-wrap">
                            {Object.entries(node.taxonomies || {}).flatMap(([tax, terms]) =>
                              terms.map((term) => <Chip key={`${tax}-${term}`}>{term}</Chip>)
                            )}
                          </div>
                        )}
                      </Td>
                      <Td>
                        {lang ? (
                          <span className="inline-flex items-center gap-1.5 text-[12px] text-slate-700" title={lang.name}>
                            <span>{lang.flag}</span>
                            {lang.code.toUpperCase()}
                          </span>
                        ) : (
                          <span className="font-mono text-[12px] text-slate-400">{node.language_code}</span>
                        )}
                      </Td>
                      <Td className="font-mono text-[12px] text-slate-500 tabular-nums">
                        {new Date(node.updated_at).toLocaleDateString()}
                      </Td>
                      <Td align="right" className="whitespace-nowrap">
                        <RowActions
                          editTo={`${basePath}/${node.id}/edit`}
                          onDelete={() => setDeleteTarget(node)}
                        />
                      </Td>
                    </Tr>
                  );
                })}
              </tbody>
            </ListTable>
            {meta && (
              <ListFooter
                page={meta.page}
                totalPages={meta.total_pages}
                total={meta.total}
                perPage={meta.per_page}
                onPage={setPage}
                onPerPage={(n) => { setPerPage(n); setPage(1); }}
                label={labelPlural.toLowerCase()}
              />
            )}
          </>
        )}
      </ListCard>

      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {label}</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.title}&quot;? This action cannot be undone.
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
