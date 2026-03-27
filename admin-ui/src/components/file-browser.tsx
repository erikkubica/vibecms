import { useCallback, useEffect, useState } from "react";
import {
  File,
  FileCode,
  FileJson,
  FileText,
  Folder,
  FolderOpen,
  ChevronRight,
  ChevronDown,
  Loader2,
  FileQuestion,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size?: number;
  /** provided by API only for files */
  language?: string;
}

interface TreeNode extends FileEntry {
  children?: TreeNode[];
  loaded?: boolean;
  expanded?: boolean;
}

export interface FileBrowserProps {
  apiBase: string;
  title: string;
  open: boolean;
  onClose: () => void;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

async function fetchDir(apiBase: string, path: string): Promise<FileEntry[]> {
  const res = await fetch(
    `${apiBase}?path=${encodeURIComponent(path)}`,
    { credentials: "include" },
  );
  if (!res.ok) throw new Error("Failed to load");
  const body = await res.json();
  return body.data;
}

async function fetchFile(
  apiBase: string,
  path: string,
): Promise<{ content: string; size: number; language: string; binary: boolean; too_large: boolean }> {
  const res = await fetch(
    `${apiBase}/content?path=${encodeURIComponent(path)}`,
    { credentials: "include" },
  );
  if (!res.ok) throw new Error("Failed to load file");
  const body = await res.json();
  return body.data;
}

function iconForFile(name: string) {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  if (["go", "ts", "tsx", "js", "jsx", "tgo", "tengo", "py", "rs", "sh"].includes(ext))
    return <FileCode className="h-4 w-4 shrink-0 text-sky-500" />;
  if (["json", "jsonc"].includes(ext))
    return <FileJson className="h-4 w-4 shrink-0 text-amber-500" />;
  if (["html", "htm", "md", "txt", "toml", "yaml", "yml", "css", "scss"].includes(ext))
    return <FileText className="h-4 w-4 shrink-0 text-emerald-500" />;
  return <File className="h-4 w-4 shrink-0 text-slate-400" />;
}

function langFromName(name: string): string {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  const map: Record<string, string> = {
    go: "Go", ts: "TypeScript", tsx: "TSX", js: "JavaScript", jsx: "JSX",
    json: "JSON", html: "HTML", htm: "HTML", css: "CSS", scss: "SCSS",
    md: "Markdown", txt: "Text", tgo: "Tengo", tengo: "Tengo",
    yaml: "YAML", yml: "YAML", toml: "TOML", py: "Python",
    sh: "Shell", sql: "SQL", xml: "XML", svg: "SVG",
  };
  return map[ext] ?? ext.toUpperCase();
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function sortEntries(entries: FileEntry[]): FileEntry[] {
  return [...entries].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
}

function entriesToNodes(entries: FileEntry[]): TreeNode[] {
  return sortEntries(entries).map((e) => ({
    ...e,
    children: e.is_dir ? [] : undefined,
    loaded: false,
    expanded: false,
  }));
}

/* ------------------------------------------------------------------ */
/*  TreeItem                                                           */
/* ------------------------------------------------------------------ */

function TreeItem({
  node,
  depth,
  selected,
  onToggle,
  onSelect,
}: {
  node: TreeNode;
  depth: number;
  selected: string | null;
  onToggle: (node: TreeNode) => void;
  onSelect: (node: TreeNode) => void;
}) {
  const isSelected = !node.is_dir && selected === node.path;

  return (
    <>
      <button
        className={`flex w-full items-center gap-1.5 py-1 px-2 text-left text-sm transition-colors hover:bg-slate-100 ${
          isSelected ? "bg-indigo-50 text-indigo-700 font-medium" : "text-slate-700"
        }`}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={() => {
          if (node.is_dir) {
            onToggle(node);
          } else {
            onSelect(node);
          }
        }}
      >
        {node.is_dir ? (
          <>
            {node.expanded ? (
              <ChevronDown className="h-3.5 w-3.5 shrink-0 text-slate-400" />
            ) : (
              <ChevronRight className="h-3.5 w-3.5 shrink-0 text-slate-400" />
            )}
            {node.expanded ? (
              <FolderOpen className="h-4 w-4 shrink-0 text-amber-500" />
            ) : (
              <Folder className="h-4 w-4 shrink-0 text-amber-500" />
            )}
          </>
        ) : (
          <>
            <span className="w-3.5 shrink-0" />
            {iconForFile(node.name)}
          </>
        )}
        <span className="truncate">{node.name}</span>
      </button>

      {/* Children */}
      {node.is_dir && node.expanded && (
        <div
          className="overflow-hidden transition-all duration-150"
        >
          {!node.loaded ? (
            <div
              className="flex items-center gap-2 py-1 text-xs text-slate-400"
              style={{ paddingLeft: `${(depth + 1) * 16 + 8}px` }}
            >
              <Loader2 className="h-3 w-3 animate-spin" />
              Loading...
            </div>
          ) : node.children && node.children.length === 0 ? (
            <div
              className="py-1 text-xs text-slate-400 italic"
              style={{ paddingLeft: `${(depth + 1) * 16 + 8}px` }}
            >
              Empty folder
            </div>
          ) : (
            node.children?.map((child) => (
              <TreeItem
                key={child.path}
                node={child}
                depth={depth + 1}
                selected={selected}
                onToggle={onToggle}
                onSelect={onSelect}
              />
            ))
          )}
        </div>
      )}
    </>
  );
}

/* ------------------------------------------------------------------ */
/*  FileBrowser                                                        */
/* ------------------------------------------------------------------ */

export default function FileBrowser({ apiBase, title, open, onClose }: FileBrowserProps) {
  const [tree, setTree] = useState<TreeNode[]>([]);
  const [treeLoading, setTreeLoading] = useState(true);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);

  // File preview state
  const [fileContent, setFileContent] = useState<string | null>(null);
  const [fileMeta, setFileMeta] = useState<{
    path: string;
    size: number;
    language: string;
    binary: boolean;
    too_large: boolean;
  } | null>(null);
  const [fileLoading, setFileLoading] = useState(false);

  // Load root directory
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setTreeLoading(true);
    fetchDir(apiBase, ".")
      .then((entries) => {
        if (!cancelled) {
          setTree(entriesToNodes(entries));
          setTreeLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) setTreeLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [apiBase, open]);

  // Reset state when dialog closes
  useEffect(() => {
    if (!open) {
      setTree([]);
      setSelectedPath(null);
      setFileContent(null);
      setFileMeta(null);
    }
  }, [open]);

  // Deep-update a node inside the tree by path
  const updateNode = useCallback(
    (nodes: TreeNode[], path: string, updater: (n: TreeNode) => TreeNode): TreeNode[] => {
      return nodes.map((n) => {
        if (n.path === path) return updater(n);
        if (n.children && path.startsWith(n.path + "/")) {
          return { ...n, children: updateNode(n.children, path, updater) };
        }
        return n;
      });
    },
    [],
  );

  // Toggle directory expand/collapse
  const handleToggle = useCallback(
    async (node: TreeNode) => {
      if (!node.is_dir) return;

      if (node.expanded) {
        // Collapse
        setTree((prev) => updateNode(prev, node.path, (n) => ({ ...n, expanded: false })));
        return;
      }

      // Expand
      setTree((prev) =>
        updateNode(prev, node.path, (n) => ({ ...n, expanded: true })),
      );

      if (!node.loaded) {
        try {
          const entries = await fetchDir(apiBase, node.path);
          setTree((prev) =>
            updateNode(prev, node.path, (n) => ({
              ...n,
              children: entriesToNodes(entries),
              loaded: true,
            })),
          );
        } catch {
          setTree((prev) =>
            updateNode(prev, node.path, (n) => ({
              ...n,
              children: [],
              loaded: true,
            })),
          );
        }
      }
    },
    [apiBase, updateNode],
  );

  // Select a file for preview
  const handleSelect = useCallback(
    async (node: TreeNode) => {
      if (node.is_dir) return;
      setSelectedPath(node.path);
      setFileLoading(true);
      setFileContent(null);
      setFileMeta(null);
      try {
        const data = await fetchFile(apiBase, node.path);
        setFileMeta({
          path: node.path,
          size: data.size,
          language: data.language || langFromName(node.name),
          binary: data.binary,
          too_large: data.too_large,
        });
        setFileContent(data.content ?? "");
      } catch {
        setFileMeta(null);
        setFileContent(null);
      } finally {
        setFileLoading(false);
      }
    },
    [apiBase],
  );

  const lines = fileContent?.split("\n") ?? [];

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-6xl w-[95vw] h-[85vh] flex flex-col p-0 gap-0 overflow-hidden">
        <DialogHeader className="shrink-0 border-b border-slate-200 px-5 py-3">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Folder className="h-4 w-4 text-amber-500" />
            {title} — Files
          </DialogTitle>
        </DialogHeader>

        <div className="flex flex-1 min-h-0">
          {/* ---- Left panel: file tree ---- */}
          <div className="w-[250px] shrink-0 border-r border-slate-200 bg-white overflow-y-auto">
            {treeLoading ? (
              <div className="flex items-center justify-center h-full">
                <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
              </div>
            ) : tree.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-slate-400 text-sm gap-2">
                <Folder className="h-8 w-8" />
                <span>No files</span>
              </div>
            ) : (
              <div className="py-1">
                {tree.map((node) => (
                  <TreeItem
                    key={node.path}
                    node={node}
                    depth={0}
                    selected={selectedPath}
                    onToggle={handleToggle}
                    onSelect={handleSelect}
                  />
                ))}
              </div>
            )}
          </div>

          {/* ---- Right panel: code preview ---- */}
          <div className="flex-1 flex flex-col min-w-0 bg-slate-900">
            {!selectedPath ? (
              /* Empty state */
              <div className="flex-1 flex flex-col items-center justify-center text-slate-500 gap-3">
                <FileQuestion className="h-12 w-12 text-slate-600" />
                <span className="text-sm">Select a file to preview</span>
              </div>
            ) : fileLoading ? (
              <div className="flex-1 flex items-center justify-center">
                <Loader2 className="h-6 w-6 animate-spin text-slate-500" />
              </div>
            ) : (
              <>
                {/* File info bar */}
                <div className="shrink-0 flex items-center gap-3 px-4 py-2 border-b border-slate-700 bg-slate-800 text-xs">
                  <span className="text-slate-300 font-mono truncate">
                    {fileMeta?.path}
                  </span>
                  <div className="flex-1" />
                  {fileMeta && (
                    <>
                      <Badge className="bg-slate-700 text-slate-300 hover:bg-slate-700 border-0 text-[10px] font-mono">
                        {fileMeta.language}
                      </Badge>
                      <span className="text-slate-500">
                        {formatSize(fileMeta.size)}
                      </span>
                    </>
                  )}
                </div>

                {/* Content area */}
                <div className="flex-1 overflow-auto">
                  {fileMeta?.binary ? (
                    <div className="flex items-center justify-center h-full text-slate-500 text-sm">
                      Binary file — cannot preview
                    </div>
                  ) : fileMeta?.too_large ? (
                    <div className="flex items-center justify-center h-full text-slate-500 text-sm">
                      File too large to preview
                    </div>
                  ) : (
                    <div className="flex min-w-fit">
                      {/* Line numbers gutter */}
                      <div className="shrink-0 sticky left-0 bg-slate-900 border-r border-slate-800 select-none pr-3 pl-3 py-3 text-right font-mono text-xs leading-5 text-slate-600">
                        {lines.map((_, i) => (
                          <div key={i}>{i + 1}</div>
                        ))}
                      </div>
                      {/* Code */}
                      <pre className="flex-1 py-3 pl-4 pr-4 font-mono text-xs leading-5 text-slate-100 whitespace-pre overflow-x-visible">
                        {lines.map((line, i) => (
                          <div key={i} className="hover:bg-slate-800/50">
                            {line || "\n"}
                          </div>
                        ))}
                      </pre>
                    </div>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
