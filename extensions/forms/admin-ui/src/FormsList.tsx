import React, { useEffect, useState } from "react";
import {
  Plus,
  Search,
  Edit2,
  Trash2,
  FileText,
  ExternalLink,
} from "@vibecms/icons";

const {
  Button,
  Card,
  CardContent,
  Badge,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Input,
} = (window as any).__VIBECMS_SHARED__.ui;
const { Link, useNavigate } = (window as any).__VIBECMS_SHARED__.ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

interface Form {
  id: number;
  name: string;
  slug: string;
  created_at: string;
}

export default function FormsList() {
  const [forms, setForms] = useState<Form[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const navigate = useNavigate();

  const fetchForms = async () => {
    try {
      const res = await fetch("/admin/api/ext/forms/", {
        credentials: "include",
      });
      const body = await res.json();
      setForms(body.rows || []);
    } catch (err) {
      toast.error("Failed to load forms");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchForms();
  }, []);

  const filteredForms = forms.filter(
    (f) =>
      f.name.toLowerCase().includes(search.toLowerCase()) ||
      f.slug.toLowerCase().includes(search.toLowerCase()),
  );

  const deleteForm = async (id: number) => {
    if (!confirm("Are you sure you want to delete this form?")) return;
    try {
      const res = await fetch(`/admin/api/ext/forms/${id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (res.ok) {
        toast.success("Form deleted");
        fetchForms();
      }
    } catch (err) {
      toast.error("Failed to delete form");
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-slate-900">
            Forms
          </h1>
          <p className="text-slate-500">
            Manage your contact forms and custom data collection points.
          </p>
        </div>
        <Button
          onClick={() => navigate("/admin/ext/forms/new")}
          className="bg-indigo-600 hover:bg-indigo-700"
        >
          <Plus className="mr-2 h-4 w-4" /> Add Form
        </Button>
      </div>

      <Card className="rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <CardContent className="p-0">
          <div className="flex items-center gap-4 p-4 border-b border-slate-100 bg-slate-50/50">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Search forms..."
                className="pl-9 bg-white border-slate-200 focus:border-indigo-500"
                value={search}
                onChange={(e: any) => setSearch(e.target.value)}
              />
            </div>
          </div>

          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent border-slate-100">
                <TableHead className="w-[300px] text-slate-500 font-medium">
                  Name
                </TableHead>
                <TableHead className="text-slate-500 font-medium">
                  Shortcode / Slug
                </TableHead>
                <TableHead className="text-slate-500 font-medium">
                  Created
                </TableHead>
                <TableHead className="text-right text-slate-500 font-medium">
                  Actions
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell
                    colSpan={4}
                    className="h-32 text-center text-slate-400"
                  >
                    Loading forms...
                  </TableCell>
                </TableRow>
              ) : filteredForms.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={4}
                    className="h-32 text-center text-slate-400"
                  >
                    No forms found.
                  </TableCell>
                </TableRow>
              ) : (
                filteredForms.map((form) => (
                  <TableRow
                    key={form.id}
                    className="group border-slate-100 hover:bg-slate-50/50 transition-colors"
                  >
                    <TableCell className="font-medium text-slate-900">
                      <div className="flex items-center gap-3">
                        <div className="h-10 w-10 rounded-lg bg-indigo-50 flex items-center justify-center text-indigo-600 group-hover:bg-indigo-100 transition-colors">
                          <FileText className="h-5 w-5" />
                        </div>
                        <Link
                          to={`/admin/ext/forms/edit/${form.id}`}
                          className="hover:text-indigo-600 hover:underline transition-colors"
                        >
                          {form.name}
                        </Link>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <code className="px-2 py-1 bg-slate-100 rounded text-xs font-mono text-slate-600">
                          [form slug="{form.slug}"]
                        </code>
                      </div>
                    </TableCell>
                    <TableCell className="text-slate-500 text-sm">
                      {new Date(form.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() =>
                            navigate(
                              `/admin/ext/forms/submissions?form_id=${form.id}`,
                            )
                          }
                          className="h-8 w-8 text-slate-400 hover:text-indigo-600"
                          title="View Submissions"
                        >
                          <ExternalLink className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() =>
                            navigate(`/admin/ext/forms/edit/${form.id}`)
                          }
                          className="h-8 w-8 text-slate-400 hover:text-indigo-600"
                          title="Edit Form"
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => deleteForm(form.id)}
                          className="h-8 w-8 text-slate-400 hover:text-red-600"
                          title="Delete Form"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
