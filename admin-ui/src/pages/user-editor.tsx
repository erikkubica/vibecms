import { useEffect, useState, type FormEvent } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, Loader2, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { getRoles, getLanguages, type Role, type Language } from "@/api/client";

// ---------- Local API helpers ----------

interface UserDetail {
  id: number;
  full_name: string;
  email: string;
  role_id: number;
  language_id: number | null;
  role: { id: number; slug: string; name: string; is_system: boolean } | string;
  last_login_at: string;
  created_at: string;
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
  if (res.status === 204) return undefined as T;
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body?.error?.message || "An unexpected error occurred");
  }
  return body as T;
}

async function getUser(id: number): Promise<UserDetail> {
  const res = await apiFetch<{ data: UserDetail }>(`/admin/api/users/${id}`);
  return res.data;
}

async function createUser(data: {
  full_name: string;
  email: string;
  password: string;
  role_id: number;
}): Promise<UserDetail> {
  const res = await apiFetch<{ data: UserDetail }>("/admin/api/users", {
    method: "POST",
    body: JSON.stringify(data),
  });
  return res.data;
}

async function updateUser(
  id: number,
  data: { full_name?: string; email?: string; password?: string; role_id?: number }
): Promise<UserDetail> {
  const res = await apiFetch<{ data: UserDetail }>(`/admin/api/users/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
  return res.data;
}

// ---------- Component ----------

export default function UserEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [roles, setRoles] = useState<Role[]>([]);
  const [languages, setLanguages] = useState<Language[]>([]);

  // Form state
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [roleId, setRoleId] = useState<string>("");
  const [languageId, setLanguageId] = useState<string>("__default__");

  useEffect(() => {
    async function load() {
      try {
        const [rolesData, langsData] = await Promise.all([getRoles(), getLanguages()]);
        setRoles(rolesData);
        setLanguages(langsData);

        if (isEdit) {
          const user = await getUser(Number(id));
          setFullName(user.full_name);
          setEmail(user.email);
          setRoleId(user.role_id ? String(user.role_id) : "");
          setLanguageId(user.language_id ? String(user.language_id) : "__default__");
        } else {
          // Default to first role for new users
          if (rolesData.length > 0) {
            setRoleId(String(rolesData[0].id));
          }
        }
      } catch {
        toast.error("Failed to load data");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id, isEdit]);

  async function handleSave(e: FormEvent) {
    e.preventDefault();

    if (!fullName.trim() || !email.trim()) {
      toast.error("Full name and email are required");
      return;
    }

    if (!isEdit && !password.trim()) {
      toast.error("Password is required for new users");
      return;
    }

    if (!roleId) {
      toast.error("Please select a role");
      return;
    }

    setSaving(true);
    try {
      if (isEdit) {
        const payload: Record<string, unknown> = {
          full_name: fullName.trim(),
          email: email.trim(),
          role_id: Number(roleId),
          language_id: languageId === "__default__" ? null : Number(languageId),
        };
        if (password.trim()) {
          payload.password = password.trim();
        }
        await updateUser(Number(id), payload as Parameters<typeof updateUser>[1]);
        toast.success("User updated successfully");
      } else {
        await createUser({
          full_name: fullName.trim(),
          email: email.trim(),
          password: password.trim(),
          role_id: Number(roleId),
          language_id: languageId === "__default__" ? null : Number(languageId),
        } as any);
        toast.success("User created successfully");
      }
      navigate("/admin/users");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save user";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9"
          onClick={() => navigate("/admin/users")}
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div className="flex items-center gap-3">
          <Users className="h-7 w-7 text-indigo-600" />
          <h1 className="text-2xl font-bold text-slate-900">
            {isEdit ? "Edit User" : "Add User"}
          </h1>
        </div>
      </div>

      {/* Form */}
      <form onSubmit={handleSave}>
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-lg font-semibold text-slate-900">
              {isEdit ? "Update user details" : "Fill in the details to create a new user"}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid gap-6 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="user-fullname" className="text-sm font-medium text-slate-700">
                  Full Name
                </Label>
                <Input
                  id="user-fullname"
                  placeholder="John Doe"
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="user-email" className="text-sm font-medium text-slate-700">
                  Email
                </Label>
                <Input
                  id="user-email"
                  type="email"
                  placeholder="john@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
              </div>
            </div>

            <div className="grid gap-6 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="user-password" className="text-sm font-medium text-slate-700">
                  Password
                </Label>
                <Input
                  id="user-password"
                  type="password"
                  placeholder={isEdit ? "Leave blank to keep current" : "Enter password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required={!isEdit}
                  className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                />
                {isEdit && (
                  <p className="text-xs text-slate-400">
                    Leave blank to keep the current password.
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="user-role" className="text-sm font-medium text-slate-700">
                  Role
                </Label>
                <Select value={roleId} onValueChange={setRoleId}>
                  <SelectTrigger
                    id="user-role"
                    className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  >
                    <SelectValue placeholder="Select a role" />
                  </SelectTrigger>
                  <SelectContent>
                    {roles.map((role) => (
                      <SelectItem key={role.id} value={String(role.id)}>
                        {role.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label className="text-sm font-medium text-slate-700">
                  Preferred Language
                </Label>
                <Select value={languageId} onValueChange={setLanguageId}>
                  <SelectTrigger className="rounded-lg border-slate-300 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20">
                    <SelectValue placeholder="Site default" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__default__">Site default</SelectItem>
                    {languages.map((lang) => (
                      <SelectItem key={lang.id} value={String(lang.id)}>
                        {lang.flag} {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-slate-400">
                  Used for sending emails in the user's preferred language.
                </p>
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center justify-end gap-3 border-t border-slate-200 pt-6">
              <Button
                type="button"
                variant="outline"
                onClick={() => navigate("/admin/users")}
                disabled={saving}
                className="rounded-lg border-slate-300"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                className="bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg"
                disabled={saving}
              >
                {saving ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : isEdit ? (
                  "Update User"
                ) : (
                  "Create User"
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      </form>
    </div>
  );
}
