import { useEffect, useState, useCallback } from "react";
import {
  Settings,
  Trash2,
  Plus,
  HardDrive,
  Image as ImageIcon,
  RefreshCw,
  Loader2,
} from "@vibecms/icons";
import {
  Button,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Separator,
  Badge,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@vibecms/ui";
import { toast } from "sonner";

// ---------- Types ----------

interface OptimizerSettings {
  normalize_enabled: boolean;
  normalize_max_dimension: number;
  upload_quality: number;
  jpeg_quality: number;
  webp_enabled: boolean;
  webp_quality: number;
}

interface ImageSize {
  id?: number;
  name: string;
  width: number;
  height: number;
  mode: string;
  source: string;
  quality: number;
  cached_files?: number;
  cache_size?: number;
}

interface CacheStats {
  total_size: number;
  total_files: number;
}

const BASE = "/admin/api/ext/media-manager/optimizer";

// ---------- API helpers ----------

async function fetchSettings(): Promise<OptimizerSettings> {
  const res = await fetch(`${BASE}/settings`, { credentials: "include" });
  if (!res.ok) throw new Error("Failed to fetch optimizer settings");
  const body = await res.json();
  return body.data ?? body;
}

async function saveSettings(s: OptimizerSettings): Promise<void> {
  const res = await fetch(`${BASE}/settings`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(s),
  });
  if (!res.ok) throw new Error("Failed to save settings");
}

async function fetchSizes(): Promise<ImageSize[]> {
  const res = await fetch(`${BASE}/sizes`, { credentials: "include" });
  if (!res.ok) throw new Error("Failed to fetch image sizes");
  const body = await res.json();
  return body.data ?? body ?? [];
}

async function createSize(size: Omit<ImageSize, "id" | "source" | "quality" | "cached_files" | "cache_size">): Promise<ImageSize> {
  const res = await fetch(`${BASE}/sizes`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(size),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.error ?? "Failed to create size");
  }
  const body = await res.json();
  return body.data ?? body;
}

async function deleteSize(name: string): Promise<void> {
  const res = await fetch(`${BASE}/sizes/${encodeURIComponent(name)}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to delete size");
}

async function clearCacheAll(): Promise<CacheStats> {
  const res = await fetch(`${BASE}/cache/clear`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to clear cache");
  const body = await res.json();
  return body.data ?? body;
}

async function clearCacheForSize(name: string): Promise<void> {
  const res = await fetch(`${BASE}/cache/clear/${encodeURIComponent(name)}`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to clear cache for size");
}

// ---------- Helpers ----------

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

// ---------- Component ----------

export default function ImageOptimizerSettings() {
  // Settings state
  const [settings, setSettings] = useState<OptimizerSettings>({
    normalize_enabled: true,
    normalize_max_dimension: 5000,
    upload_quality: 100,
    jpeg_quality: 80,
    webp_enabled: true,
    webp_quality: 75,
  });
  const [settingsLoading, setSettingsLoading] = useState(true);
  const [settingsSaving, setSettingsSaving] = useState(false);

  // Sizes state
  const [sizes, setSizes] = useState<ImageSize[]>([]);
  const [sizesLoading, setSizesLoading] = useState(true);

  // Add size form
  const [showAddForm, setShowAddForm] = useState(false);
  const [newSize, setNewSize] = useState({ name: "", width: 300, height: 300, mode: "fit" });
  const [addingSizeLoading, setAddingSizeLoading] = useState(false);

  // Cache
  const [cacheStats, setCacheStats] = useState<CacheStats>({ total_size: 0, total_files: 0 });
  const [clearingAll, setClearingAll] = useState(false);
  const [clearingSizeName, setClearingSizeName] = useState<string | null>(null);

  // Dialogs
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [deletingName, setDeletingName] = useState<string | null>(null);
  const [clearAllConfirm, setClearAllConfirm] = useState(false);

  // ---------- Load data ----------

  const loadSettings = useCallback(async () => {
    try {
      setSettingsLoading(true);
      const data = await fetchSettings();
      setSettings(data);
    } catch {
      toast.error("Failed to load optimizer settings");
    } finally {
      setSettingsLoading(false);
    }
  }, []);

  const loadSizes = useCallback(async () => {
    try {
      setSizesLoading(true);
      const data = await fetchSizes();
      setSizes(data);
      // Compute cache stats from sizes
      const totalSize = data.reduce((acc, s) => acc + (s.cache_size ?? 0), 0);
      const totalFiles = data.reduce((acc, s) => acc + (s.cached_files ?? 0), 0);
      setCacheStats({ total_size: totalSize, total_files: totalFiles });
    } catch {
      toast.error("Failed to load image sizes");
    } finally {
      setSizesLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSettings();
    loadSizes();
  }, [loadSettings, loadSizes]);

  // ---------- Handlers ----------

  const handleSaveSettings = async () => {
    try {
      setSettingsSaving(true);
      await saveSettings(settings);
      toast.success("Settings saved");
    } catch {
      toast.error("Failed to save settings");
    } finally {
      setSettingsSaving(false);
    }
  };

  const handleAddSize = async () => {
    if (!newSize.name.trim()) {
      toast.error("Size name is required");
      return;
    }
    if (!/^[a-z0-9_-]+$/.test(newSize.name)) {
      toast.error("Size name must be lowercase alphanumeric with hyphens/underscores");
      return;
    }
    if (newSize.width < 1 || newSize.height < 1) {
      toast.error("Width and height must be positive");
      return;
    }
    try {
      setAddingSizeLoading(true);
      await createSize(newSize);
      toast.success(`Size "${newSize.name}" created`);
      setNewSize({ name: "", width: 300, height: 300, mode: "fit" });
      setShowAddForm(false);
      loadSizes();
    } catch (err: any) {
      toast.error(err.message || "Failed to create size");
    } finally {
      setAddingSizeLoading(false);
    }
  };

  const handleDeleteSize = async (name: string) => {
    try {
      setDeletingName(name);
      await deleteSize(name);
      toast.success(`Size "${name}" deleted`);
      setDeleteConfirm(null);
      loadSizes();
    } catch {
      toast.error("Failed to delete size");
    } finally {
      setDeletingName(null);
    }
  };

  const handleClearCacheForSize = async (name: string) => {
    try {
      setClearingSizeName(name);
      await clearCacheForSize(name);
      toast.success(`Cache cleared for "${name}"`);
      loadSizes();
    } catch {
      toast.error("Failed to clear cache");
    } finally {
      setClearingSizeName(null);
    }
  };

  const handleClearAllCache = async () => {
    try {
      setClearingAll(true);
      const stats = await clearCacheAll();
      toast.success(`Cache cleared — freed ${formatBytes(stats.total_size ?? 0)}`);
      setClearAllConfirm(false);
      loadSizes();
    } catch {
      toast.error("Failed to clear cache");
    } finally {
      setClearingAll(false);
    }
  };

  // ---------- Render ----------

  if (settingsLoading && sizesLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-slate-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-indigo-50 text-indigo-600">
          <Settings className="h-5 w-5" />
        </div>
        <div>
          <h2 className="text-xl font-semibold text-slate-900">Image Optimizer</h2>
          <p className="text-sm text-slate-500">
            Configure upload normalization, image sizes, and cache settings
          </p>
        </div>
      </div>

      {/* ==================== Settings Section ==================== */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Upload &amp; Optimization Settings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Upload Normalization Toggle */}
          <div className="flex items-center justify-between">
            <div>
              <Label className="text-sm font-medium">Upload Normalization</Label>
              <p className="text-xs text-slate-500 mt-0.5">
                Automatically downscale, strip metadata, and compress on upload
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={settings.normalize_enabled}
              onClick={() =>
                setSettings((s) => ({ ...s, normalize_enabled: !s.normalize_enabled }))
              }
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                settings.normalize_enabled ? "bg-indigo-600" : "bg-slate-200"
              }`}
            >
              <span
                className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
                  settings.normalize_enabled ? "translate-x-6" : "translate-x-1"
                }`}
              />
            </button>
          </div>

          {/* Max Dimension */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="max-dim" className="text-sm">Max Dimension (px)</Label>
              <Input
                id="max-dim"
                type="number"
                min={100}
                max={20000}
                value={settings.normalize_max_dimension}
                onChange={(e) =>
                  setSettings((s) => ({ ...s, normalize_max_dimension: parseInt(e.target.value) || 5000 }))
                }
              />
              <p className="text-xs text-slate-400">Images exceeding this are downscaled on upload</p>
            </div>

            {/* Upload JPEG Quality */}
            <div className="space-y-1.5">
              <Label htmlFor="upload-quality" className="text-sm">
                Upload Quality: <span className="font-mono text-indigo-600">{settings.upload_quality}</span>
              </Label>
              <input
                id="upload-quality"
                type="range"
                min={50}
                max={100}
                value={settings.upload_quality}
                onChange={(e) =>
                  setSettings((s) => ({ ...s, upload_quality: parseInt(e.target.value) }))
                }
                className="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-indigo-600"
              />
              <p className="text-xs text-slate-400">100 = lossless (metadata strip only). Lower = lossy compression (JPEG + PNG)</p>
            </div>
          </div>

          <Separator />

          {/* Cache JPEG Quality */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="cache-quality" className="text-sm">
                Cache JPEG Quality: <span className="font-mono text-indigo-600">{settings.jpeg_quality}</span>
              </Label>
              <input
                id="cache-quality"
                type="range"
                min={30}
                max={100}
                value={settings.jpeg_quality}
                onChange={(e) =>
                  setSettings((s) => ({ ...s, jpeg_quality: parseInt(e.target.value) }))
                }
                className="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-indigo-600"
              />
              <p className="text-xs text-slate-400">Quality for resized/cached variants</p>
            </div>

            {/* WebP Quality */}
            <div className="space-y-1.5">
              <Label htmlFor="webp-quality" className="text-sm">
                WebP Quality: <span className="font-mono text-indigo-600">{settings.webp_quality}</span>
              </Label>
              <input
                id="webp-quality"
                type="range"
                min={30}
                max={100}
                value={settings.webp_quality}
                onChange={(e) =>
                  setSettings((s) => ({ ...s, webp_quality: parseInt(e.target.value) }))
                }
                className="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-indigo-600"
              />
              <p className="text-xs text-slate-400">Quality for WebP variants served to supported browsers</p>
            </div>
          </div>

          {/* WebP Toggle */}
          <div className="flex items-center justify-between">
            <div>
              <Label className="text-sm font-medium">WebP Auto-Conversion</Label>
              <p className="text-xs text-slate-500 mt-0.5">
                Serve WebP variants to browsers that support it
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={settings.webp_enabled}
              onClick={() =>
                setSettings((s) => ({ ...s, webp_enabled: !s.webp_enabled }))
              }
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                settings.webp_enabled ? "bg-indigo-600" : "bg-slate-200"
              }`}
            >
              <span
                className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
                  settings.webp_enabled ? "translate-x-6" : "translate-x-1"
                }`}
              />
            </button>
          </div>

          <div className="pt-2">
            <Button onClick={handleSaveSettings} disabled={settingsSaving}>
              {settingsSaving && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Save Settings
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* ==================== Registered Sizes ==================== */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">Registered Sizes</CardTitle>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowAddForm(!showAddForm)}
          >
            <Plus className="h-4 w-4 mr-1" />
            Add Size
          </Button>
        </CardHeader>
        <CardContent>
          {/* Add Size Form */}
          {showAddForm && (
            <div className="mb-4 p-4 border border-slate-200 rounded-lg bg-slate-50 space-y-3">
              <div className="grid grid-cols-4 gap-3">
                <div className="space-y-1">
                  <Label className="text-xs">Name</Label>
                  <Input
                    placeholder="e.g. hero"
                    value={newSize.name}
                    onChange={(e) => setNewSize((s) => ({ ...s, name: e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, "") }))}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Width (px)</Label>
                  <Input
                    type="number"
                    min={1}
                    value={newSize.width}
                    onChange={(e) => setNewSize((s) => ({ ...s, width: parseInt(e.target.value) || 0 }))}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Height (px)</Label>
                  <Input
                    type="number"
                    min={1}
                    value={newSize.height}
                    onChange={(e) => setNewSize((s) => ({ ...s, height: parseInt(e.target.value) || 0 }))}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Mode</Label>
                  <Select value={newSize.mode} onValueChange={(v) => setNewSize((s) => ({ ...s, mode: v }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="crop">Crop</SelectItem>
                      <SelectItem value="fit">Fit</SelectItem>
                      <SelectItem value="width">Width</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="flex gap-2">
                <Button size="sm" onClick={handleAddSize} disabled={addingSizeLoading}>
                  {addingSizeLoading && <Loader2 className="h-3 w-3 animate-spin mr-1" />}
                  Create Size
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    setShowAddForm(false);
                    setNewSize({ name: "", width: 300, height: 300, mode: "fit" });
                  }}
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}

          {/* Sizes Table */}
          {sizesLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
            </div>
          ) : sizes.length === 0 ? (
            <div className="text-center py-8 text-slate-400 text-sm">
              <ImageIcon className="h-8 w-8 mx-auto mb-2 opacity-50" />
              No image sizes registered
            </div>
          ) : (
            <div className="border rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-slate-50 border-b text-left">
                    <th className="px-4 py-2.5 font-medium text-slate-600">Name</th>
                    <th className="px-4 py-2.5 font-medium text-slate-600">Dimensions</th>
                    <th className="px-4 py-2.5 font-medium text-slate-600">Mode</th>
                    <th className="px-4 py-2.5 font-medium text-slate-600">Source</th>
                    <th className="px-4 py-2.5 font-medium text-slate-600 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {sizes.map((size) => (
                    <tr key={size.name} className="border-b last:border-b-0 hover:bg-slate-50/50">
                      <td className="px-4 py-2.5">
                        <span className="font-mono text-sm font-medium text-slate-800">{size.name}</span>
                      </td>
                      <td className="px-4 py-2.5 text-slate-600">
                        {size.width}&times;{size.height}
                      </td>
                      <td className="px-4 py-2.5">
                        <Badge variant="secondary" className="text-xs capitalize">
                          {size.mode}
                        </Badge>
                      </td>
                      <td className="px-4 py-2.5 text-slate-500">{size.source}</td>
                      <td className="px-4 py-2.5">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs text-slate-500 hover:text-indigo-600"
                            disabled={clearingSizeName === size.name}
                            onClick={() => handleClearCacheForSize(size.name)}
                          >
                            {clearingSizeName === size.name ? (
                              <Loader2 className="h-3 w-3 animate-spin mr-1" />
                            ) : (
                              <RefreshCw className="h-3 w-3 mr-1" />
                            )}
                            Clear Cache
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs text-red-500 hover:text-red-700 hover:bg-red-50"
                            onClick={() => setDeleteConfirm(size.name)}
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ==================== Cache Management ==================== */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Cache Management</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-slate-100">
                <HardDrive className="h-5 w-5 text-slate-500" />
              </div>
              <div>
                <p className="text-sm font-medium text-slate-800">
                  Total cache size: <span className="font-mono text-indigo-600">{formatBytes(cacheStats.total_size)}</span>
                </p>
                <p className="text-xs text-slate-500">
                  {cacheStats.total_files} cached file{cacheStats.total_files !== 1 ? "s" : ""} across {sizes.length} size{sizes.length !== 1 ? "s" : ""}
                </p>
              </div>
            </div>
            <Button
              variant="outline"
              className="text-red-600 border-red-200 hover:bg-red-50 hover:text-red-700"
              onClick={() => setClearAllConfirm(true)}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Clear All Cache
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* ==================== Delete Size Confirmation ==================== */}
      <Dialog open={deleteConfirm !== null} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Image Size</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the <span className="font-mono font-medium">"{deleteConfirm}"</span> size?
              This will also clear its cached files. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirm(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={deletingName !== null}
              onClick={() => deleteConfirm && handleDeleteSize(deleteConfirm)}
            >
              {deletingName && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ==================== Clear All Cache Confirmation ==================== */}
      <Dialog open={clearAllConfirm} onOpenChange={setClearAllConfirm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Clear All Image Cache</DialogTitle>
            <DialogDescription>
              This will delete all cached image variants ({cacheStats.total_files} files, {formatBytes(cacheStats.total_size)}).
              Images will be regenerated on next request.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setClearAllConfirm(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={clearingAll}
              onClick={handleClearAllCache}
            >
              {clearingAll && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Clear All Cache
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
