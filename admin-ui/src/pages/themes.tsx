import { useEffect, useState, useCallback, useRef } from "react";
import {
  Palette,
  Upload,
  GitBranch,
  RefreshCw,
  Trash2,
  Check,
  Package,
  ExternalLink,
  Loader2,
  Settings,
  FolderOpen,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import FileBrowser from "@/components/file-browser";
import {
  getThemes,
  uploadTheme,
  installThemeFromGit,
  activateTheme,
  deactivateTheme,
  pullTheme,
  deleteTheme,
  updateThemeGitConfig,
  type Theme,
} from "@/api/client";

export default function ThemesPage() {
  const [themes, setThemes] = useState<Theme[]>([]);
  const [loading, setLoading] = useState(true);
  const [installTab, setInstallTab] = useState<"upload" | "git">("upload");

  // Upload state
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Git install state
  const [gitUrl, setGitUrl] = useState("");
  const [gitBranch, setGitBranch] = useState("main");
  const [gitToken, setGitToken] = useState("");
  const [installing, setInstalling] = useState(false);

  // Delete dialog
  const [deleteTarget, setDeleteTarget] = useState<Theme | null>(null);
  const [deleting, setDeleting] = useState(false);

  // Git config dialog
  const [gitConfigTarget, setGitConfigTarget] = useState<Theme | null>(null);
  const [editGitUrl, setEditGitUrl] = useState("");
  const [editGitBranch, setEditGitBranch] = useState("");
  const [editGitToken, setEditGitToken] = useState("");
  const [savingGitConfig, setSavingGitConfig] = useState(false);

  // Pull state
  const [pullingId, setPullingId] = useState<number | null>(null);

  // File browser
  const [browseTarget, setBrowseTarget] = useState<Theme | null>(null);

  // Activate/deactivate state
  const [togglingId, setTogglingId] = useState<number | null>(null);

  const fetchThemes = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getThemes();
      setThemes(data);
    } catch {
      toast.error("Failed to load themes");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchThemes();
  }, [fetchThemes]);

  // Upload handlers
  async function handleUpload(file: File) {
    if (!file.name.endsWith(".zip")) {
      toast.error("Please upload a .zip file");
      return;
    }
    setUploading(true);
    try {
      await uploadTheme(file);
      toast.success("Theme uploaded successfully");
      fetchThemes();
    } catch {
      toast.error("Failed to upload theme");
    } finally {
      setUploading(false);
    }
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) handleUpload(file);
    if (fileInputRef.current) fileInputRef.current.value = "";
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) handleUpload(file);
  }

  // Git install
  async function handleGitInstall(e: React.FormEvent) {
    e.preventDefault();
    if (!gitUrl.trim()) {
      toast.error("Repository URL is required");
      return;
    }
    setInstalling(true);
    try {
      await installThemeFromGit({
        git_url: gitUrl.trim(),
        git_branch: gitBranch.trim() || "main",
        git_token: gitToken.trim() || undefined,
      });
      toast.success("Theme installed from Git");
      setGitUrl("");
      setGitBranch("main");
      setGitToken("");
      fetchThemes();
    } catch {
      toast.error("Failed to install theme from Git");
    } finally {
      setInstalling(false);
    }
  }

  // Activate / Deactivate
  async function handleToggleActive(theme: Theme) {
    setTogglingId(theme.id);
    try {
      if (theme.is_active) {
        await deactivateTheme(theme.id);
        toast.success(`"${theme.name}" deactivated`);
      } else {
        await activateTheme(theme.id);
        toast.success(`"${theme.name}" activated`);
      }
      fetchThemes();
    } catch {
      toast.error("Failed to update theme status");
    } finally {
      setTogglingId(null);
    }
  }

  // Pull
  async function handlePull(theme: Theme) {
    setPullingId(theme.id);
    try {
      await pullTheme(theme.id);
      toast.success(`"${theme.name}" updated from Git`);
      fetchThemes();
    } catch {
      toast.error("Failed to pull theme updates");
    } finally {
      setPullingId(null);
    }
  }

  // Delete
  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteTheme(deleteTarget.id);
      toast.success("Theme deleted");
      setDeleteTarget(null);
      fetchThemes();
    } catch {
      toast.error("Failed to delete theme");
    } finally {
      setDeleting(false);
    }
  }

  // Git config
  function openGitConfig(theme: Theme) {
    setGitConfigTarget(theme);
    setEditGitUrl(theme.git_url || "");
    setEditGitBranch(theme.git_branch || "main");
    setEditGitToken("");
  }

  async function handleSaveGitConfig(e: React.FormEvent) {
    e.preventDefault();
    if (!gitConfigTarget) return;
    setSavingGitConfig(true);
    try {
      await updateThemeGitConfig(gitConfigTarget.id, {
        git_url: editGitUrl.trim() || undefined,
        git_branch: editGitBranch.trim() || undefined,
        git_token: editGitToken.trim() || undefined,
      });
      toast.success("Git config updated");
      setGitConfigTarget(null);
      fetchThemes();
    } catch {
      toast.error("Failed to update Git config");
    } finally {
      setSavingGitConfig(false);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900">Themes</h1>
      </div>

      {/* Install Section */}
      <Card className="rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="border-b border-slate-200 bg-slate-50 px-6 py-3">
          <div className="flex gap-1">
            <button
              className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                installTab === "upload"
                  ? "bg-white text-indigo-700 shadow-sm border border-slate-200"
                  : "text-slate-500 hover:text-slate-700"
              }`}
              onClick={() => setInstallTab("upload")}
            >
              <Upload className="mr-2 inline-block h-4 w-4" />
              Upload ZIP
            </button>
            <button
              className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                installTab === "git"
                  ? "bg-white text-indigo-700 shadow-sm border border-slate-200"
                  : "text-slate-500 hover:text-slate-700"
              }`}
              onClick={() => setInstallTab("git")}
            >
              <GitBranch className="mr-2 inline-block h-4 w-4" />
              Install from Git
            </button>
          </div>
        </div>
        <CardContent className="p-6">
          {installTab === "upload" ? (
            <div
              className={`relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-8 transition-colors ${
                dragOver
                  ? "border-indigo-400 bg-indigo-50"
                  : "border-slate-300 bg-slate-50 hover:border-slate-400"
              }`}
              onDragOver={(e) => {
                e.preventDefault();
                setDragOver(true);
              }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleDrop}
            >
              <Package className="mb-3 h-10 w-10 text-slate-400" />
              <p className="mb-1 text-sm font-medium text-slate-700">
                Drag and drop a theme ZIP file here
              </p>
              <p className="mb-4 text-xs text-slate-500">or click to browse</p>
              <input
                ref={fileInputRef}
                type="file"
                accept=".zip"
                className="hidden"
                onChange={handleFileChange}
              />
              <Button
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading}
                className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
              >
                {uploading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Uploading...
                  </>
                ) : (
                  <>
                    <Upload className="mr-2 h-4 w-4" />
                    Choose File
                  </>
                )}
              </Button>
            </div>
          ) : (
            <form onSubmit={handleGitInstall} className="space-y-4 max-w-xl">
              <div className="space-y-2">
                <Label htmlFor="git-url">Repository URL</Label>
                <Input
                  id="git-url"
                  placeholder="https://github.com/user/theme.git"
                  value={gitUrl}
                  onChange={(e) => setGitUrl(e.target.value)}
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="git-branch">Branch</Label>
                  <Input
                    id="git-branch"
                    placeholder="main"
                    value={gitBranch}
                    onChange={(e) => setGitBranch(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="git-token">Deploy Token (optional)</Label>
                  <Input
                    id="git-token"
                    type="password"
                    placeholder="Token for private repos"
                    value={gitToken}
                    onChange={(e) => setGitToken(e.target.value)}
                  />
                </div>
              </div>
              <Button
                type="submit"
                disabled={installing}
                className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
              >
                {installing ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Installing...
                  </>
                ) : (
                  <>
                    <GitBranch className="mr-2 h-4 w-4" />
                    Install Theme
                  </>
                )}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>

      {/* Theme Grid */}
      {loading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
        </div>
      ) : themes.length === 0 ? (
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardContent className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400">
            <Palette className="h-12 w-12" />
            <p className="text-lg font-medium">No themes installed</p>
            <p className="text-sm">Upload a theme or install from Git to get started</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {themes.map((theme) => (
            <Card
              key={theme.id}
              className={`rounded-xl shadow-sm overflow-hidden transition-all ${
                theme.is_active
                  ? "border-2 border-indigo-500 ring-2 ring-indigo-500/20"
                  : "border border-slate-200 hover:border-slate-300"
              }`}
            >
              {/* Thumbnail area */}
              <div className="relative h-36 bg-gradient-to-br from-slate-100 to-slate-200 flex items-center justify-center">
                {theme.thumbnail ? (
                  <img
                    src={theme.thumbnail}
                    alt={theme.name}
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <Palette className="h-12 w-12 text-slate-300" />
                )}
                {theme.is_active && (
                  <div className="absolute top-3 right-3">
                    <Badge className="bg-emerald-500 text-white hover:bg-emerald-500 border-0 text-xs shadow-sm">
                      <Check className="mr-1 h-3 w-3" />
                      Active
                    </Badge>
                  </div>
                )}
                {!theme.is_active && (
                  <div className="absolute top-3 right-3">
                    <Badge className="bg-slate-400 text-white hover:bg-slate-400 border-0 text-xs">
                      Inactive
                    </Badge>
                  </div>
                )}
              </div>

              <CardContent className="p-4 space-y-3">
                {/* Name & version */}
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <h3 className="font-semibold text-slate-900 truncate">
                      {theme.name}
                    </h3>
                    {theme.author && (
                      <p className="text-xs text-slate-500 mt-0.5">
                        by {theme.author}
                      </p>
                    )}
                  </div>
                  <Badge
                    variant="outline"
                    className="shrink-0 text-xs font-mono"
                  >
                    v{theme.version}
                  </Badge>
                </div>

                {/* Description */}
                {theme.description && (
                  <p className="text-xs text-slate-500 line-clamp-2">
                    {theme.description}
                  </p>
                )}

                {/* Source badge */}
                <div className="flex items-center gap-2">
                  {theme.source === "git" ? (
                    <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100 border-0 text-xs">
                      <GitBranch className="mr-1 h-3 w-3" />
                      Git
                    </Badge>
                  ) : (
                    <Badge className="bg-blue-100 text-blue-700 hover:bg-blue-100 border-0 text-xs">
                      <Upload className="mr-1 h-3 w-3" />
                      Upload
                    </Badge>
                  )}
                  {theme.source === "git" && theme.git_url && (
                    <a
                      href={theme.git_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-slate-400 hover:text-slate-600"
                    >
                      <ExternalLink className="h-3.5 w-3.5" />
                    </a>
                  )}
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2 pt-1 border-t border-slate-100">
                  <Button
                    size="sm"
                    variant={theme.is_active ? "outline" : "default"}
                    className={
                      theme.is_active
                        ? "text-xs"
                        : "text-xs bg-indigo-600 hover:bg-indigo-700 text-white"
                    }
                    disabled={togglingId === theme.id}
                    onClick={() => handleToggleActive(theme)}
                  >
                    {togglingId === theme.id ? (
                      <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                    ) : theme.is_active ? null : (
                      <Check className="mr-1 h-3 w-3" />
                    )}
                    {theme.is_active ? "Deactivate" : "Activate"}
                  </Button>

                  {theme.source === "git" && (
                    <>
                      <Button
                        size="sm"
                        variant="outline"
                        className="text-xs"
                        disabled={pullingId === theme.id}
                        onClick={() => handlePull(theme)}
                      >
                        {pullingId === theme.id ? (
                          <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                        ) : (
                          <RefreshCw className="mr-1 h-3 w-3" />
                        )}
                        Pull
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="text-xs text-slate-500"
                        onClick={() => openGitConfig(theme)}
                      >
                        <Settings className="h-3 w-3" />
                      </Button>
                    </>
                  )}

                  <Button
                    size="sm"
                    variant="outline"
                    className="text-xs"
                    onClick={() => setBrowseTarget(theme)}
                  >
                    <FolderOpen className="mr-1 h-3 w-3" />
                    Files
                  </Button>

                  <div className="flex-1" />

                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-xs text-red-500 hover:text-red-600 hover:bg-red-50"
                    disabled={theme.is_active}
                    onClick={() => setDeleteTarget(theme)}
                    title={
                      theme.is_active
                        ? "Deactivate theme before deleting"
                        : "Delete theme"
                    }
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Delete confirmation dialog */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Theme</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.name}&quot;?
              This will remove all theme files. This action cannot be undone.
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

      {/* Git Config dialog */}
      <Dialog
        open={!!gitConfigTarget}
        onOpenChange={(open) => !open && setGitConfigTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Git Configuration</DialogTitle>
            <DialogDescription>
              Update the Git settings for &quot;{gitConfigTarget?.name}&quot;.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSaveGitConfig} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-git-url">Repository URL</Label>
              <Input
                id="edit-git-url"
                value={editGitUrl}
                onChange={(e) => setEditGitUrl(e.target.value)}
                placeholder="https://github.com/user/theme.git"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-git-branch">Branch</Label>
              <Input
                id="edit-git-branch"
                value={editGitBranch}
                onChange={(e) => setEditGitBranch(e.target.value)}
                placeholder="main"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-git-token">Deploy Token</Label>
              <Input
                id="edit-git-token"
                type="password"
                value={editGitToken}
                onChange={(e) => setEditGitToken(e.target.value)}
                placeholder={
                  gitConfigTarget?.has_git_token
                    ? "Leave blank to keep current token"
                    : "Token for private repos"
                }
              />
              {gitConfigTarget?.has_git_token && (
                <p className="text-xs text-slate-500">
                  A token is already configured. Leave blank to keep it.
                </p>
              )}
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setGitConfigTarget(null)}
                disabled={savingGitConfig}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={savingGitConfig}
                className="bg-indigo-600 hover:bg-indigo-700 text-white"
              >
                {savingGitConfig ? "Saving..." : "Save"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* File browser dialog */}
      {browseTarget && (
        <FileBrowser
          apiBase={`/admin/api/themes/${browseTarget.id}/files`}
          title={browseTarget.name}
          open={!!browseTarget}
          onClose={() => setBrowseTarget(null)}
        />
      )}
    </div>
  );
}
