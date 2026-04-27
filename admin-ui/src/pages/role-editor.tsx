import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Shield, ArrowLeft, Loader2, Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { SectionHeader } from "@/components/ui/section-header";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
  getRole,
  createRole,
  updateRole,
  getNodeTypes,
  getSystemActions,
  type Role,
  type NodeType,
  type SystemAction,
} from "@/api/client";

// Capability keys and labels
const CAPABILITY_KEYS = [
  { key: "admin_access", label: "Admin Access" },
  { key: "manage_users", label: "Manage Users" },
  { key: "manage_roles", label: "Manage Roles" },
  { key: "manage_settings", label: "Manage Settings" },
  { key: "manage_menus", label: "Manage Menus" },
  { key: "manage_layouts", label: "Manage Layouts" },
  { key: "manage_email", label: "Manage Email" },
] as const;

type AccessLevel = "default" | "none" | "read" | "write";
type AccessScope = "all" | "own";

interface NodeAccessEntry {
  access: AccessLevel;
  scope: AccessScope;
}

interface FormCapabilities {
  [key: string]: boolean;
}

interface FormNodeAccess {
  [nodeTypeSlug: string]: NodeAccessEntry;
}

function slugify(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export default function RoleEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditing = !!id;

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [role, setRole] = useState<Role | null>(null);
  const [nodeTypes, setNodeTypes] = useState<NodeType[]>([]);
  const [systemActions, setSystemActions] = useState<SystemAction[]>([]);

  // Form state -- basic info
  const [formName, setFormName] = useState("");
  const [formSlug, setFormSlug] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [autoSlug, setAutoSlug] = useState(true);

  // Form state -- capabilities (boolean flags)
  const [formCaps, setFormCaps] = useState<FormCapabilities>({});

  // Form state -- node access matrix
  const [formNodeAccess, setFormNodeAccess] = useState<FormNodeAccess>({});
  const [formDefaultNodeAccess, setFormDefaultNodeAccess] = useState<NodeAccessEntry>({
    access: "none",
    scope: "all",
  });

  // Form state -- email subscriptions
  const [formEmailSubs, setFormEmailSubs] = useState<Set<string>>(new Set());

  usePageMeta([
    "Roles",
    isEditing ? (formName ? `Edit "${formName}"` : "Edit") : "New Role",
  ]);

  useEffect(() => {
    async function fetchData() {
      try {
        const [nodeTypesData, actionsData] = await Promise.all([
          getNodeTypes(),
          getSystemActions(),
        ]);
        setNodeTypes(nodeTypesData);
        setSystemActions(actionsData);

        if (isEditing) {
          const roleData = await getRole(Number(id));
          setRole(roleData);
          populateForm(roleData);
        }
      } catch {
        toast.error("Failed to load data");
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [id, isEditing]);

  function populateForm(r: Role) {
    setFormName(r.name);
    setFormSlug(r.slug);
    setFormDescription(r.description);
    setAutoSlug(false);

    // Parse capabilities
    const caps: FormCapabilities = {};
    for (const { key } of CAPABILITY_KEYS) {
      caps[key] = !!r.capabilities?.[key];
    }
    setFormCaps(caps);

    // Parse node access
    const nodeAccess: FormNodeAccess = {};
    const nodes = (r.capabilities?.nodes as Record<string, NodeAccessEntry>) || {};
    for (const [slug, entry] of Object.entries(nodes)) {
      nodeAccess[slug] = {
        access: entry.access || "none",
        scope: entry.scope || "all",
      };
    }
    setFormNodeAccess(nodeAccess);

    const defaultAccess = (r.capabilities?.default_node_access as NodeAccessEntry) || {
      access: "none",
      scope: "all",
    };
    setFormDefaultNodeAccess({
      access: defaultAccess.access || "none",
      scope: defaultAccess.scope || "all",
    });

    // Parse email subscriptions
    const subs = (r.capabilities?.email_subscriptions as string[]) || [];
    setFormEmailSubs(new Set(subs));
  }

  function buildCapabilities(): Record<string, unknown> {
    const capabilities: Record<string, unknown> = {};

    // Boolean capabilities
    for (const { key } of CAPABILITY_KEYS) {
      capabilities[key] = !!formCaps[key];
    }

    // Node access
    const nodes: Record<string, NodeAccessEntry> = {};
    for (const [slug, entry] of Object.entries(formNodeAccess)) {
      // Skip "default" — inherits from default_node_access.
      if (entry.access === "default") continue;
      nodes[slug] = { access: entry.access, scope: entry.scope };
    }
    if (Object.keys(nodes).length > 0) {
      capabilities.nodes = nodes;
    }

    // Default node access
    capabilities.default_node_access = {
      access: formDefaultNodeAccess.access,
      scope: formDefaultNodeAccess.scope,
    };

    // Email subscriptions
    if (formEmailSubs.size > 0) {
      capabilities.email_subscriptions = Array.from(formEmailSubs);
    }

    return capabilities;
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!formName.trim()) {
      toast.error("Name is required");
      return;
    }

    const data: Partial<Role> = {
      name: formName.trim(),
      slug: (formSlug.trim() || slugify(formName)).toLowerCase(),
      description: formDescription.trim(),
      capabilities: buildCapabilities(),
    };

    setSaving(true);
    try {
      if (isEditing && role) {
        await updateRole(role.id, data);
        toast.success("Role updated successfully");
      } else {
        await createRole(data);
        toast.success("Role created successfully");
      }
      navigate("/admin/roles");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save role";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  function toggleCap(key: string) {
    setFormCaps((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  function setNodeAccess(slug: string, access: AccessLevel) {
    setFormNodeAccess((prev) => ({
      ...prev,
      [slug]: {
        access,
        scope: prev[slug]?.scope || "all",
      },
    }));
  }

  function setNodeScope(slug: string, scope: AccessScope) {
    setFormNodeAccess((prev) => ({
      ...prev,
      [slug]: {
        access: prev[slug]?.access || "none",
        scope,
      },
    }));
  }

  function toggleEmailSub(actionSlug: string) {
    setFormEmailSubs((prev) => {
      const next = new Set(prev);
      if (next.has(actionSlug)) {
        next.delete(actionSlug);
      } else {
        next.add(actionSlug);
      }
      return next;
    });
  }

  // Group system actions by category
  const actionsByCategory = systemActions.reduce<Record<string, SystemAction[]>>(
    (acc, action) => {
      const cat = action.category || "Other";
      if (!acc[cat]) acc[cat] = [];
      acc[cat].push(action);
      return acc;
    },
    {}
  );

  const isSystem = role?.is_system ?? false;

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with save bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-slate-500 hover:text-slate-700"
            onClick={() => navigate("/admin/roles")}
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <Shield className="h-7 w-7 text-indigo-600" />
          <h1 className="text-2xl font-bold text-slate-900">
            {isEditing ? "Edit Role" : "New Role"}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            className="rounded-lg border-slate-300"
            onClick={() => navigate("/admin/roles")}
            disabled={saving}
          >
            Cancel
          </Button>
          <Button
            className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow-sm"
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="mr-2 h-4 w-4" />
                {isEditing ? "Update Role" : "Create Role"}
              </>
            )}
          </Button>
        </div>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Card 1: Basic Info */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <SectionHeader title="Basic Info" />
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="role-name" className="text-sm font-medium text-slate-700">
                  Name
                </Label>
                <Input
                  id="role-name"
                  placeholder="e.g. Editor"
                  value={formName}
                  onChange={(e) => {
                    setFormName(e.target.value);
                    if (autoSlug) setFormSlug(slugify(e.target.value));
                  }}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="role-slug" className="text-sm font-medium text-slate-700">
                    Slug
                  </Label>
                  {!isEditing && (
                    <button
                      type="button"
                      className="text-xs text-indigo-600 hover:underline"
                      onClick={() => setAutoSlug(!autoSlug)}
                    >
                      {autoSlug ? "Edit" : "Auto"}
                    </button>
                  )}
                </div>
                <Input
                  id="role-slug"
                  placeholder="e.g. editor"
                  value={formSlug}
                  onChange={(e) => {
                    setAutoSlug(false);
                    setFormSlug(e.target.value);
                  }}
                  disabled={(autoSlug && !isEditing) || isSystem}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                {isSystem && (
                  <p className="text-xs text-slate-400">System role slug cannot be changed</p>
                )}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="role-description" className="text-sm font-medium text-slate-700">
                Description
              </Label>
              <Textarea
                id="role-description"
                placeholder="Brief description of this role"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                rows={3}
                className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
              />
            </div>
          </CardContent>
        </Card>

        {/* Card 2: Capabilities */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <SectionHeader title="Capabilities" />
          <CardContent>
            <div className="grid gap-2 sm:grid-cols-2">
              {CAPABILITY_KEYS.map(({ key, label }) => (
                <label
                  key={key}
                  className="flex items-center gap-2 cursor-pointer rounded-lg border border-slate-200 px-3 py-2 hover:bg-slate-50 transition-colors"
                >
                  <input
                    type="checkbox"
                    checked={!!formCaps[key]}
                    onChange={() => toggleCap(key)}
                    className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  <span className="text-sm font-medium text-slate-700">{label}</span>
                </label>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Card 3: Node Access */}
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <SectionHeader title="Node Access" />
          <CardContent className="p-0">
            <div className="rounded-lg overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow className="border-slate-200 hover:bg-transparent bg-slate-50">
                    <TableHead className="text-slate-500 font-medium text-xs">Node Type</TableHead>
                    <TableHead className="text-slate-500 font-medium text-xs">Access</TableHead>
                    <TableHead className="text-slate-500 font-medium text-xs">Scope</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {/* Default row */}
                  <TableRow className="border-slate-100 bg-indigo-50/30">
                    <TableCell className="font-medium text-slate-700 text-sm">
                      Default (other types)
                    </TableCell>
                    <TableCell>
                      <Select
                        value={formDefaultNodeAccess.access}
                        onValueChange={(val) =>
                          setFormDefaultNodeAccess((prev) => ({
                            ...prev,
                            access: val as AccessLevel,
                          }))
                        }
                      >
                        <SelectTrigger className="w-28 h-8 text-xs rounded-md border-slate-300">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">None</SelectItem>
                          <SelectItem value="read">Read</SelectItem>
                          <SelectItem value="write">Write</SelectItem>
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell>
                      <Select
                        value={formDefaultNodeAccess.scope}
                        onValueChange={(val) =>
                          setFormDefaultNodeAccess((prev) => ({
                            ...prev,
                            scope: val as AccessScope,
                          }))
                        }
                        disabled={formDefaultNodeAccess.access === "none"}
                      >
                        <SelectTrigger className="w-24 h-8 text-xs rounded-md border-slate-300">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All</SelectItem>
                          <SelectItem value="own">Own</SelectItem>
                        </SelectContent>
                      </Select>
                    </TableCell>
                  </TableRow>
                  {/* Per-node-type rows */}
                  {nodeTypes.map((nt) => {
                    const entry = formNodeAccess[nt.slug] || { access: "default" as AccessLevel, scope: "all" as AccessScope };
                    return (
                      <TableRow key={nt.slug} className="border-slate-100">
                        <TableCell className="font-medium text-slate-700 text-sm">
                          {nt.label}
                        </TableCell>
                        <TableCell>
                          <Select
                            value={entry.access}
                            onValueChange={(val) => setNodeAccess(nt.slug, val as AccessLevel)}
                          >
                            <SelectTrigger className="w-28 h-8 text-xs rounded-md border-slate-300">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="default">Default</SelectItem>
                              <SelectItem value="none">None</SelectItem>
                              <SelectItem value="read">Read</SelectItem>
                              <SelectItem value="write">Write</SelectItem>
                            </SelectContent>
                          </Select>
                        </TableCell>
                        <TableCell>
                          <Select
                            value={entry.scope}
                            onValueChange={(val) => setNodeScope(nt.slug, val as AccessScope)}
                            disabled={entry.access === "none" || entry.access === "default"}
                          >
                            <SelectTrigger className="w-24 h-8 text-xs rounded-md border-slate-300">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="all">All</SelectItem>
                              <SelectItem value="own">Own</SelectItem>
                            </SelectContent>
                          </Select>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                  {nodeTypes.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3} className="text-center py-4 text-slate-400 text-sm">
                        No node types defined yet.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        {/* Card 4: Email Subscriptions */}
        {systemActions.length > 0 && (
          <Card className="rounded-xl border border-slate-200 shadow-sm">
            <SectionHeader title="Email Subscriptions" />
            <CardContent className="space-y-4">
              <p className="text-xs text-slate-500">
                Users with this role will receive email notifications for checked actions.
              </p>
              {Object.entries(actionsByCategory).map(([category, actions]) => (
                <div key={category} className="space-y-2">
                  <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                    {category}
                  </h4>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {actions.map((action) => (
                      <label
                        key={action.slug}
                        className="flex items-start gap-2 cursor-pointer rounded-lg border border-slate-200 px-3 py-2 hover:bg-slate-50 transition-colors"
                      >
                        <input
                          type="checkbox"
                          checked={formEmailSubs.has(action.slug)}
                          onChange={() => toggleEmailSub(action.slug)}
                          className="h-4 w-4 mt-0.5 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="text-sm font-medium text-slate-700">
                            {action.label}
                          </span>
                          {action.description && (
                            <p className="text-xs text-slate-400">{action.description}</p>
                          )}
                        </div>
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        )}
      </form>
    </div>
  );
}
