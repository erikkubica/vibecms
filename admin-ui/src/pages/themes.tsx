import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  Palette,
  Upload,
  GitBranch,
  RefreshCw,
  Trash2,
  Check,
  Package,
  Loader2,
  Settings,
  FolderOpen,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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

  // Install dialog
  const [installOpen, setInstallOpen] = useState(false);
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

  // Navigation
  const navigate = useNavigate();

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
      setInstallOpen(false);
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
      setInstallOpen(false);
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
        <Button
          onClick={() => setInstallOpen(true)}
          className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm rounded-lg font-medium"
        >
          <Plus className="mr-2 h-4 w-4" />
          Install
        </Button>
      </div>

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
              className={`group rounded-xl overflow-hidden transition-all duration-200 ${
                theme.is_active
                  ? "border-2 border-indigo-500/70 shadow-md shadow-indigo-500/5"
                  : "border border-slate-200/80 shadow-sm hover:shadow-md hover:border-slate-300"
              }`}
            >
              {/* Preview area — fixed aspect with graceful fallback */}
              <div className="relative bg-slate-100 overflow-hidden">
                <img
                  src={theme.thumbnail || `/admin/api/themes/${theme.id}/preview`}
                  alt={theme.name}
                  className="w-full h-auto block"
                  onError={(e) => { e.currentTarget.style.display = "none"; }}
                />
                {/* Status + source overlays */}
                <div className="absolute top-2.5 left-2.5 flex items-center gap-1.5">
                  {theme.source === "git" ? (
                    <span className="inline-flex items-center gap-1 rounded-full bg-slate-900/50 px-2 py-0.5 text-[10px] font-medium text-white/90 backdrop-blur-sm">
                      <GitBranch className="h-2.5 w-2.5" />
                      Git
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 rounded-full bg-slate-900/50 px-2 py-0.5 text-[10px] font-medium text-white/90 backdrop-blur-sm">
                      <Upload className="h-2.5 w-2.5" />
                      Upload
                    </span>
                  )}
                </div>
                <div className="absolute top-2.5 right-2.5">
                  {theme.is_active ? (
                    <span className="inline-flex items-center gap-1 rounded-full bg-indigo-500 px-2.5 py-1 text-[11px] font-semibold text-white shadow-sm backdrop-blur-sm">
                      <Check className="h-3 w-3" />
                      Active
                    </span>
                  ) : (
                    <span className="inline-flex items-center rounded-full bg-slate-900/50 px-2.5 py-1 text-[11px] font-medium text-white/80 backdrop-blur-sm">
                      Inactive
                    </span>
                  )}
                </div>
              </div>

              {/* Content */}
              <div className="p-4 space-y-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h3 className="font-semibold text-[15px] text-slate-900 truncate leading-tight">{theme.name}</h3>
                    <p className="text-xs text-slate-400 mt-1">
                      {theme.author ? `by ${theme.author}` : theme.slug}
                    </p>
                  </div>
                  <span className="shrink-0 rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-mono font-medium text-slate-500 tracking-wide">
                    {theme.version}
                  </span>
                </div>

                {theme.description && (
                  <p className="text-[13px] text-slate-500 leading-relaxed line-clamp-2">{theme.description}</p>
                )}

                {/* Actions */}
                <div className="flex items-center gap-1.5 pt-2">
                  <Button
                    size="sm"
                    className={`text-xs h-8 rounded-lg flex-1 ${
                      theme.is_active
                        ? "bg-white border border-slate-200 text-slate-600 hover:bg-slate-50 hover:border-slate-300 shadow-none"
                        : "bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm"
                    }`}
                    disabled={togglingId === theme.id}
                    onClick={() => handleToggleActive(theme)}
                  >
                    {togglingId === theme.id ? (
                      <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                    ) : theme.is_active ? null : (
                      <Check className="mr-1.5 h-3 w-3" />
                    )}
                    {theme.is_active ? "Deactivate" : "Activate"}
                  </Button>

                  {theme.source === "git" && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-xs h-8 rounded-lg border-slate-200"
                      disabled={pullingId === theme.id}
                      onClick={() => handlePull(theme)}
                    >
                      {pullingId === theme.id ? (
                        <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                      ) : (
                        <RefreshCw className="mr-1.5 h-3 w-3" />
                      )}
                      Pull
                    </Button>
                  )}

                  <Button
                    size="sm"
                    variant="outline"
                    className="text-xs h-8 rounded-lg border-slate-200"
                    onClick={() => navigate(`/admin/themes/${theme.id}/files`)}
                  >
                    <FolderOpen className="mr-1.5 h-3 w-3" />
                    Files
                  </Button>

                  {theme.source === "git" && (
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-8 w-8 p-0 text-slate-400 hover:text-slate-600 rounded-lg"
                      onClick={() => openGitConfig(theme)}
                      title="Git settings"
                    >
                      <Settings className="h-3.5 w-3.5" />
                    </Button>
                  )}

                  <div className="flex-1" />

                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-8 w-8 p-0 text-slate-400 hover:text-red-500 hover:bg-red-50 rounded-lg"
                    disabled={theme.is_active}
                    onClick={() => setDeleteTarget(theme)}
                    title={theme.is_active ? "Deactivate theme before deleting" : "Delete theme"}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Install dialog */}
      <Dialog open={installOpen} onOpenChange={setInstallOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Install Theme</DialogTitle>
            <DialogDescription>
              Upload a ZIP archive or install from a Git repository.
            </DialogDescription>
          </DialogHeader>
          <div className="flex gap-1 border-b border-slate-200 pb-3">
            <button
              className={`cursor-pointer rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                installTab === "upload"
                  ? "bg-slate-100 text-indigo-700 shadow-sm border border-slate-200"
                  : "text-slate-500 hover:text-slate-700"
              }`}
              onClick={() => setInstallTab("upload")}
            >
              <Upload className="mr-2 inline-block h-4 w-4" />
              Upload ZIP
            </button>
            <button
              className={`cursor-pointer rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                installTab === "git"
                  ? "bg-slate-100 text-indigo-700 shadow-sm border border-slate-200"
                  : "text-slate-500 hover:text-slate-700"
              }`}
              onClick={() => setInstallTab("git")}
            >
              <GitBranch className="mr-2 inline-block h-4 w-4" />
              From Git
            </button>
          </div>
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
            <form onSubmit={handleGitInstall} className="space-y-4">
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
        </DialogContent>
      </Dialog>

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

    </div>
  );
}
